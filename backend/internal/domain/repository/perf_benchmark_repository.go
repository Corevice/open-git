package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type PerfBenchmarkRepository interface {
	Create(ctx context.Context, b *entity.PerfBenchmark) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.PerfBenchmark, error)
	ListByScenario(ctx context.Context, scenarioName string, limit int, cursor string) ([]*entity.PerfBenchmark, string, error)
	GetLatestByScenario(ctx context.Context, scenarioName string) (*entity.PerfBenchmark, error)
	GetLatestPerScenario(ctx context.Context) ([]*entity.PerfBenchmark, error)
}
