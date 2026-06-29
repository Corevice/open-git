package workflow_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/queue"
	wfparser "github.com/open-git/backend/internal/infrastructure/workflow"
	"github.com/open-git/backend/internal/usecase/workflow"
)

type mockWorkflowRepo struct {
	workflows []*entity.Workflow
}

func (m *mockWorkflowRepo) ListActiveByRepository(_ context.Context, _, _ uuid.UUID) ([]*entity.Workflow, error) {
	return m.workflows, nil
}

var _ domainrepo.IWorkflowRepository = (*mockWorkflowRepo)(nil)

type mockWorkflowRunRepo struct {
	created       []*entity.WorkflowRun
	incremented   int
	lastRunNumber int
}

func (m *mockWorkflowRunRepo) ListByHeadSHA(context.Context, uuid.UUID, string) ([]*entity.WorkflowRun, error) {
	return nil, nil
}

func (m *mockWorkflowRunRepo) Create(_ context.Context, run *entity.WorkflowRun) error {
	copyRun := *run
	m.created = append(m.created, &copyRun)
	return nil
}

func (m *mockWorkflowRunRepo) GetByID(context.Context, uuid.UUID, uuid.UUID) (*entity.WorkflowRun, error) {
	return nil, domain.ErrNotFound
}

func (m *mockWorkflowRunRepo) Update(context.Context, *entity.WorkflowRun) error {
	return nil
}

func (m *mockWorkflowRunRepo) IncrementRunNumber(_ context.Context, _, _ uuid.UUID) (int, error) {
	m.incremented++
	m.lastRunNumber = m.incremented
	return m.lastRunNumber, nil
}

func (m *mockWorkflowRunRepo) IncrementRunAttempt(context.Context, uuid.UUID, uuid.UUID) (int, error) {
	return 0, nil
}

var _ domainrepo.IWorkflowRunRepository = (*mockWorkflowRunRepo)(nil)

type mockWorkflowJobRepo struct {
	created []*entity.WorkflowJob
}

func (m *mockWorkflowJobRepo) Create(_ context.Context, job *entity.WorkflowJob) error {
	copyJob := *job
	m.created = append(m.created, &copyJob)
	return nil
}

func (m *mockWorkflowJobRepo) CreateBatch(_ context.Context, jobs []*entity.WorkflowJob) error {
	for _, job := range jobs {
		copyJob := *job
		m.created = append(m.created, &copyJob)
	}
	return nil
}

func (m *mockWorkflowJobRepo) GetByID(context.Context, uuid.UUID) (*entity.WorkflowJob, error) {
	return nil, domain.ErrNotFound
}

func (m *mockWorkflowJobRepo) AcquireForRunner(context.Context, uuid.UUID, uuid.UUID, int) (bool, error) {
	return false, nil
}

func (m *mockWorkflowJobRepo) UpdateStatus(context.Context, uuid.UUID, string, string) error {
	return nil
}

func (m *mockWorkflowJobRepo) Complete(context.Context, uuid.UUID, string, time.Time) error {
	return nil
}

func (m *mockWorkflowJobRepo) Cancel(context.Context, uuid.UUID) error {
	return nil
}

func (m *mockWorkflowJobRepo) ListQueued(context.Context, uuid.UUID) ([]*entity.WorkflowJob, error) {
	return nil, nil
}

func (m *mockWorkflowJobRepo) ListByRunID(context.Context, uuid.UUID, uuid.UUID) ([]*entity.WorkflowJob, error) {
	return nil, nil
}

var _ domainrepo.IWorkflowJobRepository = (*mockWorkflowJobRepo)(nil)

type mockWorkflowStepRepo struct {
	batches [][]*entity.WorkflowStep
}

func (m *mockWorkflowStepRepo) CreateBatch(_ context.Context, steps []*entity.WorkflowStep) error {
	copied := make([]*entity.WorkflowStep, len(steps))
	for i, step := range steps {
		copyStep := *step
		copied[i] = &copyStep
	}
	m.batches = append(m.batches, copied)
	return nil
}

