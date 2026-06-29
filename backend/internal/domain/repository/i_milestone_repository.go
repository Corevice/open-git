package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IMilestoneRepository interface {
	Create(ctx context.Context, milestone *entity.Milestone) error
	GetByNumber(ctx context.Context, repoID uuid.UUID, number int) (*entity.Milestone, error)
	ListByRepo(ctx context.Context, repoID uuid.UUID, state string, page, perPage int) ([]*entity.Milestone, int, error)
	Update(ctx context.Context, milestone *entity.Milestone) error
	Delete(ctx context.Context, id uuid.UUID) error
	NextNumber(ctx context.Context, repoID uuid.UUID) (int, error)
	IncrOpenCount(ctx context.Context, id uuid.UUID) error
	DecrOpenCount(ctx context.Context, id uuid.UUID) error
}
