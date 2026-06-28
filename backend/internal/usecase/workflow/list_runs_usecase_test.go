package workflow

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
)

type mockListRunsRepo struct {
	filter repository.ListWorkflowRunsFilter
	runs   []*entity.WorkflowRun
	total  int
}

func (m *mockListRunsRepo) ListByHeadSHA(_ context.Context, _ uuid.UUID, _ string) ([]*entity.WorkflowRun, error) {
	return nil, nil
}

func (m *mockListRunsRepo) ListByRepo(_ context.Context, filter repository.ListWorkflowRunsFilter) ([]*entity.WorkflowRun, int, error) {
	m.filter = filter
	return m.runs, m.total, nil
}

func (m *mockListRunsRepo) GetByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*entity.WorkflowRun, error) {
	return nil, nil
}

func (m *mockListRunsRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _, _ string, _ *time.Time) error {
	return nil
}

func (m *mockListRunsRepo) Create(_ context.Context, _ *entity.WorkflowRun) error {
	return nil
}

func TestListWorkflowRuns_DefaultPage(t *testing.T) {
	repo := &mockListRunsRepo{runs: []*entity.WorkflowRun{}, total: 0}
	uc := NewListWorkflowRunsUsecase(repo)

	out, err := uc.Execute(context.Background(), ListWorkflowRunsInput{
		OrganizationID: uuid.New(),
		RepositoryID:   uuid.New(),
		Page:           0,
		PerPage:        10,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Page != 1 {
		t.Fatalf("expected page 1, got %d", out.Page)
	}
	if repo.filter.Page != 1 {
		t.Fatalf("expected filter page 1, got %d", repo.filter.Page)
	}
}

func TestListWorkflowRuns_ClampPerPage(t *testing.T) {
	repo := &mockListRunsRepo{runs: []*entity.WorkflowRun{}, total: 0}
	uc := NewListWorkflowRunsUsecase(repo)

	out, err := uc.Execute(context.Background(), ListWorkflowRunsInput{
		OrganizationID: uuid.New(),
		RepositoryID:   uuid.New(),
		Page:           1,
		PerPage:        200,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.PerPage != 100 {
		t.Fatalf("expected per_page 100, got %d", out.PerPage)
	}
	if repo.filter.PerPage != 100 {
		t.Fatalf("expected filter per_page 100, got %d", repo.filter.PerPage)
	}
}

func TestListWorkflowRuns_EmptyResultNonNilSlice(t *testing.T) {
	repo := &mockListRunsRepo{runs: nil, total: 0}
	uc := NewListWorkflowRunsUsecase(repo)

	out, err := uc.Execute(context.Background(), ListWorkflowRunsInput{
		OrganizationID: uuid.New(),
		RepositoryID:   uuid.New(),
		Page:           1,
		PerPage:        30,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Runs == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(out.Runs) != 0 {
		t.Fatalf("expected empty slice, got %d runs", len(out.Runs))
	}
}
