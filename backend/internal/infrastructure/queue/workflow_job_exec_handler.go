package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
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
	conclusionTimedOut       = "timed_out"
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

	jobID, err := uuid.Parse(payload.JobID)
	if err != nil {
		return fmt.Errorf("parse job id: %w", asynq.SkipRetry)
	}
	orgID, err := uuid.Parse(payload.OrgID)
	if err != nil {
		return fmt.Errorf("parse org id: %w", asynq.SkipRetry)
	}
	runID, err := uuid.Parse(payload.RunID)
	if err != nil {
		return fmt.Errorf("parse run id: %w", asynq.SkipRetry)
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

	steps, err := h.stepRepo.ListByJobID(ctx, payload.OrgID, payload.JobID)
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
		secretMap, err = h.secrets.GetSecrets(ctx, payload.OrgID, job.RepositoryID.String())
		if err != nil {
			return fmt.Errorf("load secrets: %w", err)
		}
	}

	logFn, logErrFn := h.buildLogCallback(execCtx, payload, job)
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
	switch status {
	case entity.WorkflowJobStatusCompleted:
		if err := h.jobRepo.Complete(ctx, jobID, conclusion, completedAt); err != nil {
			return fmt.Errorf("update job terminal status: %w", err)
		}
	default:
		if err := h.jobRepo.UpdateStatus(ctx, jobID, status, conclusion); err != nil {
			return fmt.Errorf("update job terminal status: %w", err)
		}
	}

	if err := h.enqueuer.EnqueueSchedule(ctx, WorkflowSchedulePayload{
		RunID: payload.RunID,
		OrgID: payload.OrgID,
	}); err != nil {
		return fmt.Errorf("enqueue next workflow schedule: %w", err)
	}

	return nil
}

func (h *WorkflowJobExecHandler) buildLogCallback(
	ctx context.Context,
	payload WorkflowJobExecPayload,
	job *entity.WorkflowJob,
) (func(offset int64, chunk string), func() error) {
	var (
		appendErr error
		appendMu  sync.Mutex
	)
	return func(offset int64, chunk string) {
		if h.logRepo == nil {
			return
		}
		line := &entity.JobLogLine{
			OrganizationID: payload.OrgID,
			RepositoryID:   job.RepositoryID.String(),
			RunID:          payload.RunID,
			JobID:          payload.JobID,
			LineNumber:     offset,
			Stream:         entity.LogStreamStdout,
			Text:           chunk,
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
