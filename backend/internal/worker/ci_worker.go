package worker

import (
	"bufio"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/queue"
	"github.com/open-git/backend/internal/infrastructure/workflow"
	"github.com/open-git/backend/internal/middleware"
)

// int64CompatibleUUID returns a UUID whose upper 64 bits are zero, so it
// survives the int64<->UUID bridge the Actions API uses to expose numeric job
// ids. Mirrors the repository helper of the same purpose.
func int64CompatibleUUID() uuid.UUID {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return uuid.New()
	}
	id := int64(binary.BigEndian.Uint64(buf[:]) & 0x7fffffffffffffff)
	if id == 0 {
		id = 1
	}
	return middleware.Int64ToUUID(id)
}

const (
	TypeCIRun = "ci:run"

	ciStepTimeout = 10 * time.Minute

	// maxParallelJobsPerRun bounds how many jobs within a single run execute
	// steps concurrently. Independent jobs run in parallel up to this cap.
	maxParallelJobsPerRun = 8

	planTierFree = "free"
	planTierPro  = "pro"

	freeTierConcurrentLimit = 1
	proTierConcurrentLimit  = 20

	ciStatusQueued     = "queued"
	ciStatusInProgress = "in_progress"
	ciStatusCompleted  = "completed"
	ciStatusFailed     = "failed"

	ciConclusionSuccess        = "success"
	ciConclusionFailure        = "failure"
	ciConclusionRateLimited    = "rate_limited"
	ciConclusionInvalidPayload = "invalid_payload"

	logMask = "***"

	jobLogStatusSuccess = "success"
	jobLogStatusFailure = "failure"

	jobResultSuccess = "success"
	jobResultFailure = "failure"
	jobResultSkipped = "skipped"
)

var (
	ErrConcurrentLimitExceeded = errors.New("ci concurrent run limit exceeded")
	ErrSkipPathCreateFailure   = errors.New("skip path: failed to register skipped job")
)

type CIRunPayload struct {
	WorkflowRunID  string `json:"workflow_run_id"`
	RepositoryID   string `json:"repository_id"`
	OrganizationID string `json:"organization_id"`
	WorkflowYAML   []byte `json:"workflow_yaml"`

	// Run metadata used to build the `github` expression context. Older queued
	// payloads may omit these; the worker treats missing fields as empty.
	HeadSHA    string `json:"head_sha,omitempty"`
	HeadBranch string `json:"head_branch,omitempty"`
	Event      string `json:"event,omitempty"`
	Actor      string `json:"actor,omitempty"`
	Workflow   string `json:"workflow,omitempty"`
	RunNumber  int    `json:"run_number,omitempty"`
	Repository string `json:"repository,omitempty"`

	// RepoGitPath is the on-disk bare repository, used by the built-in
	// actions/checkout to populate a job's workspace. Empty disables checkout.
	RepoGitPath string `json:"repo_git_path,omitempty"`
}

// SecretDecrypter decrypts a stored secret value. Replace with a real KMS-backed
// implementation in production; the default identity decrypter is suitable for
// tests where secrets are stored in plaintext for assertion purposes.
type SecretDecrypter func(ctx context.Context, encrypted string) (string, error)

// CommandRunner executes a shell command in workdir and returns the combined
// stdout/stderr. Exposed for testability.
type CommandRunner func(ctx context.Context, workdir string, env []string, script string) ([]byte, error)

// StreamingCommandRunner executes a shell command in workdir and invokes sink
// for each output line.
type StreamingCommandRunner func(ctx context.Context, workdir string, env []string, script string, step int, sink func(stream, line string)) error

// CheckoutFunc populates dest with the repository at gitPath checked out at ref.
// Implements the built-in actions/checkout; injectable for tests.
type CheckoutFunc func(ctx context.Context, gitPath, ref, dest string) error

// ContainerActionFunc runs a `uses: docker://<image>` container action with the
// workspace mounted and the given env (INPUT_* action inputs plus step env),
// returning its combined output. Injectable for tests.
type ContainerActionFunc func(ctx context.Context, image, workdir string, env []string) ([]byte, error)

type CIWorker struct {
	db              *sql.DB
	decrypt         SecretDecrypter
	runStep         CommandRunner
	runStepStream   StreamingCommandRunner
	checkout        CheckoutFunc
	containerAction ContainerActionFunc
	stepWait        time.Duration
	logRepo         domainrepo.IJobLogRepository
	jobRepo         domainrepo.IWorkflowJobRepository
	logPublisher    *queue.JobLogPublisher
	sandbox         sandbox
}

