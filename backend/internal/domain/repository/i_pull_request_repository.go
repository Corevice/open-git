package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type ListPullRequestsFilter struct {
	State   string
	HeadRef string
	BaseRef string
	Page    int
	PerPage int
}

type IPullRequestRepository interface {
	Create(ctx context.Context, pr *entity.PullRequest) error
	GetByNumber(ctx context.Context, repoID uuid.UUID, number int) (*entity.PullRequest, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entity.PullRequest, error)
	ListByRepo(ctx context.Context, repoID uuid.UUID, filter ListPullRequestsFilter) ([]*entity.PullRequest, int, error)
	NextNumber(ctx context.Context, repoID uuid.UUID) (int, error)
	Update(ctx context.Context, pr *entity.PullRequest) error
	SetMerged(ctx context.Context, id uuid.UUID, mergedAt time.Time, mergedBy uuid.UUID, sha string) error
}
