package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
)

type sqlxScanJobRepository struct {
	db *sqlx.DB
}

var _ domainrepo.IScanJobRepository = (*sqlxScanJobRepository)(nil)

func NewScanJobRepository(db *sqlx.DB) domainrepo.IScanJobRepository {
	return &sqlxScanJobRepository{db: db}
}

const scanJobSelectColumns = `
	id, organization_id, repository_id, type, status, retry_count, started_at, finished_at, error
`

func (r *sqlxScanJobRepository) Create(ctx context.Context, job *entity.ScanJob) error {
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	if job.Status == "" {
		job.Status = entity.ScanJobStatusQueued
	}

	const query = `
		INSERT INTO scan_jobs (
			id, organization_id, repository_id, type, status, retry_count, started_at, finished_at, error
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?
		)
	`
	q := r.db.Rebind(query)

	var startedAt, finishedAt any
	if job.StartedAt != nil {
		startedAt = *job.StartedAt
	}
	if job.FinishedAt != nil {
		finishedAt = *job.FinishedAt
	}

	_, err := r.db.ExecContext(ctx, q,
		job.ID,
		job.OrganizationID,
		job.RepositoryID,
		job.Type,
		job.Status,
		job.RetryCount,
		startedAt,
		finishedAt,
		job.Error,
	)
	return dbErrors.MapDBError(err)
}

func (r *sqlxScanJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.ScanJob, error) {
	query := `SELECT ` + scanJobSelectColumns + ` FROM scan_jobs WHERE id = ?`
	query = r.db.Rebind(query)

	job, err := scanScanJob(r.db.QueryRowxContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return job, nil
}

func (r *sqlxScanJobRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entity.ScanJobStatus, errMsg string) error {
	now := time.Now().UTC()

	var startedAt, finishedAt any
	switch status {
	case entity.ScanJobStatusRunning:
		startedAt = now
	case entity.ScanJobStatusCompleted, entity.ScanJobStatusScanFailed, entity.ScanJobStatusParseError:
		finishedAt = now
	}

	query := `
		UPDATE scan_jobs
		SET status = ?, error = ?,
			started_at = COALESCE(?, started_at),
			finished_at = COALESCE(?, finished_at)
		WHERE id = ?
	`
	query = r.db.Rebind(query)

	_, err := r.db.ExecContext(ctx, query, status, errMsg, startedAt, finishedAt, id)
	return dbErrors.MapDBError(err)
}

type scanJobScanner interface {
	Scan(dest ...any) error
}

func scanScanJob(row scanJobScanner) (*entity.ScanJob, error) {
	var (
		job        entity.ScanJob
		startedAt  sql.NullTime
		finishedAt sql.NullTime
	)

	err := row.Scan(
		&job.ID,
		&job.OrganizationID,
		&job.RepositoryID,
		&job.Type,
		&job.Status,
		&job.RetryCount,
		&startedAt,
		&finishedAt,
		&job.Error,
	)
	if err != nil {
		return nil, err
	}

	if startedAt.Valid {
		t := startedAt.Time
		job.StartedAt = &t
	}
	if finishedAt.Valid {
		t := finishedAt.Time
		job.FinishedAt = &t
	}

	return &job, nil
}