func NewCIWorker(db *sql.DB) *CIWorker {
	w := &CIWorker{
		db:       db,
		decrypt:  identityDecrypter,
		stepWait: ciStepTimeout,
		sandbox:  newSandbox(SandboxModeNone, ""),
	}
	w.runStep = w.defaultCommandRunner
	w.runStepStream = w.defaultStreamingCommandRunner
	w.checkout = defaultCheckout
	w.containerAction = defaultContainerAction
	return w
}

// WithCheckout overrides the actions/checkout implementation (used in tests).
func (w *CIWorker) WithCheckout(fn CheckoutFunc) *CIWorker {
	w.checkout = fn
	return w
}

// WithContainerAction overrides the docker:// action runner (used in tests).
func (w *CIWorker) WithContainerAction(fn ContainerActionFunc) *CIWorker {
	w.containerAction = fn
	return w
}

// defaultContainerAction runs a container action image with the workspace bind
// mounted, isolated from the network, passing INPUT_*/step env via -e.
func defaultContainerAction(ctx context.Context, image, workdir string, env []string) ([]byte, error) {
	args := []string{"run", "--rm", "--network", "none", "-v", workdir + ":/github/workspace", "-w", "/github/workspace"}
	for _, e := range env {
		args = append(args, "-e", e)
	}
	args = append(args, image)
	return exec.CommandContext(ctx, "docker", args...).CombinedOutput()
}

// actionInputEnv converts a step's `with:` inputs into GitHub-style INPUT_*
// environment variables (uppercased, non-alphanumerics to underscores),
// interpolating any ${{ }} expressions with the step context.
func actionInputEnv(with map[string]string, evalCtx *workflow.EvalContext) []string {
	env := make([]string, 0, len(with))
	for k, v := range with {
		if strings.Contains(v, "${{") {
			if iv, err := workflow.InterpolateString(v, evalCtx); err == nil {
				v = iv
			}
		}
		env = append(env, "INPUT_"+inputKeyToEnv(k)+"="+v)
	}
	sort.Strings(env)
	return env
}

