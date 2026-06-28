package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
)

type mockCancelRunRepo struct {
	run          *entity.WorkflowRun
	updateStatus func(ctx context.Context, id uuid.UUID, status, conclusion string, completedAt *time.Time) error
	lastStatus   string
	lastConclusion string
	lastCompletedAt *time.Time
}

func (m *mockCancelRunRepo) ListByHeadSHA(_ context.Context, _ uuid.UUID, _ string) ([]*entity.WorkflowRun, error) {
	return nil, nil
}

func (m *mockCancelRunRepo) ListByRepo(_ context.Context, _ repository.ListWorkflowRunsFilter) ([]*entity.WorkflowRun, int, error) {
	return nil, 0, nil
}

func (m *mockCancelRunRepo) GetByID(_ context.Context, _, _ uuid.UUID) (*entity.WorkflowRun, error) {
	return m.run, nil
}

func (m *mockCancelRunRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status, conclusion string, completedAt *time.Time) error {
	m.lastStatus = status
	m.lastConclusion = conclusion
	m.lastCompletedAt = completedAt
	if m.updateStatus != nil {
		return m.updateStatus(ctx, id, status, conclusion, completedAt)
	}
	return nil
}

func (m *mockCancelRunRepo) Create(_ context.Context, _ *entity.WorkflowRun) error {
	return nil
}

func TestCancelRunUsecase_ErrConflictWhenCompleted(t *testing.T) {
	runID := uuid.New()
	repo := &mockCancelRunRepo{
		run: &entity.WorkflowRun{
			ID:     runID,
			Status: "completed",
		},
	}
	uc := NewCancelRunUsecase(repo)

	err := uc.Execute(context.Background(), CancelRunInput{
		OrganizationID: uuid.New(),
		RunID:          runID,
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestCancelRunUsecase_SuccessUpdatesStatus(t *testing.T) {
	runID := uuid.New()
	repo := &mockCancelRunRepo{
		run: &entity.WorkflowRun{
			ID:     runID,
			Status: "in_progress",
		},
	}
	uc := NewCancelRunUsecase(repo)

	err := uc.Execute(context.Background(), CancelRunInput{
		OrganizationID: uuid.New(),
		RunID:          runID,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if repo.lastStatus != "completed" {
		t.Fatalf("expected status completed, got %q", repo.lastStatus)
	}
	if repo.lastConclusion != "cancelled" {
		t.Fatalf("expected conclusion cancelled, got %q", repo.lastConclusion)
	}
	if repo.lastCompletedAt == nil {
		t.Fatal("expected non-nil completedAt")
	}
}
