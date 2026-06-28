package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type ICommentRepository interface {
	Create(ctx context.Context, comment *entity.Comment) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Comment, error)
	ListByIssue(ctx context.Context, issueID uuid.UUID, page, perPage int) ([]*entity.Comment, int, error)
	Update(ctx context.Context, comment *entity.Comment) error
	Delete(ctx context.Context, id uuid.UUID) error
}
