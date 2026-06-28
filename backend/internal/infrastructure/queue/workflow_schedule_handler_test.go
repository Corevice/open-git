package queue

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/domain/entity"
)

type mockScheduleJobRepo struct {
	jobs []*schedulableJob
}

func (m *mockScheduleJobRepo) ListByRunID(_ context.Context, _, _ string) ([]*schedulableJob, error) {
	return m.jobs, nil
}

type mockScheduleRunRepo struct {
	statusUpdates    []string
	conclusionUpdate struct {
		status     string
		conclusion string
		called     bool
	}
}

func (m *mockScheduleRunRepo) UpdateStatus(_ context.Context, _, _, status string) error {
	m.statusUpdates = append(m.statusUpdates, status)
	return nil
}

func (m *mockScheduleRunRepo) UpdateConclusion(_ context.Context, _, _, status, conclusion string) error {
	m.conclusionUpdate.status = status
	m.conclusionUpdate.conclusion = conclusion
	m.conclusionUpdate.called = true
	return nil
}

type mockJobExecEnqueuer struct {
	payloads []WorkflowJobExecPayload
}

func (m *mockJobExecEnqueuer) EnqueueJobExec(_ context.Context, payload WorkflowJobExecPayload) error {
	m.payloads = append(m.payloads, payload)
	return nil
}

func TestScheduleHandler_EnqueuesReadyJobs(t *testing.T) {
	jobRepo := &mockScheduleJobRepo{
		jobs: []*schedulableJob{
			{ID: "job-a", Name: "A", Status: entity.WorkflowJobStatusQueued},
			{ID: "job-b", Name: "B", Status: entity.WorkflowJobStatusQueued, Needs: []string{"A"}},
			{ID: "job-c", Name: "C", Status: entity.WorkflowJobStatusQueued, Needs: []string{"B"}},
		},
	}
	runRepo := &mockScheduleRunRepo{}
	enqueuer := &mockJobExecEnqueuer{}

	handler := NewWorkflowScheduleHandlerWithEnqueuer(jobRepo, runRepo, enqueuer)

	task, err := newScheduleTask(WorkflowSchedulePayload{RunID: "run-1", OrgID: "org-1"})
	if err != nil {
		t.Fatalf("newScheduleTask: %v", err)
	}

	if err := handler.HandleWorkflowSchedule(context.Background(), task); err != nil {
		t.Fatalf("HandleWorkflowSchedule: %v", err)
	}

	if len(enqueuer.payloads) != 1 {
		t.Fatalf("expected 1 enqueued job, got %d", len(enqueuer.payloads))
	}
	if enqueuer.payloads[0].JobID != "job-a" {
		t.Fatalf("expected job-a enqueued, got %q", enqueuer.payloads[0].JobID)
	}
	if len(runRepo.statusUpdates) != 1 || runRepo.statusUpdates[0] != runStatusInProgress {
		t.Fatalf("expected run status in_progress, got %v", runRepo.statusUpdates)
	}
}

func TestScheduleHandler_AllJobsComplete_UpdatesRunConclusion(t *testing.T) {
	jobRepo := &mockScheduleJobRepo{
		jobs: []*schedulableJob{
			{ID: "job-a", Name: "A", Status: entity.WorkflowJobStatusCompleted, Conclusion: conclusionSuccess},
			{ID: "job-b", Name: "B", Status: entity.WorkflowJobStatusCompleted, Conclusion: conclusionSuccess, Needs: []string{"A"}},
		},
	}
	runRepo := &mockScheduleRunRepo{}
	enqueuer := &mockJobExecEnqueuer{}

	handler := NewWorkflowScheduleHandlerWithEnqueuer(jobRepo, runRepo, enqueuer)

	task, err := newScheduleTask(WorkflowSchedulePayload{RunID: "run-1", OrgID: "org-1"})
	if err != nil {
		t.Fatalf("newScheduleTask: %v", err)
	}

	if err := handler.HandleWorkflowSchedule(context.Background(), task); err != nil {
		t.Fatalf("HandleWorkflowSchedule: %v", err)
	}

	if len(enqueuer.payloads) != 0 {
		t.Fatalf("expected no enqueued jobs, got %d", len(enqueuer.payloads))
	}
	if !runRepo.conclusionUpdate.called {
		t.Fatal("expected run conclusion update")
	}
	if runRepo.conclusionUpdate.conclusion != conclusionSuccess {
		t.Fatalf("expected success conclusion, got %q", runRepo.conclusionUpdate.conclusion)
	}
}

