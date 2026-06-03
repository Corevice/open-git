package repository

import (
	"context"

	"github.com/Corevice/open-git/backend/internal/domain/entity"
	"github.com/google/uuid"
)

type ListPullRequestsFilter struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	State          string
	Page           int
	PerPage        int
}

type IPullRequestRepository interface {
	Create(ctx context.Context, pr *entity.PullRequest) error
	GetByNumber(ctx context.Context, repoID uuid.UUID, number int) (*entity.PullRequest, error)
	ListByRepo(ctx context.Context, filter ListPullRequestsFilter) ([]*entity.PullRequest, int, error)
	Update(ctx context.Context, pr *entity.PullRequest) error
	NextNumber(ctx context.Context, repoID uuid.UUID) (int, error)
}