func (m *mockWorkflowStepRepo) ResetQueuedByRunID(context.Context, uuid.UUID) error {
	return nil
}

var _ domainrepo.IWorkflowStepRepository = (*mockWorkflowStepRepo)(nil)

type mockRepoLookup struct {
	repo *entity.Repository
}

func (m *mockRepoLookup) GetByID(_ context.Context, _, _ uuid.UUID) (*entity.Repository, error) {
	return m.repo, nil
}

type mockScheduleEnqueuer struct {
	payloads []queue.WorkflowSchedulePayload
}

func (m *mockScheduleEnqueuer) EnqueueSchedule(_ context.Context, payload queue.WorkflowSchedulePayload) error {
	m.payloads = append(m.payloads, payload)
	return nil
}

type stubWorkflowParser struct {
	ir  *wfparser.WorkflowIR
	err error
}

func (s stubWorkflowParser) Parse([]byte) (*wfparser.Workflow, error) {
	return &wfparser.Workflow{Name: "CI"}, nil
}

func (s stubWorkflowParser) Analyze(*wfparser.Workflow) (*wfparser.WorkflowIR, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.ir, nil
}

func newTestTriggerUsecase(
	workflowRepo *mockWorkflowRepo,
	runRepo *mockWorkflowRunRepo,
	jobRepo *mockWorkflowJobRepo,
	stepRepo *mockWorkflowStepRepo,
	enqueuer *mockScheduleEnqueuer,
	parser stubWorkflowParser,
) *workflow.TriggerWorkflowUsecase {
	return workflow.NewTriggerWorkflowUsecaseWithDeps(
		workflowRepo,
		runRepo,
		jobRepo,
		stepRepo,
		&mockRepoLookup{repo: &entity.Repository{GitPath: "/tmp/repo.git"}},
		enqueuer,
		parser,
		func(_, _, _ string) ([]byte, error) {
			return []byte("on: push\njobs:\n  build:\n    steps:\n      - run: echo ok\n"), nil
		},
	)
}

func TestTrigger_PushMatchesBranchFilter_CreatesRun(t *testing.T) {
	orgID := uuid.New()
	repoID := uuid.New()
	workflowID := uuid.New()

	workflowRepo := &mockWorkflowRepo{
		workflows: []*entity.Workflow{{
			ID:             workflowID,
			OrganizationID: orgID,
			RepositoryID:   repoID,
			Path:           ".github/workflows/ci.yml",
			State:          "active",
		}},
	}
	runRepo := &mockWorkflowRunRepo{}
	jobRepo := &mockWorkflowJobRepo{}
	stepRepo := &mockWorkflowStepRepo{}
	enqueuer := &mockScheduleEnqueuer{}
	parser := stubWorkflowParser{
		ir: &wfparser.WorkflowIR{
			On: map[string]any{
				"push": map[string]any{
					"branches": []any{"main"},
				},
			},
			Jobs: map[string]wfparser.IRJob{
				"build": {
					RunsOn: "ubuntu-latest",
					Steps: []wfparser.IRStep{
						{Run: "echo ok"},
					},
				},
			},
			DAG: wfparser.DAGInfo{Order: []string{"build"}},
		},
	}

	uc := newTestTriggerUsecase(workflowRepo, runRepo, jobRepo, stepRepo, enqueuer, parser)
	out, err := uc.Execute(context.Background(), workflow.TriggerWorkflowInput{
		RepositoryID: repoID,
		OrgID:        orgID,
		ActorID:      uuid.New(),
		Event:        "push",
		HeadSHA:      "abc123",
		HeadBranch:   "main",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if out.RunID == uuid.Nil {
		t.Fatal("expected run ID to be set")
	}
	if len(runRepo.created) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runRepo.created))
	}
	if runRepo.created[0].Status != "queued" {
		t.Fatalf("expected queued status, got %q", runRepo.created[0].Status)
	}
	if len(enqueuer.payloads) != 1 {
		t.Fatalf("expected 1 schedule enqueue, got %d", len(enqueuer.payloads))
	}
	if enqueuer.payloads[0].RunID != out.RunID.String() {
		t.Fatalf("expected enqueued run ID %s, got %s", out.RunID, enqueuer.payloads[0].RunID)
	}
	if len(jobRepo.created) != 1 {
		t.Fatalf("expected 1 workflow job to be created")
	}
}

