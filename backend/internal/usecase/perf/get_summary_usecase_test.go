package perf_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/usecase/perf"
)

type summaryBenchmarkRepo struct {
	latest []*entity.PerfBenchmark
	err    error
}

func (m *summaryBenchmarkRepo) Create(context.Context, *entity.PerfBenchmark) error {
	return nil
}

func (m *summaryBenchmarkRepo) GetLatestPerScenario(context.Context) ([]*entity.PerfBenchmark, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.latest, nil
}

func TestGetSummaryUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("all scenarios pass", func(t *testing.T) {
		repo := &summaryBenchmarkRepo{
			latest: []*entity.PerfBenchmark{
				{
					ID:           uuid.New(),
					ScenarioName: "rest-repos-read",
					SLOResult:    entity.SLOPass,
					CreatedAt:    now,
				},
				{
					ID:           uuid.New(),
					ScenarioName: "graphql-query",
					SLOResult:    entity.SLOPass,
					CreatedAt:    now,
				},
			},
		}
		uc := &perf.GetSummaryUseCase{
			BenchmarkRepo: repo,
			GrafanaURL:    "https://grafana.example.com",
		}

		got, err := uc.Execute(ctx)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if got.SLOStatus.Overall != perf.SLOOverallPass {
			t.Fatalf("SLOStatus.Overall = %q, want %q", got.SLOStatus.Overall, perf.SLOOverallPass)
		}
		if len(got.SLOStatus.Violations) != 0 {
			t.Fatalf("SLOStatus.Violations = %v, want empty", got.SLOStatus.Violations)
		}
		if got.GrafanaURL != "https://grafana.example.com" {
			t.Fatalf("GrafanaURL = %q, want https://grafana.example.com", got.GrafanaURL)
		}
		if len(got.Latest) != 2 {
			t.Fatalf("len(Latest) = %d, want 2", len(got.Latest))
		}
	})

	t.Run("failed scenario marks overall fail and records violation", func(t *testing.T) {
		repo := &summaryBenchmarkRepo{
			latest: []*entity.PerfBenchmark{
				{
					ID:           uuid.New(),
					ScenarioName: "rest-repos-read",
					SLOResult:    entity.SLOPass,
					CreatedAt:    now,
				},
				{
					ID:           uuid.New(),
					ScenarioName: "graphql-query",
					SLOResult:    entity.SLOFail,
					CreatedAt:    now,
				},
			},
		}
		uc := &perf.GetSummaryUseCase{BenchmarkRepo: repo}

		got, err := uc.Execute(ctx)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if got.SLOStatus.Overall != perf.SLOOverallFail {
			t.Fatalf("SLOStatus.Overall = %q, want %q", got.SLOStatus.Overall, perf.SLOOverallFail)
		}
		if len(got.SLOStatus.Violations) != 1 || got.SLOStatus.Violations[0] != "graphql-query" {
			t.Fatalf("SLOStatus.Violations = %v, want [graphql-query]", got.SLOStatus.Violations)
		}
	})

	t.Run("repository error propagates", func(t *testing.T) {
		repoErr := errors.New("db unavailable")
		uc := &perf.GetSummaryUseCase{
			BenchmarkRepo: &summaryBenchmarkRepo{err: repoErr},
		}

		_, err := uc.Execute(ctx)
		if !errors.Is(err, repoErr) {
			t.Fatalf("Execute() error = %v, want %v", err, repoErr)
		}
	})
}
