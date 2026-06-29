package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain/entity"
	repo "github.com/open-git/backend/internal/repository"
)

type sqlxWorkflowRepository struct {
	*sqlx.DB
}

func NewWorkflowRepository(db *sqlx.DB) repo.IWorkflowRepository {
	return &sqlxWorkflowRepository{DB: db}
}

const workflowSelectColumns = `id, organization_id, repository_id, name, path, state, created_at, updated_at`

const workflowRevisionSelectColumns = `id, workflow_id, commit_sha, raw_content_hash, parse_status, ir, parsed_at`

const workflowDiagnosticSelectColumns = `id, workflow_revision_id, line, col, severity, message`

func (r *sqlxWorkflowRepository) Upsert(ctx context.Context, wf *entity.Workflow) error {
	if wf.ID == uuid.Nil {
		wf.ID = uuid.New()
	}
	now := time.Now().UTC()
	if wf.CreatedAt.IsZero() {
		wf.CreatedAt = now
	}
	wf.UpdatedAt = now

	query := `
		INSERT INTO workflows (id, organization_id, repository_id, name, path, state, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(repository_id, path) DO UPDATE SET
			name = excluded.name,
			state = excluded.state,
			updated_at = excluded.updated_at
	`
	query = r.DB.Rebind(query)

	_, err := r.DB.ExecContext(ctx, query,
		wf.ID,
		wf.OrganizationID,
		wf.RepositoryID,
		wf.Name,
		wf.Path,
		wf.State,
		wf.CreatedAt,
		wf.UpdatedAt,
	)
	return err
}

func (r *sqlxWorkflowRepository) GetByID(ctx context.Context, orgID, id uuid.UUID) (*entity.Workflow, error) {
	query := `SELECT ` + workflowSelectColumns + ` FROM workflows WHERE id = ? AND organization_id = ?`
	query = r.DB.Rebind(query)

	row := r.DB.QueryRowxContext(ctx, query, id, orgID)
	wf, err := scanWorkflow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (r *sqlxWorkflowRepository) GetByPath(ctx context.Context, orgID, repoID uuid.UUID, path string) (*entity.Workflow, error) {
	query := `SELECT ` + workflowSelectColumns + ` FROM workflows WHERE organization_id = ? AND repository_id = ? AND path = ?`
	query = r.DB.Rebind(query)

	row := r.DB.QueryRowxContext(ctx, query, orgID, repoID, path)
	wf, err := scanWorkflow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (r *sqlxWorkflowRepository) ListByRepo(ctx context.Context, orgID, repoID uuid.UUID) ([]*entity.Workflow, error) {
	query := `SELECT ` + workflowSelectColumns + ` FROM workflows WHERE organization_id = ? AND repository_id = ? ORDER BY path ASC`
	query = r.DB.Rebind(query)

	rows, err := r.DB.QueryxContext(ctx, query, orgID, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workflows []*entity.Workflow
	for rows.Next() {
		wf, err := scanWorkflow(rows)
		if err != nil {
			return nil, err
		}
		workflows = append(workflows, wf)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return workflows, nil
}

func (r *sqlxWorkflowRepository) SaveRevision(ctx context.Context, rev *entity.WorkflowRevision) error {
	if rev.ID == uuid.Nil {
		rev.ID = uuid.New()
	}
	if rev.IR == "" {
		rev.IR = "{}"
	}

	query := `
		INSERT INTO workflow_revisions (id, workflow_id, commit_sha, raw_content_hash, parse_status, ir, parsed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workflow_id, commit_sha) DO NOTHING
	`
	query = r.DB.Rebind(query)

	_, err := r.DB.ExecContext(ctx, query,
		rev.ID,
		rev.WorkflowID,
		rev.CommitSHA,
		rev.RawContentHash,
		rev.ParseStatus,
		rev.IR,
		rev.ParsedAt,
	)
	return err
}

func (r *sqlxWorkflowRepository) GetLatestRevision(ctx context.Context, workflowID uuid.UUID) (*entity.WorkflowRevision, error) {
	query := `SELECT ` + workflowRevisionSelectColumns + ` FROM workflow_revisions WHERE workflow_id = ? ORDER BY parsed_at DESC LIMIT 1`
	query = r.DB.Rebind(query)

	row := r.DB.QueryRowxContext(ctx, query, workflowID)
	rev, err := scanWorkflowRevision(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rev, nil
}

func (r *sqlxWorkflowRepository) SaveDiagnostics(ctx context.Context, revID uuid.UUID, diags []*entity.WorkflowDiagnostic) error {
	deleteQuery := `DELETE FROM workflow_diagnostics WHERE workflow_revision_id = ?`
	deleteQuery = r.DB.Rebind(deleteQuery)
	if _, err := r.DB.ExecContext(ctx, deleteQuery, revID); err != nil {
		return err
	}

	insertQuery := `
		INSERT INTO workflow_diagnostics (id, workflow_revision_id, line, col, severity, message)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	insertQuery = r.DB.Rebind(insertQuery)

	for _, diag := range diags {
		diag.ID = uuid.New()
		diag.WorkflowRevisionID = revID
		if _, err := r.DB.ExecContext(ctx, insertQuery,
			diag.ID,
			diag.WorkflowRevisionID,
			diag.Line,
			diag.Col,
			diag.Severity,
			diag.Message,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *sqlxWorkflowRepository) ListDiagnosticsByRevision(ctx context.Context, revID uuid.UUID) ([]*entity.WorkflowDiagnostic, error) {
	query := `SELECT ` + workflowDiagnosticSelectColumns + ` FROM workflow_diagnostics WHERE workflow_revision_id = ? ORDER BY line, col`
	query = r.DB.Rebind(query)

	rows, err := r.DB.QueryxContext(ctx, query, revID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var diags []*entity.WorkflowDiagnostic
	for rows.Next() {
		diag, err := scanWorkflowDiagnostic(rows)
		if err != nil {
			return nil, err
		}
		diags = append(diags, diag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return diags, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanWorkflow(row rowScanner) (*entity.Workflow, error) {
	var wf entity.Workflow
	err := row.Scan(
		&wf.ID,
		&wf.OrganizationID,
		&wf.RepositoryID,
		&wf.Name,
		&wf.Path,
		&wf.State,
		&wf.CreatedAt,
		&wf.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &wf, nil
}

func scanWorkflowRevision(row rowScanner) (*entity.WorkflowRevision, error) {
	var rev entity.WorkflowRevision
	var parsedAt sql.NullTime
	err := row.Scan(
		&rev.ID,
		&rev.WorkflowID,
		&rev.CommitSHA,
		&rev.RawContentHash,
		&rev.ParseStatus,
		&rev.IR,
		&parsedAt,
	)
	if err != nil {
		return nil, err
	}
	if parsedAt.Valid {
		t := parsedAt.Time
		rev.ParsedAt = &t
	}
	return &rev, nil
}

func scanWorkflowDiagnostic(row rowScanner) (*entity.WorkflowDiagnostic, error) {
	var diag entity.WorkflowDiagnostic
	err := row.Scan(
		&diag.ID,
		&diag.WorkflowRevisionID,
		&diag.Line,
		&diag.Col,
		&diag.Severity,
		&diag.Message,
	)
	if err != nil {
		return nil, err
	}
	return &diag, nil
}
