package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type sqlxDependabotAlertRepository struct {
	db *sqlx.DB
}

var _ domainrepo.IDependabotAlertRepository = (*sqlxDependabotAlertRepository)(nil)

func NewDependabotAlertRepository(db *sqlx.DB) domainrepo.IDependabotAlertRepository {
	return &sqlxDependabotAlertRepository{db: db}
}

const dependabotAlertSelectColumns = `
	id, organization_id, repository_id, alert_number, advisory_id, manifest_path, state, auto_dismissed_at
`

func scanDependabotAlert(scanner interface {
	Scan(dest ...any) error
}) (*entity.DependabotAlert, error) {
	var (
		alert           entity.DependabotAlert
		autoDismissedAt sql.NullTime
	)
	if err := scanner.Scan(
		&alert.ID,
		&alert.OrganizationID,
		&alert.RepositoryID,
		&alert.AlertNumber,
		&alert.AdvisoryID,
		&alert.ManifestPath,
		&alert.State,
		&autoDismissedAt,
	); err != nil {
		return nil, err
	}
	if autoDismissedAt.Valid {
		t := autoDismissedAt.Time
		alert.AutoDismissedAt = &t
	}
	return &alert, nil
}

func (r *sqlxDependabotAlertRepository) ListByRepo(ctx context.Context, orgID, repoID uuid.UUID, state string, page, perPage int) ([]*entity.DependabotAlert, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	offset := (page - 1) * perPage

	baseWhere := `WHERE organization_id = :org_id AND repository_id = :repo_id`
	args := map[string]any{
		"org_id":  orgID,
		"repo_id": repoID,
	}
	if state != "" {
		baseWhere += ` AND state = :state`
		args["state"] = state
	}

	countQuery := `SELECT COUNT(*) FROM dependabot_alerts ` + baseWhere
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
		"org_id":  orgID,
		"repo_id": repoID,
		"limit":   perPage,
		"offset":  offset,
	}
	for k, v := range args {
		listArgs[k] = v
	}
	listQuery := `SELECT ` + dependabotAlertSelectColumns + ` FROM dependabot_alerts ` + baseWhere +
		` ORDER BY alert_number ASC LIMIT :limit OFFSET :offset`

	rows, err := r.db.NamedQueryContext(ctx, listQuery, listArgs)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	alerts := make([]*entity.DependabotAlert, 0)
	for rows.Next() {
		alert, err := scanDependabotAlert(rows)
		if err != nil {
			return nil, 0, err
		}
		alerts = append(alerts, alert)
	}
	return alerts, total, rows.Err()
}

func (r *sqlxDependabotAlertRepository) GetByAlertNumber(ctx context.Context, orgID, repoID uuid.UUID, alertNumber int) (*entity.DependabotAlert, error) {
	query := `SELECT ` + dependabotAlertSelectColumns + `
		FROM dependabot_alerts
		WHERE organization_id = $1 AND repository_id = $2 AND alert_number = $3`
	row := r.db.QueryRowxContext(ctx, query, orgID, repoID, alertNumber)
	alert, err := scanDependabotAlert(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return alert, nil
}

func (r *sqlxDependabotAlertRepository) UpdateState(ctx context.Context, orgID, repoID uuid.UUID, alertNumber int, state entity.DependabotAlertState, reason *entity.DismissedReason) (*entity.DependabotAlert, error) {
	_ = reason

	const query = `
		UPDATE dependabot_alerts
		SET state = $1
		WHERE organization_id = $2 AND repository_id = $3 AND alert_number = $4
	`
	result, err := r.db.ExecContext(ctx, query, state, orgID, repoID, alertNumber)
	if err != nil {
		return nil, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, fmt.Errorf("dependabot alert not found")
	}
	return r.GetByAlertNumber(ctx, orgID, repoID, alertNumber)
}
