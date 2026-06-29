package queue

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/runner"
)

type mockExecJobRepo struct {
	job           *entity.WorkflowJob
	statusUpdates []string
}

func (m *mockExecJobRepo) Create(context.Context, *entity.WorkflowJob) error { return nil }

func (m *mockExecJobRepo) CreateBatch(context.Context, []*entity.WorkflowJob) error { return nil }

func (m *mockExecJobRepo) GetByID(_ context.Context, jobID uuid.UUID) (*entity.WorkflowJob, error) {
	if m.job == nil || m.job.ID != jobID {
		return nil, errors.New("not found")
	}
	return m.job, nil
}

func (m *mockExecJobRepo) AcquireForRunner(context.Context, uuid.UUID, uuid.UUID, int) (bool, error) {
	return false, nil
}

func (m *mockExecJobRepo) UpdateStatus(_ context.Context, jobID uuid.UUID, status, conclusion string) error {
	if m.job != nil && m.job.ID == jobID {
		m.job.Status = status
		m.job.Conclusion = conclusion
	}
	m.statusUpdates = append(m.statusUpdates, status)
	return nil
}

func (m *mockExecJobRepo) Complete(_ context.Context, jobID uuid.UUID, conclusion string, _ time.Time) error {
	return m.UpdateStatus(context.Background(), jobID, entity.WorkflowJobStatusCompleted, conclusion)
}

func (m *mockExecJobRepo) Cancel(context.Context, uuid.UUID) error { return nil }

func (m *mockExecJobRepo) ListQueued(context.Context, uuid.UUID) ([]*entity.WorkflowJob, error) {
	return nil, nil
}

func (m *mockExecJobRepo) ListByRunID(context.Context, uuid.UUID, uuid.UUID) ([]*entity.WorkflowJob, error) {
	return nil, nil
}

type mockExecStepRepo struct {
	steps []*runner.Step
}

func (m *mockExecStepRepo) ListByJobID(context.Context, string, string) ([]*runner.Step, error) {
	return m.steps, nil
}

type mockExecLogRepo struct {
	mu    sync.Mutex
	lines []*entity.JobLogLine
	err   error
}

func (m *mockExecLogRepo) ListLines(context.Context, string, string, int64, int) ([]*entity.JobLogLine, error) {
	return nil, nil
}

func (m *mockExecLogRepo) CountLines(context.Context, string, string) (int64, error) {
	return 0, nil
}

func (m *mockExecLogRepo) SetMeta(context.Context, *domainrepo.JobLogMeta) error {
	return nil
}

func (m *mockExecLogRepo) GetMeta(context.Context, string, string) (*domainrepo.JobLogMeta, error) {
	return nil, nil
}

func (m *mockExecLogRepo) AppendLines(_ context.Context, lines []*entity.JobLogLine) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.lines = append(m.lines, lines...)
	return nil
}

type mockExecRunner struct {
	err error
}

func (m *mockExecRunner) ExecuteJob(
	_ context.Context,
	_ *entity.WorkflowJob,
	_ []*runner.Step,
	_ map[string]string,
	logFn func(int64, string),
) error {
	logFn(0, "line-1")
	logFn(1, "line-2")
	return m.err
}

type mockExecScheduleEnqueuer struct {
	payloads []WorkflowSchedulePayload
}

func (m *mockExecScheduleEnqueuer) EnqueueSchedule(_ context.Context, payload WorkflowSchedulePayload) error {
	m.payloads = append(m.payloads, payload)
	return nil
}

func TestJobTimeoutMinutes_UsesReflectionField(t *testing.T) {
	job := &entity.WorkflowJob{TimeoutMinutes: 42}
	if got := jobTimeoutMinutes(job); got != 42 {
		t.Fatalf("jobTimeoutMinutes() = %d, want 42", got)
	}
	if got := jobTimeoutMinutes(nil); got != defaultJobTimeoutMinutes {
		t.Fatalf("jobTimeoutMinutes(nil) = %d, want default", got)
	}
}

