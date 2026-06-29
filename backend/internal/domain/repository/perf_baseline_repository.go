package repository

import (
	"context"

	"github.com/open-git/backend/internal/domain/entity"
)

type PerfBaselineRepository interface {
	GetByScenario(ctx context.Context, scenarioName string) (*entity.PerfBaseline, error)
	Upsert(ctx context.Context, b *entity.PerfBaseline) error
}
