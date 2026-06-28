package issue

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
)

const maxCommentBodyLength = 65536

type UpdateCommentInput struct {
	CommentID      uuid.UUID
	OrganizationID uuid.UUID
	ActorID        uuid.UUID
	Body           string
}

type UpdateCommentUsecase struct {
	commentRepo  repository.ICommentRepository
	auditLogRepo repository.IAuditLogRepository
}

func NewUpdateCommentUsecase(
	commentRepo repository.ICommentRepository,
	auditLogRepo repository.IAuditLogRepository,
) *UpdateCommentUsecase {
	return &UpdateCommentUsecase{
		commentRepo:  commentRepo,
		auditLogRepo: auditLogRepo,
	}
}

func (uc *UpdateCommentUsecase) Execute(ctx context.Context, input UpdateCommentInput) (*entity.Comment, error) {
	comment, err := uc.commentRepo.GetByID(ctx, input.CommentID)
	if err != nil {
		return nil, err
	}
	if comment == nil {
		return nil, apperror.ErrNotFound
	}

	if len(input.Body) > maxCommentBodyLength {
		return nil, apperror.ErrValidation
	}

	comment.Body = input.Body

	if err := uc.commentRepo.Update(ctx, comment); err != nil {
		return nil, err
	}

	if err := uc.auditLogRepo.InsertAuditLog(
		ctx,
		input.OrganizationID,
		input.ActorID,
		"comment.update",
		"comment",
		comment.ID,
		json.RawMessage(`{}`),
	); err != nil {
		return nil, err
	}

	return comment, nil
}
