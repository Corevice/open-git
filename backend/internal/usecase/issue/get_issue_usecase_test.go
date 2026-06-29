package issue_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
	issueusecase "github.com/open-git/backend/internal/usecase/issue"
)

type getIssueRepo struct {
	issue *entity.Issue
}

func (m *getIssueRepo) Create(_ context.Context, _ *entity.Issue) error {
	return nil
}

func (m *getIssueRepo) GetByNumber(_ context.Context, _ uuid.UUID, _ int) (*entity.Issue, error) {
	return m.issue, nil
}

func (m *getIssueRepo) ListByRepo(_ context.Context, _ repository.ListIssuesFilter) ([]*entity.Issue, int, error) {
	return nil, 0, nil
}

func (m *getIssueRepo) NextNumber(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (m *getIssueRepo) Update(_ context.Context, _ *entity.Issue) error {
	return nil
}

func (m *getIssueRepo) GetByID(_ context.Context, _ uuid.UUID) (*entity.Issue, error) {
	return m.issue, nil
}

func (m *getIssueRepo) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *getIssueRepo) Count(_ context.Context, _ repository.ListIssuesFilter) (int, error) {
	return 0, nil
}

func TestGetIssueNotFoundWhenNil(t *testing.T) {
	uc := issueusecase.NewGetIssueUsecase(&getIssueRepo{issue: nil})

	_, err := uc.Execute(context.Background(), issueusecase.GetIssueInput{
		OrganizationID: uuid.New(),
		RepositoryID:   uuid.New(),
		IssueNumber:    1,
	})
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetIssueNotFoundWhenDeleted(t *testing.T) {
	uc := issueusecase.NewGetIssueUsecase(&getIssueRepo{
		issue: &entity.Issue{
			ID:    uuid.New(),
			State: "deleted",
		},
	})

	_, err := uc.Execute(context.Background(), issueusecase.GetIssueInput{
		OrganizationID: uuid.New(),
		RepositoryID:   uuid.New(),
		IssueNumber:    1,
	})
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetIssueSuccess(t *testing.T) {
	repoID := uuid.New()
	issue := &entity.Issue{
		ID:           uuid.New(),
		RepositoryID: repoID,
		Number:       1,
		Title:        "Open issue",
		State:        "open",
	}

	uc := issueusecase.NewGetIssueUsecase(&getIssueRepo{issue: issue})

	got, err := uc.Execute(context.Background(), issueusecase.GetIssueInput{
		OrganizationID: uuid.New(),
		RepositoryID:   repoID,
		IssueNumber:    1,
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if got.ID != issue.ID {
		t.Fatalf("expected issue %v, got %v", issue.ID, got.ID)
	}
}
