package issue_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
	issueusecase "github.com/open-git/backend/internal/usecase/issue"
	"github.com/google/uuid"
)

type mockCommentRepo struct {
	comments []*entity.Comment
}

func (m *mockCommentRepo) Create(_ context.Context, comment *entity.Comment) error {
	m.comments = append(m.comments, comment)
	return nil
}

func (m *mockCommentRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.Comment, error) {
	for _, comment := range m.comments {
		if comment.ID == id {
			return comment, nil
		}
	}
	return nil, nil
}

func (m *mockCommentRepo) ListByIssue(_ context.Context, _ uuid.UUID, _, _ int) ([]*entity.Comment, int, error) {
	return m.comments, len(m.comments), nil
}

func (m *mockCommentRepo) Update(_ context.Context, comment *entity.Comment) error {
	for i, existing := range m.comments {
		if existing.ID == comment.ID {
			m.comments[i] = comment
			return nil
		}
	}
	return nil
}

func (m *mockCommentRepo) Delete(_ context.Context, id uuid.UUID) error {
	for i, comment := range m.comments {
		if comment.ID == id {
			m.comments = append(m.comments[:i], m.comments[i+1:]...)
			return nil
		}
	}
	return nil
}

type commentIssueRepo struct {
	issue *entity.Issue
}

func (m *commentIssueRepo) Create(_ context.Context, _ *entity.Issue) error {
	return nil
}

func (m *commentIssueRepo) GetByNumber(_ context.Context, _ uuid.UUID, _ int) (*entity.Issue, error) {
	return m.issue, nil
}

func (m *commentIssueRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.Issue, error) {
	if m.issue != nil && m.issue.ID == id {
		return m.issue, nil
	}
	return nil, nil
}

func (m *commentIssueRepo) ListByRepo(_ context.Context, _ repository.ListIssuesFilter) ([]*entity.Issue, int, error) {
	return nil, 0, nil
}

func (m *commentIssueRepo) Update(_ context.Context, _ *entity.Issue) error {
	return nil
}

func (m *commentIssueRepo) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *commentIssueRepo) Count(_ context.Context, _ repository.ListIssuesFilter) (int, error) {
	return 0, nil
}

func (m *commentIssueRepo) NextNumber(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

type commentAuditLogRepo struct{}

func (commentAuditLogRepo) InsertAuditLog(
	_ context.Context,
	_, _ uuid.UUID,
	_, _ string,
	_ uuid.UUID,
	_ json.RawMessage,
) error {
	return nil
}

func TestCommentOnDeletedIssue(t *testing.T) {
	repoID := uuid.New()
	issue := &entity.Issue{
		ID:           uuid.New(),
		RepositoryID: repoID,
		Number:       1,
		State:        "deleted",
	}

	uc := issueusecase.NewCreateCommentUsecase(
		&commentIssueRepo{issue: issue},
		&mockCommentRepo{},
		commentAuditLogRepo{},
	)

	_, err := uc.Execute(context.Background(), issueusecase.CreateCommentInput{
		OrganizationID: uuid.New(),
		RepositoryID:   repoID,
		IssueNumber:    1,
		ActorID:        uuid.New(),
		Body:           "should fail",
	})
	if !errors.Is(err, apperror.ErrGone) {
		t.Fatalf("expected ErrGone, got %v", err)
	}
}
