package repository

import (
	"context"

	"github.com/google/uuid"
)

type IReviewRepository interface {
	CountSatisfiedReviews(ctx context.Context, pullRequestID uuid.UUID) (int, error)
}
