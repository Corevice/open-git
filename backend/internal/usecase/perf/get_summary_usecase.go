package perf

import (
	"context"

	"github.com/open-git/backend/internal/domain/entity"
)

type SLOStatusSummary struct {
	Overall    string
	Violations []string
}

type SummaryResult struct {
	Latest     []*entity.PerfBenchmark
	SLOStatus  SLOStatusSummary
	GrafanaURL string
}

type GetSummaryUseCase struct {
	BenchmarkRepo BenchmarkRepository
	GrafanaURL    string
}

func (uc *GetSummaryUseCase) Execute(ctx context.Context) (*SummaryResult, error) {
	latest, err := uc.BenchmarkRepo.GetLatestPerScenario(ctx)
	if err != nil {
		return nil, err
	}

	sloStatus := SLOStatusSummary{
		Overall:    "pass",
		Violations: []string{},
	}

	for _, benchmark := range latest {
		if benchmark.SLOResult == entity.SLOFail {
			sloStatus.Overall = "fail"
			sloStatus.Violations = append(sloStatus.Violations, benchmark.ScenarioName)
		}
	}

	return &SummaryResult{
		Latest:     latest,
		SLOStatus:  sloStatus,
		GrafanaURL: uc.GrafanaURL,
	}, nil
}
