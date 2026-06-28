package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
)

type mockRerunRunRepo struct {
	run    *entity.WorkflowRun
	created *entity.WorkflowRun
}

func (m *mockRerunRunRepo) ListByHeadSHA(_ context.Context, _ uuid.UUID, _ string) ([]*entity.WorkflowRun, error) {
	return nil, nil
}

func (m *mockRerunRunRepo) ListByRepo(_ context.Context, _ ListWorkflowRunsFilter) ([]*entity.WorkflowRun, int, error) {
	return nil, 0, nil
}

func (m *mockRerunRunRepo) GetByID(_ context.Context, _, _ uuid.UUID) (*entity.WorkflowRun, error) {
	return m.run, nil
}

func (m *mockRerunRunRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _, _ string, _ *time.Time) error {
	return nil
}

func (m *mockRerunRunRepo) Create(_ context.Context, run *entity.WorkflowRun) error {
	m.created = run
	return nil
}

func TestRerunRunUsecase_ErrConflictWhenQueued(t *testing.T) {
	runID := uuid.New()
	repo := &mockRerunRunRepo{
		run: &entity.WorkflowRun{
			ID:     runID,
			Status: "queued",
		},
	}
	uc := NewRerunRunUsecase(repo)

	_, err := uc.Execute(context.Background(), RerunRunInput{
		OrganizationID: uuid.New(),
		RunID:          runID,
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestRerunRunUsecase_SuccessCreatesQueuedRun(t *testing.T) {
	runID := uuid.New()
	orgID := uuid.New()
	repoID := uuid.New()
	repo := &mockRerunRunRepo{
		run: &entity.WorkflowRun{
			ID:             runID,
			OrganizationID: orgID,
			RepositoryID:   repoID,
			Workflow:       "ci",
			Status:         "completed",
			RunNumber:      3,
			HeadSHA:        "abc123",
			HeadBranch:     "main",
			Event:          "push",
		},
	}
	uc := NewRerunRunUsecase(repo)

	out, err := uc.Execute(context.Background(), RerunRunInput{
		OrganizationID: orgID,
		RunID:          runID,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Status != "queued" {
		t.Fatalf("expected status queued, got %q", out.Status)
	}
	if repo.created == nil {
		t.Fatal("expected Create to be called")
	}
	if repo.created.Status != "queued" {
		t.Fatalf("expected created run status queued, got %q", repo.created.Status)
	}
	if repo.created.RunNumber != 4 {
		t.Fatalf("expected run number 4, got %d", repo.created.RunNumber)
	}
	if repo.created.Event != "workflow_dispatch" {
		t.Fatalf("expected event workflow_dispatch, got %q", repo.created.Event)
	}
}
