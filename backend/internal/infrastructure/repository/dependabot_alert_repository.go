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

func (r *sqlxDependabotAlertRepository) ListByRepo(
	ctx context.Context,
	orgID, repoID uuid.UUID,
	state string,
	page, perPage int,
) ([]*entity.DependabotAlert, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	offset := (page - 1) * perPage

	baseQuery := `SELECT ` + dependabotAlertSelectColumns + ` FROM dependabot_alerts WHERE organization_id = ? AND repository_id = ?`
	countQuery := `SELECT COUNT(*) FROM dependabot_alerts WHERE organization_id = ? AND repository_id = ?`
	args := []any{orgID, repoID}

	if state != "" {
		baseQuery += ` AND state = ?`
		countQuery += ` AND state = ?`
		args = append(args, state)
	}

	countQuery = r.db.Rebind(countQuery)
	var total int
	if err := r.db.QueryRowxContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, dbErrors.MapDBError(err)
	}

	listQuery := baseQuery + ` ORDER BY alert_number ASC LIMIT ? OFFSET ?`
	listArgs := append(append([]any{}, args...), perPage, offset)
	listQuery = r.db.Rebind(listQuery)

	rows, err := r.db.QueryxContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	alerts, err := scanDependabotAlerts(rows)
	if err != nil {
		return nil, 0, err
	}
	return alerts, total, nil
}

func (r *sqlxDependabotAlertRepository) GetByAlertNumber(
	ctx context.Context,
	orgID, repoID uuid.UUID,
	alertNumber int,
) (*entity.DependabotAlert, error) {
	_ = orgID
	query := `SELECT ` + dependabotAlertSelectColumns + `
		FROM dependabot_alerts
		WHERE repository_id = ? AND alert_number = ?`
	query = r.db.Rebind(query)

	alert, err := scanDependabotAlert(r.db.QueryRowxContext(ctx, query, repoID, alertNumber))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return alert, nil
}

func (r *sqlxDependabotAlertRepository) UpdateState(
	ctx context.Context,
	orgID, repoID uuid.UUID,
	alertNumber int,
	state entity.DependabotAlertState,
	reason *entity.DismissedReason,
) (*entity.DependabotAlert, error) {
	var autoDismissedAt any
	if state == entity.DependabotAlertStateDismissed {
		now := time.Now().UTC()
		autoDismissedAt = now
	}

	query := `
		UPDATE dependabot_alerts
		SET state = ?, auto_dismissed_at = ?
		WHERE organization_id = ? AND repository_id = ? AND alert_number = ?
	`
	query = r.db.Rebind(query)

	result, err := r.db.ExecContext(ctx, query, state, autoDismissedAt, orgID, repoID, alertNumber)
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

	alert, err := r.GetByAlertNumber(ctx, orgID, repoID, alertNumber)
	if err != nil {
		return nil, err
	}
	if alert != nil && reason != nil {
		alert.DismissedReason = reason
	}
	return alert, nil
}

type dependabotAlertScanner interface {
	Scan(dest ...any) error
}

func scanDependabotAlert(row dependabotAlertScanner) (*entity.DependabotAlert, error) {
	var (
		alert           entity.DependabotAlert
		autoDismissedAt sql.NullTime
	)

	err := row.Scan(
		&alert.ID,
		&alert.OrganizationID,
		&alert.RepositoryID,
		&alert.AlertNumber,
		&alert.AdvisoryID,
		&alert.ManifestPath,
		&alert.State,
		&autoDismissedAt,
	)
	if err != nil {
		return nil, err
	}

	if autoDismissedAt.Valid {
		t := autoDismissedAt.Time
		alert.AutoDismissedAt = &t
	}

	return &alert, nil
}

func scanDependabotAlerts(rows *sqlx.Rows) ([]*entity.DependabotAlert, error) {
	alerts := make([]*entity.DependabotAlert, 0)
	for rows.Next() {
		alert, err := scanDependabotAlert(rows)
		if err != nil {
			return nil, dbErrors.MapDBError(err)
		}
		alerts = append(alerts, alert)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return alerts, nil
}
