package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/infrastructure/workflow"
)

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

// CommandRunner executes a shell command and returns the combined stdout/stderr.
// Exposed for testability.
type CommandRunner func(ctx context.Context, env []string, script string) ([]byte, error)

type CIWorker struct {
	db       *sql.DB
	decrypt  SecretDecrypter
	runStep  CommandRunner
	stepWait time.Duration
}

func NewCIWorker(db *sql.DB) *CIWorker {
	return &CIWorker{
		db:       db,
		decrypt:  identityDecrypter,
		runStep:  defaultCommandRunner,
		stepWait: ciStepTimeout,
	}
}

func (w *CIWorker) WithDecrypter(d SecretDecrypter) *CIWorker {
	w.decrypt = d
	return w
}

func (w *CIWorker) WithCommandRunner(r CommandRunner) *CIWorker {
	w.runStep = r
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

	wf, err := workflow.ParseWorkflow(payload.WorkflowYAML)
	if err != nil {
		w.markRun(ctx, payload.WorkflowRunID, ciStatusFailed, ciConclusionFailure, err.Error())
		return fmt.Errorf("parse workflow: %w: %w", err, asynq.SkipRetry)
	}

	secretEnv, secretValues, err := w.loadSecrets(ctx, payload.RepositoryID)
	if err != nil {
		return fmt.Errorf("load secrets: %w", err)
	}

	logBuf := &strings.Builder{}
	conclusion := ciConclusionSuccess
runLoop:
	for jobID, job := range wf.Jobs {
		for i, step := range job.Steps {
			if step.Run == "" {
				continue
			}
			stepCtx, cancel := context.WithTimeout(ctx, w.stepWait)
			out, runErr := w.runStep(stepCtx, secretEnv, step.Run)
			cancel()

			masked := maskSecrets(string(out), secretValues)
			fmt.Fprintf(logBuf, "[job=%s step=%d name=%s]\n%s\n", jobID, i, step.Name, masked)

			if runErr != nil {
				conclusion = ciConclusionFailure
				fmt.Fprintf(logBuf, "[job=%s step=%d] error: %s\n", jobID, i, maskSecrets(runErr.Error(), secretValues))
				break runLoop
			}
		}
	}

	finalStatus := ciStatusCompleted
	if conclusion != ciConclusionSuccess {
		finalStatus = ciStatusFailed
	}
	w.markRun(ctx, payload.WorkflowRunID, finalStatus, conclusion, logBuf.String())
	return nil
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
			return "", fmt.Errorf("organization %s not found", orgID)
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
	rows, err := w.db.QueryContext(
		ctx,
		`SELECT name, encrypted_value FROM secrets WHERE repository_id = $1`,
		repoID,
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var env []string
	var values []string
	for rows.Next() {
		var name, encrypted string
		if err := rows.Scan(&name, &encrypted); err != nil {
			return nil, nil, err
		}
		plain, err := w.decrypt(ctx, encrypted)
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
	_, err := w.db.ExecContext(
		ctx,
		`UPDATE workflow_runs SET status = $1 WHERE id = $2`,
		status, runID,
	)
	return err
}

func (w *CIWorker) markRun(ctx context.Context, runID, status, conclusion, logs string) {
	if w.db == nil || runID == "" {
		return
	}
	conclusionField := conclusion
	if logs != "" {
		conclusionField = conclusion + "\n" + logs
	}
	_, _ = w.db.ExecContext(
		ctx,
		`UPDATE workflow_runs SET status = $1, conclusion = $2, completed_at = $3 WHERE id = $4`,
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

func defaultCommandRunner(ctx context.Context, env []string, script string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", script)
	cmd.Env = append(cmd.Env, env...)
	return cmd.CombinedOutput()
}
