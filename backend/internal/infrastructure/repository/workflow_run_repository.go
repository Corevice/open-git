package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
	workflowusecase "github.com/open-git/backend/internal/usecase/workflow"
)

type sqlxWorkflowRunRepository struct {
	db *sqlx.DB
	// onRerun re-dispatches execution for a run that Rerun reset to queued.
	// Injected from composition (main.go) so the repository does not depend on
	// the executor.
	onRerun func(ctx context.Context, organizationID uuid.UUID, run *entity.WorkflowRun)
}

var _ domainrepo.IWorkflowRunRepository = (*sqlxWorkflowRunRepository)(nil)

func NewWorkflowRunRepository(db *sqlx.DB) *sqlxWorkflowRunRepository {
	return &sqlxWorkflowRunRepository{db: db}
}

// SetRerunDispatcher wires the callback invoked after a successful Rerun.
func (r *sqlxWorkflowRunRepository) SetRerunDispatcher(fn func(ctx context.Context, organizationID uuid.UUID, run *entity.WorkflowRun)) {
	r.onRerun = fn
}

const workflowRunSelectColumns = `
	id, repository_id, workflow_id, workflow, head_sha, head_branch, event,
	actor_login, run_number, status, conclusion, created_at, updated_at
`

// Create inserts a run and assigns it the next per-repository run number.
func (r *sqlxWorkflowRunRepository) Create(ctx context.Context, organizationID uuid.UUID, run *entity.WorkflowRun) error {
	if run.ID == uuid.Nil {
		// Int64-compatible id (upper 64 bits zero) so it round-trips the
		// int64<->UUID bridge the Actions API uses to expose numeric run ids.
		run.ID = newInt64CompatibleUUID()
	}
	now := time.Now().UTC()
	if run.CreatedAt.IsZero() {
		run.CreatedAt = now
	}
	run.UpdatedAt = now
	if run.Status == "" {
		run.Status = "queued"
	}

	// Per-repo monotonic run number. MAX+1 has a benign race under concurrent
	// pushes to the same repo (run numbers are display metadata, not keys).
	if run.RunNumber == 0 {
		query := r.db.Rebind(`SELECT COALESCE(MAX(run_number), 0) + 1 FROM workflow_runs WHERE repository_id = ?`)
		if err := r.db.GetContext(ctx, &run.RunNumber, query, run.RepositoryID); err != nil {
			return dbErrors.MapDBError(err)
		}
	}

	const query = `
		INSERT INTO workflow_runs (
			id, organization_id, repository_id, workflow_id, workflow, head_sha,
			head_branch, event, actor_login, run_number, status, conclusion,
			started_at, created_at, updated_at
		) VALUES (
			:id, :organization_id, :repository_id, :workflow_id, :workflow, :head_sha,
			:head_branch, :event, :actor_login, :run_number, :status, :conclusion,
			:started_at, :created_at, :updated_at
		)
	`
	conclusion := sql.NullString{}
	if run.Conclusion != "" {
		conclusion = sql.NullString{String: run.Conclusion, Valid: true}
	}
	_, err := r.db.NamedExecContext(ctx, query, map[string]any{
		"id":              run.ID,
		"organization_id": organizationID,
		"repository_id":   run.RepositoryID,
		"workflow_id":     run.WorkflowID,
		"workflow":        run.Workflow,
		"head_sha":        run.HeadSHA,
		"head_branch":     run.HeadBranch,
		"event":           run.Event,
		"actor_login":     run.ActorLogin,
		"run_number":      run.RunNumber,
		"status":          run.Status,
		"conclusion":      conclusion,
		"started_at":      run.CreatedAt,
		"created_at":      run.CreatedAt,
		"updated_at":      run.UpdatedAt,
	})
	return dbErrors.MapDBError(err)
}

