package perf

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/open-git/backend/internal/domain/entity"
)

// AllowedBenchmarkEnvironments lists valid benchmark execution targets.
var AllowedBenchmarkEnvironments = []string{
	"docker-compose",
	"k8s",
	"ci",
}

var allowedEnvironments = func() map[string]struct{} {
	allowed := make(map[string]struct{}, len(AllowedBenchmarkEnvironments))
	for _, env := range AllowedBenchmarkEnvironments {
		allowed[env] = struct{}{}
	}
	return allowed
}()

const defaultRegressionPctMax = 20.0

type BenchmarkRepository interface {
	Create(ctx context.Context, benchmark *entity.PerfBenchmark) error
	GetLatestPerScenario(ctx context.Context) ([]*entity.PerfBenchmark, error)
}

type SLOThresholdRepository interface {
	GetByScenario(ctx context.Context, scenarioName string) (*entity.PerfSLOThreshold, error)
}

type BaselineRepository interface {
	GetByScenario(ctx context.Context, scenarioName string) (*entity.PerfBenchmark, error)
}

type CreateBenchmarkInput struct {
	ScenarioName string
	Environment  string
	StartedAt    time.Time
	FinishedAt   *time.Time
	GitSHA       string
	Metrics      entity.PerfMetrics
	Partial      bool
}

type CreateBenchmarkUseCase struct {
	BenchmarkRepo    BenchmarkRepository
	SLOThresholdRepo SLOThresholdRepository
	BaselineRepo     BaselineRepository
}

func (uc *CreateBenchmarkUseCase) Execute(ctx context.Context, input CreateBenchmarkInput) (*entity.PerfBenchmark, error) {
	if err := input.Metrics.Validate(); err != nil {
		return nil, err
	}

	if _, ok := allowedEnvironments[input.Environment]; !ok {
		return nil, fmt.Errorf("invalid environment")
	}

	threshold, err := uc.SLOThresholdRepo.GetByScenario(ctx, input.ScenarioName)
	if err != nil {
		return nil, err
	}

	sloResult := EvaluateSLO(input.Metrics, threshold)

	baseline, err := uc.BaselineRepo.GetByScenario(ctx, input.ScenarioName)
	if err != nil {
		return nil, err
	}

	maxPct := defaultRegressionPctMax
	if threshold != nil && threshold.RegressionPctMax != nil {
		maxPct = *threshold.RegressionPctMax
	}

	regression := DetectRegression(input.Metrics, baseline, maxPct)

	var status entity.BenchmarkStatus
	switch {
	case input.Metrics.TotalRequests == 0:
		status = entity.BenchmarkStatusInvalid
	case input.Partial:
		status = entity.BenchmarkStatusPartial
	default:
		status = entity.BenchmarkStatusCompleted
	}

	if status == entity.BenchmarkStatusInvalid {
		sloResult = entity.SLOSkipped
	}

	benchmark := &entity.PerfBenchmark{
		ID:           uuid.New(),
		ScenarioName: input.ScenarioName,
		Environment:  input.Environment,
		Status:       status,
		SLOResult:    sloResult,
		StartedAt:    input.StartedAt,
		FinishedAt:   input.FinishedAt,
		GitSHA:       input.GitSHA,
		Metrics:      input.Metrics,
		Regression:   regression,
		CreatedAt:    time.Now().UTC(),
	}

	if err := uc.BenchmarkRepo.Create(ctx, benchmark); err != nil {
		return nil, err
	}

	return benchmark, nil
}
