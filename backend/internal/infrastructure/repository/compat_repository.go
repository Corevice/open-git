package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/entity"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
)

type sqlxCompatRepository struct {
	*sqlx.DB
}

var _ domainrepo.ICompatRepository = (*sqlxCompatRepository)(nil)

func NewCompatRepository(db *sqlx.DB) domainrepo.ICompatRepository {
	return &sqlxCompatRepository{DB: db}
}

func (r *sqlxCompatRepository) CreateRun(ctx context.Context, run *entity.CompatTestRun) error {
	if run.ID == uuid.Nil {
		run.ID = uuid.New()
	}
	if run.CreatedAt.IsZero() {
		run.CreatedAt = time.Now().UTC()
	}

	const query = `
		INSERT INTO compat_test_run (
			id, suite, status, triggered_by, organization_id,
			total_endpoints, passing, failing, unimplemented, coverage_rate,
			started_at, finished_at, created_at
		) VALUES (
			:id, :suite, :status, :triggered_by, :organization_id,
			:total_endpoints, :passing, :failing, :unimplemented, :coverage_rate,
			:started_at, :finished_at, :created_at
		)
	`

	_, err := r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":               run.ID,
		"suite":            run.Suite,
		"status":           run.Status,
		"triggered_by":     run.TriggeredBy,
		"organization_id":  run.OrganizationID,
		"total_endpoints":  run.TotalEndpoints,
		"passing":          run.Passing,
		"failing":          run.Failing,
		"unimplemented":    run.Unimplemented,
		"coverage_rate":    run.CoverageRate,
		"started_at":       run.StartedAt,
		"finished_at":      run.FinishedAt,
		"created_at":       run.CreatedAt,
	})
	return dbErrors.MapDBError(err)
}

func (r *sqlxCompatRepository) UpdateRun(ctx context.Context, run *entity.CompatTestRun) error {
	const query = `
		UPDATE compat_test_run
		SET status = ?, total_endpoints = ?, passing = ?, failing = ?,
		    unimplemented = ?, coverage_rate = ?, started_at = ?, finished_at = ?
		WHERE id = ?
	`

	q := r.DB.Rebind(query)
	_, err := r.DB.ExecContext(
		ctx,
		q,
		run.Status,
		run.TotalEndpoints,
		run.Passing,
		run.Failing,
		run.Unimplemented,
		run.CoverageRate,
		run.StartedAt,
		run.FinishedAt,
		run.ID,
	)
	return dbErrors.MapDBError(err)
}

const compatTestRunSelectColumns = `
	id, suite, status, triggered_by, organization_id,
	total_endpoints, passing, failing, unimplemented, coverage_rate,
	started_at, finished_at, created_at
`

