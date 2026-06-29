package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/entity"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
)

type sqlxActionVerificationRepository struct {
	db *sqlx.DB
}

var _ domainrepo.IActionVerificationRepository = (*sqlxActionVerificationRepository)(nil)

func NewActionVerificationRepository(db *sqlx.DB) domainrepo.IActionVerificationRepository {
	return &sqlxActionVerificationRepository{db: db}
}

func (r *sqlxActionVerificationRepository) Create(ctx context.Context, v *entity.ActionVerification) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	if v.CreatedAt.IsZero() {
		v.CreatedAt = time.Now().UTC()
	}

	const query = `
		INSERT INTO action_verifications (
			id, organization_id, "trigger", status, requested_by,
			started_at, finished_at, created_at
		) VALUES (
			:id, :organization_id, :trigger, :status, :requested_by,
			:started_at, :finished_at, :created_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, map[string]any{
		"id":              v.ID,
		"organization_id": v.OrganizationID,
		"trigger":         v.Trigger,
		"status":          v.Status,
		"requested_by":    v.RequestedBy,
		"started_at":      v.StartedAt,
		"finished_at":     v.FinishedAt,
		"created_at":      v.CreatedAt,
	})
	return dbErrors.MapDBError(err)
}

func (r *sqlxActionVerificationRepository) Update(ctx context.Context, v *entity.ActionVerification) error {
	const query = `
		UPDATE action_verifications
		SET "trigger" = :trigger, status = :status, requested_by = :requested_by,
		    started_at = :started_at, finished_at = :finished_at
		WHERE id = :id
	`

	_, err := r.db.NamedExecContext(ctx, query, map[string]any{
		"id":           v.ID,
		"trigger":      v.Trigger,
		"status":       v.Status,
		"requested_by": v.RequestedBy,
		"started_at":   v.StartedAt,
		"finished_at":  v.FinishedAt,
	})
	return dbErrors.MapDBError(err)
}

const actionVerificationSelectColumns = `
	id, organization_id, "trigger", status, requested_by,
	started_at, finished_at, created_at
`

func (r *sqlxActionVerificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.ActionVerification, error) {
	const query = `SELECT ` + actionVerificationSelectColumns + ` FROM action_verifications WHERE id = ?`
	q := r.db.Rebind(query)

	var (
		v             entity.ActionVerification
		requestedByRaw any
		startedAt     sql.NullTime
		finishedAt    sql.NullTime
	)

	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&v.ID,
		&v.OrganizationID,
		&v.Trigger,
		&v.Status,
		&requestedByRaw,
		&startedAt,
		&finishedAt,
		&v.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}

	requestedBy, err := parseOptionalUUID(requestedByRaw)
	if err != nil {
		return nil, err
	}
	v.RequestedBy = requestedBy
	if startedAt.Valid {
		v.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		v.FinishedAt = &finishedAt.Time
	}

	return &v, nil
}

func (r *sqlxActionVerificationRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*entity.ActionVerification, error) {
	const query = `
		SELECT ` + actionVerificationSelectColumns + `
		FROM action_verifications
		WHERE organization_id = ?
		ORDER BY created_at DESC
	`
	q := r.db.Rebind(query)

	rows, err := r.db.QueryxContext(ctx, q, orgID)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	verifications := make([]*entity.ActionVerification, 0)
	for rows.Next() {
		var (
			v              entity.ActionVerification
			requestedByRaw any
			startedAt      sql.NullTime
			finishedAt     sql.NullTime
		)

		if err := rows.Scan(
			&v.ID,
			&v.OrganizationID,
			&v.Trigger,
			&v.Status,
			&requestedByRaw,
			&startedAt,
			&finishedAt,
			&v.CreatedAt,
		); err != nil {
			return nil, dbErrors.MapDBError(err)
		}

		requestedBy, err := parseOptionalUUID(requestedByRaw)
		if err != nil {
			return nil, err
		}
		v.RequestedBy = requestedBy
		if startedAt.Valid {
			v.StartedAt = &startedAt.Time
		}
		if finishedAt.Valid {
			v.FinishedAt = &finishedAt.Time
		}

		verifications = append(verifications, &v)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}

	return verifications, nil
}
