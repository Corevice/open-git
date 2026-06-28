package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/entity"
)

type sqlxWorkflowRunRepository struct {
	db *sqlx.DB
}

var _ domainrepo.IWorkflowRunRepository = (*sqlxWorkflowRunRepository)(nil)

func NewWorkflowRunRepository(db *sqlx.DB) domainrepo.IWorkflowRunRepository {
	return &sqlxWorkflowRunRepository{db: db}
}

func (r *sqlxWorkflowRunRepository) ListByHeadSHA(ctx context.Context, repoID uuid.UUID, sha string) ([]*entity.WorkflowRun, error) {
	const query = `
		SELECT id, repository_id, head_sha, workflow, status, conclusion, started_at, completed_at
		FROM workflow_runs
		WHERE repository_id = $1 AND head_sha = $2
	`

	rows, err := r.db.QueryxContext(ctx, query, repoID, sha)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	runs := make([]*entity.WorkflowRun, 0)
	for rows.Next() {
		run, err := scanWorkflowRunRow(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return runs, nil
}

type workflowRunScanner interface {
	Scan(dest ...any) error
}

func scanWorkflowRunRow(scanner workflowRunScanner) (*entity.WorkflowRun, error) {
	var (
		run         entity.WorkflowRun
		conclusion  sql.NullString
		completedAt sql.NullTime
	)

	if err := scanner.Scan(
		&run.ID,
		&run.RepositoryID,
		&run.HeadSHA,
		&run.Workflow,
		&run.Status,
		&conclusion,
		&run.CreatedAt,
		&completedAt,
	); err != nil {
		return nil, err
	}

	if conclusion.Valid {
		run.Conclusion = conclusion.String
	}
	if completedAt.Valid {
		run.UpdatedAt = completedAt.Time
	} else {
		run.UpdatedAt = run.CreatedAt
	}

	return &run, nil
}

func (r *sqlxWorkflowRunRepository) Create(ctx context.Context, run *entity.WorkflowRun) error {
	if run.ID == uuid.Nil {
		run.ID = uuid.New()
	}
	const query = `
		INSERT INTO workflow_runs (
			id, organization_id, repository_id, workflow, status, conclusion, started_at, completed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	startedAt := run.CreatedAt
	if run.StartedAt != nil {
		startedAt = *run.StartedAt
	}
	q := r.db.Rebind(query)
	_, err := r.db.ExecContext(ctx, q,
		run.ID,
		run.OrganizationID,
		run.RepositoryID,
		run.Workflow,
		run.Status,
		run.Conclusion,
		startedAt,
		run.CompletedAt,
	)
	return err
}

func (r *sqlxWorkflowRunRepository) GetByID(ctx context.Context, runID, orgID uuid.UUID) (*entity.WorkflowRun, error) {
	const query = `
		SELECT id, repository_id, '' AS head_sha, workflow, status, conclusion, started_at, completed_at
		FROM workflow_runs
		WHERE id = ? AND organization_id = ?
	`
	q := r.db.Rebind(query)
	row := r.db.QueryRowxContext(ctx, q, runID, orgID)
	run, err := scanWorkflowRunRow(row)
	if err == sql.ErrNoRows {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	run.OrganizationID = orgID
	return run, nil
}

func (r *sqlxWorkflowRunRepository) Update(ctx context.Context, run *entity.WorkflowRun) error {
	const query = `
		UPDATE workflow_runs
		SET status = ?, conclusion = ?, completed_at = ?
		WHERE id = ? AND organization_id = ?
	`
	q := r.db.Rebind(query)
	_, err := r.db.ExecContext(ctx, q, run.Status, run.Conclusion, run.CompletedAt, run.ID, run.OrganizationID)
	return err
}

func (r *sqlxWorkflowRunRepository) IncrementRunNumber(ctx context.Context, orgID, repoID uuid.UUID) (int, error) {
	const query = `
		SELECT COUNT(*) FROM workflow_runs WHERE organization_id = ? AND repository_id = ?
	`
	q := r.db.Rebind(query)
	var count int
	if err := r.db.QueryRowxContext(ctx, q, orgID, repoID).Scan(&count); err != nil {
		return 0, err
	}
	return count + 1, nil
}

func (r *sqlxWorkflowRunRepository) IncrementRunAttempt(ctx context.Context, runID, orgID uuid.UUID) (int, error) {
	run, err := r.GetByID(ctx, runID, orgID)
	if err != nil {
		return 0, err
	}
	return run.RunAttempt + 1, nil
}
