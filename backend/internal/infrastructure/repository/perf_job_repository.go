package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
)

type sqlPerfJobRepository struct {
	db *sql.DB
}

var _ domainrepo.PerfJobRepository = (*sqlPerfJobRepository)(nil)

func NewPerfJobRepository(db *sql.DB) *sqlPerfJobRepository {
	return &sqlPerfJobRepository{db: db}
}

const perfJobSelectColumns = `id, status, triggered_by, benchmark_id, created_at, updated_at`

func (r *sqlPerfJobRepository) Create(ctx context.Context, job *entity.PerfJob) error {
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	now := time.Now().UTC()
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}
	if job.UpdatedAt.IsZero() {
		job.UpdatedAt = now
	}
	if job.Status == "" {
		job.Status = entity.JobQueued
	}

	const query = `
		INSERT INTO perf_jobs (id, status, triggered_by, benchmark_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		job.ID.String(),
		string(job.Status),
		nullUUIDPtr(job.TriggeredBy),
		nullUUIDPtr(job.BenchmarkID),
		job.CreatedAt,
		job.UpdatedAt,
	)
	return dbErrors.MapDBError(err)
}

func (r *sqlPerfJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.PerfJob, error) {
	query := `SELECT ` + perfJobSelectColumns + ` FROM perf_jobs WHERE id = $1`

	job, err := r.scanJob(r.db.QueryRowContext(ctx, query, id.String()))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return job, nil
}

func (r *sqlPerfJobRepository) GetActiveJob(ctx context.Context) (*entity.PerfJob, error) {
	query := `
		SELECT ` + perfJobSelectColumns + `
		FROM perf_jobs
		WHERE status IN ('queued', 'running')
		LIMIT 1
	`

	job, err := r.scanJob(r.db.QueryRowContext(ctx, query))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return job, nil
}

func (r *sqlPerfJobRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entity.JobStatus, benchmarkID *uuid.UUID) error {
	const query = `
		UPDATE perf_jobs
		SET status = $1, benchmark_id = $2, updated_at = NOW()
		WHERE id = $3
	`

	result, err := r.db.ExecContext(ctx, query, string(status), nullUUIDPtr(benchmarkID), id.String())
	if err != nil {
		return dbErrors.MapDBError(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

type jobRowScanner interface {
	Scan(dest ...any) error
}

func (r *sqlPerfJobRepository) scanJob(row jobRowScanner) (*entity.PerfJob, error) {
	var (
		job           entity.PerfJob
		idRaw         string
		triggeredBy   sql.NullString
		benchmarkID   sql.NullString
	)

	err := row.Scan(
		&idRaw,
		&job.Status,
		&triggeredBy,
		&benchmarkID,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	job.ID, err = uuid.Parse(idRaw)
	if err != nil {
		return nil, err
	}
	if triggeredBy.Valid {
		parsed, err := uuid.Parse(triggeredBy.String)
		if err != nil {
			return nil, err
		}
		job.TriggeredBy = &parsed
	}
	if benchmarkID.Valid {
		parsed, err := uuid.Parse(benchmarkID.String)
		if err != nil {
			return nil, err
		}
		job.BenchmarkID = &parsed
	}

	return &job, nil
}

func nullUUIDPtr(id *uuid.UUID) sql.NullString {
	if id == nil || *id == uuid.Nil {
		return sql.NullString{}
	}
	return sql.NullString{String: id.String(), Valid: true}
}
