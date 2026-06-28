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

type listCommentsIssueRepo struct {
	issue *entity.Issue
}

func (m *listCommentsIssueRepo) Create(_ context.Context, _ *entity.Issue) error {
	return nil
}

func (m *listCommentsIssueRepo) GetByNumber(_ context.Context, _ uuid.UUID, _ int) (*entity.Issue, error) {
	return m.issue, nil
}

func (m *listCommentsIssueRepo) ListByRepo(_ context.Context, _ repository.ListIssuesFilter) ([]*entity.Issue, int, error) {
	return nil, 0, nil
}

func (m *listCommentsIssueRepo) NextNumber(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (m *listCommentsIssueRepo) GetByID(_ context.Context, _ uuid.UUID) (*entity.Issue, error) {
	return nil, nil
}

func (m *listCommentsIssueRepo) Update(_ context.Context, _ *entity.Issue) error {
	return nil
}

func (m *listCommentsIssueRepo) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *listCommentsIssueRepo) Count(_ context.Context, _ repository.ListIssuesFilter) (int, error) {
	return 0, nil
}

type listCommentsRepo struct {
	comments []*entity.Comment
	total    int
}

func (m *listCommentsRepo) Create(_ context.Context, _ *entity.Comment) error {
	return nil
}

func (m *listCommentsRepo) ListByIssue(_ context.Context, _ uuid.UUID, page, perPage int) ([]*entity.Comment, int, error) {
	start := (page - 1) * perPage
	if start >= len(m.comments) {
		return []*entity.Comment{}, m.total, nil
	}
	end := start + perPage
	if end > len(m.comments) {
		end = len(m.comments)
	}
	return m.comments[start:end], m.total, nil
}

func (m *listCommentsRepo) GetByID(_ context.Context, _ uuid.UUID) (*entity.Comment, error) {
	return nil, nil
}

func (m *listCommentsRepo) Update(_ context.Context, _ *entity.Comment) error {
	return nil
}

func (m *listCommentsRepo) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

func TestListCommentsNotFoundWhenIssueMissing(t *testing.T) {
	uc := issueusecase.NewListCommentsUsecase(
		&listCommentsIssueRepo{issue: nil},
		&listCommentsRepo{},
	)

	_, err := uc.Execute(context.Background(), issueusecase.ListCommentsInput{
		OrganizationID: uuid.New(),
		RepositoryID:   uuid.New(),
		IssueNumber:    1,
		Page:           1,
		PerPage:        30,
	})
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListCommentsEmptySliceWhenNoComments(t *testing.T) {
	orgID := uuid.New()
	repoID := uuid.New()
	issue := &entity.Issue{
		ID:             uuid.New(),
		OrganizationID: orgID,
		RepositoryID:   repoID,
		Number:         1,
		State:          "open",
	}

	uc := issueusecase.NewListCommentsUsecase(
		&listCommentsIssueRepo{issue: issue},
		&listCommentsRepo{comments: nil, total: 0},
	)

	output, err := uc.Execute(context.Background(), issueusecase.ListCommentsInput{
		OrganizationID: orgID,
		RepositoryID:   repoID,
		IssueNumber:    1,
		Page:           1,
		PerPage:        30,
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if output.Comments == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(output.Comments) != 0 {
		t.Fatalf("expected 0 comments, got %d", len(output.Comments))
	}
}

func TestListCommentsReturnsCommentsAndPage(t *testing.T) {
	orgID := uuid.New()
	repoID := uuid.New()
	issueID := uuid.New()
	issue := &entity.Issue{
		ID:             issueID,
		OrganizationID: orgID,
		RepositoryID:   repoID,
		Number:         1,
		State:          "open",
	}
	comments := []*entity.Comment{
		{ID: uuid.New(), IssueID: issueID, Body: "first"},
		{ID: uuid.New(), IssueID: issueID, Body: "second"},
	}

	uc := issueusecase.NewListCommentsUsecase(
		&listCommentsIssueRepo{issue: issue},
		&listCommentsRepo{comments: comments, total: 2},
	)

	output, err := uc.Execute(context.Background(), issueusecase.ListCommentsInput{
		OrganizationID: orgID,
		RepositoryID:   repoID,
		IssueNumber:    1,
		Page:           1,
		PerPage:        30,
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(output.Comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(output.Comments))
	}
	if output.Total != 2 {
		t.Fatalf("expected total 2, got %d", output.Total)
	}
	if output.Page != 1 {
		t.Fatalf("expected page 1, got %d", output.Page)
	}
	if output.PerPage != 30 {
		t.Fatalf("expected perPage 30, got %d", output.PerPage)
	}
}
