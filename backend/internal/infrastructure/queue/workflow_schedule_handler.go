package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

const (
	runStatusInProgress = "in_progress"
	runStatusCompleted  = "completed"

	conclusionSuccess   = "success"
	conclusionFailure   = "failure"
	conclusionCancelled = "cancelled"
)

type schedulableJob struct {
	ID         string
	Name       string
	Status     string
	Conclusion string
	Needs      []string
}

type scheduleJobRepository interface {
	ListByRunID(ctx context.Context, orgID, runID string) ([]*schedulableJob, error)
}

type workflowRunRepository interface {
	UpdateStatus(ctx context.Context, orgID, runID, status string) error
	UpdateConclusion(ctx context.Context, orgID, runID, status, conclusion string) error
}

type workflowJobExecEnqueuer interface {
	EnqueueJobExec(ctx context.Context, payload WorkflowJobExecPayload) error
}

type WorkflowScheduleHandler struct {
	jobRepo  scheduleJobRepository
	runRepo  workflowRunRepository
	enqueuer workflowJobExecEnqueuer
}

func NewWorkflowScheduleHandler(
	jobRepo domainrepo.IWorkflowJobRepository,
	runRepo workflowRunRepository,
	client *asynq.Client,
) *WorkflowScheduleHandler {
	return NewWorkflowScheduleHandlerWithEnqueuer(
		&domainJobRepoAdapter{repo: jobRepo},
		runRepo,
		&asynqJobExecEnqueuer{client: client},
	)
}

func NewWorkflowScheduleHandlerWithEnqueuer(
	jobRepo scheduleJobRepository,
	runRepo workflowRunRepository,
	enqueuer workflowJobExecEnqueuer,
) *WorkflowScheduleHandler {
	return &WorkflowScheduleHandler{
		jobRepo:  jobRepo,
		runRepo:  runRepo,
		enqueuer: enqueuer,
	}
}

type domainJobRepoAdapter struct {
	repo domainrepo.IWorkflowJobRepository
}

func (a *domainJobRepoAdapter) ListByRunID(ctx context.Context, orgID, runID string) ([]*schedulableJob, error) {
	jobs, err := a.repo.ListByRunID(ctx, orgID, runID)
	if err != nil {
		return nil, err
	}
	out := make([]*schedulableJob, len(jobs))
	for i, job := range jobs {
		out[i] = &schedulableJob{
			ID:         job.ID,
			Name:       job.Name,
			Status:     job.Status,
			Conclusion: job.Conclusion,
			Needs:      workflowJobNeeds(job),
		}
	}
	return out, nil
}

type asynqJobExecEnqueuer struct {
	client *asynq.Client
}

func (e *asynqJobExecEnqueuer) EnqueueJobExec(ctx context.Context, payload WorkflowJobExecPayload) error {
	_, err := EnqueueWorkflowJobExec(ctx, e.client, payload)
	return err
}

func (h *WorkflowScheduleHandler) HandleWorkflowSchedule(ctx context.Context, task *asynq.Task) error {
	var payload WorkflowSchedulePayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal workflow schedule payload: %w: %w", err, asynq.SkipRetry)
	}
	if payload.RunID == "" || payload.OrgID == "" {
		return fmt.Errorf("workflow schedule payload missing identifiers: %w", asynq.SkipRetry)
	}

	jobs, err := h.jobRepo.ListByRunID(ctx, payload.OrgID, payload.RunID)
	if err != nil {
		return fmt.Errorf("list workflow jobs: %w", err)
	}

	completed := buildCompletedJobSet(jobs)
	ready := findReadyJobs(jobs, completed)

	enqueued := 0
	for _, job := range ready {
		if err := h.enqueuer.EnqueueJobExec(ctx, WorkflowJobExecPayload{
			JobID: job.ID,
			RunID: payload.RunID,
			OrgID: payload.OrgID,
		}); err != nil {
			return fmt.Errorf("enqueue workflow job exec for %q: %w", job.Name, err)
		}
		enqueued++
	}

	if enqueued > 0 {
		if err := h.runRepo.UpdateStatus(ctx, payload.OrgID, payload.RunID, runStatusInProgress); err != nil {
			return fmt.Errorf("update run status in_progress: %w", err)
		}
		return nil
	}

	if allJobsTerminal(jobs) {
		status, conclusion := computeRunConclusion(jobs)
		if err := h.runRepo.UpdateConclusion(ctx, payload.OrgID, payload.RunID, status, conclusion); err != nil {
			return fmt.Errorf("update run conclusion: %w", err)
		}
		return nil
	}

	if err := h.runRepo.UpdateConclusion(ctx, payload.OrgID, payload.RunID, runStatusCompleted, conclusionFailure); err != nil {
		return fmt.Errorf("mark run failure for deadlock: %w", err)
	}
	return nil
}

func buildCompletedJobSet(jobs []*schedulableJob) map[string]struct{} {
	completed := make(map[string]struct{})
	for _, job := range jobs {
		if isNeedSatisfied(job) {
			completed[job.Name] = struct{}{}
		}
	}
	return completed
}

func findReadyJobs(jobs []*schedulableJob, completed map[string]struct{}) []*schedulableJob {
	ready := make([]*schedulableJob, 0)
	for _, job := range jobs {
		if job.Status != entity.WorkflowJobStatusQueued {
			continue
		}
		if allNeedsCompleted(job, completed) {
			ready = append(ready, job)
		}
	}
	return ready
}

func allNeedsCompleted(job *schedulableJob, completed map[string]struct{}) bool {
	for _, need := range job.Needs {
		if _, ok := completed[need]; !ok {
			return false
		}
	}
	return true
}

func isNeedSatisfied(job *schedulableJob) bool {
	return job.Status == entity.WorkflowJobStatusCompleted && job.Conclusion == conclusionSuccess
}

func isJobTerminal(job *schedulableJob) bool {
	switch job.Status {
	case entity.WorkflowJobStatusCompleted, entity.WorkflowJobStatusFailed, "cancelled":
		return true
	default:
		return false
	}
}

func allJobsTerminal(jobs []*schedulableJob) bool {
	if len(jobs) == 0 {
		return true
	}
	for _, job := range jobs {
		if !isJobTerminal(job) {
			return false
		}
	}
	return true
}

func computeRunConclusion(jobs []*schedulableJob) (status, conclusion string) {
	status = runStatusCompleted
	conclusion = conclusionSuccess

	for _, job := range jobs {
		if job.Conclusion == conclusionFailure || job.Status == entity.WorkflowJobStatusFailed {
			return runStatusCompleted, conclusionFailure
		}
		if job.Conclusion == conclusionCancelled || job.Status == "cancelled" {
			return runStatusCompleted, conclusionCancelled
		}
	}
	return status, conclusion
}

func workflowJobNeeds(job *entity.WorkflowJob) []string {
	if job == nil {
		return nil
	}
	field := reflect.ValueOf(job).Elem().FieldByName("Needs")
	if !field.IsValid() || field.Kind() != reflect.Slice {
		return nil
	}
	out := make([]string, field.Len())
	for i := 0; i < field.Len(); i++ {
		out[i] = field.Index(i).String()
	}
	return out
}
