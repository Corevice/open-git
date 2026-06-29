package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type ICompatRepository interface {
	CreateRun(ctx context.Context, run *entity.CompatTestRun) error
	UpdateRun(ctx context.Context, run *entity.CompatTestRun) error
	GetRun(ctx context.Context, id uuid.UUID) (*entity.CompatTestRun, error)
	ListRuns(ctx context.Context, orgID uuid.UUID, limit int) ([]*entity.CompatTestRun, error)
	CreateEndpointResult(ctx context.Context, r *entity.CompatEndpointResult) error
	ListEndpointResults(ctx context.Context, runID uuid.UUID) ([]*entity.CompatEndpointResult, error)
}