// List returns every run for the org+repo, newest first. Status/branch/event
// filtering and pagination happen in the usecase.
func (r *sqlxWorkflowRunRepository) List(ctx context.Context, filter workflowusecase.ListRunsFilter) ([]*entity.WorkflowRun, int, error) {
	query := r.db.Rebind(`
		SELECT ` + workflowRunSelectColumns + `
		FROM workflow_runs
		WHERE organization_id = ? AND repository_id = ?
		ORDER BY run_number DESC, created_at DESC
	`)

	rows, err := r.db.QueryxContext(ctx, query, filter.OrganizationID, filter.RepositoryID)
	if err != nil {
		return nil, 0, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	runs := make([]*entity.WorkflowRun, 0)
	for rows.Next() {
		run, err := scanFullWorkflowRun(rows)
		if err != nil {
			return nil, 0, dbErrors.MapDBError(err)
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, dbErrors.MapDBError(err)
	}
	return runs, len(runs), nil
}

func (r *sqlxWorkflowRunRepository) GetByID(ctx context.Context, organizationID, repositoryID, runID uuid.UUID) (*entity.WorkflowRun, error) {
	query := r.db.Rebind(`
		SELECT ` + workflowRunSelectColumns + `
		FROM workflow_runs
		WHERE id = ? AND organization_id = ? AND repository_id = ?
	`)

	run, err := scanFullWorkflowRun(r.db.QueryRowxContext(ctx, query, runID, organizationID, repositoryID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return run, nil
}

// Cancel marks a run cancelled and cancels its outstanding jobs.
func (r *sqlxWorkflowRunRepository) Cancel(ctx context.Context, organizationID, repositoryID, runID, _ uuid.UUID) error {
	now := time.Now().UTC()

	query := r.db.Rebind(`
		UPDATE workflow_runs
		SET status = 'completed', conclusion = 'cancelled', completed_at = ?, updated_at = ?
		WHERE id = ? AND organization_id = ? AND repository_id = ?
	`)
	result, err := r.db.ExecContext(ctx, query, now, now, runID, organizationID, repositoryID)
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}

	jobsQuery := r.db.Rebind(`
		UPDATE workflow_jobs
		SET status = 'cancelled', conclusion = 'cancelled', finished_at = ?
		WHERE workflow_run_id = ? AND status IN ('queued', 'in_progress')
	`)
	if _, err := r.db.ExecContext(ctx, jobsQuery, now, runID); err != nil {
		return dbErrors.MapDBError(err)
	}
	return nil
}

// Rerun resets the run to queued and re-dispatches execution via the injected
// dispatcher.
func (r *sqlxWorkflowRunRepository) Rerun(ctx context.Context, organizationID, repositoryID, runID, _ uuid.UUID) (*entity.WorkflowRun, error) {
	now := time.Now().UTC()

	query := r.db.Rebind(`
		UPDATE workflow_runs
		SET status = 'queued', conclusion = NULL, completed_at = NULL, updated_at = ?
		WHERE id = ? AND organization_id = ? AND repository_id = ?
	`)
	result, err := r.db.ExecContext(ctx, query, now, runID, organizationID, repositoryID)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	if affected == 0 {
		return nil, domain.ErrNotFound
	}

	run, err := r.GetByID(ctx, organizationID, repositoryID, runID)
	if err != nil {
		return nil, err
	}
	if r.onRerun != nil {
		r.onRerun(ctx, organizationID, run)
	}
	return run, nil
}

func (r *sqlxWorkflowRunRepository) ListByHeadSHA(ctx context.Context, repoID uuid.UUID, sha string) ([]*entity.WorkflowRun, error) {
	query := r.db.Rebind(`
		SELECT ` + workflowRunSelectColumns + `
		FROM workflow_runs
		WHERE repository_id = ? AND head_sha = ?
	`)

	rows, err := r.db.QueryxContext(ctx, query, repoID, sha)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	runs := make([]*entity.WorkflowRun, 0)
	for rows.Next() {
		run, err := scanFullWorkflowRun(rows)
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

func scanFullWorkflowRun(scanner workflowRunScanner) (*entity.WorkflowRun, error) {
	var (
		run           entity.WorkflowRun
		workflowIDRaw string
		conclusion    sql.NullString
		updatedAt     sql.NullTime
	)

	if err := scanner.Scan(
		&run.ID,
		&run.RepositoryID,
		&workflowIDRaw,
		&run.Workflow,
		&run.HeadSHA,
		&run.HeadBranch,
		&run.Event,
		&run.ActorLogin,
		&run.RunNumber,
		&run.Status,
		&conclusion,
		&run.CreatedAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}

	if parsed, err := uuid.Parse(workflowIDRaw); err == nil {
		run.WorkflowID = parsed
	}
	if conclusion.Valid {
		run.Conclusion = conclusion.String
	}
	if updatedAt.Valid {
		run.UpdatedAt = updatedAt.Time
	} else {
		run.UpdatedAt = run.CreatedAt
	}

	return &run, nil
}