func TestBuildLogCallback_ConcurrentAppendIsSafe(t *testing.T) {
	logRepo := &mockExecLogRepo{}
	handler := &WorkflowJobExecHandler{logRepo: logRepo}
	jobID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	runID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	repoID := uuid.MustParse("00000000-0000-0000-0000-000000000004")
	payload := WorkflowJobExecPayload{OrgID: orgID.String(), RunID: runID.String(), JobID: jobID.String()}
	job := &entity.WorkflowJob{RepositoryID: repoID}

	logFn, logErrFn := handler.buildLogCallback(context.Background(), payload, job)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int64) {
			defer wg.Done()
			logFn(i, "chunk")
		}(int64(i))
	}
	wg.Wait()

	if err := logErrFn(); err != nil {
		t.Fatalf("logErrFn() = %v", err)
	}
	if len(logRepo.lines) != 8 {
		t.Fatalf("expected 8 log lines, got %d", len(logRepo.lines))
	}
}

func TestWorkflowJobExecHandler_RejectsRunMismatch(t *testing.T) {
	jobID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	runID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	repoID := uuid.MustParse("00000000-0000-0000-0000-000000000004")

	jobRepo := &mockExecJobRepo{
		job: &entity.WorkflowJob{
			ID:             jobID,
			WorkflowRunID:  &runID,
			OrganizationID: orgID,
			RepositoryID:   repoID,
		},
	}
	handler := NewWorkflowJobExecHandlerWithEnqueuer(
		jobRepo,
		&mockExecStepRepo{},
		nil,
		&mockExecRunner{},
		&mockExecScheduleEnqueuer{},
		nil,
	)

	task, err := newJobExecTask(WorkflowJobExecPayload{
		JobID: jobID.String(),
		RunID: uuid.MustParse("00000000-0000-0000-0000-000000000099").String(),
		OrgID: orgID.String(),
	})
	if err != nil {
		t.Fatalf("newJobExecTask: %v", err)
	}

	err = handler.HandleWorkflowJobExec(context.Background(), task)
	if err == nil {
		t.Fatal("expected run mismatch error")
	}
}

func TestWorkflowJobExecHandler_MarksTimedOutJobCompleted(t *testing.T) {
	jobID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	runID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	repoID := uuid.MustParse("00000000-0000-0000-0000-000000000004")

	jobRepo := &mockExecJobRepo{
		job: &entity.WorkflowJob{
			ID:             jobID,
			WorkflowRunID:  &runID,
			OrganizationID: orgID,
			RepositoryID:   repoID,
		},
	}
	handler := NewWorkflowJobExecHandlerWithEnqueuer(
		jobRepo,
		&mockExecStepRepo{},
		nil,
		&timeoutExecRunner{},
		&mockExecScheduleEnqueuer{},
		nil,
	)

	task, err := newJobExecTask(WorkflowJobExecPayload{
		JobID: jobID.String(),
		RunID: runID.String(),
		OrgID: orgID.String(),
	})
	if err != nil {
		t.Fatalf("newJobExecTask: %v", err)
	}

	if err := handler.HandleWorkflowJobExec(context.Background(), task); err != nil {
		t.Fatalf("HandleWorkflowJobExec: %v", err)
	}

	if jobRepo.job.Status != entity.WorkflowJobStatusCompleted {
		t.Fatalf("expected completed status on timeout, got %q", jobRepo.job.Status)
	}
	if jobRepo.job.Conclusion != conclusionTimedOut {
		t.Fatalf("expected timed_out conclusion, got %q", jobRepo.job.Conclusion)
	}
}

type timeoutExecRunner struct{}

func (timeoutExecRunner) ExecuteJob(context.Context, *entity.WorkflowJob, []*runner.Step, map[string]string, func(int64, string)) error {
	return context.DeadlineExceeded
}

func newJobExecTask(payload WorkflowJobExecPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeWorkflowJobExec, data), nil
}

func TestWorkflowJobNeeds_ReflectsNeedsField(t *testing.T) {
	// workflowJobNeeds reflects a "Needs" slice field off *entity.WorkflowJob.
	// The current entity.WorkflowJob exposes no such field, so the helper must
	// return no dependencies rather than panicking. (A nil job must also be
	// handled safely.)
	if got := workflowJobNeeds(&entity.WorkflowJob{}); len(got) != 0 {
		t.Fatalf("workflowJobNeeds() = %v, want no needs", got)
	}
	if got := workflowJobNeeds(nil); len(got) != 0 {
		t.Fatalf("workflowJobNeeds(nil) = %v, want no needs", got)
	}
}