func TestTrigger_BranchMismatch_NoRun(t *testing.T) {
	orgID := uuid.New()
	repoID := uuid.New()

	workflowRepo := &mockWorkflowRepo{
		workflows: []*entity.Workflow{{
			ID:           uuid.New(),
			RepositoryID: repoID,
			Path:         ".github/workflows/ci.yml",
			State:        "active",
		}},
	}
	runRepo := &mockWorkflowRunRepo{}
	parser := stubWorkflowParser{
		ir: &wfparser.WorkflowIR{
			On: map[string]any{
				"push": map[string]any{
					"branches": []any{"main"},
				},
			},
			Jobs: map[string]wfparser.IRJob{
				"build": {Steps: []wfparser.IRStep{{Run: "echo ok"}}},
			},
			DAG: wfparser.DAGInfo{Order: []string{"build"}},
		},
	}

	uc := newTestTriggerUsecase(workflowRepo, runRepo, &mockWorkflowJobRepo{}, &mockWorkflowStepRepo{}, &mockScheduleEnqueuer{}, parser)
	out, err := uc.Execute(context.Background(), workflow.TriggerWorkflowInput{
		RepositoryID: repoID,
		OrgID:        orgID,
		ActorID:      uuid.New(),
		Event:        "push",
		HeadBranch:   "dev",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if out.RunID != uuid.Nil {
		t.Fatalf("expected no run ID, got %s", out.RunID)
	}
	if len(runRepo.created) != 0 {
		t.Fatalf("expected no runs created, got %d", len(runRepo.created))
	}
}

func TestTrigger_MissingJobsKey_RunIsFailure(t *testing.T) {
	orgID := uuid.New()
	repoID := uuid.New()

	workflowRepo := &mockWorkflowRepo{
		workflows: []*entity.Workflow{{
			ID:           uuid.New(),
			RepositoryID: repoID,
			Path:         ".github/workflows/ci.yml",
			State:        "active",
		}},
	}
	runRepo := &mockWorkflowRunRepo{}
	parser := stubWorkflowParser{
		ir: &wfparser.WorkflowIR{
			On:   map[string]any{"push": map[string]any{}},
			Jobs: map[string]wfparser.IRJob{},
			DAG:  wfparser.DAGInfo{},
		},
	}

	uc := newTestTriggerUsecase(workflowRepo, runRepo, &mockWorkflowJobRepo{}, &mockWorkflowStepRepo{}, &mockScheduleEnqueuer{}, parser)
	out, err := uc.Execute(context.Background(), workflow.TriggerWorkflowInput{
		RepositoryID: repoID,
		OrgID:        orgID,
		ActorID:      uuid.New(),
		Event:        "push",
		HeadBranch:   "main",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if out.RunID == uuid.Nil {
		t.Fatal("expected failure run ID")
	}
	if len(runRepo.created) != 1 {
		t.Fatalf("expected 1 failure run, got %d", len(runRepo.created))
	}
	run := runRepo.created[0]
	if run.Conclusion != "failure" {
		t.Fatalf("expected failure conclusion, got %q", run.Conclusion)
	}
	if run.ErrorMessage == "" {
		t.Fatal("expected error message on failure run")
	}
}

func TestTrigger_CircularNeeds_Rejected(t *testing.T) {
	orgID := uuid.New()
	repoID := uuid.New()

	workflowRepo := &mockWorkflowRepo{
		workflows: []*entity.Workflow{{
			ID:           uuid.New(),
			RepositoryID: repoID,
			Path:         ".github/workflows/ci.yml",
			State:        "active",
		}},
	}
	runRepo := &mockWorkflowRunRepo{}
	parser := stubWorkflowParser{
		ir: &wfparser.WorkflowIR{
			On: map[string]any{"push": map[string]any{}},
			Jobs: map[string]wfparser.IRJob{
				"a": {Needs: []string{"b"}, Steps: []wfparser.IRStep{{Run: "a"}}},
				"b": {Needs: []string{"a"}, Steps: []wfparser.IRStep{{Run: "b"}}},
			},
			DAG: wfparser.DAGInfo{Order: []string{"a"}},
		},
	}

	uc := newTestTriggerUsecase(workflowRepo, runRepo, &mockWorkflowJobRepo{}, &mockWorkflowStepRepo{}, &mockScheduleEnqueuer{}, parser)
	_, err := uc.Execute(context.Background(), workflow.TriggerWorkflowInput{
		RepositoryID: repoID,
		OrgID:        orgID,
		ActorID:      uuid.New(),
		Event:        "push",
		HeadBranch:   "main",
	})
	if err == nil {
		t.Fatal("expected circular dependency error")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if len(runRepo.created) != 0 {
		t.Fatalf("expected no run records, got %d", len(runRepo.created))
	}
}

func TestTrigger_RefsHeadsMainMatchesMainFilter(t *testing.T) {
	orgID := uuid.New()
	repoID := uuid.New()

	workflowRepo := &mockWorkflowRepo{
		workflows: []*entity.Workflow{{
			ID:           uuid.New(),
			RepositoryID: repoID,
			Path:         ".github/workflows/ci.yml",
			State:        "active",
		}},
	}
	runRepo := &mockWorkflowRunRepo{}
	parser := stubWorkflowParser{
		ir: &wfparser.WorkflowIR{
			On: map[string]any{
				"push": map[string]any{
					"branches": []any{"main"},
				},
			},
			Jobs: map[string]wfparser.IRJob{
				"build": {Steps: []wfparser.IRStep{{Run: "echo ok"}}},
			},
			DAG: wfparser.DAGInfo{Order: []string{"build"}},
		},
	}

	uc := newTestTriggerUsecase(workflowRepo, runRepo, &mockWorkflowJobRepo{}, &mockWorkflowStepRepo{}, &mockScheduleEnqueuer{}, parser)
	out, err := uc.Execute(context.Background(), workflow.TriggerWorkflowInput{
		RepositoryID: repoID,
		OrgID:        orgID,
		ActorID:      uuid.New(),
		Event:        "push",
		HeadBranch:   "refs/heads/main",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if out.RunID == uuid.Nil {
		t.Fatal("expected run to be created for refs/heads/main")
	}
}

func TestTrigger_FeatureFilterDoesNotMatchMainBranch(t *testing.T) {
	orgID := uuid.New()
	repoID := uuid.New()

	workflowRepo := &mockWorkflowRepo{
		workflows: []*entity.Workflow{{
			ID:           uuid.New(),
			RepositoryID: repoID,
			Path:         ".github/workflows/ci.yml",
			State:        "active",
		}},
	}
	runRepo := &mockWorkflowRunRepo{}
	parser := stubWorkflowParser{
		ir: &wfparser.WorkflowIR{
			On: map[string]any{
				"push": map[string]any{
					"branches": []any{"feature"},
				},
			},
			Jobs: map[string]wfparser.IRJob{
				"build": {Steps: []wfparser.IRStep{{Run: "echo ok"}}},
			},
			DAG: wfparser.DAGInfo{Order: []string{"build"}},
		},
	}

	uc := newTestTriggerUsecase(workflowRepo, runRepo, &mockWorkflowJobRepo{}, &mockWorkflowStepRepo{}, &mockScheduleEnqueuer{}, parser)
	out, err := uc.Execute(context.Background(), workflow.TriggerWorkflowInput{
		RepositoryID: repoID,
		OrgID:        orgID,
		ActorID:      uuid.New(),
		Event:        "push",
		HeadBranch:   "refs/heads/main",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if out.RunID != uuid.Nil {
		t.Fatalf("expected no run, got %s", out.RunID)
	}
}
