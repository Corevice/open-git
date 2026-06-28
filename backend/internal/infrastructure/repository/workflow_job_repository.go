package repository

import (
	"context"
	"database/sql"
	"time"

	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/entity"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
)

type WorkflowJobRepository struct {
	db *sql.DB
}

var _ domainrepo.IWorkflowJobRepository = (*WorkflowJobRepository)(nil)

func NewWorkflowJobRepository(db *sql.DB) *WorkflowJobRepository {
	return &WorkflowJobRepository{db: db}
}

func (r *WorkflowJobRepository) Create(ctx context.Context, job *entity.WorkflowJob) error {
	const query = `
		INSERT INTO workflow_jobs(id, workflow_run_id, organization_id, repository_id, name, status, conclusion, created_at)
		VALUES(?,?,?,?,?,?,?,?)
	`
	_, err := r.db.ExecContext(ctx, query,
		job.ID,
		job.WorkflowRunID,
		job.OrganizationID,
		job.RepositoryID,
		job.Name,
		job.Status,
		job.Conclusion,
		job.CreatedAt,
	)
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	return nil
}

func (r *WorkflowJobRepository) GetByID(ctx context.Context, orgID, jobID string) (*entity.WorkflowJob, error) {
	const query = `
		SELECT id, workflow_run_id, organization_id, repository_id, name, status, conclusion, created_at, completed_at
		FROM workflow_jobs
		WHERE organization_id=? AND id=?
	`
	row := r.db.QueryRowContext(ctx, query, orgID, jobID)
	job, err := scanWorkflowJob(row)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return job, nil
}

func (r *WorkflowJobRepository) UpdateStatus(ctx context.Context, jobID, status, conclusion string, completedAt *time.Time) error {
	const query = `
		UPDATE workflow_jobs
		SET status=?, conclusion=?, completed_at=?
		WHERE id=?
	`
	_, err := r.db.ExecContext(ctx, query, status, conclusion, completedAt, jobID)
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	return nil
}

func (r *WorkflowJobRepository) ListByRunID(ctx context.Context, orgID, runID string) ([]*entity.WorkflowJob, error) {
	const query = `
		SELECT id, workflow_run_id, organization_id, repository_id, name, status, conclusion, created_at, completed_at
		FROM workflow_jobs
		WHERE organization_id=? AND workflow_run_id=?
		ORDER BY created_at
	`
	rows, err := r.db.QueryContext(ctx, query, orgID, runID)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	jobs := make([]*entity.WorkflowJob, 0)
	for rows.Next() {
		job, err := scanWorkflowJob(rows)
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

type workflowJobScanner interface {
	Scan(dest ...any) error
}

func scanWorkflowJob(scanner workflowJobScanner) (*entity.WorkflowJob, error) {
	var (
		job         entity.WorkflowJob
		completedAt sql.NullTime
	)
	if err := scanner.Scan(
		&job.ID,
		&job.WorkflowRunID,
		&job.OrganizationID,
		&job.RepositoryID,
		&job.Name,
		&job.Status,
		&job.Conclusion,
		&job.CreatedAt,
		&completedAt,
	); err != nil {
		return nil, err
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	return &job, nil
}
