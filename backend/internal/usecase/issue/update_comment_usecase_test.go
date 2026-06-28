package issue_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	issueusecase "github.com/open-git/backend/internal/usecase/issue"
)

type updateCommentRepo struct {
	comment    *entity.Comment
	updateCall bool
}

func (m *updateCommentRepo) Create(_ context.Context, _ *entity.Comment) error {
	return nil
}

func (m *updateCommentRepo) ListByIssue(_ context.Context, _ uuid.UUID, _, _ int) ([]*entity.Comment, int, error) {
	return nil, 0, nil
}

func (m *updateCommentRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.Comment, error) {
	if m.comment != nil && m.comment.ID == id {
		return m.comment, nil
	}
	return nil, nil
}

func (m *updateCommentRepo) Update(_ context.Context, comment *entity.Comment) error {
	m.updateCall = true
	m.comment = comment
	return nil
}

func (m *updateCommentRepo) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

type updateCommentAuditLogRepo struct {
	calls []auditLogCall
}

func (m *updateCommentAuditLogRepo) InsertAuditLog(
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

func TestUpdateCommentBodyTooLong(t *testing.T) {
	commentID := uuid.New()
	commentRepo := &updateCommentRepo{
		comment: &entity.Comment{ID: commentID, Body: "original"},
	}
	longBody := strings.Repeat("a", 65537)

	uc := issueusecase.NewUpdateCommentUsecase(commentRepo, &updateCommentAuditLogRepo{})

	_, err := uc.Execute(context.Background(), issueusecase.UpdateCommentInput{
		CommentID:      commentID,
		OrganizationID: uuid.New(),
		ActorID:        uuid.New(),
		Body:           longBody,
	})
	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
	if commentRepo.updateCall {
		t.Fatal("expected Update not to be called for invalid body")
	}
}

func TestUpdateCommentSuccess(t *testing.T) {
	commentID := uuid.New()
	orgID := uuid.New()
	actorID := uuid.New()
	commentRepo := &updateCommentRepo{
		comment: &entity.Comment{ID: commentID, Body: "original"},
	}
	auditRepo := &updateCommentAuditLogRepo{}
	newBody := "updated body"

	uc := issueusecase.NewUpdateCommentUsecase(commentRepo, auditRepo)

	got, err := uc.Execute(context.Background(), issueusecase.UpdateCommentInput{
		CommentID:      commentID,
		OrganizationID: orgID,
		ActorID:        actorID,
		Body:           newBody,
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if got.Body != newBody {
		t.Fatalf("expected body %q, got %q", newBody, got.Body)
	}
	if !commentRepo.updateCall {
		t.Fatal("expected commentRepo.Update to be called")
	}
	if len(auditRepo.calls) != 1 {
		t.Fatalf("expected 1 audit log call, got %d", len(auditRepo.calls))
	}
	call := auditRepo.calls[0]
	if call.action != "comment.update" || call.targetType != "comment" || call.targetID != commentID {
		t.Fatalf("unexpected audit payload: %+v", call)
	}
}
