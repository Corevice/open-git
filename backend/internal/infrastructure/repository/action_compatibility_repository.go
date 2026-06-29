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

type sqlxActionCompatibilityRepository struct {
	db *sqlx.DB
}

var _ domainrepo.IActionCompatibilityRepository = (*sqlxActionCompatibilityRepository)(nil)

func NewActionCompatibilityRepository(db *sqlx.DB) domainrepo.IActionCompatibilityRepository {
	return &sqlxActionCompatibilityRepository{db: db}
}

func (r *sqlxActionCompatibilityRepository) UpsertResult(ctx context.Context, result *entity.ActionCompatibilityResult) error {
	if result.ID == uuid.Nil {
		result.ID = uuid.New()
	}
	now := time.Now().UTC()
	if result.CreatedAt.IsZero() {
		result.CreatedAt = now
	}
	result.UpdatedAt = now

	goldenDiffJSON, err := marshalGoldenDiff(result.GoldenDiff)
	if err != nil {
		return err
	}

	const query = `
		INSERT INTO action_compatibility_results (
			id, organization_id, repository_id, action_name, action_version,
			status, note, golden_diff, verified_at, verification_id,
			created_at, updated_at
		) VALUES (
			:id, :organization_id, :repository_id, :action_name, :action_version,
			:status, :note, :golden_diff, :verified_at, :verification_id,
			:created_at, :updated_at
		)
		ON CONFLICT (organization_id, action_name, action_version) DO UPDATE SET
			status = EXCLUDED.status,
			note = EXCLUDED.note,
			golden_diff = EXCLUDED.golden_diff,
			verified_at = EXCLUDED.verified_at,
			verification_id = EXCLUDED.verification_id,
			updated_at = EXCLUDED.updated_at
	`

	_, err = r.db.NamedExecContext(ctx, query, map[string]any{
		"id":              result.ID,
		"organization_id": result.OrganizationID,
		"repository_id":   result.RepositoryID,
		"action_name":     result.ActionName,
		"action_version":  result.ActionVersion,
		"status":          result.Status,
		"note":            result.Note,
		"golden_diff":     goldenDiffJSON,
		"verified_at":     result.VerifiedAt,
		"verification_id": result.VerificationID,
		"created_at":      result.CreatedAt,
		"updated_at":      result.UpdatedAt,
	})
	return dbErrors.MapDBError(err)
}

const actionCompatibilityResultSelectColumns = `
	id, organization_id, repository_id, action_name, action_version,
	status, note, golden_diff, verified_at, verification_id,
	created_at, updated_at
`

func (r *sqlxActionCompatibilityRepository) ListResults(ctx context.Context, orgID uuid.UUID, repoID *uuid.UUID) ([]*entity.ActionCompatibilityResult, error) {
	query := `
		SELECT ` + actionCompatibilityResultSelectColumns + `
		FROM action_compatibility_results
		WHERE organization_id = ?
	`
	args := []any{orgID}
	if repoID != nil {
		query += ` AND repository_id = ?`
		args = append(args, *repoID)
	}
	query += ` ORDER BY action_name, action_version`

	q := r.db.Rebind(query)
	rows, err := r.db.QueryxContext(ctx, q, args...)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	results := make([]*entity.ActionCompatibilityResult, 0)
	for rows.Next() {
		result, err := scanActionCompatibilityResult(rows)
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

func (r *sqlxActionCompatibilityRepository) GetResult(ctx context.Context, orgID uuid.UUID, actionName, actionVersion string) (*entity.ActionCompatibilityResult, error) {
	const query = `
		SELECT ` + actionCompatibilityResultSelectColumns + `
		FROM action_compatibility_results
		WHERE organization_id = ? AND action_name = ? AND action_version = ?
	`
	q := r.db.Rebind(query)

	result, err := scanActionCompatibilityResult(r.db.QueryRowxContext(ctx, q, orgID, actionName, actionVersion))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return result, nil
}

type actionCompatibilityResultScanner interface {
	Scan(dest ...any) error
}

func scanActionCompatibilityResult(scanner actionCompatibilityResultScanner) (*entity.ActionCompatibilityResult, error) {
	var (
		result         entity.ActionCompatibilityResult
		repositoryIDRaw any
		note           sql.NullString
		goldenDiffRaw  any
		verifiedAt     sql.NullTime
	)

	if err := scanner.Scan(
		&result.ID,
		&result.OrganizationID,
		&repositoryIDRaw,
		&result.ActionName,
		&result.ActionVersion,
		&result.Status,
		&note,
		&goldenDiffRaw,
		&verifiedAt,
		&result.VerificationID,
		&result.CreatedAt,
		&result.UpdatedAt,
	); err != nil {
		return nil, err
	}

	repositoryID, err := parseOptionalUUID(repositoryIDRaw)
	if err != nil {
		return nil, err
	}
	result.RepositoryID = repositoryID
	if note.Valid {
		result.Note = &note.String
	}
	if verifiedAt.Valid {
		result.VerifiedAt = &verifiedAt.Time
	}

	goldenDiffJSON, err := jsonBytesFromDBValue(goldenDiffRaw)
	if err != nil {
		return nil, err
	}
	if len(goldenDiffJSON) > 0 {
		var diff map[string]any
		if err := json.Unmarshal(goldenDiffJSON, &diff); err != nil {
			return nil, err
		}
		result.GoldenDiff = diff
	}

	return &result, nil
}

func marshalGoldenDiff(diff map[string]any) (any, error) {
	if diff == nil {
		return nil, nil
	}
	data, err := json.Marshal(diff)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}
