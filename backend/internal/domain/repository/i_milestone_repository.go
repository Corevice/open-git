package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IMilestoneRepository interface {
	GetByNumber(ctx context.Context, repoID uuid.UUID, number int) (*entity.Milestone, error)
}
