package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain/entity"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
	repo "github.com/open-git/backend/internal/repository"
)

type sqlxRepositoryCollaboratorRepository struct {
	*sqlx.DB
}

var _ repo.IRepositoryCollaboratorRepository = (*sqlxRepositoryCollaboratorRepository)(nil)

func NewRepositoryCollaboratorRepository(db *sqlx.DB) *sqlxRepositoryCollaboratorRepository {
	return &sqlxRepositoryCollaboratorRepository{DB: db}
}

func (r *sqlxRepositoryCollaboratorRepository) AddCollaborator(ctx context.Context, repoID, userID uuid.UUID, permission string) error {
	const query = `
		INSERT INTO repository_collaborators (repository_id, user_id, permission)
		VALUES (:repository_id, :user_id, :permission)
		ON CONFLICT(repository_id, user_id) DO UPDATE SET permission = excluded.permission
	`

	_, err := r.DB.NamedExecContext(ctx, query, map[string]any{
		"repository_id": repoID,
		"user_id":       userID,
		"permission":    permission,
	})
	return dbErrors.MapDBError(err)
}

func (r *sqlxRepositoryCollaboratorRepository) RemoveCollaborator(ctx context.Context, repoID, userID uuid.UUID) error {
	query := `DELETE FROM repository_collaborators WHERE repository_id = ? AND user_id = ?`
	query = r.DB.Rebind(query)

	_, err := r.DB.ExecContext(ctx, query, repoID, userID)
	return dbErrors.MapDBError(err)
}

func (r *sqlxRepositoryCollaboratorRepository) GetPermission(ctx context.Context, repoID, userID uuid.UUID) (string, error) {
	query := `SELECT permission FROM repository_collaborators WHERE repository_id = ? AND user_id = ?`
	query = r.DB.Rebind(query)

	var permission string
	err := r.DB.QueryRowxContext(ctx, query, repoID, userID).Scan(&permission)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", dbErrors.MapDBError(err)
	}
	return permission, nil
}

func (r *sqlxRepositoryCollaboratorRepository) ListCollaborators(ctx context.Context, repoID uuid.UUID) ([]*entity.RepositoryCollaborator, error) {
	query := `
		SELECT repository_id, user_id, permission
		FROM repository_collaborators
		WHERE repository_id = ?
		ORDER BY permission ASC
	`
	query = r.DB.Rebind(query)

	rows, err := r.DB.QueryxContext(ctx, query, repoID)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	var collaborators []*entity.RepositoryCollaborator
	for rows.Next() {
		var collab entity.RepositoryCollaborator
		if err := rows.Scan(&collab.RepositoryID, &collab.UserID, &collab.Permission); err != nil {
			return nil, dbErrors.MapDBError(err)
		}
		collaborators = append(collaborators, &collab)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return collaborators, nil
}
