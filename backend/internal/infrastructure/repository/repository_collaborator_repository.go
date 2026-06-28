package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain/entity"
)

type sqlxRepositoryCollaboratorRepository struct {
	*sqlx.DB
}

func NewRepositoryCollaboratorRepository(db *sqlx.DB) *sqlxRepositoryCollaboratorRepository {
	return &sqlxRepositoryCollaboratorRepository{DB: db}
}

func (r *sqlxRepositoryCollaboratorRepository) AddCollaborator(ctx context.Context, repoID, userID uuid.UUID, permission string) error {
	var query string
	if r.DriverName() == "postgres" {
		query = `
			INSERT INTO repository_collaborators (repository_id, user_id, permission)
			VALUES ($1, $2, $3)
			ON CONFLICT (repository_id, user_id) DO UPDATE SET permission = EXCLUDED.permission
		`
	} else {
		query = `
			INSERT INTO repository_collaborators (repository_id, user_id, permission)
			VALUES (?, ?, ?)
			ON CONFLICT (repository_id, user_id) DO UPDATE SET permission = excluded.permission
		`
	}
	query = r.Rebind(query)
	_, err := r.ExecContext(ctx, query, repoID, userID, permission)
	return err
}

func (r *sqlxRepositoryCollaboratorRepository) RemoveCollaborator(ctx context.Context, repoID, userID uuid.UUID) error {
	query := `DELETE FROM repository_collaborators WHERE repository_id = ? AND user_id = ?`
	query = r.Rebind(query)
	_, err := r.ExecContext(ctx, query, repoID, userID)
	return err
}

func (r *sqlxRepositoryCollaboratorRepository) GetPermission(ctx context.Context, repoID, userID uuid.UUID) (string, error) {
	query := `SELECT permission FROM repository_collaborators WHERE repository_id = ? AND user_id = ?`
	query = r.Rebind(query)

	var permission string
	err := r.QueryRowxContext(ctx, query, repoID, userID).Scan(&permission)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return permission, nil
}

func (r *sqlxRepositoryCollaboratorRepository) ListCollaborators(ctx context.Context, repoID uuid.UUID) ([]*entity.RepositoryCollaborator, error) {
	query := `
		SELECT repository_id, user_id, permission
		FROM repository_collaborators
		WHERE repository_id = ?
		ORDER BY user_id
	`
	query = r.Rebind(query)

	rows, err := r.QueryxContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collaborators []*entity.RepositoryCollaborator
	for rows.Next() {
		var c entity.RepositoryCollaborator
		if err := rows.Scan(&c.RepositoryID, &c.UserID, &c.Permission); err != nil {
			return nil, err
		}
		collaborators = append(collaborators, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return collaborators, nil
}