func TestScheduleHandler_AnyJobFailed_ConclusionIsFailure(t *testing.T) {
	jobRepo := &mockScheduleJobRepo{
		jobs: []*schedulableJob{
			{ID: "job-a", Name: "A", Status: entity.WorkflowJobStatusCompleted, Conclusion: conclusionSuccess},
			{ID: "job-b", Name: "B", Status: entity.WorkflowJobStatusFailed, Conclusion: conclusionFailure, Needs: []string{"A"}},
		},
	}
	runRepo := &mockScheduleRunRepo{}
	enqueuer := &mockJobExecEnqueuer{}

	handler := NewWorkflowScheduleHandlerWithEnqueuer(jobRepo, runRepo, enqueuer)

	task, err := newScheduleTask(WorkflowSchedulePayload{RunID: "run-1", OrgID: "org-1"})
	if err != nil {
		t.Fatalf("newScheduleTask: %v", err)
	}

	if err := handler.HandleWorkflowSchedule(context.Background(), task); err != nil {
		t.Fatalf("HandleWorkflowSchedule: %v", err)
	}

	if runRepo.conclusionUpdate.conclusion != conclusionFailure {
		t.Fatalf("expected failure conclusion, got %q", runRepo.conclusionUpdate.conclusion)
	}
}

func TestScheduleHandler_DependencyCycle_MarksRunFailure(t *testing.T) {
	jobRepo := &mockScheduleJobRepo{
		jobs: []*schedulableJob{
			{ID: "job-a", Name: "A", Status: entity.WorkflowJobStatusQueued, Needs: []string{"B"}},
			{ID: "job-b", Name: "B", Status: entity.WorkflowJobStatusQueued, Needs: []string{"A"}},
		},
	}
	runRepo := &mockScheduleRunRepo{}
	enqueuer := &mockJobExecEnqueuer{}

	handler := NewWorkflowScheduleHandlerWithEnqueuer(jobRepo, runRepo, enqueuer)

	task, err := newScheduleTask(WorkflowSchedulePayload{RunID: "run-1", OrgID: "org-1"})
	if err != nil {
		t.Fatalf("newScheduleTask: %v", err)
	}

	if err := handler.HandleWorkflowSchedule(context.Background(), task); err != nil {
		t.Fatalf("HandleWorkflowSchedule: %v", err)
	}

	if len(enqueuer.payloads) != 0 {
		t.Fatalf("expected no enqueued jobs for cycle, got %d", len(enqueuer.payloads))
	}
	if !runRepo.conclusionUpdate.called {
		t.Fatal("expected run conclusion update for dependency cycle")
	}
	if runRepo.conclusionUpdate.conclusion != conclusionFailure {
		t.Fatalf("expected failure conclusion for cycle, got %q", runRepo.conclusionUpdate.conclusion)
	}
}

func TestComputeRunConclusion_FailureOverCancelled(t *testing.T) {
	_, conclusion := computeRunConclusion([]*schedulableJob{
		{Conclusion: conclusionCancelled, Status: "cancelled"},
		{Conclusion: conclusionFailure, Status: entity.WorkflowJobStatusFailed},
	})
	if conclusion != conclusionFailure {
		t.Fatalf("expected failure conclusion, got %q", conclusion)
	}
}

func TestComputeRunConclusion_CancelledWhenNoFailure(t *testing.T) {
	_, conclusion := computeRunConclusion([]*schedulableJob{
		{Conclusion: conclusionSuccess, Status: entity.WorkflowJobStatusCompleted},
		{Conclusion: conclusionCancelled, Status: "cancelled"},
	})
	if conclusion != conclusionCancelled {
		t.Fatalf("expected cancelled conclusion, got %q", conclusion)
	}
}

func TestComputeRunConclusion_TimedOutIsFailure(t *testing.T) {
	_, conclusion := computeRunConclusion([]*schedulableJob{
		{Conclusion: conclusionTimedOut, Status: entity.WorkflowJobStatusCompleted},
	})
	if conclusion != conclusionFailure {
		t.Fatalf("expected failure conclusion for timed out job, got %q", conclusion)
	}
}

func newScheduleTask(payload WorkflowSchedulePayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeWorkflowSchedule, data), nil
}
