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
