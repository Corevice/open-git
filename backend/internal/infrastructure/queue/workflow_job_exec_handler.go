package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/runner"
)

const (
	defaultJobTimeoutMinutes = 360
)

type workflowStepRepository interface {
	ListByJobID(ctx context.Context, orgID, jobID string) ([]*runner.Step, error)
}

type workflowScheduleEnqueuer interface {
	EnqueueSchedule(ctx context.Context, payload WorkflowSchedulePayload) error
}

type secretProvider interface {
	GetSecrets(ctx context.Context, orgID, repoID string) (map[string]string, error)
}

type WorkflowJobExecHandler struct {
	jobRepo  domainrepo.IWorkflowJobRepository
	stepRepo workflowStepRepository
	logRepo  domainrepo.IJobLogRepository
	runner   runner.Runner
	enqueuer workflowScheduleEnqueuer
	secrets  secretProvider
}

func NewWorkflowJobExecHandler(
	jobRepo domainrepo.IWorkflowJobRepository,
	stepRepo workflowStepRepository,
	logRepo domainrepo.IJobLogRepository,
	jobRunner runner.Runner,
	client *asynq.Client,
	secrets secretProvider,
) *WorkflowJobExecHandler {
	return &WorkflowJobExecHandler{
		jobRepo:  jobRepo,
		stepRepo: stepRepo,
		logRepo:  logRepo,
		runner:   jobRunner,
		enqueuer: &asynqScheduleEnqueuer{client: client},
		secrets:  secrets,
	}
}

func NewWorkflowJobExecHandlerWithEnqueuer(
	jobRepo domainrepo.IWorkflowJobRepository,
	stepRepo workflowStepRepository,
	logRepo domainrepo.IJobLogRepository,
	jobRunner runner.Runner,
	enqueuer workflowScheduleEnqueuer,
	secrets secretProvider,
) *WorkflowJobExecHandler {
	return &WorkflowJobExecHandler{
		jobRepo:  jobRepo,
		stepRepo: stepRepo,
		logRepo:  logRepo,
		runner:   jobRunner,
		enqueuer: enqueuer,
		secrets:  secrets,
	}
}

type asynqScheduleEnqueuer struct {
	client *asynq.Client
}

func (e *asynqScheduleEnqueuer) EnqueueSchedule(ctx context.Context, payload WorkflowSchedulePayload) error {
	_, err := EnqueueWorkflowSchedule(ctx, e.client, payload)
	return err
}