func inputKeyToEnv(k string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(k) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

// defaultCheckout clones the bare repository into dest and checks out ref using
// the git binary.
func defaultCheckout(ctx context.Context, gitPath, ref, dest string) error {
	if gitPath == "" {
		return fmt.Errorf("no repository path configured for checkout")
	}
	if out, err := exec.CommandContext(ctx, "git", "clone", "--quiet", gitPath, dest).CombinedOutput(); err != nil {
		return fmt.Errorf("git clone: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if ref != "" {
		if out, err := exec.CommandContext(ctx, "git", "-C", dest, "checkout", "--quiet", ref).CombinedOutput(); err != nil {
			return fmt.Errorf("git checkout %s: %w: %s", ref, err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

// isCheckoutAction reports whether a `uses:` reference is actions/checkout
// (any version).
func isCheckoutAction(uses string) bool {
	ref := uses
	if at := strings.LastIndex(ref, "@"); at >= 0 {
		ref = ref[:at]
	}
	return strings.EqualFold(strings.TrimSpace(ref), "actions/checkout")
}

// WithSandbox selects how steps are isolated: SandboxModeNone (direct on host,
// trusted instances) or SandboxModeDocker (ephemeral container per job).
func (w *CIWorker) WithSandbox(mode, image string) *CIWorker {
	w.sandbox = newSandbox(mode, image)
	return w
}

func (w *CIWorker) WithDecrypter(d SecretDecrypter) *CIWorker {
	w.decrypt = d
	return w
}

func (w *CIWorker) WithCommandRunner(r CommandRunner) *CIWorker {
	w.runStep = r
	return w
}

func (w *CIWorker) WithLogRepository(r domainrepo.IJobLogRepository) *CIWorker {
	w.logRepo = r
	return w
}

func (w *CIWorker) WithJobRepository(r domainrepo.IWorkflowJobRepository) *CIWorker {
	w.jobRepo = r
	return w
}

func (w *CIWorker) WithLogPublisher(p *queue.JobLogPublisher) *CIWorker {
	w.logPublisher = p
	return w
}

func (w *CIWorker) WithStreamingCommandRunner(r StreamingCommandRunner) *CIWorker {
	w.runStepStream = r
	return w
}

func (w *CIWorker) HandleCIRun(ctx context.Context, task *asynq.Task) error {
	var payload CIRunPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal ci payload: %w: %w", err, asynq.SkipRetry)
	}
	if payload.WorkflowRunID == "" || payload.RepositoryID == "" || payload.OrganizationID == "" {
		w.markRun(ctx, payload.WorkflowRunID, ciStatusFailed, ciConclusionInvalidPayload, "")
		return fmt.Errorf("ci payload missing required identifiers: %w", asynq.SkipRetry)
	}

	tier, err := w.loadPlanTier(ctx, payload.OrganizationID)
	if err != nil {
		return fmt.Errorf("load plan tier: %w", err)
	}
	running, err := w.countRunning(ctx, payload.OrganizationID, payload.WorkflowRunID)
	if err != nil {
		return fmt.Errorf("count running ci jobs: %w", err)
	}
	if exceeded := concurrentLimitExceeded(tier, running); exceeded {
		w.markRun(ctx, payload.WorkflowRunID, ciStatusFailed, ciConclusionRateLimited, "")
		return fmt.Errorf("plan tier %q: %w (running=%d)", tier, ErrConcurrentLimitExceeded, running)
	}

	if err := w.setStatus(ctx, payload.WorkflowRunID, ciStatusInProgress); err != nil {
		return fmt.Errorf("set status in_progress: %w", err)
	}

	ir, diags, parseErr := workflow.ParseWorkflowFull(payload.WorkflowYAML)
	if parseErr != nil {
		w.markRun(ctx, payload.WorkflowRunID, ciStatusFailed, ciConclusionFailure, parseErr.Error())
		return fmt.Errorf("parse workflow: %w: %w", parseErr, asynq.SkipRetry)
	}
	for _, d := range diags {
		if d.Severity == "error" {
			w.markRun(ctx, payload.WorkflowRunID, ciStatusFailed, ciConclusionFailure, d.Message)
			return fmt.Errorf("invalid workflow: %s: %w", d.Message, asynq.SkipRetry)
		}
	}
	if ir == nil || len(ir.Jobs) == 0 {
		w.markRun(ctx, payload.WorkflowRunID, ciStatusFailed, ciConclusionFailure, "workflow has no jobs")
		return fmt.Errorf("workflow has no jobs: %w", asynq.SkipRetry)
	}

	secretEnv, secretValues, err := w.loadSecrets(ctx, payload.RepositoryID)
	if err != nil {
		return fmt.Errorf("load secrets: %w", err)
	}

	// Ephemeral working directory for this run: steps execute here (or with it
	// bind-mounted, in docker mode), never in the server's own working
	// directory, and it is removed when the run finishes.
	workdir, err := os.MkdirTemp("", "og-ci-run-*")
	if err != nil {
		return fmt.Errorf("create ci workdir: %w", err)
	}
	defer os.RemoveAll(workdir)

	useStreaming := w.logRepo != nil
	streamRunner := w.runStepStream
	if useStreaming && streamRunner == nil {
		streamRunner = w.defaultStreamingCommandRunner
	}

	// Identifiers are parsed once; a malformed id fails the whole run.
	var runUUID, repoUUID, orgUUID uuid.UUID
	if w.jobRepo != nil {
		var perr error
		if runUUID, perr = uuid.Parse(payload.WorkflowRunID); perr != nil {
			w.markRun(ctx, payload.WorkflowRunID, ciStatusFailed, ciConclusionFailure, perr.Error())
			return fmt.Errorf("parse workflow run id: %w: %w", perr, asynq.SkipRetry)
		}
		if repoUUID, perr = uuid.Parse(payload.RepositoryID); perr != nil {
			w.markRun(ctx, payload.WorkflowRunID, ciStatusFailed, ciConclusionFailure, perr.Error())
			return fmt.Errorf("parse repository id: %w: %w", perr, asynq.SkipRetry)
		}
		if orgUUID, perr = uuid.Parse(payload.OrganizationID); perr != nil {
			w.markRun(ctx, payload.WorkflowRunID, ciStatusFailed, ciConclusionFailure, perr.Error())
			return fmt.Errorf("parse organization id: %w: %w", perr, asynq.SkipRetry)
		}
	}

	logBuf := &strings.Builder{}
	conclusion := ciConclusionSuccess
	jobLogStatus := make(map[string]string)
	jobLineCounts := make(map[string]int64)
	jobIDs := make(map[string]string)
	jobResults := make(map[string]string) // jobName -> success | failure | skipped

	// Static expression contexts, shared across all steps of this run.
	githubCtx := map[string]string{
		"sha":        payload.HeadSHA,
		"ref":        refFromBranch(payload.HeadBranch),
		"ref_name":   payload.HeadBranch,
		"event_name": payload.Event,
		"actor":      payload.Actor,
		"workflow":   payload.Workflow,
		"run_number": strconv.Itoa(payload.RunNumber),
		"repository": payload.Repository,
	}
	runnerCtx := map[string]string{"os": "Linux", "arch": "X64"}
	secretsCtx := parseEnvToMap(secretEnv)

	// Run jobs in dependency (topological) order so `needs` is honored. On a
	// broken DAG (already reported as a diagnostic error above) Order may be
	// short; fall back to a stable order defensively.
	order := ir.DAG.Order
	if len(order) < len(ir.Jobs) {
		order = sortedJobNames(ir.Jobs)
	}

	// skipJob records a job as skipped (status completed / conclusion skipped)
	// and notes the reason. Used for both unmet `needs` and a false job-level
	// `if:`.
	skipJob := func(jobName, reason string) error {
		jobResults[jobName] = jobResultSkipped
		jobLogStatus[jobName] = jobResultSkipped
		fmt.Fprintf(logBuf, "[job=%s] skipped: %s\n", jobName, reason)
		if w.jobRepo != nil {
			skipUUID := int64CompatibleUUID()
			jobIDs[jobName] = skipUUID.String()
			now := time.Now().UTC()
			if createErr := w.jobRepo.Create(ctx, &entity.WorkflowJob{
				ID: skipUUID, WorkflowRunID: &runUUID, OrganizationID: orgUUID, RepositoryID: repoUUID,
				Name: jobName, Status: entity.WorkflowJobStatusInProgress, StartedAt: &now, CreatedAt: now,
			}); createErr != nil {
				return fmt.Errorf("create workflow job: %w", createErr)
			}
			if completeErr := w.jobRepo.Complete(ctx, skipUUID, entity.WorkflowJobConclusionSkipped, time.Now().UTC()); completeErr != nil {
				return fmt.Errorf("complete workflow job: %w", completeErr)
			}
		}
		return nil
	}

	// Independent jobs run concurrently; a job starts only once every job in its
	// `needs` has finished. Shared run state (the maps, the aggregate log
	// buffer, the overall conclusion) is guarded by mu; the actual step
	// execution happens outside the lock so jobs run in parallel. A bounded
	// semaphore caps how many jobs execute at once.
	var mu sync.Mutex
	var fatalErr error
	var fatalOnce sync.Once
	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()
	setFatal := func(err error) {
		fatalOnce.Do(func() {
			fatalErr = err
			cancelRun()
		})
	}

	done := make(map[string]chan struct{}, len(order))
	for _, name := range order {
		done[name] = make(chan struct{})
	}
	sem := make(chan struct{}, maxParallelJobsPerRun)
	var wg sync.WaitGroup

	for _, jobName := range order {
		wg.Add(1)
		go func(jobName string) {
			defer wg.Done()
			defer close(done[jobName])
			job := ir.Jobs[jobName]

			// Wait for every dependency to finish before starting.
			for _, need := range job.Needs {
				if ch, ok := done[need]; ok {
					select {
					case <-ch:
					case <-runCtx.Done():
						return
					}
				}
			}
			if runCtx.Err() != nil {
				return
			}

			// Skip if a dependency did not succeed (GitHub semantics).
			mu.Lock()
			dep, unmet := firstUnsatisfiedNeed(job.Needs, jobResults)
			mu.Unlock()
			if unmet {
				mu.Lock()
				err := skipJob(jobName, fmt.Sprintf("dependency %q did not succeed", dep))
				mu.Unlock()
				if err != nil {
					setFatal(err)
				}
				return
			}

			// Skip on a false job-level `if:` (evaluated against github + env).
			if job.If != "" {
				jobEnv := mergeStringMaps(ir.Env, job.Env)
				ec := &workflow.EvalContext{Contexts: map[string]map[string]string{
					"github": githubCtx, "runner": runnerCtx, "secrets": secretsCtx, "env": jobEnv, "matrix": {},
				}}
				run, ifErr := workflow.EvaluateCondition(job.If, ec)
				reason := ""
				if ifErr != nil {
					reason = "invalid job if: " + ifErr.Error()
				} else if !run {
					reason = fmt.Sprintf("job if condition %q evaluated false", job.If)
				}
				if reason != "" {
					mu.Lock()
					err := skipJob(jobName, reason)
					mu.Unlock()
					if err != nil {
						setFatal(err)
					}
					return
				}
			}

			// Bound the number of jobs executing steps at once.
			select {
			case sem <- struct{}{}:
			case <-runCtx.Done():
				return
			}
			defer func() { <-sem }()

			// Expand the matrix into one instance per combination (run
			// sequentially within the job); a job without a matrix runs once.
			// The logical job succeeds only if every instance succeeds.
			combos := job.MatrixExpansion
			if len(combos) == 0 {
				combos = []map[string]any{nil}
			}
			logicalFailed := false

			for _, combo := range combos {
				if runCtx.Err() != nil {
					return
				}
				instanceName := jobName
				matrixCtx := map[string]string{}
				if combo != nil {
					matrixCtx = stringifyMatrixCombo(combo)
					instanceName = fmt.Sprintf("%s (%s)", jobName, matrixLabel(combo))
				}

				var jobUUID uuid.UUID
				jobIDStr := instanceName
				if w.jobRepo != nil {
					jobUUID = int64CompatibleUUID()
					jobIDStr = jobUUID.String()
					now := time.Now().UTC()
					if createErr := w.jobRepo.Create(runCtx, &entity.WorkflowJob{
						ID: jobUUID, WorkflowRunID: &runUUID, OrganizationID: orgUUID, RepositoryID: repoUUID,
						Name: instanceName, Status: entity.WorkflowJobStatusInProgress, StartedAt: &now, CreatedAt: now,
					}); createErr != nil {
						setFatal(fmt.Errorf("create workflow job: %w", createErr))
						return
					}
				}

				acc := &jobAccumulator{}
				instanceFailed := w.runJobSteps(runCtx, runJobSpec{
					payload:      payload,
					wfEnv:        ir.Env,
					job:          job,
					instanceName: instanceName,
					jobIDStr:     jobIDStr,
					matrixCtx:    matrixCtx,
					githubCtx:    githubCtx,
					runnerCtx:    runnerCtx,
					secretsCtx:   secretsCtx,
					secretEnv:    secretEnv,
					secretValues: secretValues,
					workdir:      workdir,
					useStreaming: useStreaming,
					streamRunner: streamRunner,
					acc:          acc,
				})

				if w.jobRepo != nil && jobUUID != uuid.Nil {
					jobConclusion := entity.WorkflowJobConclusionSuccess
					if instanceFailed {
						jobConclusion = entity.WorkflowJobConclusionFailure
					}
					_ = w.jobRepo.Complete(runCtx, jobUUID, jobConclusion, time.Now().UTC())
				}

				status := jobLogStatusSuccess
				if instanceFailed {
					status = jobLogStatusFailure
					logicalFailed = true
				}
				mu.Lock()
				jobIDs[instanceName] = jobIDStr
				jobLogStatus[instanceName] = status
				jobLineCounts[instanceName] = acc.lineCount
				logBuf.WriteString(acc.buf.String())
				mu.Unlock()
			}

			mu.Lock()
			if logicalFailed {
				jobResults[jobName] = jobResultFailure
				conclusion = ciConclusionFailure
			} else {
				jobResults[jobName] = jobResultSuccess
			}
			mu.Unlock()
		}(jobName)
	}

	wg.Wait()
	if fatalErr != nil {
		return fatalErr
	}

	finalStatus := ciStatusCompleted
	if conclusion != ciConclusionSuccess {
		finalStatus = ciStatusFailed
	}

	if w.logRepo != nil {
		for jobName, status := range jobLogStatus {
			jobIDStr := jobIDs[jobName]
			if jobIDStr == "" {
				jobIDStr = jobName
			}
			meta := &domainrepo.JobLogMeta{
				JobID:          jobIDStr,
				OrganizationID: payload.OrganizationID,
				Status:         status,
				TotalLines:     jobLineCounts[jobName],
			}
			_ = w.logRepo.SetMeta(ctx, meta)
		}
	}

	w.markRun(ctx, payload.WorkflowRunID, finalStatus, conclusion, logBuf.String())
	return nil
}

// jobAccumulator holds a single job instance's private log buffer and line
// counter. Each instance owns its own accumulator so instances can run
// concurrently without sharing mutable state.
type jobAccumulator struct {
	buf       strings.Builder
	lineCount int64
}

// runJobSpec bundles the per-run state a single job instance needs to execute.
type runJobSpec struct {
	payload      CIRunPayload
	wfEnv        map[string]string
	job          workflow.IRJob
	instanceName string
	jobIDStr     string
	matrixCtx    map[string]string
	githubCtx    map[string]string
	runnerCtx    map[string]string
	secretsCtx   map[string]string
	secretEnv    []string
	secretValues []string
	workdir      string
	useStreaming bool
	streamRunner StreamingCommandRunner
	acc          *jobAccumulator
}

// runJobSteps executes one job instance's steps, evaluating expressions against
// the supplied contexts (including the matrix combination), and returns whether
// the instance failed.
func (w *CIWorker) runJobSteps(ctx context.Context, s runJobSpec) bool {
	jobFailed := false

	// Each job instance gets its own workspace under the run's temp dir, so
	// actions/checkout populates a clean tree and parallel matrix instances
	// don't clobber each other.
	instWorkdir := s.workdir
	if dir, err := os.MkdirTemp(s.workdir, "job-*"); err == nil {
		instWorkdir = dir
	}

	for i, step := range s.job.Steps {
		// Merge env (workflow < job < step) and expose it, with the
		// github/runner/secrets/matrix contexts, for expression evaluation.
		mergedEnv := mergeStringMaps(s.wfEnv, s.job.Env, step.Env)
		evalCtx := &workflow.EvalContext{
			Contexts: map[string]map[string]string{
				"github":  s.githubCtx,
				"runner":  s.runnerCtx,
				"secrets": s.secretsCtx,
				"env":     mergedEnv,
				"matrix":  s.matrixCtx,
			},
			Failed: jobFailed,
		}
		for k, v := range mergedEnv {
			if strings.Contains(v, "${{") {
				if iv, ierr := workflow.InterpolateString(v, evalCtx); ierr == nil {
					mergedEnv[k] = iv
				}
			}
		}

		// Decide whether the step runs. An explicit `if:` is evaluated (with
		// the current failure state, so always()/failure() work); otherwise a
		// step is skipped once the job has failed.
		if step.If != "" {
			run, ifErr := workflow.EvaluateCondition(step.If, evalCtx)
			if ifErr != nil {
				jobFailed = true
				fmt.Fprintf(&s.acc.buf, "[job=%s step=%d] if error: %s\n", s.instanceName, i, ifErr.Error())
				continue
			}
			if !run {
				continue
			}
		} else if jobFailed {
			continue
		}

		stepEnv := buildStepEnv([]map[string]string{mergedEnv}, s.secretEnv)

		if step.Uses != "" {
			if w.runUsesStep(ctx, s, step, i, instWorkdir, evalCtx, stepEnv) {
				jobFailed = true
			}
			continue
		}
		if step.Run == "" {
			continue
		}

		script, _ := workflow.InterpolateString(step.Run, evalCtx)
		stepCtx, cancel := context.WithTimeout(ctx, w.stepWait)

		if s.useStreaming {
			runErr := s.streamRunner(stepCtx, instWorkdir, stepEnv, script, i, func(stream, line string) {
				s.acc.lineCount++
				lineNum := s.acc.lineCount
				masked := maskSecrets(line, s.secretValues)
				fmt.Fprintf(&s.acc.buf, "[job=%s step=%d name=%s]\n%s\n", s.instanceName, i, step.Name, masked)

				logLine := &entity.JobLogLine{
					OrganizationID: s.payload.OrganizationID,
					RepositoryID:   s.payload.RepositoryID,
					RunID:          s.payload.WorkflowRunID,
					JobID:          s.jobIDStr,
					StepIndex:      i,
					LineNumber:     lineNum,
					Stream:         stream,
					Text:           masked,
					CreatedAt:      time.Now().UTC(),
				}
				if appendErr := w.logRepo.AppendLines(ctx, []*entity.JobLogLine{logLine}); appendErr != nil {
					return
				}
				if w.logPublisher != nil {
					_ = w.logPublisher.Publish(ctx, logLine)
				}
			})
			cancel()

			if runErr != nil {
				jobFailed = true
				fmt.Fprintf(&s.acc.buf, "[job=%s step=%d] error: %s\n", s.instanceName, i, maskSecrets(runErr.Error(), s.secretValues))
			}
		} else {
			out, runErr := w.runStep(stepCtx, instWorkdir, stepEnv, script)
			cancel()

			masked := maskSecrets(string(out), s.secretValues)
			fmt.Fprintf(&s.acc.buf, "[job=%s step=%d name=%s]\n%s\n", s.instanceName, i, step.Name, masked)

			if runErr != nil {
				jobFailed = true
				fmt.Fprintf(&s.acc.buf, "[job=%s step=%d] error: %s\n", s.instanceName, i, maskSecrets(runErr.Error(), s.secretValues))
			}
		}
	}

	return jobFailed
}

// runUsesStep handles a `uses:` step. The built-in actions/checkout populates
// the workspace from the run's commit; any other action is logged as
// unsupported and skipped (not failed), so workflows that reference marketplace
// actions still make progress. Returns true only if a supported action failed.
func (w *CIWorker) runUsesStep(ctx context.Context, s runJobSpec, step workflow.IRStep, i int, workdir string, evalCtx *workflow.EvalContext, stepEnv []string) bool {
	if isCheckoutAction(step.Uses) {
		ref := s.payload.HeadSHA
		if ref == "" {
			ref = s.payload.HeadBranch
		}
		w.emitUsesLine(ctx, s, i, fmt.Sprintf("Run actions/checkout (%s)", ref))
		if err := w.checkout(ctx, s.payload.RepoGitPath, ref, workdir); err != nil {
			w.emitUsesLine(ctx, s, i, "checkout failed: "+err.Error())
			return true
		}
		return false
	}

	// Container actions (uses: docker://<image>) are self-contained: run the
	// image with the workspace mounted and `with:` inputs passed as INPUT_*.
	if step.UsesRef != nil && step.UsesRef.Kind == "docker" && step.UsesRef.Image != "" {
		w.emitUsesLine(ctx, s, i, "Run "+step.Uses)
		env := append(append([]string{}, stepEnv...), actionInputEnv(step.With, evalCtx)...)
		out, err := w.containerAction(ctx, step.UsesRef.Image, workdir, env)
		for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
			if line != "" {
				w.emitUsesLine(ctx, s, i, maskSecrets(line, s.secretValues))
			}
		}
		if err != nil {
			w.emitUsesLine(ctx, s, i, "action failed: "+maskSecrets(err.Error(), s.secretValues))
			return true
		}
		return false
	}

	w.emitUsesLine(ctx, s, i, fmt.Sprintf("Skipping unsupported action %q (built-in: actions/checkout and docker:// container actions)", step.Uses))
	return false
}

// emitUsesLine records a synthetic log line for a `uses:` step so the web UI
// shows checkout/unsupported-action notes, mirroring how run-step output is
// streamed. Falls back to the buffered log when no log repository is wired.
func (w *CIWorker) emitUsesLine(ctx context.Context, s runJobSpec, stepIdx int, text string) {
	fmt.Fprintf(&s.acc.buf, "[job=%s step=%d name=%s]\n%s\n", s.instanceName, stepIdx, "", text)
	if w.logRepo == nil {
		return
	}
	s.acc.lineCount++
	line := &entity.JobLogLine{
		OrganizationID: s.payload.OrganizationID,
		RepositoryID:   s.payload.RepositoryID,
		RunID:          s.payload.WorkflowRunID,
		JobID:          s.jobIDStr,
		StepIndex:      stepIdx,
		LineNumber:     s.acc.lineCount,
		Stream:         entity.LogStreamStdout,
		Text:           text,
		CreatedAt:      time.Now().UTC(),
	}
	if err := w.logRepo.AppendLines(ctx, []*entity.JobLogLine{line}); err != nil {
		return
	}
	if w.logPublisher != nil {
		_ = w.logPublisher.Publish(ctx, line)
	}
}

// sortedJobNames returns job names in a stable order (used only as a fallback
// when the topological order is unavailable).
func sortedJobNames(jobs map[string]workflow.IRJob) []string {
	names := make([]string, 0, len(jobs))
	for name := range jobs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// firstUnsatisfiedNeed returns the first dependency that has not completed
// successfully, indicating the dependent job must be skipped.
func firstUnsatisfiedNeed(needs []string, results map[string]string) (string, bool) {
	for _, need := range needs {
		if results[need] != jobResultSuccess {
			return need, true
		}
	}
	return "", false
}

// stringifyMatrixCombo renders a matrix combination's values as strings for the
// `matrix` expression context.
func stringifyMatrixCombo(combo map[string]any) map[string]string {
	out := make(map[string]string, len(combo))
	for k, v := range combo {
		out[k] = fmt.Sprintf("%v", v)
	}
	return out
}

// matrixLabel builds the GitHub-style suffix for a matrix job instance, e.g.
// "ubuntu, 1.22". Keys are sorted for deterministic naming since the combination
// map is unordered.
func matrixLabel(combo map[string]any) string {
	keys := make([]string, 0, len(combo))
	for k := range combo {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%v", combo[k]))
	}
	return strings.Join(parts, ", ")
}

// refFromBranch returns the full git ref for a branch name (empty for empty).
func refFromBranch(branch string) string {
	if branch == "" {
		return ""
	}
	return "refs/heads/" + branch
}

// mergeStringMaps merges maps left-to-right (later overrides earlier) into a
// new map, so callers can freely mutate the result.
func mergeStringMaps(maps ...map[string]string) map[string]string {
	out := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// parseEnvToMap turns a KEY=VALUE slice into a map (used to expose secrets as
// an expression context).
func parseEnvToMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, kv := range env {
		if eq := strings.IndexByte(kv, '='); eq >= 0 {
			m[kv[:eq]] = kv[eq+1:]
		}
	}
	return m
}

// buildStepEnv flattens the env layers (lowest precedence first) and the
// decrypted secrets (highest precedence) into a deduplicated, sorted KEY=VALUE
// slice so the executed step sees a single unambiguous value per key.
func buildStepEnv(layers []map[string]string, secretEnv []string) []string {
	m := make(map[string]string)
	for _, layer := range layers {
		for k, v := range layer {
			m[k] = v
		}
	}
	for _, kv := range secretEnv {
		if eq := strings.IndexByte(kv, '='); eq >= 0 {
			m[kv[:eq]] = kv[eq+1:]
		}
	}
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	sort.Strings(out)
	return out
}

func (w *CIWorker) loadPlanTier(ctx context.Context, orgID string) (string, error) {
	if w.db == nil {
		return planTierPro, nil
	}
	var tier string
	err := w.db.QueryRowContext(
		ctx,
		`SELECT plan_tier FROM organizations WHERE id = $1`,
		orgID,
	).Scan(&tier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Personal repositories use the owner's user id as the organization
			// id and have no organizations row; treat them as the free tier.
			return planTierFree, nil
		}
		return "", err
	}
	return strings.ToLower(tier), nil
}

func (w *CIWorker) countRunning(ctx context.Context, orgID, excludeRunID string) (int, error) {
	if w.db == nil {
		return 0, nil
	}
	var n int
	err := w.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM workflow_runs
		 WHERE organization_id = $1 AND status = $2 AND id <> $3`,
		orgID, ciStatusInProgress, excludeRunID,
	).Scan(&n)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (w *CIWorker) loadSecrets(ctx context.Context, repoID string) ([]string, []string, error) {
	if w.db == nil {
		return nil, nil, nil
	}
	// action_secrets is the table the /actions/secrets API writes; encrypted
	// values are bytes sealed by the configured ActionSecretEncryptor, which
	// the injected decrypter reverses.
	rows, err := w.db.QueryContext(
		ctx,
		`SELECT name, encrypted_value FROM action_secrets WHERE repository_id = $1`,
		repoID,
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var env []string
	var values []string
	for rows.Next() {
		var name string
		var encrypted []byte
		if err := rows.Scan(&name, &encrypted); err != nil {
			return nil, nil, err
		}
		plain, err := w.decrypt(ctx, string(encrypted))
		if err != nil {
			return nil, nil, fmt.Errorf("decrypt secret %s: %w", name, err)
		}
		env = append(env, name+"="+plain)
		values = append(values, plain)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return env, values, nil
}

func (w *CIWorker) setStatus(ctx context.Context, runID, status string) error {
	if w.db == nil || runID == "" {
		return nil
	}
	// Never resurrect a run the user cancelled while it was queued/starting.
	_, err := w.db.ExecContext(
		ctx,
		`UPDATE workflow_runs SET status = $1
		 WHERE id = $2 AND (conclusion IS NULL OR conclusion <> 'cancelled')`,
		status, runID,
	)
	return err
}

func (w *CIWorker) markRun(ctx context.Context, runID, status, conclusion, logs string) {
	if w.db == nil || runID == "" {
		return
	}
	conclusionField := conclusion
	// Legacy fallback: with no log repository wired, step output is stashed in
	// the conclusion column so it isn't lost entirely. With a log repository
	// the lines live in job_log_lines and conclusion stays a clean enum value.
	if logs != "" && w.logRepo == nil {
		conclusionField = conclusion + "\n" + logs
	}
	// The cancel guard keeps a completed executor from overwriting a run the
	// user cancelled mid-flight.
	_, _ = w.db.ExecContext(
		ctx,
		`UPDATE workflow_runs SET status = $1, conclusion = $2, completed_at = $3, updated_at = $3
		 WHERE id = $4 AND (conclusion IS NULL OR conclusion <> 'cancelled')`,
		status, conclusionField, time.Now().UTC(), runID,
	)
}

func concurrentLimitExceeded(tier string, running int) bool {
	switch strings.ToLower(tier) {
	case planTierFree:
		return running >= freeTierConcurrentLimit
	case planTierPro:
		return running >= proTierConcurrentLimit
	default:
		return running >= proTierConcurrentLimit
	}
}

func maskSecrets(s string, secretValues []string) string {
	for _, v := range secretValues {
		if v == "" {
			continue
		}
		s = strings.ReplaceAll(s, v, logMask)
	}
	return s
}

func identityDecrypter(_ context.Context, encrypted string) (string, error) {
	return encrypted, nil
}

func (w *CIWorker) defaultCommandRunner(ctx context.Context, workdir string, env []string, script string) ([]byte, error) {
	cmd := w.sandbox.buildCommand(ctx, workdir, env, script)
	return cmd.CombinedOutput()
}

func (w *CIWorker) defaultStreamingCommandRunner(ctx context.Context, workdir string, env []string, script string, _ int, sink func(stream, line string)) error {
	cmd := w.sandbox.buildCommand(ctx, workdir, env, script)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	scanLines := func(r io.Reader, stream string) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			sink(stream, scanner.Text())
		}
	}

	wg.Add(2)
	go scanLines(stdout, entity.LogStreamStdout)
	go scanLines(stderr, entity.LogStreamStderr)
	wg.Wait()

	return cmd.Wait()
}
