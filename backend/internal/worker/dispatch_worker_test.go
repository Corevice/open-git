package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/queue"
	"github.com/open-git/backend/internal/infrastructure/runner"
)

type mockRunnerAdapter struct {
	executeCalls []runner.ActJobPayload
	cancelCalls  []string
	executeErr   error
}

func (m *mockRunnerAdapter) Execute(_ context.Context, job runner.ActJobPayload) error {
	m.executeCalls = append(m.executeCalls, job)
	return m.executeErr
}

func (m *mockRunnerAdapter) Cancel(_ context.Context, jobID string) error {
	m.cancelCalls = append(m.cancelCalls, jobID)
	return nil
}

var _ runner.RunnerAdapter = (*mockRunnerAdapter)(nil)

type mockRunnerRepo struct {
	findAvailableLabels []string
	findAvailableResult *entity.Runner
	findAvailableErr    error
	findAvailableCalls  int
}

func (m *mockRunnerRepo) Create(_ context.Context, _ *entity.Runner) error { return nil }

func (m *mockRunnerRepo) GetByID(_ context.Context, _ uuid.UUID) (*entity.Runner, error) {
	return nil, domain.ErrNotFound
}

func (m *mockRunnerRepo) ListByOrg(_ context.Context, _ uuid.UUID) ([]*entity.Runner, error) {
	return nil, nil
}

func (m *mockRunnerRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}

func (m *mockRunnerRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }

func (m *mockRunnerRepo) FindAvailable(_ context.Context, _ uuid.UUID, labels []string) (*entity.Runner, error) {
	m.findAvailableCalls++
	m.findAvailableLabels = append([]string(nil), labels...)
	if m.findAvailableErr != nil {
		return nil, m.findAvailableErr
	}
	return m.findAvailableResult, nil
}

var _ domainrepo.IRunnerRepository = (*mockRunnerRepo)(nil)

type mockWorkflowJobRepo struct {
	jobs              map[uuid.UUID]*entity.WorkflowJob
	completeCalls     int
	completeJobID     uuid.UUID
	completeConclusion string
	acquireCalls      int
	cancelCalls       int
	cancelJobID       uuid.UUID
}

func (m *mockWorkflowJobRepo) Create(_ context.Context, job *entity.WorkflowJob) error {
	if m.jobs == nil {
		m.jobs = make(map[uuid.UUID]*entity.WorkflowJob)
	}
	m.jobs[job.ID] = job
	return nil
}

