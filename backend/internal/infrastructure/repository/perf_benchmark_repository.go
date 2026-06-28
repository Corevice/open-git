package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
)

type sqlPerfBenchmarkRepository struct {
	db *sql.DB
}

var _ domainrepo.PerfBenchmarkRepository = (*sqlPerfBenchmarkRepository)(nil)

func NewPerfBenchmarkRepository(db *sql.DB) *sqlPerfBenchmarkRepository {
	return &sqlPerfBenchmarkRepository{db: db}
}

const perfBenchmarkSelectColumns = `
	id, scenario_name, environment, status, slo_result,
	started_at, finished_at, git_sha, metrics, regression, created_at
`

type metricsJSON struct {
	P50Ms         int     `json:"p50_ms"`
	P95Ms         int     `json:"p95_ms"`
	P99Ms         int     `json:"p99_ms"`
	ThroughputRPS float64 `json:"throughput_rps"`
	ErrorRate     float64 `json:"error_rate"`
	TotalRequests int64   `json:"total_requests"`
}

type regressionJSON struct {
	VsBaseline string  `json:"vs_baseline"`
	Flagged    bool    `json:"flagged"`
	DeltaPct   float64 `json:"delta_pct"`
}

func (r *sqlPerfBenchmarkRepository) Create(ctx context.Context, b *entity.PerfBenchmark) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	if b.CreatedAt.IsZero() {
		b.CreatedAt = time.Now().UTC()
	}

	metricsRaw, err := marshalMetrics(b.Metrics)
	if err != nil {
		return err
	}
	regressionRaw, err := marshalRegression(b.Regression)
	if err != nil {
		return err
	}

	const query = `
		INSERT INTO perf_benchmarks (
			id, scenario_name, environment, status, slo_result,
			started_at, finished_at, git_sha, metrics, regression, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = r.db.ExecContext(ctx, query,
		b.ID.String(),
		b.ScenarioName,
		b.Environment,
		string(b.Status),
		nullString(string(b.SLOResult)),
		b.StartedAt,
		nullTimePtr(b.FinishedAt),
		nullString(b.GitSHA),
		metricsRaw,
		regressionRaw,
		b.CreatedAt,
	)
	return dbErrors.MapDBError(err)
}

func (r *sqlPerfBenchmarkRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.PerfBenchmark, error) {
	query := `SELECT ` + perfBenchmarkSelectColumns + ` FROM perf_benchmarks WHERE id = $1`

	b, err := r.scanBenchmark(r.db.QueryRowContext(ctx, query, id.String()))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return b, nil
}

func (r *sqlPerfBenchmarkRepository) ListByScenario(ctx context.Context, scenarioName string, limit int, cursor string) ([]*entity.PerfBenchmark, string, error) {
	if limit <= 0 {
		limit = 30
	}

	var (
		rows *sql.Rows
		err  error
	)

	if cursor == "" {
		query := `
			SELECT ` + perfBenchmarkSelectColumns + `
			FROM perf_benchmarks
			WHERE scenario_name = $1
			ORDER BY created_at DESC
			LIMIT $2
		`
		rows, err = r.db.QueryContext(ctx, query, scenarioName, limit+1)
	} else {
		cursorTime, parseErr := parseCursorTime(cursor)
		if parseErr != nil {
			return nil, "", parseErr
		}
		query := `
			SELECT ` + perfBenchmarkSelectColumns + `
			FROM perf_benchmarks
			WHERE scenario_name = $1 AND created_at < $2
			ORDER BY created_at DESC
			LIMIT $3
		`
		rows, err = r.db.QueryContext(ctx, query, scenarioName, cursorTime, limit+1)
	}
	if err != nil {
		return nil, "", dbErrors.MapDBError(err)
	}
	defer rows.Close()

	benchmarks, err := r.scanBenchmarkRows(rows)
	if err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(benchmarks) > limit {
		nextCursor = benchmarks[limit-1].CreatedAt.UTC().Format(time.RFC3339Nano)
		benchmarks = benchmarks[:limit]
	}

	return benchmarks, nextCursor, nil
}

func (r *sqlPerfBenchmarkRepository) GetLatestByScenario(ctx context.Context, scenarioName string) (*entity.PerfBenchmark, error) {
	query := `
		SELECT ` + perfBenchmarkSelectColumns + `
		FROM perf_benchmarks
		WHERE scenario_name = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	b, err := r.scanBenchmark(r.db.QueryRowContext(ctx, query, scenarioName))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return b, nil
}