func (r *sqlxCompatRepository) GetRun(ctx context.Context, id uuid.UUID) (*entity.CompatTestRun, error) {
	const query = `SELECT ` + compatTestRunSelectColumns + ` FROM compat_test_run WHERE id = ?`
	q := r.DB.Rebind(query)

	run, err := scanCompatTestRun(r.DB.QueryRowxContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return run, nil
}

func (r *sqlxCompatRepository) ListRuns(ctx context.Context, orgID uuid.UUID, limit int) ([]*entity.CompatTestRun, error) {
	const query = `
		SELECT ` + compatTestRunSelectColumns + `
		FROM compat_test_run
		WHERE organization_id = ?
		ORDER BY started_at DESC
		LIMIT ?
	`
	q := r.DB.Rebind(query)

	rows, err := r.DB.QueryxContext(ctx, q, orgID, limit)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	runs := make([]*entity.CompatTestRun, 0)
	for rows.Next() {
		run, err := scanCompatTestRun(rows)
		if err != nil {
			return nil, dbErrors.MapDBError(err)
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return runs, nil
}

func (r *sqlxCompatRepository) CreateEndpointResult(ctx context.Context, result *entity.CompatEndpointResult) error {
	if result.ID == uuid.Nil {
		result.ID = uuid.New()
	}

	checksJSON, err := marshalCompatEndpointChecks(result.Checks)
	if err != nil {
		return err
	}

	diffJSON, err := marshalCompatEndpointDiff(result.Diff)
	if err != nil {
		return err
	}

	const query = `
		INSERT INTO compat_endpoint_result (id, run_id, method, path, status, checks, diff)
		VALUES (:id, :run_id, :method, :path, :status, :checks, :diff)
	`

	_, err = r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":     result.ID,
		"run_id": result.RunID,
		"method": result.Method,
		"path":   result.Path,
		"status": result.Status,
		"checks": checksJSON,
		"diff":   diffJSON,
	})
	return dbErrors.MapDBError(err)
}

func (r *sqlxCompatRepository) ListEndpointResults(ctx context.Context, runID uuid.UUID) ([]*entity.CompatEndpointResult, error) {
	const query = `
		SELECT id, run_id, method, path, status, checks, diff
		FROM compat_endpoint_result
		WHERE run_id = ?
		ORDER BY path, method
	`
	q := r.DB.Rebind(query)

	rows, err := r.DB.QueryxContext(ctx, q, runID)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	results := make([]*entity.CompatEndpointResult, 0)
	for rows.Next() {
		result, err := scanCompatEndpointResult(rows)
		if err != nil {
			return nil, dbErrors.MapDBError(err)
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return results, nil
}

type compatTestRunScanner interface {
	Scan(dest ...any) error
}

func scanCompatTestRun(scanner compatTestRunScanner) (*entity.CompatTestRun, error) {
	var (
		run            entity.CompatTestRun
		triggeredByRaw any
		startedAt      nullTime
		finishedAt     nullTime
		createdAt      nullTime
	)

	if err := scanner.Scan(
		&run.ID,
		&run.Suite,
		&run.Status,
		&triggeredByRaw,
		&run.OrganizationID,
		&run.TotalEndpoints,
		&run.Passing,
		&run.Failing,
		&run.Unimplemented,
		&run.CoverageRate,
		&startedAt,
		&finishedAt,
		&createdAt,
	); err != nil {
		return nil, err
	}
	run.CreatedAt = createdAt.Time

	triggeredBy, err := parseOptionalUUID(triggeredByRaw)
	if err != nil {
		return nil, err
	}
	run.TriggeredBy = triggeredBy
	if startedAt.Valid {
		run.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		run.FinishedAt = &finishedAt.Time
	}

	return &run, nil
}

type compatEndpointResultScanner interface {
	Scan(dest ...any) error
}

func scanCompatEndpointResult(scanner compatEndpointResultScanner) (*entity.CompatEndpointResult, error) {
	var (
		result    entity.CompatEndpointResult
		checksRaw any
		diffRaw   any
	)

	if err := scanner.Scan(
		&result.ID,
		&result.RunID,
		&result.Method,
		&result.Path,
		&result.Status,
		&checksRaw,
		&diffRaw,
	); err != nil {
		return nil, err
	}

	checksJSON, err := jsonBytesFromDBValue(checksRaw)
	if err != nil {
		return nil, err
	}
	if len(checksJSON) > 0 {
		checks := &entity.CompatEndpointChecks{}
		if err := json.Unmarshal(checksJSON, checks); err != nil {
			return nil, err
		}
		result.Checks = checks
	}

	diffJSON, err := jsonBytesFromDBValue(diffRaw)
	if err != nil {
		return nil, err
	}
	if len(diffJSON) > 0 {
		result.Diff = json.RawMessage(diffJSON)
	}

	return &result, nil
}

func parseOptionalUUID(raw any) (*uuid.UUID, error) {
	switch value := raw.(type) {
	case nil:
		return nil, nil
	case []byte:
		if len(value) == 0 {
			return nil, nil
		}
		id, err := uuid.Parse(string(value))
		if err != nil {
			return nil, err
		}
		return &id, nil
	case string:
		if value == "" {
			return nil, nil
		}
		id, err := uuid.Parse(value)
		if err != nil {
			return nil, err
		}
		return &id, nil
	default:
		return nil, errors.New("unsupported uuid column type")
	}
}

func jsonBytesFromDBValue(raw any) ([]byte, error) {
	switch value := raw.(type) {
	case nil:
		return nil, nil
	case []byte:
		return value, nil
	case string:
		return []byte(value), nil
	default:
		return nil, errors.New("unsupported json column type")
	}
}

func marshalCompatEndpointChecks(checks *entity.CompatEndpointChecks) (any, error) {
	if checks == nil {
		return nil, nil
	}
	data, err := json.Marshal(checks)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

func marshalCompatEndpointDiff(diff json.RawMessage) (any, error) {
	if len(diff) == 0 {
		return nil, nil
	}
	if !json.Valid(diff) {
		return nil, errors.New("invalid diff json")
	}
	return string(diff), nil
}
