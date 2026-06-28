package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/runner"
)

const defaultJobTimeoutMinutes = 360

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
	jobRepo    domainrepo.IWorkflowJobRepository
	stepRepo   workflowStepRepository
	logRepo    domainrepo.IJobLogRepository
	runner     runner.Runner
	enqueuer   workflowScheduleEnqueuer
	secrets    secretProvider
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
		return fmt.Errorf("unmarshal workflow job exec payload: %w: %w", err, asynq.SkipRetry)
	}
	if payload.JobID == "" || payload.RunID == "" || payload.OrgID == "" {
		return fmt.Errorf("workflow job exec payload missing identifiers: %w", asynq.SkipRetry)
	}

	job, err := h.jobRepo.GetByID(ctx, payload.OrgID, payload.JobID)
	if err != nil {
		return fmt.Errorf("load workflow job: %w", err)
	}

	steps, err := h.stepRepo.ListByJobID(ctx, payload.OrgID, payload.JobID)
	if err != nil {
		return fmt.Errorf("load workflow steps: %w", err)
	}

	if err := h.jobRepo.UpdateStatus(ctx, payload.JobID, entity.WorkflowJobStatusInProgress, "", nil); err != nil {
		return fmt.Errorf("set job in_progress: %w", err)
	}

	timeoutMinutes := jobTimeoutMinutes(job)
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMinutes)*time.Minute)
	defer cancel()

	secretMap := map[string]string{}
	if h.secrets != nil {
		secretMap, err = h.secrets.GetSecrets(ctx, payload.OrgID, job.RepositoryID)
		if err != nil {
			return fmt.Errorf("load secrets: %w", err)
		}
	}

	logFn := h.buildLogCallback(execCtx, payload, job)
	runErr := h.runner.ExecuteJob(execCtx, job, steps, secretMap, logFn)

	status, conclusion := entity.WorkflowJobStatusCompleted, conclusionSuccess
	if runErr != nil {
		status = entity.WorkflowJobStatusFailed
		if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
			conclusion = fmt.Sprintf("job timed out after %d minutes", timeoutMinutes)
		} else {
			conclusion = conclusionFailure
		}
	}

	completedAt := time.Now().UTC()
	if err := h.jobRepo.UpdateStatus(ctx, payload.JobID, status, conclusion, &completedAt); err != nil {
		return fmt.Errorf("update job terminal status: %w", err)
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
) func(offset int64, chunk string) {
	return func(offset int64, chunk string) {
		if h.logRepo == nil {
			return
		}
		line := &entity.JobLogLine{
			OrganizationID: payload.OrgID,
			RepositoryID:   job.RepositoryID,
			RunID:          payload.RunID,
			JobID:          payload.JobID,
			LineNumber:     offset,
			Stream:         entity.LogStreamStdout,
			Text:           chunk,
			CreatedAt:      time.Now().UTC(),
		}
		_ = h.logRepo.AppendLines(ctx, []*entity.JobLogLine{line})
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