func (r *sqlPerfBenchmarkRepository) GetLatestPerScenario(ctx context.Context) ([]*entity.PerfBenchmark, error) {
	query := `
		SELECT DISTINCT ON (scenario_name) ` + perfBenchmarkSelectColumns + `
		FROM perf_benchmarks
		ORDER BY scenario_name, created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	return r.scanBenchmarkRows(rows)
}

type benchmarkRowScanner interface {
	Scan(dest ...any) error
}

func (r *sqlPerfBenchmarkRepository) scanBenchmark(row benchmarkRowScanner) (*entity.PerfBenchmark, error) {
	var (
		b            entity.PerfBenchmark
		idRaw        string
		sloResult    sql.NullString
		finishedAt   sql.NullTime
		gitSHA       sql.NullString
		metricsRaw   []byte
		regressionRaw []byte
	)

	err := row.Scan(
		&idRaw,
		&b.ScenarioName,
		&b.Environment,
		&b.Status,
		&sloResult,
		&b.StartedAt,
		&finishedAt,
		&gitSHA,
		&metricsRaw,
		&regressionRaw,
		&b.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	b.ID, err = uuid.Parse(idRaw)
	if err != nil {
		return nil, err
	}
	if sloResult.Valid {
		b.SLOResult = entity.SLOResult(sloResult.String)
	}
	if finishedAt.Valid {
		t := finishedAt.Time
		b.FinishedAt = &t
	}
	if gitSHA.Valid {
		b.GitSHA = gitSHA.String
	}

	b.Metrics, err = unmarshalMetrics(metricsRaw)
	if err != nil {
		return nil, err
	}
	b.Regression, err = unmarshalRegression(regressionRaw)
	if err != nil {
		return nil, err
	}

	return &b, nil
}

func (r *sqlPerfBenchmarkRepository) scanBenchmarkRows(rows *sql.Rows) ([]*entity.PerfBenchmark, error) {
	var benchmarks []*entity.PerfBenchmark
	for rows.Next() {
		b, err := r.scanBenchmark(rows)
		if err != nil {
			return nil, dbErrors.MapDBError(err)
		}
		benchmarks = append(benchmarks, b)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	if benchmarks == nil {
		benchmarks = []*entity.PerfBenchmark{}
	}
	return benchmarks, nil
}

func marshalMetrics(m entity.PerfMetrics) ([]byte, error) {
	return json.Marshal(metricsJSON{
		P50Ms:         m.P50Ms,
		P95Ms:         m.P95Ms,
		P99Ms:         m.P99Ms,
		ThroughputRPS: m.ThroughputRPS,
		ErrorRate:     m.ErrorRate,
		TotalRequests: m.TotalRequests,
	})
}

func unmarshalMetrics(raw []byte) (entity.PerfMetrics, error) {
	if len(raw) == 0 {
		return entity.PerfMetrics{}, nil
	}
	var m metricsJSON
	if err := json.Unmarshal(raw, &m); err != nil {
		return entity.PerfMetrics{}, err
	}
	return entity.PerfMetrics{
		P50Ms:         m.P50Ms,
		P95Ms:         m.P95Ms,
		P99Ms:         m.P99Ms,
		ThroughputRPS: m.ThroughputRPS,
		ErrorRate:     m.ErrorRate,
		TotalRequests: m.TotalRequests,
	}, nil
}

func marshalRegression(r *entity.PerfRegressionResult) ([]byte, error) {
	if r == nil {
		return nil, nil
	}
	return json.Marshal(regressionJSON{
		VsBaseline: r.VsBaseline,
		Flagged:    r.Flagged,
		DeltaPct:   r.DeltaPct,
	})
}

func unmarshalRegression(raw []byte) (*entity.PerfRegressionResult, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var r regressionJSON
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	return &entity.PerfRegressionResult{
		VsBaseline: r.VsBaseline,
		Flagged:    r.Flagged,
		DeltaPct:   r.DeltaPct,
	}, nil
}

func parseCursorTime(cursor string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, cursor); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, cursor)
}

func nullString(v string) sql.NullString {
	if v == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: v, Valid: true}
}

func nullTimePtr(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
