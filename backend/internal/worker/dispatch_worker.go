package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/queue"
	"github.com/open-git/backend/internal/infrastructure/runner"
	"github.com/open-git/backend/internal/usecase/actions"
)

const (
	maxRunsOnJSONLen    = 4096
	dispatchRetryDelay  = 30 * time.Second
)

type DispatchWorker struct {
	runnerRepo  domainrepo.IRunnerRepository
	jobRepo     domainrepo.IWorkflowJobRepository
	actAdapter  runner.RunnerAdapter
	asynqClient *asynq.Client
}

func NewDispatchWorker(
	runnerRepo domainrepo.IRunnerRepository,
	jobRepo domainrepo.IWorkflowJobRepository,
	actAdapter runner.RunnerAdapter,
	asynqClient *asynq.Client,
) *DispatchWorker {
	return &DispatchWorker{
		runnerRepo:  runnerRepo,
		jobRepo:     jobRepo,
		actAdapter:  actAdapter,
		asynqClient: asynqClient,
	}
}

func (w *DispatchWorker) scheduleDispatchRetry(ctx context.Context, payload queue.DispatchJobPayload) error {
	if w.asynqClient == nil {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal dispatch job payload: %w", err)
	}
	task := asynq.NewTask(queue.TypeDispatchJob, data)
	_, err = w.asynqClient.EnqueueContext(ctx, task, asynq.ProcessIn(dispatchRetryDelay))
	if err != nil {
		return fmt.Errorf("enqueue dispatch retry: %w", err)
	}
	return nil
}

func (w *DispatchWorker) resolveRunsOnLabels(job *entity.WorkflowJob, payload queue.DispatchJobPayload) ([]string, error) {
	labels := append([]string(nil), job.RunsOn...)
	if len(labels) > 0 {
		return labels, nil
	}
	if payload.RunsOn == "" {
		return labels, nil
	}
	if len(payload.RunsOn) > maxRunsOnJSONLen {
		return nil, fmt.Errorf("runs_on payload exceeds size limit")
	}
	if err := json.Unmarshal([]byte(payload.RunsOn), &labels); err != nil {
		return nil, fmt.Errorf("parse runs_on: %w", err)
	}
	return labels, nil
}

func (w *DispatchWorker) HandleDispatchJob(ctx context.Context, task *asynq.Task) error {
	var payload queue.DispatchJobPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal dispatch job payload: %w", err)
	}

	jobID, err := uuid.Parse(payload.JobID)
	if err != nil {
		return fmt.Errorf("parse job_id: %w", err)
	}
	orgID, err := uuid.Parse(payload.OrganizationID)
	if err != nil {
		return fmt.Errorf("parse organization_id: %w", err)
	}

	job, err := w.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("get job: %w", err)
	}
	if job.OrganizationID != orgID {
		return fmt.Errorf("job organization mismatch")
	}

	labels, err := w.resolveRunsOnLabels(job, payload)
	if err != nil {
		return err
	}

	if actions.UsesActAdapter(labels) {
		actPayload := runner.RunnerJobPayload{
			JobID:          job.ID.String(),
			WorkflowYAML:   runner.BuildActWorkflowYAML(job, nil),
			TimeoutMinutes: job.TimeoutMinutes,
		}
		conclusion := entity.WorkflowJobConclusionSuccess
		execErr := w.actAdapter.Execute(ctx, actPayload)
		if execErr != nil {
			conclusion = entity.WorkflowJobConclusionFailure
		}
		if err := w.jobRepo.Complete(ctx, jobID, conclusion, time.Now().UTC()); err != nil {
			return fmt.Errorf("complete job: %w", err)
		}
		if execErr != nil {
			return fmt.Errorf("act adapter execute: %w", execErr)
		}
		return nil
	}

	runnerEntity, err := w.runnerRepo.FindAvailable(ctx, orgID, labels)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			if err := w.scheduleDispatchRetry(ctx, payload); err != nil {
				return fmt.Errorf("schedule dispatch retry: %w", err)
			}
			return nil
		}
		return fmt.Errorf("find available runner: %w", err)
	}
	if runnerEntity == nil {
		if err := w.scheduleDispatchRetry(ctx, payload); err != nil {
			return fmt.Errorf("schedule dispatch retry: %w", err)
		}
		return nil
	}

	acquired, err := w.jobRepo.AcquireForRunner(ctx, jobID, runnerEntity.ID, job.AcquireLockVersion)
	if err != nil {
		return fmt.Errorf("acquire job for runner: %w", err)
	}
	if !acquired {
		if err := w.scheduleDispatchRetry(ctx, payload); err != nil {
			return fmt.Errorf("schedule dispatch retry: %w", err)
		}
		return nil
	}
	return nil
}

func (w *DispatchWorker) HandleCancelJob(ctx context.Context, task *asynq.Task) error {
	var payload queue.CancelJobPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal cancel job payload: %w", err)
	}

	jobID, err := uuid.Parse(payload.JobID)
	if err != nil {
		return fmt.Errorf("parse job_id: %w", err)
	}

	job, err := w.jobRepo.GetByID(ctx, jobID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("get job: %w", err)
	}

	useActAdapter := err != nil
	if err == nil {
		useActAdapter = actions.UsesActAdapter(job.RunsOn)
	}
	if useActAdapter {
		if err := w.actAdapter.Cancel(ctx, payload.JobID); err != nil {
			return fmt.Errorf("cancel act adapter job: %w", err)
		}
	}

	if err := w.jobRepo.Cancel(ctx, jobID); err != nil {
		return fmt.Errorf("cancel job: %w", err)
	}
	return nil
}
