package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/entity"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
)

type sqlxWorkflowJobRepository struct {
	db *sqlx.DB
}

var _ domainrepo.IWorkflowJobRepository = (*sqlxWorkflowJobRepository)(nil)

func NewWorkflowJobRepository(db *sqlx.DB) domainrepo.IWorkflowJobRepository {
	return &sqlxWorkflowJobRepository{db: db}
}

func (r *sqlxWorkflowJobRepository) Create(ctx context.Context, job *entity.WorkflowJob) error {
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	now := time.Now().UTC()
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}

	runsOnJSON, err := json.Marshal(job.RunsOn)
	if err != nil {
		return err
	}

	runID := workflowJobRunID(job)
	assignedRunnerID := workflowJobAssignedRunnerID(job)

	const query = `
		INSERT INTO workflow_jobs (
			id, workflow_run_id, organization_id, repository_id, name, status, conclusion,
			assigned_runner_id, runs_on, acquire_lock_version, started_at, finished_at,
			timeout_minutes, created_at
		) VALUES (
			:id, :workflow_run_id, :organization_id, :repository_id, :name, :status, :conclusion,
			:assigned_runner_id, :runs_on, :acquire_lock_version, :started_at, :finished_at,
			:timeout_minutes, :created_at
		)
	`

	_, err = r.db.NamedExecContext(ctx, query, map[string]any{
		"id":                   job.ID,
		"workflow_run_id":      runID,
		"organization_id":      job.OrganizationID,
		"repository_id":        job.RepositoryID,
		"name":                 job.Name,
		"status":               job.Status,
		"conclusion":           job.Conclusion,
		"assigned_runner_id":   assignedRunnerID,
		"runs_on":              string(runsOnJSON),
		"acquire_lock_version": job.AcquireLockVersion,
		"started_at":           job.StartedAt,
		"finished_at":          job.FinishedAt,
		"timeout_minutes":      job.TimeoutMinutes,
		"created_at":           job.CreatedAt,
	})
	return dbErrors.MapDBError(err)
}

func (r *sqlxWorkflowJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.WorkflowJob, error) {
	const query = `
		SELECT id, workflow_run_id, organization_id, repository_id, name, status, conclusion,
			assigned_runner_id, runs_on, acquire_lock_version, started_at, finished_at,
			timeout_minutes, created_at
		FROM workflow_jobs
		WHERE id = ?
	`
	q := r.db.Rebind(query)
	row := r.db.QueryRowxContext(ctx, q, id)

	job, err := scanWorkflowJobRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return job, nil
}

func (r *sqlxWorkflowJobRepository) AcquireForRunner(ctx context.Context, jobID uuid.UUID, runnerID uuid.UUID, lockVersion int) (bool, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return false, dbErrors.MapDBError(err)
	}
	defer func() { _ = tx.Rollback() }()

	startedAt := time.Now().UTC()
	const query = `
		UPDATE workflow_jobs
		SET assigned_runner_id = ?, status = 'in_progress', acquire_lock_version = acquire_lock_version + 1, started_at = ?
		WHERE id = ? AND status = 'queued' AND acquire_lock_version = ?
	`
	q := tx.Rebind(query)
	result, err := tx.ExecContext(ctx, q, runnerID, startedAt, jobID, lockVersion)
	if err != nil {
		return false, dbErrors.MapDBError(err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, dbErrors.MapDBError(err)
	}
	if err := tx.Commit(); err != nil {
		return false, dbErrors.MapDBError(err)
	}
	return rowsAffected == 1, nil
}

func (r *sqlxWorkflowJobRepository) UpdateStatus(ctx context.Context, jobID uuid.UUID, status, conclusion string) error {
	const query = `
		UPDATE workflow_jobs
		SET status = ?, conclusion = ?
		WHERE id = ?
	`
	q := r.db.Rebind(query)
	result, err := r.db.ExecContext(ctx, q, status, conclusion, jobID)
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *sqlxWorkflowJobRepository) Complete(ctx context.Context, jobID uuid.UUID, conclusion string, finishedAt time.Time) error {
	const query = `
		UPDATE workflow_jobs
		SET status = 'completed', conclusion = ?, finished_at = ?
		WHERE id = ? AND status = 'in_progress'
	`
	q := r.db.Rebind(query)
	result, err := r.db.ExecContext(ctx, q, conclusion, finishedAt, jobID)
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *sqlxWorkflowJobRepository) Cancel(ctx context.Context, jobID uuid.UUID) error {
	const query = `
		UPDATE workflow_jobs
		SET status = 'cancelled'
		WHERE id = ? AND status IN ('queued', 'in_progress')
	`
	q := r.db.Rebind(query)
	result, err := r.db.ExecContext(ctx, q, jobID)
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *sqlxWorkflowJobRepository) ListQueued(ctx context.Context, orgID uuid.UUID) ([]*entity.WorkflowJob, error) {
	const query = `
		SELECT id, workflow_run_id, organization_id, repository_id, name, status, conclusion,
			assigned_runner_id, runs_on, acquire_lock_version, started_at, finished_at,
			timeout_minutes, created_at
		FROM workflow_jobs
		WHERE organization_id = ? AND status = 'queued'
		ORDER BY created_at ASC
	`
	q := r.db.Rebind(query)
	rows, err := r.db.QueryxContext(ctx, q, orgID)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	return scanWorkflowJobRows(rows)
}

