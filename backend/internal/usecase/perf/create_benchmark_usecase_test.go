package perf_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/usecase/perf"
)

type mockBenchmarkRepo struct {
	created []*entity.PerfBenchmark
}

func (m *mockBenchmarkRepo) Create(_ context.Context, benchmark *entity.PerfBenchmark) error {
	m.created = append(m.created, benchmark)
	return nil
}

func (m *mockBenchmarkRepo) GetLatestPerScenario(_ context.Context) ([]*entity.PerfBenchmark, error) {
	return nil, nil
}

type mockSLOThresholdRepo struct {
	threshold *entity.PerfSLOThreshold
}

func (m *mockSLOThresholdRepo) GetByScenario(_ context.Context, _ string) (*entity.PerfSLOThreshold, error) {
	return m.threshold, nil
}

type mockBaselineRepo struct {
	baseline *entity.PerfBenchmark
}

func (m *mockBaselineRepo) GetByScenario(_ context.Context, _ string) (*entity.PerfBenchmark, error) {
	return m.baseline, nil
}

func validMetrics() entity.PerfMetrics {
	return entity.PerfMetrics{
		P50Ms:         80,
		P95Ms:         250,
		P99Ms:         600,
		ThroughputRPS: 1200,
		ErrorRate:     0.001,
		TotalRequests: 360000,
	}
}

func newUseCase(benchmarkRepo *mockBenchmarkRepo, thresholdRepo *mockSLOThresholdRepo, baselineRepo *mockBaselineRepo) *perf.CreateBenchmarkUseCase {
	return &perf.CreateBenchmarkUseCase{
		BenchmarkRepo:    benchmarkRepo,
		SLOThresholdRepo: thresholdRepo,
		BaselineRepo:     baselineRepo,
	}
}

func TestCreateBenchmarkUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	startedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("valid input status completed slo result set", func(t *testing.T) {
		benchmarkRepo := &mockBenchmarkRepo{}
		thresholdRepo := &mockSLOThresholdRepo{
			threshold: &entity.PerfSLOThreshold{
				P95MsMax: intPtr(300),
			},
		}
		baselineRepo := &mockBaselineRepo{}

		uc := newUseCase(benchmarkRepo, thresholdRepo, baselineRepo)
		got, err := uc.Execute(ctx, perf.CreateBenchmarkInput{
			ScenarioName: "rest-repos-read",
			Environment:  "ci",
			StartedAt:    startedAt,
			Metrics:      validMetrics(),
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if got.Status != entity.BenchmarkStatusCompleted {
			t.Fatalf("Status = %q, want completed", got.Status)
		}
		if got.SLOResult != entity.SLOPass {
			t.Fatalf("SLOResult = %q, want pass", got.SLOResult)
		}
		if len(benchmarkRepo.created) != 1 {
			t.Fatalf("Create called %d times, want 1", len(benchmarkRepo.created))
		}
	})

	t.Run("invalid metrics error create never called", func(t *testing.T) {
		benchmarkRepo := &mockBenchmarkRepo{}
		uc := newUseCase(benchmarkRepo, &mockSLOThresholdRepo{}, &mockBaselineRepo{})

		metrics := validMetrics()
		metrics.P50Ms = 300

		_, err := uc.Execute(ctx, perf.CreateBenchmarkInput{
			ScenarioName: "rest-repos-read",
			Environment:  "ci",
			StartedAt:    startedAt,
			Metrics:      metrics,
		})
		if err == nil {
			t.Fatal("Execute() error = nil, want validation error")
		}
		if len(benchmarkRepo.created) != 0 {
			t.Fatalf("Create called %d times, want 0", len(benchmarkRepo.created))
		}
	})

	t.Run("total requests zero status invalid slo skipped", func(t *testing.T) {
		benchmarkRepo := &mockBenchmarkRepo{}
		thresholdRepo := &mockSLOThresholdRepo{
			threshold: &entity.PerfSLOThreshold{
				P95MsMax: intPtr(300),
			},
		}
		uc := newUseCase(benchmarkRepo, thresholdRepo, &mockBaselineRepo{})

		metrics := validMetrics()
		metrics.TotalRequests = 0

		got, err := uc.Execute(ctx, perf.CreateBenchmarkInput{
			ScenarioName: "rest-repos-read",
			Environment:  "ci",
			StartedAt:    startedAt,
			Metrics:      metrics,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if got.Status != entity.BenchmarkStatusInvalid {
			t.Fatalf("Status = %q, want invalid", got.Status)
		}
		if got.SLOResult != entity.SLOSkipped {
			t.Fatalf("SLOResult = %q, want skipped", got.SLOResult)
		}
	})

	t.Run("nil baseline regression nil", func(t *testing.T) {
		benchmarkRepo := &mockBenchmarkRepo{}
		uc := newUseCase(benchmarkRepo, &mockSLOThresholdRepo{}, &mockBaselineRepo{})

		got, err := uc.Execute(ctx, perf.CreateBenchmarkInput{
			ScenarioName: "rest-repos-read",
			Environment:  "ci",
			StartedAt:    startedAt,
			Metrics:      validMetrics(),
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if got.Regression != nil {
			t.Fatalf("Regression = %+v, want nil", got.Regression)
		}
	})

	t.Run("baseline with big p95 increase regression flagged", func(t *testing.T) {
		benchmarkRepo := &mockBenchmarkRepo{}
		baselineRepo := &mockBaselineRepo{
			baseline: &entity.PerfBenchmark{
				ID:           uuid.New(),
				ScenarioName: "rest-repos-read",
				Metrics: entity.PerfMetrics{
					P95Ms: 200,
				},
			},
		}
		uc := newUseCase(benchmarkRepo, &mockSLOThresholdRepo{}, baselineRepo)

		metrics := validMetrics()
		metrics.P95Ms = 260

		got, err := uc.Execute(ctx, perf.CreateBenchmarkInput{
			ScenarioName: "rest-repos-read",
			Environment:  "ci",
			StartedAt:    startedAt,
			Metrics:      metrics,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if got.Regression == nil {
			t.Fatal("Regression = nil, want result")
		}
		if !got.Regression.Flagged {
			t.Fatalf("Regression.Flagged = false, want true")
		}
	})

	t.Run("p95 exceeds threshold slo fail", func(t *testing.T) {
		benchmarkRepo := &mockBenchmarkRepo{}
		thresholdRepo := &mockSLOThresholdRepo{
			threshold: &entity.PerfSLOThreshold{
				P95MsMax: intPtr(200),
			},
		}
		uc := newUseCase(benchmarkRepo, thresholdRepo, &mockBaselineRepo{})

		got, err := uc.Execute(ctx, perf.CreateBenchmarkInput{
			ScenarioName: "rest-repos-read",
			Environment:  "ci",
			StartedAt:    startedAt,
			Metrics:      validMetrics(),
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if got.SLOResult != entity.SLOFail {
			t.Fatalf("SLOResult = %q, want fail", got.SLOResult)
		}
	})
}
