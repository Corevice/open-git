package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IDependabotAlertRepository interface {
	ListByRepo(ctx context.Context, orgID, repoID uuid.UUID, state string, page, perPage int) ([]*entity.DependabotAlert, int, error)
	GetByAlertNumber(ctx context.Context, orgID, repoID uuid.UUID, alertNumber int) (*entity.DependabotAlert, error)
	UpdateState(ctx context.Context, orgID, repoID uuid.UUID, alertNumber int, state entity.DependabotAlertState, reason *entity.DismissedReason) (*entity.DependabotAlert, error)
}
