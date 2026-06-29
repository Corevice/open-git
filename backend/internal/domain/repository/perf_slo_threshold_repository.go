package repository

import (
	"context"

	"github.com/open-git/backend/internal/domain/entity"
)

type PerfSLOThresholdRepository interface {
	GetByScenario(ctx context.Context, scenarioName string) (*entity.PerfSLOThreshold, error)
	Upsert(ctx context.Context, t *entity.PerfSLOThreshold) error
}
