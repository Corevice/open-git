package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IReviewCommentRepository interface {
	Create(ctx context.Context, c *entity.ReviewComment) error
	ListByPR(ctx context.Context, prID uuid.UUID) ([]*entity.ReviewComment, error)
	ListByReview(ctx context.Context, reviewID uuid.UUID) ([]*entity.ReviewComment, error)
}
