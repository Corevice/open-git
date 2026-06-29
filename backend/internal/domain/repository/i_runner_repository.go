package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IRunnerRepository interface {
	Create(ctx context.Context, r *entity.Runner) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Runner, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*entity.Runner, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, lastSeenAt time.Time) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindAvailable(ctx context.Context, orgID uuid.UUID, labels []string) (*entity.Runner, error)
}
