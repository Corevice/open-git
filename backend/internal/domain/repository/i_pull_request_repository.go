package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IPullRequestRepository interface {
	Create(ctx context.Context, pr *entity.PullRequest) error
	GetByNumber(ctx context.Context, repoID uuid.UUID, number int) (*entity.PullRequest, error)
	ListByRepo(ctx context.Context, repoID uuid.UUID, state string, page, perPage int) ([]*entity.PullRequest, error)
	UpdateState(ctx context.Context, id uuid.UUID, state string) error
	SetMerged(ctx context.Context, id uuid.UUID, mergedAt time.Time) error
}
