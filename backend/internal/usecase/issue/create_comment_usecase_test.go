package issue_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Corevice/open-git/backend/internal/apperror"
	"github.com/Corevice/open-git/backend/internal/domain/entity"
	"github.com/Corevice/open-git/backend/internal/domain/repository"
	issueusecase "github.com/Corevice/open-git/backend/internal/usecase/issue"
	"github.com/google/uuid"
)

type mockCommentRepo struct {
	comments []*entity.Comment
}

func (m *mockCommentRepo) Create(_ context.Context, comment *entity.Comment) error {
	m.comments = append(m.comments, comment)
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

func (m *commentIssueRepo) ListByRepo(_ context.Context, _ repository.ListIssuesFilter) ([]*entity.Issue, int, error) {
	return nil, 0, nil
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
