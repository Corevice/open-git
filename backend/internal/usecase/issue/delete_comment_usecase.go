package issue

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/repository"
)

type DeleteCommentInput struct {
	CommentID      uuid.UUID
	OrganizationID uuid.UUID
	ActorID        uuid.UUID
}

type DeleteCommentUsecase struct {
	commentRepo  repository.ICommentRepository
	auditLogRepo repository.IAuditLogRepository
}

func NewDeleteCommentUsecase(
	commentRepo repository.ICommentRepository,
	auditLogRepo repository.IAuditLogRepository,
) *DeleteCommentUsecase {
	return &DeleteCommentUsecase{
		commentRepo:  commentRepo,
		auditLogRepo: auditLogRepo,
	}
}

func (uc *DeleteCommentUsecase) Execute(ctx context.Context, input DeleteCommentInput) error {
	comment, err := uc.commentRepo.GetByID(ctx, input.CommentID)
	if err != nil {
		return err
	}
	if comment == nil {
		return apperror.ErrNotFound
	}
	if comment.OrganizationID != input.OrganizationID {
		return apperror.ErrNotFound
	}
	if comment.AuthorID != input.ActorID {
		return domain.ErrForbidden
	}

	if err := uc.commentRepo.Delete(ctx, input.CommentID); err != nil {
		return err
	}

	if err := uc.auditLogRepo.InsertAuditLog(
		ctx,
		input.OrganizationID,
		input.ActorID,
		"comment.delete",
		"comment",
		input.CommentID,
		json.RawMessage(`{}`),
	); err != nil {
		return err
	}
	return nil
}
