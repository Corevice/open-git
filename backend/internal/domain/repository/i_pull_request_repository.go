package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IPullRequestRepository interface {
	Create(ctx context.Context, pr *entity.PullRequest) error
	Update(ctx context.Context, pr *entity.PullRequest) error
	GetByNumber(ctx context.Context, repoID uuid.UUID, number int) (*entity.PullRequest, error)
	ListByRepo(ctx context.Context, filter ListPullRequestsFilter) ([]*entity.PullRequest, int, error)
	NextNumber(ctx context.Context, repoID uuid.UUID) (int, error)
}
