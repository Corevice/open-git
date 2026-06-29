package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type sqlxSecurityAdvisoryRepository struct {
	db *sqlx.DB
}

var _ domainrepo.ISecurityAdvisoryRepository = (*sqlxSecurityAdvisoryRepository)(nil)

func NewSecurityAdvisoryRepository(db *sqlx.DB) domainrepo.ISecurityAdvisoryRepository {
	return &sqlxSecurityAdvisoryRepository{db: db}
}

const securityAdvisorySelectColumns = `
	id, organization_id, repository_id, ghsa_id, cve_id, severity, summary, description,
	affected_package, affected_versions, patched_versions, state, dismissed_reason,
	created_at, updated_at
`

func scanSecurityAdvisory(scanner interface {
	Scan(dest ...any) error
}) (*entity.SecurityAdvisory, error) {
	var (
		advisory  entity.SecurityAdvisory
		repoID    sql.NullString
		cveID     sql.NullString
		dismissed sql.NullString
		createdAt time.Time
		updatedAt time.Time
	)
	if err := scanner.Scan(
		&advisory.ID,
		&advisory.OrganizationID,
		&repoID,
		&advisory.GHSAPID,
		&cveID,
		&advisory.Severity,
		&advisory.Summary,
		&advisory.Description,
		&advisory.AffectedPackage,
		&advisory.AffectedVersions,
		&advisory.PatchedVersions,
		&advisory.State,
		&dismissed,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}
	if repoID.Valid {
		parsed, err := uuid.Parse(repoID.String)
		if err != nil {
			return nil, err
		}
		advisory.RepositoryID = &parsed
	}
	if cveID.Valid {
		advisory.CVEID = cveID.String
	}
	if dismissed.Valid {
		reason := entity.DismissedReason(dismissed.String)
		advisory.DismissedReason = &reason
	}
	advisory.CreatedAt = createdAt
	advisory.UpdatedAt = updatedAt
	return &advisory, nil
}

func (r *sqlxSecurityAdvisoryRepository) ListByOrg(ctx context.Context, orgID uuid.UUID, state, severity string, page, perPage int) ([]*entity.SecurityAdvisory, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	offset := (page - 1) * perPage

	baseWhere := `WHERE organization_id = :org_id`
	args := map[string]any{"org_id": orgID}
	if state != "" {
		baseWhere += ` AND state = :state`
		args["state"] = state
	}
	if severity != "" {
		baseWhere += ` AND severity = :severity`
		args["severity"] = severity
	}

	countQuery := `SELECT COUNT(*) FROM security_advisories ` + baseWhere
	countRows, err := r.db.NamedQueryContext(ctx, countQuery, args)
	if err != nil {
		return nil, 0, err
	}
	defer countRows.Close()
	var total int
	if countRows.Next() {
		if err := countRows.Scan(&total); err != nil {
			return nil, 0, err
		}
	}

	listArgs := map[string]any{
		"org_id": orgID,
		"limit":  perPage,
		"offset": offset,
	}
	for k, v := range args {
		listArgs[k] = v
	}
	listQuery := `SELECT ` + securityAdvisorySelectColumns + ` FROM security_advisories ` + baseWhere +
		` ORDER BY created_at DESC LIMIT :limit OFFSET :offset`

	rows, err := r.db.NamedQueryContext(ctx, listQuery, listArgs)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	advisories := make([]*entity.SecurityAdvisory, 0)
	for rows.Next() {
		advisory, err := scanSecurityAdvisory(rows)
		if err != nil {
			return nil, 0, err
		}
		advisories = append(advisories, advisory)
	}
	return advisories, total, rows.Err()
}

func (r *sqlxSecurityAdvisoryRepository) GetByGHSAPID(ctx context.Context, orgID uuid.UUID, ghsaID string) (*entity.SecurityAdvisory, error) {
	query := `SELECT ` + securityAdvisorySelectColumns + ` FROM security_advisories WHERE organization_id = $1 AND ghsa_id = $2`
	row := r.db.QueryRowxContext(ctx, query, orgID, ghsaID)
	advisory, err := scanSecurityAdvisory(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return advisory, nil
}

func (r *sqlxSecurityAdvisoryRepository) UpdateState(ctx context.Context, orgID uuid.UUID, ghsaID string, state entity.AdvisoryState, reason *entity.DismissedReason) (*entity.SecurityAdvisory, error) {
	var dismissed sql.NullString
	if reason != nil && *reason != "" {
		dismissed = sql.NullString{String: string(*reason), Valid: true}
	}

	const query = `
		UPDATE security_advisories
		SET state = $1, dismissed_reason = $2, updated_at = $3
		WHERE organization_id = $4 AND ghsa_id = $5
	`
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, query, state, dismissed, now, orgID, ghsaID)
	if err != nil {
		return nil, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, fmt.Errorf("security advisory not found")
	}
	return r.GetByGHSAPID(ctx, orgID, ghsaID)
}
