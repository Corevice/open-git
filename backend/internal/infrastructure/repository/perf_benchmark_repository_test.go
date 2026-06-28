package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

var perfBenchmarkRowColumns = []string{
	"id", "scenario_name", "environment", "status", "slo_result",
	"started_at", "finished_at", "git_sha", "metrics", "regression", "created_at",
}

func TestPerfBenchmarkRepository_Create(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	repo := repository.NewPerfBenchmarkRepository(mockDB)

	benchmarkID := uuid.New()
	startedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(5 * time.Minute)

	b := &entity.PerfBenchmark{
		ID:           benchmarkID,
		ScenarioName: "rest-repos-read",
		Environment:  "ci",
		Status:       entity.StatusCompleted,
		SLOResult:    entity.SLOPass,
		StartedAt:    startedAt,
		FinishedAt:   &finishedAt,
		GitSHA:       "abc1234",
		Metrics: entity.PerfMetrics{
			P50Ms:         80,
			P95Ms:         250,
			P99Ms:         600,
			ThroughputRPS: 1200,
			ErrorRate:     0.001,
			TotalRequests: 360000,
		},
		Regression: &entity.PerfRegressionResult{
			VsBaseline: "+3%",
			Flagged:    false,
			DeltaPct:   3.0,
		},
		CreatedAt: startedAt,
	}

	mock.ExpectExec(`INSERT INTO perf_benchmarks`).
		WithArgs(
			benchmarkID.String(),
			"rest-repos-read",
			"ci",
			"completed",
			sqlmock.AnyArg(),
			startedAt,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			startedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.Create(context.Background(), b); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestPerfBenchmarkRepository_GetByID(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	repo := repository.NewPerfBenchmarkRepository(mockDB)

	benchmarkID := uuid.New()
	startedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(5 * time.Minute)
	createdAt := startedAt.Add(time.Minute)
	metricsRaw := []byte(`{"p50_ms":80,"p95_ms":250,"p99_ms":600,"throughput_rps":1200,"error_rate":0.001,"total_requests":360000}`)
	regressionRaw := []byte(`{"vs_baseline":"+3%","flagged":false,"delta_pct":3}`)

	mock.ExpectQuery(`SELECT .+ FROM perf_benchmarks WHERE id = \$1`).
		WithArgs(benchmarkID.String()).
		WillReturnRows(sqlmock.NewRows(perfBenchmarkRowColumns).
			AddRow(
				benchmarkID.String(),
				"rest-repos-read",
				"ci",
				"completed",
				"pass",
				startedAt,
				finishedAt,
				"abc1234",
				metricsRaw,
				regressionRaw,
				createdAt,
			))

	got, err := repo.GetByID(context.Background(), benchmarkID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected benchmark, got nil")
	}
	if got.ID != benchmarkID {
		t.Fatalf("expected id %s, got %s", benchmarkID, got.ID)
	}
	if got.ScenarioName != "rest-repos-read" {
		t.Fatalf("unexpected scenario_name: %s", got.ScenarioName)
	}
	if got.Metrics.P95Ms != 250 {
		t.Fatalf("expected p95_ms 250, got %d", got.Metrics.P95Ms)
	}
	if got.Regression == nil || got.Regression.VsBaseline != "+3%" {
		t.Fatalf("unexpected regression: %+v", got.Regression)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestPerfBenchmarkRepository_ListByScenario_Empty(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	repo := repository.NewPerfBenchmarkRepository(mockDB)

	mock.ExpectQuery(`SELECT .+ FROM perf_benchmarks WHERE scenario_name = \$1`).
		WithArgs("missing-scenario", 31).
		WillReturnRows(sqlmock.NewRows(perfBenchmarkRowColumns))

	items, nextCursor, err := repo.ListByScenario(context.Background(), "missing-scenario", 30, "")
	if err != nil {
		t.Fatalf("ListByScenario: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty slice, got %d items", len(items))
	}
	if nextCursor != "" {
		t.Fatalf("expected empty next cursor, got %q", nextCursor)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestPerfJobRepository_GetActiveJob_NoActiveJob(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	repo := repository.NewPerfJobRepository(mockDB)

	mock.ExpectQuery(`SELECT .+ FROM perf_jobs WHERE status IN`).
		WillReturnError(sql.ErrNoRows)

	got, err := repo.GetActiveJob(context.Background())
	if err != nil {
		t.Fatalf("GetActiveJob: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil job, got %+v", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}
