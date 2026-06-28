package repository

import (
	"context"

	"github.com/google/uuid"
)

type IReviewRepository interface {
	CountSatisfiedReviews(ctx context.Context, prID uuid.UUID) (int, error)
}