func (r *sqlxWorkflowJobRepository) CreateBatch(ctx context.Context, jobs []*entity.WorkflowJob) error {
	for _, job := range jobs {
		if err := r.Create(ctx, job); err != nil {
			return err
		}
	}
	return nil
}

func (r *sqlxWorkflowJobRepository) ResetQueuedByRunID(ctx context.Context, runID uuid.UUID) error {
	const query = `
		UPDATE workflow_jobs
		SET status = 'queued', conclusion = '', started_at = NULL, finished_at = NULL
		WHERE workflow_run_id = ? AND status IN ('queued', 'failed', 'completed')
	`
	q := r.db.Rebind(query)
	_, err := r.db.ExecContext(ctx, q, runID)
	return dbErrors.MapDBError(err)
}

func (r *sqlxWorkflowJobRepository) CancelInProgressByRunID(ctx context.Context, orgID, runID uuid.UUID) error {
	const query = `
		UPDATE workflow_jobs
		SET status = 'cancelled'
		WHERE organization_id = ? AND workflow_run_id = ? AND status = 'in_progress'
	`
	q := r.db.Rebind(query)
	_, err := r.db.ExecContext(ctx, q, orgID, runID)
	return dbErrors.MapDBError(err)
}

func (r *sqlxWorkflowJobRepository) ListByRunID(ctx context.Context, orgID, runID uuid.UUID) ([]*entity.WorkflowJob, error) {
	const query = `
		SELECT id, workflow_run_id, organization_id, repository_id, name, status, conclusion,
			assigned_runner_id, runs_on, acquire_lock_version, started_at, finished_at,
			timeout_minutes, created_at
		FROM workflow_jobs
		WHERE organization_id = ? AND workflow_run_id = ?
		ORDER BY created_at ASC
	`
	q := r.db.Rebind(query)
	rows, err := r.db.QueryxContext(ctx, q, orgID, runID)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	return scanWorkflowJobRows(rows)
}

func workflowJobRunID(job *entity.WorkflowJob) uuid.UUID {
	if job.WorkflowRunID != nil && *job.WorkflowRunID != uuid.Nil {
		return *job.WorkflowRunID
	}
	return uuid.Nil
}

func workflowJobAssignedRunnerID(job *entity.WorkflowJob) *uuid.UUID {
	if job.AssignedRunnerID == nil {
		return nil
	}
	if *job.AssignedRunnerID == uuid.Nil {
		return nil
	}
	return job.AssignedRunnerID
}

type workflowJobScanner interface {
	Scan(dest ...any) error
}

func scanWorkflowJobRow(scanner workflowJobScanner) (*entity.WorkflowJob, error) {
	var (
		job              entity.WorkflowJob
		runID            uuid.UUID
		assignedRunnerID sql.NullString
		runsOnJSON       string
		startedAt        sql.NullTime
		finishedAt       sql.NullTime
	)
	if err := scanner.Scan(
		&job.ID,
		&runID,
		&job.OrganizationID,
		&job.RepositoryID,
		&job.Name,
		&job.Status,
		&job.Conclusion,
		&assignedRunnerID,
		&runsOnJSON,
		&job.AcquireLockVersion,
		&startedAt,
		&finishedAt,
		&job.TimeoutMinutes,
		&job.CreatedAt,
	); err != nil {
		return nil, err
	}

	if runID != uuid.Nil {
		job.WorkflowRunID = &runID
	}
	if assignedRunnerID.Valid {
		parsed, err := uuid.Parse(assignedRunnerID.String)
		if err != nil {
			return nil, err
		}
		job.AssignedRunnerID = &parsed
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		job.FinishedAt = &finishedAt.Time
	}
	if runsOnJSON != "" {
		if err := json.Unmarshal([]byte(runsOnJSON), &job.RunsOn); err != nil {
			return nil, err
		}
	}
	return &job, nil
}

func scanWorkflowJobRows(rows *sqlx.Rows) ([]*entity.WorkflowJob, error) {
	jobs := make([]*entity.WorkflowJob, 0)
	for rows.Next() {
		job, err := scanWorkflowJobRow(rows)
		if err != nil {
			return nil, dbErrors.MapDBError(err)
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return jobs, nil
}