func (m *mockWorkflowJobRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.WorkflowJob, error) {
	if m.jobs == nil {
		return nil, domain.ErrNotFound
	}
	job, ok := m.jobs[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return job, nil
}

func (m *mockWorkflowJobRepo) AcquireForRunner(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ int) (bool, error) {
	m.acquireCalls++
	return true, nil
}

func (m *mockWorkflowJobRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _, _ string) error {
	return nil
}

func (m *mockWorkflowJobRepo) Complete(_ context.Context, jobID uuid.UUID, conclusion string, _ time.Time) error {
	m.completeCalls++
	m.completeJobID = jobID
	m.completeConclusion = conclusion
	return nil
}

func (m *mockWorkflowJobRepo) Cancel(_ context.Context, jobID uuid.UUID) error {
	m.cancelCalls++
	m.cancelJobID = jobID
	return nil
}

func (m *mockWorkflowJobRepo) ListQueued(_ context.Context, _ uuid.UUID) ([]*entity.WorkflowJob, error) {
	return nil, nil
}

var _ domainrepo.IWorkflowJobRepository = (*mockWorkflowJobRepo)(nil)

func TestHandleDispatchJob_GitHubHostedRoutesToActAdapter(t *testing.T) {
	ctx := context.Background()
	jobID := uuid.New()
	orgID := uuid.New()

	jobRepo := &mockWorkflowJobRepo{
		jobs: map[uuid.UUID]*entity.WorkflowJob{
			jobID: {
				ID:             jobID,
				OrganizationID: orgID,
				Name:           "build",
				RunsOn:         []string{"ubuntu-latest"},
			},
		},
	}
	runnerRepo := &mockRunnerRepo{}
	actAdapter := &mockRunnerAdapter{}

	worker := NewDispatchWorker(runnerRepo, jobRepo, actAdapter, nil)

	payload, err := json.Marshal(queue.DispatchJobPayload{
		JobID:          jobID.String(),
		OrganizationID: orgID.String(),
		RunsOn:         `["ubuntu-latest"]`,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	task := asynq.NewTask(queue.TypeDispatchJob, payload)
	if err := worker.HandleDispatchJob(ctx, task); err != nil {
		t.Fatalf("HandleDispatchJob returned error: %v", err)
	}

	if len(actAdapter.executeCalls) != 1 {
		t.Fatalf("expected act adapter Execute to be called once, got %d", len(actAdapter.executeCalls))
	}
	if actAdapter.executeCalls[0].JobID != jobID.String() {
		t.Fatalf("expected act payload job id %q, got %q", jobID, actAdapter.executeCalls[0].JobID)
	}
	if len(actAdapter.executeCalls[0].WorkflowYAML) == 0 {
		t.Fatal("expected act payload workflow YAML to be populated")
	}
	if runnerRepo.findAvailableCalls != 0 {
		t.Fatalf("expected FindAvailable not to be called, got %d calls", runnerRepo.findAvailableCalls)
	}
	if jobRepo.completeCalls != 1 {
		t.Fatalf("expected Complete to be called once, got %d", jobRepo.completeCalls)
	}
	if jobRepo.completeConclusion != entity.WorkflowJobConclusionSuccess {
		t.Fatalf("expected success conclusion, got %q", jobRepo.completeConclusion)
	}
}

func TestHandleDispatchJob_SelfHostedCallsFindAvailable(t *testing.T) {
	ctx := context.Background()
	jobID := uuid.New()
	orgID := uuid.New()
	runnerID := uuid.New()

	jobRepo := &mockWorkflowJobRepo{
		jobs: map[uuid.UUID]*entity.WorkflowJob{
			jobID: {
				ID:                 jobID,
				OrganizationID:     orgID,
				RunsOn:             []string{"self-hosted", "linux"},
				AcquireLockVersion: 0,
			},
		},
	}
	runnerRepo := &mockRunnerRepo{
		findAvailableResult: &entity.Runner{ID: runnerID},
	}
	actAdapter := &mockRunnerAdapter{}

	worker := NewDispatchWorker(runnerRepo, jobRepo, actAdapter, nil)

	payload, err := json.Marshal(queue.DispatchJobPayload{
		JobID:          jobID.String(),
		OrganizationID: orgID.String(),
		RunsOn:         `["self-hosted","linux"]`,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	task := asynq.NewTask(queue.TypeDispatchJob, payload)
	if err := worker.HandleDispatchJob(ctx, task); err != nil {
		t.Fatalf("HandleDispatchJob returned error: %v", err)
	}

	if runnerRepo.findAvailableCalls != 1 {
		t.Fatalf("expected FindAvailable once, got %d", runnerRepo.findAvailableCalls)
	}
	if len(actAdapter.executeCalls) != 0 {
		t.Fatalf("expected act adapter not to be called, got %d calls", len(actAdapter.executeCalls))
	}
	if jobRepo.acquireCalls != 1 {
		t.Fatalf("expected AcquireForRunner once, got %d", jobRepo.acquireCalls)
	}
}

func TestHandleDispatchJob_NoAvailableRunnerLeavesQueued(t *testing.T) {
	ctx := context.Background()
	jobID := uuid.New()
	orgID := uuid.New()

	jobRepo := &mockWorkflowJobRepo{
		jobs: map[uuid.UUID]*entity.WorkflowJob{
			jobID: {
				ID:             jobID,
				OrganizationID: orgID,
				RunsOn:         []string{"self-hosted", "linux"},
			},
		},
	}
	runnerRepo := &mockRunnerRepo{
		findAvailableErr: domain.ErrNotFound,
	}
	actAdapter := &mockRunnerAdapter{}

	worker := NewDispatchWorker(runnerRepo, jobRepo, actAdapter, nil)

	payload, err := json.Marshal(queue.DispatchJobPayload{
		JobID:          jobID.String(),
		OrganizationID: orgID.String(),
		RunsOn:         `["self-hosted","linux"]`,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	task := asynq.NewTask(queue.TypeDispatchJob, payload)
	if err := worker.HandleDispatchJob(ctx, task); err != nil {
		t.Fatalf("HandleDispatchJob returned error: %v", err)
	}

	if jobRepo.acquireCalls != 0 {
		t.Fatalf("expected AcquireForRunner not to be called, got %d", jobRepo.acquireCalls)
	}
	if jobRepo.completeCalls != 0 {
		t.Fatalf("expected Complete not to be called, got %d", jobRepo.completeCalls)
	}
}

func TestHandleCancelJob_CallsJobRepoCancelAndActAdapterCancel(t *testing.T) {
	ctx := context.Background()
	jobID := uuid.New()

	jobRepo := &mockWorkflowJobRepo{}
	actAdapter := &mockRunnerAdapter{}
	worker := NewDispatchWorker(&mockRunnerRepo{}, jobRepo, actAdapter, nil)

	payload, err := json.Marshal(queue.CancelJobPayload{JobID: jobID.String()})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	task := asynq.NewTask(queue.TypeCancelJob, payload)
	if err := worker.HandleCancelJob(ctx, task); err != nil {
		t.Fatalf("HandleCancelJob returned error: %v", err)
	}

	if jobRepo.cancelCalls != 1 {
		t.Fatalf("expected Cancel once, got %d", jobRepo.cancelCalls)
	}
	if jobRepo.cancelJobID != jobID {
		t.Fatalf("expected cancel job id %q, got %q", jobID, jobRepo.cancelJobID)
	}
	if len(actAdapter.cancelCalls) != 1 {
		t.Fatalf("expected act adapter Cancel once, got %d", len(actAdapter.cancelCalls))
	}
	if actAdapter.cancelCalls[0] != jobID.String() {
		t.Fatalf("expected act cancel job id %q, got %q", jobID, actAdapter.cancelCalls[0])
	}
}

func TestHandleDispatchJob_ActAdapterFailureCompletesWithFailure(t *testing.T) {
	ctx := context.Background()
	jobID := uuid.New()
	orgID := uuid.New()

	jobRepo := &mockWorkflowJobRepo{
		jobs: map[uuid.UUID]*entity.WorkflowJob{
			jobID: {
				ID:             jobID,
				OrganizationID: orgID,
				Name:           "build",
				RunsOn:         []string{"ubuntu-latest"},
			},
		},
	}
	actAdapter := &mockRunnerAdapter{executeErr: errors.New("act failed")}

	worker := NewDispatchWorker(&mockRunnerRepo{}, jobRepo, actAdapter, nil)

	payload, err := json.Marshal(queue.DispatchJobPayload{
		JobID:          jobID.String(),
		OrganizationID: orgID.String(),
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	task := asynq.NewTask(queue.TypeDispatchJob, payload)
	err = worker.HandleDispatchJob(ctx, task)
	if err == nil {
		t.Fatal("expected HandleDispatchJob to return act adapter error")
	}
	if jobRepo.completeCalls != 1 {
		t.Fatalf("expected Complete to be called once, got %d", jobRepo.completeCalls)
	}
	if jobRepo.completeConclusion != entity.WorkflowJobConclusionFailure {
		t.Fatalf("expected failure conclusion, got %q", jobRepo.completeConclusion)
	}
}

func TestHandleDispatchJob_OrganizationMismatch(t *testing.T) {
	ctx := context.Background()
	jobID := uuid.New()
	orgID := uuid.New()
	otherOrgID := uuid.New()

	jobRepo := &mockWorkflowJobRepo{
		jobs: map[uuid.UUID]*entity.WorkflowJob{
			jobID: {
				ID:             jobID,
				OrganizationID: orgID,
				Name:           "build",
				RunsOn:         []string{"ubuntu-latest"},
			},
		},
	}
	actAdapter := &mockRunnerAdapter{}
	worker := NewDispatchWorker(&mockRunnerRepo{}, jobRepo, actAdapter, nil)

	payload, err := json.Marshal(queue.DispatchJobPayload{
		JobID:          jobID.String(),
		OrganizationID: otherOrgID.String(),
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	task := asynq.NewTask(queue.TypeDispatchJob, payload)
	if err := worker.HandleDispatchJob(ctx, task); err == nil {
		t.Fatal("expected organization mismatch error")
	}
	if len(actAdapter.executeCalls) != 0 {
		t.Fatalf("expected act adapter not to be called, got %d calls", len(actAdapter.executeCalls))
	}
}
