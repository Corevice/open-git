package database

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"

	"github.com/google/uuid"

	"github.com/open-git/backend/internal/domain"
	repo "github.com/open-git/backend/internal/repository"
)

type organizationRepository struct {
	db *sql.DB
}

func NewOrganizationRepository(db *sql.DB) repo.IOrganizationRepository {
	return &organizationRepository{db: db}
}

// Organizations (like users) are stored with TEXT UUID primary keys, while the
// domain layer identifies them by int64. These helpers bridge the two using the
// lower 64 bits, matching middleware.Int64ToUUID / UUIDToInt64.
func orgInt64ToUUIDString(id int64) string {
	var u uuid.UUID
	binary.BigEndian.PutUint64(u[8:], uint64(id))
	return u.String()
}

func orgUUIDToInt64(s string) int64 {
	u, err := uuid.Parse(s)
	if err != nil {
		return 0
	}
	return int64(binary.BigEndian.Uint64(u[8:]))
}

func (r *organizationRepository) GetByLogin(ctx context.Context, login string) (*domain.Organization, error) {
	const query = `SELECT id, login, name, created_at FROM organizations WHERE login = $1`

	row := r.db.QueryRowContext(ctx, query, login)

	var org domain.Organization
	var idStr string
	err := row.Scan(&idStr, &org.Login, &org.Name, &org.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	org.ID = orgUUIDToInt64(idStr)
	return &org, nil
}

func (r *organizationRepository) ListByUserID(ctx context.Context, userID int64) ([]*domain.Organization, error) {
	const query = `
		SELECT o.id, o.login, o.name, o.created_at
		FROM organizations o
		JOIN memberships m ON o.id = m.organization_id
		WHERE m.user_id = $1
	`

	rows, err := r.db.QueryContext(ctx, query, orgInt64ToUUIDString(userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orgs := make([]*domain.Organization, 0)
	for rows.Next() {
		var org domain.Organization
		var idStr string
		if err := rows.Scan(&idStr, &org.Login, &org.Name, &org.CreatedAt); err != nil {
			return nil, err
		}
		org.ID = orgUUIDToInt64(idStr)
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
	err := r.db.QueryRowContext(ctx, query, orgInt64ToUUIDString(orgID), orgInt64ToUUIDString(userID)).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return role, nil
}
