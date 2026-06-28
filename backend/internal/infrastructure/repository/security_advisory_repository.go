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

type sqlxSecurityAdvisoryRepository struct {
	db *sqlx.DB
}

var _ domainrepo.ISecurityAdvisoryRepository = (*sqlxSecurityAdvisoryRepository)(nil)

// SecurityAdvisoryFullRepository includes write methods used by scanners.
type SecurityAdvisoryFullRepository interface {
	domainrepo.ISecurityAdvisoryRepository
	Create(ctx context.Context, advisory *entity.SecurityAdvisory) error
	Upsert(ctx context.Context, advisory *entity.SecurityAdvisory) error
}

var _ SecurityAdvisoryFullRepository = (*sqlxSecurityAdvisoryRepository)(nil)

func NewSecurityAdvisoryRepository(db *sqlx.DB) SecurityAdvisoryFullRepository {
	return &sqlxSecurityAdvisoryRepository{db: db}
}

const securityAdvisorySelectColumns = `
	id, organization_id, repository_id, ghsa_id, cve_id, severity, summary, description,
	affected_package, affected_versions, patched_versions, state, dismissed_reason, created_at, updated_at
`

func (r *sqlxSecurityAdvisoryRepository) Create(ctx context.Context, advisory *entity.SecurityAdvisory) error {
	if advisory.ID == uuid.Nil {
		advisory.ID = uuid.New()
	}
	now := time.Now().UTC()
	if advisory.CreatedAt.IsZero() {
		advisory.CreatedAt = now
	}
	if advisory.UpdatedAt.IsZero() {
		advisory.UpdatedAt = now
	}
	if advisory.State == "" {
		advisory.State = entity.AdvisoryStateOpen
	}

	const query = `
		INSERT INTO security_advisories (
			id, organization_id, repository_id, ghsa_id, cve_id, severity, summary, description,
			affected_package, affected_versions, patched_versions, state, dismissed_reason, created_at, updated_at
		) VALUES (
			:id, :organization_id, :repository_id, :ghsa_id, :cve_id, :severity, :summary, :description,
			:affected_package, :affected_versions, :patched_versions, :state, :dismissed_reason, :created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, securityAdvisoryParams(advisory))
	return dbErrors.MapDBError(err)
}

func (r *sqlxSecurityAdvisoryRepository) Upsert(ctx context.Context, advisory *entity.SecurityAdvisory) error {
	existing, err := r.GetByGHSAPID(ctx, advisory.OrganizationID, advisory.GHSAPID)
	if err != nil {
		return err
	}
	if existing != nil {
		advisory.ID = existing.ID
		advisory.CreatedAt = existing.CreatedAt
	} else if advisory.ID == uuid.Nil {
		advisory.ID = uuid.New()
	}

	now := time.Now().UTC()
	if advisory.CreatedAt.IsZero() {
		advisory.CreatedAt = now
	}
	advisory.UpdatedAt = now
	if advisory.State == "" {
		advisory.State = entity.AdvisoryStateOpen
	}

	const query = `
		INSERT OR REPLACE INTO security_advisories (
			id, organization_id, repository_id, ghsa_id, cve_id, severity, summary, description,
			affected_package, affected_versions, patched_versions, state, dismissed_reason, created_at, updated_at
		) VALUES (
			:id, :organization_id, :repository_id, :ghsa_id, :cve_id, :severity, :summary, :description,
			:affected_package, :affected_versions, :patched_versions, :state, :dismissed_reason, :created_at, :updated_at
		)
	`

	_, err = r.db.NamedExecContext(ctx, query, securityAdvisoryParams(advisory))
	return dbErrors.MapDBError(err)
}

func (r *sqlxSecurityAdvisoryRepository) ListByOrg(
	ctx context.Context,
	orgID uuid.UUID,
	state, severity string,
	page, perPage int,
) ([]*entity.SecurityAdvisory, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	offset := (page - 1) * perPage

	baseQuery := `SELECT ` + securityAdvisorySelectColumns + ` FROM security_advisories WHERE organization_id = ?`
	countQuery := `SELECT COUNT(*) FROM security_advisories WHERE organization_id = ?`
	args := []any{orgID}

	if state != "" {
		baseQuery += ` AND state = ?`
		countQuery += ` AND state = ?`
		args = append(args, state)
	}
	if severity != "" {
		baseQuery += ` AND severity = ?`
		countQuery += ` AND severity = ?`
		args = append(args, severity)
	}

	countQuery = r.db.Rebind(countQuery)
	var total int
	if err := r.db.QueryRowxContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, dbErrors.MapDBError(err)
	}

	listQuery := baseQuery + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	listArgs := append(append([]any{}, args...), perPage, offset)
	listQuery = r.db.Rebind(listQuery)

	rows, err := r.db.QueryxContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	advisories := make([]*entity.SecurityAdvisory, 0)
	for rows.Next() {
		advisory, err := scanSecurityAdvisory(rows)
		if err != nil {
			return nil, 0, dbErrors.MapDBError(err)
		}
		advisories = append(advisories, advisory)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, dbErrors.MapDBError(err)
	}
	return advisories, total, nil
}

func (r *sqlxSecurityAdvisoryRepository) GetByGHSAPID(ctx context.Context, orgID uuid.UUID, ghsaID string) (*entity.SecurityAdvisory, error) {
	query := `SELECT ` + securityAdvisorySelectColumns + ` FROM security_advisories WHERE organization_id = ? AND ghsa_id = ?`
	query = r.db.Rebind(query)

	advisory, err := scanSecurityAdvisory(r.db.QueryRowxContext(ctx, query, orgID, ghsaID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return advisory, nil
}

func (r *sqlxSecurityAdvisoryRepository) UpdateState(
	ctx context.Context,
	orgID uuid.UUID,
	ghsaID string,
	state entity.AdvisoryState,
	reason *entity.DismissedReason,
) (*entity.SecurityAdvisory, error) {
	var dismissedReason any
	if state == entity.AdvisoryStateDismissed && reason != nil {
		dismissedReason = string(*reason)
	}

	query := `
		UPDATE security_advisories
		SET state = ?, dismissed_reason = ?, updated_at = ?
		WHERE organization_id = ? AND ghsa_id = ?
	`
	query = r.db.Rebind(query)
	now := time.Now().UTC()

	result, err := r.db.ExecContext(ctx, query, state, dismissedReason, now, orgID, ghsaID)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	if rows == 0 {
		return nil, nil
	}

	return r.GetByGHSAPID(ctx, orgID, ghsaID)
}

func securityAdvisoryParams(advisory *entity.SecurityAdvisory) map[string]any {
	var repoID any
	if advisory.RepositoryID != nil {
		repoID = *advisory.RepositoryID
	}

	var dismissedReason any
	if advisory.DismissedReason != nil {
		dismissedReason = string(*advisory.DismissedReason)
	}

	return map[string]any{
		"id":                advisory.ID,
		"organization_id":   advisory.OrganizationID,
		"repository_id":     repoID,
		"ghsa_id":           advisory.GHSAPID,
		"cve_id":            advisory.CVEID,
		"severity":          advisory.Severity,
		"summary":           advisory.Summary,
		"description":       advisory.Description,
		"affected_package":  advisory.AffectedPackage,
		"affected_versions": advisory.AffectedVersions,
		"patched_versions":  advisory.PatchedVersions,
		"state":             advisory.State,
		"dismissed_reason":  dismissedReason,
		"created_at":        advisory.CreatedAt,
		"updated_at":        advisory.UpdatedAt,
	}
}

type securityAdvisoryScanner interface {
	Scan(dest ...any) error
}

func scanSecurityAdvisory(row securityAdvisoryScanner) (*entity.SecurityAdvisory, error) {
	var (
		advisory        entity.SecurityAdvisory
		repositoryID    sql.NullString
		dismissedReason sql.NullString
	)

	err := row.Scan(
		&advisory.ID,
		&advisory.OrganizationID,
		&repositoryID,
		&advisory.GHSAPID,
		&advisory.CVEID,
		&advisory.Severity,
		&advisory.Summary,
		&advisory.Description,
		&advisory.AffectedPackage,
		&advisory.AffectedVersions,
		&advisory.PatchedVersions,
		&advisory.State,
		&dismissedReason,
		&advisory.CreatedAt,
		&advisory.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if repositoryID.Valid {
		parsed, err := uuid.Parse(repositoryID.String)
		if err != nil {
			return nil, err
		}
		advisory.RepositoryID = &parsed
	}
	if dismissedReason.Valid {
		reason := entity.DismissedReason(dismissedReason.String)
		advisory.DismissedReason = &reason
	}

	return &advisory, nil
}
