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

type createMockCommentRepo struct {
	comments []*entity.Comment
}

func (m *createMockCommentRepo) Create(_ context.Context, comment *entity.Comment) error {
	m.comments = append(m.comments, comment)
	return nil
}

func (m *createMockCommentRepo) GetByID(context.Context, uuid.UUID) (*entity.Comment, error) {
	return nil, nil
}

func (m *createMockCommentRepo) ListByIssue(context.Context, uuid.UUID, int, int) ([]*entity.Comment, int, error) {
	return nil, 0, nil
}

func (m *createMockCommentRepo) Update(context.Context, *entity.Comment) error {
	return nil
}

func (m *createMockCommentRepo) Delete(context.Context, uuid.UUID) error {
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

func (m *commentIssueRepo) GetByID(context.Context, uuid.UUID) (*entity.Issue, error) {
	return m.issue, nil
}

func (m *commentIssueRepo) Update(context.Context, *entity.Issue) error {
	return nil
}

func (m *commentIssueRepo) Delete(context.Context, uuid.UUID) error {
	return nil
}

func (m *commentIssueRepo) Count(context.Context, repository.ListIssuesFilter) (int, error) {
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

func (commentAuditLogRepo) Create(context.Context, *entity.AuditLog) error {
	return nil
}

func (commentAuditLogRepo) List(context.Context, uuid.UUID, string, int, int) ([]*entity.AuditLog, int, error) {
	return nil, 0, nil
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
		&createMockCommentRepo{},
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