func (h *WorkflowJobExecHandler) HandleWorkflowJobExec(ctx context.Context, task *asynq.Task) error {
	var payload WorkflowJobExecPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal workflow job exec payload: %v: %w", err, asynq.SkipRetry)
	}
	if payload.JobID == "" || payload.RunID == "" || payload.OrgID == "" {
		return fmt.Errorf("workflow job exec payload missing identifiers: %w", asynq.SkipRetry)
	}

	orgID, err := uuid.Parse(payload.OrgID)
	if err != nil {
		return fmt.Errorf("invalid org id: %w", asynq.SkipRetry)
	}
	runID, err := uuid.Parse(payload.RunID)
	if err != nil {
		return fmt.Errorf("invalid run id: %w", asynq.SkipRetry)
	}
	jobID, err := uuid.Parse(payload.JobID)
	if err != nil {
		return fmt.Errorf("invalid job id: %w", asynq.SkipRetry)
	}

	job, err := h.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("load workflow job: %w", err)
	}
	if job.OrganizationID != orgID {
		return fmt.Errorf("job organization mismatch: %w", asynq.SkipRetry)
	}
	if job.RepositoryID == uuid.Nil {
		return fmt.Errorf("job repository id missing: %w", asynq.SkipRetry)
	}
	if job.WorkflowRunID == nil || *job.WorkflowRunID != runID {
		return fmt.Errorf("job run mismatch: %w", asynq.SkipRetry)
	}

	steps, err := h.stepRepo.ListByJobID(ctx, orgID.String(), jobID.String())
	if err != nil {
		return fmt.Errorf("load workflow steps: %w", err)
	}

	if err := h.jobRepo.UpdateStatus(ctx, jobID, entity.WorkflowJobStatusInProgress, ""); err != nil {
		return fmt.Errorf("set job in_progress: %w", err)
	}

	timeoutMinutes := jobTimeoutMinutes(job)
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMinutes)*time.Minute)
	defer cancel()

	secretMap := map[string]string{}
	if h.secrets != nil {
		secretMap, err = h.secrets.GetSecrets(ctx, orgID.String(), job.RepositoryID.String())
		if err != nil {
			return fmt.Errorf("load secrets: %w", err)
		}
	}

	logFn, logErrFn := h.buildLogCallback(execCtx, orgID, runID, jobID, job)
	runErr := h.runner.ExecuteJob(execCtx, job, steps, secretMap, logFn)
	if logErr := logErrFn(); logErr != nil {
		return fmt.Errorf("append job log lines: %w", logErr)
	}

	status, conclusion := entity.WorkflowJobStatusCompleted, conclusionSuccess
	if runErr != nil {
		if errors.Is(runErr, context.DeadlineExceeded) || errors.Is(execCtx.Err(), context.DeadlineExceeded) {
			status = entity.WorkflowJobStatusCompleted
			conclusion = conclusionTimedOut
		} else {
			status = entity.WorkflowJobStatusFailed
			conclusion = conclusionFailure
		}
	}

	completedAt := time.Now().UTC()
	if status == entity.WorkflowJobStatusCompleted {
		if err := h.jobRepo.Complete(ctx, jobID, conclusion, completedAt); err != nil {
			return fmt.Errorf("update job terminal status: %w", err)
		}
	} else if err := h.jobRepo.UpdateStatus(ctx, jobID, status, conclusion); err != nil {
		return fmt.Errorf("update job terminal status: %w", err)
	}

	if err := h.enqueuer.EnqueueSchedule(ctx, WorkflowSchedulePayload{
		RunID: runID.String(),
		OrgID: orgID.String(),
	}); err != nil {
		return fmt.Errorf("enqueue next workflow schedule: %w", err)
	}

	return nil
}

func (h *WorkflowJobExecHandler) buildLogCallback(
	ctx context.Context,
	orgID uuid.UUID,
	runID uuid.UUID,
	jobID uuid.UUID,
	job *entity.WorkflowJob,
) (func(offset int64, chunk string), func() error) {
	var (
		appendErr error
		appendMu  sync.Mutex
	)
	orgIDStr := orgID.String()
	runIDStr := runID.String()
	jobIDStr := jobID.String()
	return func(offset int64, chunk string) {
		if h.logRepo == nil {
			return
		}
		line := &entity.JobLogLine{
			OrganizationID: orgIDStr,
			RepositoryID:   job.RepositoryID.String(),
			RunID:          runIDStr,
			JobID:          jobIDStr,
			LineNumber:     offset,
			Stream:         entity.LogStreamStdout,
			Text:           sanitizeLogLine(chunk),
			CreatedAt:      time.Now().UTC(),
		}
		appendMu.Lock()
		defer appendMu.Unlock()
		if err := h.logRepo.AppendLines(ctx, []*entity.JobLogLine{line}); err != nil && appendErr == nil {
			appendErr = err
		}
	}, func() error {
		appendMu.Lock()
		defer appendMu.Unlock()
		return appendErr
	}
}

func sanitizeLogLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, s)
}

func jobTimeoutMinutes(job *entity.WorkflowJob) int {
	if job == nil {
		return defaultJobTimeoutMinutes
	}
	field := reflect.ValueOf(job).Elem().FieldByName("TimeoutMinutes")
	if field.IsValid() && field.Kind() == reflect.Int && field.Int() > 0 {
		return int(field.Int())
	}
	return defaultJobTimeoutMinutes
}
