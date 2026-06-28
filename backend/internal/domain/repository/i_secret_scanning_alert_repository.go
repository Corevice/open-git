package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type ISecretScanningAlertRepository interface {
	Create(ctx context.Context, alert *entity.SecretScanningAlert) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.SecretScanningAlert, error)
	ListByRepo(ctx context.Context, orgID, repoID uuid.UUID, page, perPage int) ([]*entity.SecretScanningAlert, int, error)
	UpdateState(ctx context.Context, id uuid.UUID, state string) error
	Upsert(ctx context.Context, alert *entity.SecretScanningAlert) error
}
