package issue_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	issueusecase "github.com/open-git/backend/internal/usecase/issue"
)

type mockCommentRepo struct {
	comment    *entity.Comment
	updateCall bool
	deleteCall bool
}

func (m *mockCommentRepo) Create(_ context.Context, _ *entity.Comment) error {
	return nil
}

func (m *mockCommentRepo) ListByIssue(_ context.Context, _ uuid.UUID, _, _ int) ([]*entity.Comment, int, error) {
	return nil, 0, nil
}

func (m *mockCommentRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.Comment, error) {
	if m.comment != nil && m.comment.ID == id {
		return m.comment, nil
	}
	return nil, nil
}

func (m *mockCommentRepo) Update(_ context.Context, comment *entity.Comment) error {
	m.updateCall = true
	m.comment = comment
	return nil
}

func (m *mockCommentRepo) Delete(_ context.Context, _ uuid.UUID) error {
	m.deleteCall = true
	return nil
}

type mockCommentAuditLogRepo struct {
	calls []auditLogCall
}

func (m *mockCommentAuditLogRepo) InsertAuditLog(
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

func (m *mockCommentAuditLogRepo) Create(context.Context, *entity.AuditLog) error {
	return nil
}

func (m *mockCommentAuditLogRepo) List(context.Context, uuid.UUID, string, int, int) ([]*entity.AuditLog, int, error) {
	return nil, 0, nil
}

func TestUpdateCommentNotFoundWhenNil(t *testing.T) {
	uc := issueusecase.NewUpdateCommentUsecase(
		&mockCommentRepo{comment: nil},
		&mockCommentAuditLogRepo{},
	)

	_, err := uc.Execute(context.Background(), issueusecase.UpdateCommentInput{
		CommentID:      uuid.New(),
		OrganizationID: uuid.New(),
		ActorID:        uuid.New(),
		Body:           "updated",
	})
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateCommentNotFoundWhenOrgMismatch(t *testing.T) {
	commentID := uuid.New()
	orgID := uuid.New()
	otherOrgID := uuid.New()
	actorID := uuid.New()

	uc := issueusecase.NewUpdateCommentUsecase(
		&mockCommentRepo{
			comment: &entity.Comment{
				ID:             commentID,
				OrganizationID: orgID,
				AuthorID:       actorID,
				Body:           "original",
			},
		},
		&mockCommentAuditLogRepo{},
	)

	_, err := uc.Execute(context.Background(), issueusecase.UpdateCommentInput{
		CommentID:      commentID,
		OrganizationID: otherOrgID,
		ActorID:        actorID,
		Body:           "updated",
	})
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateCommentForbiddenWhenActorMismatch(t *testing.T) {
	commentID := uuid.New()
	orgID := uuid.New()
	authorID := uuid.New()
	otherActorID := uuid.New()

	uc := issueusecase.NewUpdateCommentUsecase(
		&mockCommentRepo{
			comment: &entity.Comment{
				ID:             commentID,
				OrganizationID: orgID,
				AuthorID:       authorID,
				Body:           "original",
			},
		},
		&mockCommentAuditLogRepo{},
	)

	_, err := uc.Execute(context.Background(), issueusecase.UpdateCommentInput{
		CommentID:      commentID,
		OrganizationID: orgID,
		ActorID:        otherActorID,
		Body:           "updated",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestUpdateCommentBodyTooLong(t *testing.T) {
	commentID := uuid.New()
	orgID := uuid.New()
	actorID := uuid.New()
	commentRepo := &mockCommentRepo{
		comment: &entity.Comment{
			ID:             commentID,
			OrganizationID: orgID,
			AuthorID:       actorID,
			Body:           "original",
		},
	}
	longBody := strings.Repeat("a", 65537)

	uc := issueusecase.NewUpdateCommentUsecase(commentRepo, &mockCommentAuditLogRepo{})

	_, err := uc.Execute(context.Background(), issueusecase.UpdateCommentInput{
		CommentID:      commentID,
		OrganizationID: orgID,
		ActorID:        actorID,
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
	commentRepo := &mockCommentRepo{
		comment: &entity.Comment{
			ID:             commentID,
			OrganizationID: orgID,
			AuthorID:       actorID,
			Body:           "original",
		},
	}
	auditRepo := &mockCommentAuditLogRepo{}
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
