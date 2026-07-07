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
	"sort"
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
)

type CIRunPayload struct {
	WorkflowRunID  string `json:"workflow_run_id"`
	RepositoryID   string `json:"repository_id"`
	OrganizationID string `json:"organization_id"`
	WorkflowYAML   []byte `json:"workflow_yaml"`
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

type CIWorker struct {
	db            *sql.DB
	decrypt       SecretDecrypter
	runStep       CommandRunner
	runStepStream StreamingCommandRunner
	stepWait      time.Duration
	logRepo       domainrepo.IJobLogRepository
	jobRepo       domainrepo.IWorkflowJobRepository
	logPublisher  *queue.JobLogPublisher
	sandbox       sandbox
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
	return w
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

	// Run jobs in dependency (topological) order so `needs` is honored. On a
	// broken DAG (already reported as a diagnostic error above) Order may be
	// short; fall back to a stable order defensively.
	order := ir.DAG.Order
	if len(order) < len(ir.Jobs) {
		order = sortedJobNames(ir.Jobs)
	}

	for _, jobName := range order {
		job := ir.Jobs[jobName]

		var jobUUID uuid.UUID
		if w.jobRepo != nil {
			jobUUID = int64CompatibleUUID()
			jobIDs[jobName] = jobUUID.String()
			now := time.Now().UTC()
			wfJob := &entity.WorkflowJob{
				ID:             jobUUID,
				WorkflowRunID:  &runUUID,
				OrganizationID: orgUUID,
				RepositoryID:   repoUUID,
				Name:           jobName,
				Status:         entity.WorkflowJobStatusInProgress,
				StartedAt:      &now,
				CreatedAt:      now,
			}
			if createErr := w.jobRepo.Create(ctx, wfJob); createErr != nil {
				return fmt.Errorf("create workflow job: %w", createErr)
			}
		}

		// A job whose dependencies did not all succeed is skipped, like GitHub
		// Actions. The dependency's own failure already set the run conclusion,
		// so a skip adds no new failure — but independent jobs still run.
		if dep, unmet := firstUnsatisfiedNeed(job.Needs, jobResults); unmet {
			jobResults[jobName] = jobResultSkipped
			jobLogStatus[jobName] = jobResultSkipped
			fmt.Fprintf(logBuf, "[job=%s] skipped: dependency %q did not succeed\n", jobName, dep)
			if w.jobRepo != nil && jobUUID != uuid.Nil {
				_ = w.jobRepo.Complete(ctx, jobUUID, entity.WorkflowJobConclusionSkipped, time.Now().UTC())
			}
			continue
		}

		jobLogStatus[jobName] = jobLogStatusSuccess
		jobFailed := false

		for i, step := range job.Steps {
			if step.Run == "" {
				// `uses` steps are not executed yet; skip without failing.
				continue
			}
			// Env precedence (low to high): workflow env, job env, step env,
			// then decrypted secrets injected by name.
			stepEnv := buildStepEnv([]map[string]string{ir.Env, job.Env, step.Env}, secretEnv)
			stepCtx, cancel := context.WithTimeout(ctx, w.stepWait)

			if useStreaming {
				jobIDStr := jobIDs[jobName]
				if jobIDStr == "" {
					jobIDStr = jobName
				}
				runErr := streamRunner(stepCtx, workdir, stepEnv, step.Run, i, func(stream, line string) {
					jobLineCounts[jobName]++
					lineNum := jobLineCounts[jobName]
					masked := maskSecrets(line, secretValues)
					fmt.Fprintf(logBuf, "[job=%s step=%d name=%s]\n%s\n", jobName, i, step.Name, masked)

					logLine := &entity.JobLogLine{
						OrganizationID: payload.OrganizationID,
						RepositoryID:   payload.RepositoryID,
						RunID:          payload.WorkflowRunID,
						JobID:          jobIDStr,
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
					fmt.Fprintf(logBuf, "[job=%s step=%d] error: %s\n", jobName, i, maskSecrets(runErr.Error(), secretValues))
					break
				}
			} else {
				out, runErr := w.runStep(stepCtx, workdir, stepEnv, step.Run)
				cancel()

				masked := maskSecrets(string(out), secretValues)
				fmt.Fprintf(logBuf, "[job=%s step=%d name=%s]\n%s\n", jobName, i, step.Name, masked)

				if runErr != nil {
					jobFailed = true
					fmt.Fprintf(logBuf, "[job=%s step=%d] error: %s\n", jobName, i, maskSecrets(runErr.Error(), secretValues))
					break
				}
			}
		}

		if jobFailed {
			jobResults[jobName] = jobResultFailure
			jobLogStatus[jobName] = jobLogStatusFailure
			conclusion = ciConclusionFailure
		} else {
			jobResults[jobName] = jobResultSuccess
		}

		// Record the job's terminal state so run detail pages show per-job
		// results rather than jobs stuck in_progress forever.
		if w.jobRepo != nil && jobUUID != uuid.Nil {
			jobConclusion := entity.WorkflowJobConclusionSuccess
			if jobFailed {
				jobConclusion = entity.WorkflowJobConclusionFailure
			}
			_ = w.jobRepo.Complete(ctx, jobUUID, jobConclusion, time.Now().UTC())
		}
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
