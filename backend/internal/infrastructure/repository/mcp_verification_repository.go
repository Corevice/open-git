package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
)

type sqlxMCPVerificationRepository struct {
	db *sqlx.DB
}

var _ domainrepo.IMCPVerificationRepository = (*sqlxMCPVerificationRepository)(nil)

func NewSQLxMCPVerificationRepository(db *sqlx.DB) *sqlxMCPVerificationRepository {
	return &sqlxMCPVerificationRepository{db: db}
}

const mcpVerificationRunSelectColumns = `
	id, organization_id, repository_id, triggered_by, status, overall_status,
	targets, started_at, finished_at, created_at
`

func (r *sqlxMCPVerificationRepository) CreateRun(ctx context.Context, run *entity.MCPVerificationRun) error {
	if run.ID == uuid.Nil {
		run.ID = uuid.New()
	}
	if run.CreatedAt.IsZero() {
		run.CreatedAt = time.Now().UTC()
	}

	targets := run.Targets
	if len(targets) == 0 {
		targets = json.RawMessage("[]")
	}

	const query = `
		INSERT INTO mcp_verification_runs (
			id, organization_id, repository_id, triggered_by, status, overall_status,
			targets, started_at, finished_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		run.ID,
		run.OrganizationID,
		run.RepositoryID,
		run.TriggeredBy,
		run.Status,
		run.OverallStatus,
		targets,
		run.StartedAt,
		run.FinishedAt,
		run.CreatedAt,
	)
	return dbErrors.MapDBError(err)
}

func (r *sqlxMCPVerificationRepository) GetRunByID(ctx context.Context, id, orgID uuid.UUID) (*entity.MCPVerificationRun, error) {
	query := `SELECT ` + mcpVerificationRunSelectColumns + `
		FROM mcp_verification_runs
		WHERE id = $1 AND organization_id = $2`

	run, err := scanMCPVerificationRun(r.db.QueryRowxContext(ctx, query, id, orgID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return run, nil
}

func (r *sqlxMCPVerificationRepository) UpdateRun(ctx context.Context, run *entity.MCPVerificationRun) error {
	targets := run.Targets
	if len(targets) == 0 {
		targets = json.RawMessage("[]")
	}

	const query = `
		UPDATE mcp_verification_runs
		SET repository_id = $1,
		    triggered_by = $2,
		    status = $3,
		    overall_status = $4,
		    targets = $5,
		    started_at = $6,
		    finished_at = $7
		WHERE id = $8 AND organization_id = $9
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		run.RepositoryID,
		run.TriggeredBy,
		run.Status,
		run.OverallStatus,
		targets,
		run.StartedAt,
		run.FinishedAt,
		run.ID,
		run.OrganizationID,
	)
	if err != nil {
		return dbErrors.MapDBError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	if rowsAffected == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func (r *sqlxMCPVerificationRepository) DeleteRun(ctx context.Context, id, orgID uuid.UUID) error {
	const query = `DELETE FROM mcp_verification_runs WHERE id = $1 AND organization_id = $2`

	result, err := r.db.ExecContext(ctx, query, id, orgID)
	if err != nil {
		return dbErrors.MapDBError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	if rowsAffected == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func (r *sqlxMCPVerificationRepository) GetLatestRun(ctx context.Context, orgID uuid.UUID) (*entity.MCPVerificationRun, error) {
	query := `SELECT ` + mcpVerificationRunSelectColumns + `
		FROM mcp_verification_runs
		WHERE organization_id = $1
		ORDER BY created_at DESC
		LIMIT 1`

	run, err := scanMCPVerificationRun(r.db.QueryRowxContext(ctx, query, orgID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return run, nil
}

func (r *sqlxMCPVerificationRepository) ListRuns(ctx context.Context, orgID uuid.UUID, page, perPage int) ([]*entity.MCPVerificationRun, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	offset := (page - 1) * perPage

	const countQuery = `SELECT COUNT(*) FROM mcp_verification_runs WHERE organization_id = $1`

	var total int64
	if err := r.db.GetContext(ctx, &total, countQuery, orgID); err != nil {
		return nil, 0, dbErrors.MapDBError(err)
	}

	query := `SELECT ` + mcpVerificationRunSelectColumns + `
		FROM mcp_verification_runs
		WHERE organization_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryxContext(ctx, query, orgID, perPage, offset)
	if err != nil {
		return nil, 0, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	runs := make([]*entity.MCPVerificationRun, 0)
	for rows.Next() {
		run, err := scanMCPVerificationRun(rows)
		if err != nil {
			return nil, 0, dbErrors.MapDBError(err)
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, dbErrors.MapDBError(err)
	}
	return runs, total, nil
}

func (r *sqlxMCPVerificationRepository) GetActiveRun(ctx context.Context, orgID uuid.UUID) (*entity.MCPVerificationRun, error) {
	const query = `
		SELECT * FROM mcp_verification_runs
		WHERE organization_id = $1 AND status IN ('queued', 'running')
		LIMIT 1`

	run, err := scanMCPVerificationRun(r.db.QueryRowxContext(ctx, query, orgID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return run, nil
}

func (r *sqlxMCPVerificationRepository) CountRunsThisMonth(ctx context.Context, orgID uuid.UUID) (int64, error) {
	const query = `
		SELECT COUNT(*)
		FROM mcp_verification_runs
		WHERE organization_id = $1 AND created_at >= date_trunc('month', NOW())`

	var count int64
	if err := r.db.GetContext(ctx, &count, query, orgID); err != nil {
		return 0, dbErrors.MapDBError(err)
	}
	return count, nil
}

func (r *sqlxMCPVerificationRepository) BatchCreateChecks(ctx context.Context, checks []*entity.MCPVerificationCheck) error {
	if len(checks) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	defer func() { _ = tx.Rollback() }()

	const query = `
		INSERT INTO mcp_verification_checks (
			id, run_id, organization_id, check_id, category, status,
			expected, actual, error, duration_ms, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`

	for _, check := range checks {
		if check == nil {
			continue
		}
		if check.ID == uuid.Nil {
			check.ID = uuid.New()
		}
		if check.CreatedAt.IsZero() {
			check.CreatedAt = time.Now().UTC()
		}

		_, err := tx.ExecContext(
			ctx,
			query,
			check.ID,
			check.RunID,
			check.OrganizationID,
			check.CheckID,
			check.Category,
			check.Status,
			check.Expected,
			check.Actual,
			check.Error,
			check.DurationMS,
			check.CreatedAt,
		)
		if err != nil {
			return dbErrors.MapDBError(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return dbErrors.MapDBError(err)
	}
	return nil
}

func (r *sqlxMCPVerificationRepository) ListChecksByRun(ctx context.Context, runID, orgID uuid.UUID) ([]*entity.MCPVerificationCheck, error) {
	const query = `
		SELECT id, run_id, organization_id, check_id, category, status,
		       expected, actual, error, duration_ms, created_at
		FROM mcp_verification_checks
		WHERE run_id = $1 AND organization_id = $2
		ORDER BY created_at ASC`

	rows, err := r.db.QueryxContext(ctx, query, runID, orgID)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	checks := make([]*entity.MCPVerificationCheck, 0)
	for rows.Next() {
		check, err := scanMCPVerificationCheck(rows)
		if err != nil {
			return nil, dbErrors.MapDBError(err)
		}
		checks = append(checks, check)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return checks, nil
}

type mcpVerificationRunScanner interface {
	Scan(dest ...any) error
}

func scanMCPVerificationRun(scanner mcpVerificationRunScanner) (*entity.MCPVerificationRun, error) {
	var (
		run             entity.MCPVerificationRun
		repositoryIDRaw any
		triggeredByRaw  any
		overallStatus   sql.NullString
		targetsRaw      any
		startedAt       sql.NullTime
		finishedAt      sql.NullTime
	)

	if err := scanner.Scan(
		&run.ID,
		&run.OrganizationID,
		&repositoryIDRaw,
		&triggeredByRaw,
		&run.Status,
		&overallStatus,
		&targetsRaw,
		&startedAt,
		&finishedAt,
		&run.CreatedAt,
	); err != nil {
		return nil, err
	}

	repositoryID, err := parseOptionalUUID(repositoryIDRaw)
	if err != nil {
		return nil, err
	}
	run.RepositoryID = repositoryID

	triggeredBy, err := parseOptionalUUID(triggeredByRaw)
	if err != nil {
		return nil, err
	}
	run.TriggeredBy = triggeredBy

	if overallStatus.Valid {
		status := entity.OverallStatus(overallStatus.String)
		run.OverallStatus = &status
	}

	targetsJSON, err := jsonBytesFromDBValue(targetsRaw)
	if err != nil {
		return nil, err
	}
	if len(targetsJSON) > 0 {
		run.Targets = json.RawMessage(targetsJSON)
	} else {
		run.Targets = json.RawMessage("[]")
	}

	if startedAt.Valid {
		run.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		run.FinishedAt = &finishedAt.Time
	}

	return &run, nil
}

type mcpVerificationCheckScanner interface {
	Scan(dest ...any) error
}

func scanMCPVerificationCheck(scanner mcpVerificationCheckScanner) (*entity.MCPVerificationCheck, error) {
	var (
		check       entity.MCPVerificationCheck
		expectedRaw any
		actualRaw   any
		errorRaw    sql.NullString
	)

	if err := scanner.Scan(
		&check.ID,
		&check.RunID,
		&check.OrganizationID,
		&check.CheckID,
		&check.Category,
		&check.Status,
		&expectedRaw,
		&actualRaw,
		&errorRaw,
		&check.DurationMS,
		&check.CreatedAt,
	); err != nil {
		return nil, err
	}

	expectedJSON, err := jsonBytesFromDBValue(expectedRaw)
	if err != nil {
		return nil, err
	}
	if len(expectedJSON) > 0 {
		check.Expected = json.RawMessage(expectedJSON)
	}

	actualJSON, err := jsonBytesFromDBValue(actualRaw)
	if err != nil {
		return nil, err
	}
	if len(actualJSON) > 0 {
		check.Actual = json.RawMessage(actualJSON)
	}

	if errorRaw.Valid {
		msg := errorRaw.String
		check.Error = &msg
	}

	return &check, nil
}
