package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IReviewRepository interface {
	Create(ctx context.Context, r *entity.Review) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Review, error)
	ListByPR(ctx context.Context, prID uuid.UUID) ([]*entity.Review, error)
	CountSatisfiedReviews(ctx context.Context, prID uuid.UUID) (int, error)
	HasBlockingReviews(ctx context.Context, prID uuid.UUID) (bool, error)
}
