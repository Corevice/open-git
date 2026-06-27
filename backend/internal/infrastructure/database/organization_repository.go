package database

import (
	"context"
	"database/sql"

	"github.com/open-git/backend/internal/domain"
)

type organizationRepository struct {
	db *sql.DB
}

func NewOrganizationRepository(db *sql.DB) *organizationRepository {
	return &organizationRepository{db: db}
}

func (r *organizationRepository) GetByLogin(ctx context.Context, login string) (*domain.Organization, error) {
	const query = `SELECT id, login, name, created_at FROM organizations WHERE login = $1`

	row := r.db.QueryRowContext(ctx, query, login)

	var org domain.Organization
	err := row.Scan(&org.ID, &org.Login, &org.Name, &org.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &org, nil
}

func (r *organizationRepository) ListByUserID(ctx context.Context, userID int64) ([]*domain.Organization, error) {
	const query = `
		SELECT o.id, o.login, o.name, o.created_at
		FROM organizations o
		JOIN memberships m ON o.id = m.organization_id
		WHERE m.user_id = $1
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orgs := make([]*domain.Organization, 0)
	for rows.Next() {
		var org domain.Organization
		if err := rows.Scan(&org.ID, &org.Login, &org.Name, &org.CreatedAt); err != nil {
			return nil, err
		}
		orgs = append(orgs, &org)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orgs, nil
}

func (r *organizationRepository) GetMemberRole(ctx context.Context, orgID, userID int64) (string, error) {
	const query = `SELECT role FROM memberships WHERE organization_id = $1 AND user_id = $2`

	var role string
	err := r.db.QueryRowContext(ctx, query, orgID, userID).Scan(&role)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return role, nil
}
