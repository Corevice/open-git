package issue_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	issueusecase "github.com/open-git/backend/internal/usecase/issue"
)

type deleteCommentRepo struct {
	comment    *entity.Comment
	deleteCall bool
}

func (m *deleteCommentRepo) Create(_ context.Context, _ *entity.Comment) error {
	return nil
}

func (m *deleteCommentRepo) ListByIssue(_ uuid.UUID, _, _ int) ([]*entity.Comment, int, error) {
	return nil, 0, nil
}

func (m *deleteCommentRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.Comment, error) {
	if m.comment != nil && m.comment.ID == id {
		return m.comment, nil
	}
	return nil, nil
}

func (m *deleteCommentRepo) Update(_ context.Context, _ *entity.Comment) error {
	return nil
}

func (m *deleteCommentRepo) Delete(_ context.Context, _ uuid.UUID) error {
	m.deleteCall = true
	return nil
}

type deleteCommentAuditLogRepo struct {
	calls []auditLogCall
}

func (m *deleteCommentAuditLogRepo) InsertAuditLog(
	_ context.Context,
	orgID, actorID uuid.UUID,
	action, targetType string,
	targetID uuid.UUID,
	_ json.RawMessage,
) error {
	m.calls = append(m.calls, auditLogCall{
		orgID:      orgID,
		actorID:    actorID,
		action:     action,
		targetType: targetType,
		targetID:   targetID,
	})
	return nil
}

func TestDeleteCommentNotFoundWhenNil(t *testing.T) {
	uc := issueusecase.NewDeleteCommentUsecase(
		&deleteCommentRepo{comment: nil},
		&deleteCommentAuditLogRepo{},
	)

	err := uc.Execute(context.Background(), issueusecase.DeleteCommentInput{
		CommentID:      uuid.New(),
		OrganizationID: uuid.New(),
		ActorID:        uuid.New(),
	})
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteCommentSuccess(t *testing.T) {
	commentID := uuid.New()
	orgID := uuid.New()
	actorID := uuid.New()
	commentRepo := &deleteCommentRepo{
		comment: &entity.Comment{ID: commentID, Body: "to delete"},
	}
	auditRepo := &deleteCommentAuditLogRepo{}

	uc := issueusecase.NewDeleteCommentUsecase(commentRepo, auditRepo)

	err := uc.Execute(context.Background(), issueusecase.DeleteCommentInput{
		CommentID:      commentID,
		OrganizationID: orgID,
		ActorID:        actorID,
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !commentRepo.deleteCall {
		t.Fatal("expected commentRepo.Delete to be called")
	}
	if len(auditRepo.calls) != 1 {
		t.Fatalf("expected 1 audit log call, got %d", len(auditRepo.calls))
	}
	call := auditRepo.calls[0]
	if call.action != "comment.delete" || call.targetType != "comment" || call.targetID != commentID {
		t.Fatalf("unexpected audit payload: %+v", call)
	}
}
