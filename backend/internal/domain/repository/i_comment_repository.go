package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type ICommentRepository interface {
	Create(ctx context.Context, comment *entity.Comment) error
	ListByIssue(ctx context.Context, issueID uuid.UUID) ([]*entity.Comment, error)
}
