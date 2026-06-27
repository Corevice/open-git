package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type sqlxMembershipRepository struct {
	*sqlx.DB
}

func NewMembershipRepository(db *sqlx.DB) *sqlxMembershipRepository {
	return &sqlxMembershipRepository{DB: db}
}

func (r *sqlxMembershipRepository) HasReadAccess(ctx context.Context, userID, organizationID uuid.UUID) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1 FROM memberships
			WHERE organization_id = $1 AND user_id = $2
		)
	`

	var exists bool
	if err := r.DB.QueryRowxContext(ctx, query, organizationID, userID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (r *sqlxMembershipRepository) HasWriteAccess(ctx context.Context, userID, organizationID uuid.UUID) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1 FROM memberships
			WHERE organization_id = $1 AND user_id = $2
			AND role IN ('owner', 'admin')
		)
	`

	var exists bool
	if err := r.DB.QueryRowxContext(ctx, query, organizationID, userID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}
