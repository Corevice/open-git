package issue_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	issueusecase "github.com/open-git/backend/internal/usecase/issue"
)

func TestDeleteCommentNotFoundWhenNil(t *testing.T) {
	uc := issueusecase.NewDeleteCommentUsecase(
		&mockCommentRepo{comment: nil},
		&mockCommentAuditLogRepo{},
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

func TestDeleteCommentNotFoundWhenOrgMismatch(t *testing.T) {
	commentID := uuid.New()
	orgID := uuid.New()
	otherOrgID := uuid.New()
	actorID := uuid.New()

	uc := issueusecase.NewDeleteCommentUsecase(
		&mockCommentRepo{
			comment: &entity.Comment{
				ID:             commentID,
				OrganizationID: orgID,
				AuthorID:       actorID,
				Body:           "to delete",
			},
		},
		&mockCommentAuditLogRepo{},
	)

	err := uc.Execute(context.Background(), issueusecase.DeleteCommentInput{
		CommentID:      commentID,
		OrganizationID: otherOrgID,
		ActorID:        actorID,
	})
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteCommentForbiddenWhenActorMismatch(t *testing.T) {
	commentID := uuid.New()
	orgID := uuid.New()
	authorID := uuid.New()
	otherActorID := uuid.New()

	uc := issueusecase.NewDeleteCommentUsecase(
		&mockCommentRepo{
			comment: &entity.Comment{
				ID:             commentID,
				OrganizationID: orgID,
				AuthorID:       authorID,
				Body:           "to delete",
			},
		},
		&mockCommentAuditLogRepo{},
	)

	err := uc.Execute(context.Background(), issueusecase.DeleteCommentInput{
		CommentID:      commentID,
		OrganizationID: orgID,
		ActorID:        otherActorID,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestDeleteCommentSuccess(t *testing.T) {
	commentID := uuid.New()
	orgID := uuid.New()
	actorID := uuid.New()
	commentRepo := &mockCommentRepo{
		comment: &entity.Comment{
			ID:             commentID,
			OrganizationID: orgID,
			AuthorID:       actorID,
			Body:           "to delete",
		},
	}
	auditRepo := &mockCommentAuditLogRepo{}

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
