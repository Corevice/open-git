package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IDependabotAlertRepository interface {
	Create(ctx context.Context, alert *entity.DependabotAlert) error
	GetByNumber(ctx context.Context, repoID uuid.UUID, alertNumber int) (*entity.DependabotAlert, error)
	ListByRepo(ctx context.Context, orgID, repoID uuid.UUID, page, perPage int) ([]*entity.DependabotAlert, int, error)
	UpdateState(ctx context.Context, id uuid.UUID, state, dismissedReason string) error
	Upsert(ctx context.Context, alert *entity.DependabotAlert) error
}
