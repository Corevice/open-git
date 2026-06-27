package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type MembershipRepository struct {
	*sqlx.DB
}

func NewMembershipRepository(db *sqlx.DB) *MembershipRepository {
	return &MembershipRepository{DB: db}
}

func (r *MembershipRepository) HasReadAccess(ctx context.Context, userID, organizationID uuid.UUID) (bool, error) {
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

func (r *MembershipRepository) HasWriteAccess(ctx context.Context, userID, organizationID uuid.UUID) (bool, error) {
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
