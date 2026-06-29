package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
)

type sqlxMembershipRepository struct {
	*sqlx.DB
}

func NewMembershipRepository(db *sqlx.DB) *sqlxMembershipRepository {
	return &sqlxMembershipRepository{DB: db}
}

func (r *sqlxMembershipRepository) Add(ctx context.Context, m *entity.Membership) error {
	const query = `
		INSERT INTO memberships (organization_id, user_id, role)
		VALUES (:organization_id, :user_id, :role)
	`

	_, err := r.DB.NamedExecContext(ctx, query, map[string]any{
		"organization_id": m.OrganizationID,
		"user_id":         m.UserID,
		"role":            m.Role,
	})
	return dbErrors.MapDBError(err)
}

func (r *sqlxMembershipRepository) GetRole(ctx context.Context, orgID, userID uuid.UUID) (string, error) {
	query := `SELECT role FROM memberships WHERE organization_id = ? AND user_id = ?`
	query = r.DB.Rebind(query)

	var role string
	err := r.DB.QueryRowxContext(ctx, query, orgID, userID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", domain.ErrNotFound
	}
	if err != nil {
		return "", dbErrors.MapDBError(err)
	}
	return role, nil
}

func (r *sqlxMembershipRepository) ListByOrg(ctx context.Context, orgID uuid.UUID, page, perPage int) ([]*entity.Membership, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	offset := (page - 1) * perPage

	query := `
		SELECT organization_id, user_id, role
		FROM memberships
		WHERE organization_id = ?
		ORDER BY role ASC
		LIMIT ? OFFSET ?
	`
	query = r.DB.Rebind(query)

	rows, err := r.DB.QueryxContext(ctx, query, orgID, perPage, offset)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	var memberships []*entity.Membership
	for rows.Next() {
		var m entity.Membership
		if err := rows.Scan(&m.OrganizationID, &m.UserID, &m.Role); err != nil {
			return nil, dbErrors.MapDBError(err)
		}
		memberships = append(memberships, &m)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return memberships, nil
}

func (r *sqlxMembershipRepository) UpdateRole(ctx context.Context, orgID, userID uuid.UUID, role string) error {
	query := `UPDATE memberships SET role = ? WHERE organization_id = ? AND user_id = ?`
	query = r.DB.Rebind(query)
	_, err := r.DB.ExecContext(ctx, query, role, orgID, userID)
	return dbErrors.MapDBError(err)
}

func (r *sqlxMembershipRepository) Remove(ctx context.Context, orgID, userID uuid.UUID) error {
	query := `DELETE FROM memberships WHERE organization_id = ? AND user_id = ?`
	query = r.DB.Rebind(query)
	_, err := r.DB.ExecContext(ctx, query, orgID, userID)
	return dbErrors.MapDBError(err)
}
