package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain/entity"
)

type sqlxRepositoryRepository struct {
	*sqlx.DB
}

func NewRepositoryRepository(db *sqlx.DB) *sqlxRepositoryRepository {
	return &sqlxRepositoryRepository{DB: db}
}

func (r *sqlxRepositoryRepository) Create(ctx context.Context, repo *entity.Repository) error {
	if repo.ID == uuid.Nil {
		repo.ID = uuid.New()
	}
	if repo.CreatedAt.IsZero() {
		repo.CreatedAt = time.Now().UTC()
	}

	const query = `
		INSERT INTO repositories (id, organization_id, owner_id, name, description, git_path, owner_login, visibility, default_branch, created_at)
		VALUES (:id, :organization_id, :owner_id, :name, :description, :git_path, :owner_login, :visibility, :default_branch, :created_at)
	`

	_, err := r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":              repo.ID,
		"organization_id": repo.OrganizationID,
		"owner_id":        repo.OwnerID,
		"name":            repo.Name,
		"description":     repo.Description,
		"git_path":        repo.GitPath,
		"owner_login":     repo.OwnerLogin,
		"visibility":      repo.Visibility,
		"default_branch":  repo.DefaultBranch,
		"created_at":      repo.CreatedAt,
	})
	return err
}

func (r *sqlxRepositoryRepository) GetByOwnerAndName(ctx context.Context, ownerID uuid.UUID, name string) (*entity.Repository, error) {
	const query = `
		SELECT id, organization_id, owner_id, name, description, git_path, owner_login, visibility, default_branch, created_at
		FROM repositories
		WHERE owner_id = $1 AND name = $2
	`

	row := r.DB.QueryRowxContext(ctx, query, ownerID, name)

	var repo entity.Repository
	err := row.Scan(
		&repo.ID,
		&repo.OrganizationID,
		&repo.OwnerID,
		&repo.Name,
		&repo.Description,
		&repo.GitPath,
		&repo.OwnerLogin,
		&repo.Visibility,
		&repo.DefaultBranch,
		&repo.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

func (r *sqlxRepositoryRepository) GetByOwnerLoginAndName(ctx context.Context, ownerLogin, name string) (*entity.Repository, error) {
	const query = `
		SELECT id, organization_id, owner_id, name, description, git_path, owner_login, visibility, default_branch, created_at
		FROM repositories
		WHERE owner_login = $1 AND name = $2
	`

	row := r.DB.QueryRowxContext(ctx, query, ownerLogin, name)

	var repo entity.Repository
	err := row.Scan(
		&repo.ID,
		&repo.OrganizationID,
		&repo.OwnerID,
		&repo.Name,
		&repo.Description,
		&repo.GitPath,
		&repo.OwnerLogin,
		&repo.Visibility,
		&repo.DefaultBranch,
		&repo.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

func (r *sqlxRepositoryRepository) ListByOrg(ctx context.Context, orgID uuid.UUID, page, perPage int) ([]*entity.Repository, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	offset := (page - 1) * perPage

	const query = `
		SELECT id, organization_id, owner_id, name, description, git_path, owner_login, visibility, default_branch, created_at
		FROM repositories
		WHERE organization_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.DB.QueryxContext(ctx, query, orgID, perPage, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []*entity.Repository
	for rows.Next() {
		var repo entity.Repository
		if err := rows.Scan(
			&repo.ID,
			&repo.OrganizationID,
			&repo.OwnerID,
			&repo.Name,
			&repo.Description,
			&repo.GitPath,
			&repo.OwnerLogin,
			&repo.Visibility,
			&repo.DefaultBranch,
			&repo.CreatedAt,
		); err != nil {
			return nil, err
		}
		repos = append(repos, &repo)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return repos, nil
}

func (r *sqlxRepositoryRepository) ListByOwner(ctx context.Context, ownerID uuid.UUID, page, perPage int) ([]*entity.Repository, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	offset := (page - 1) * perPage

	const query = `
		SELECT id, organization_id, owner_id, name, description, git_path, owner_login, visibility, default_branch, created_at
		FROM repositories
		WHERE owner_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.DB.QueryxContext(ctx, query, ownerID, perPage, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []*entity.Repository
	for rows.Next() {
		var repo entity.Repository
		if err := rows.Scan(
			&repo.ID,
			&repo.OrganizationID,
			&repo.OwnerID,
			&repo.Name,
			&repo.Description,
			&repo.GitPath,
			&repo.OwnerLogin,
			&repo.Visibility,
			&repo.DefaultBranch,
			&repo.CreatedAt,
		); err != nil {
			return nil, err
		}
		repos = append(repos, &repo)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return repos, nil
}

func (r *sqlxRepositoryRepository) UpdateVisibility(ctx context.Context, id uuid.UUID, visibility string) error {
	const query = `UPDATE repositories SET visibility = $1 WHERE id = $2`
	_, err := r.DB.ExecContext(ctx, query, visibility, id)
	return err
}

func (r *sqlxRepositoryRepository) UpdateName(ctx context.Context, id uuid.UUID, newName string) error {
	const query = `UPDATE repositories SET name = $1 WHERE id = $2`
	_, err := r.DB.ExecContext(ctx, query, newName, id)
	return err
}

func (r *sqlxRepositoryRepository) UpdateDefaultBranch(ctx context.Context, id uuid.UUID, branch string) error {
	const query = `UPDATE repositories SET default_branch = $1 WHERE id = $2`
	_, err := r.DB.ExecContext(ctx, query, branch, id)
	return err
}

func (r *sqlxRepositoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM repositories WHERE id = $1`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

func (r *sqlxRepositoryRepository) GetByID(ctx context.Context, repositoryID, organizationID uuid.UUID) (*entity.Repository, error) {
	const query = `
		SELECT id, organization_id, owner_id, name, description, git_path, owner_login, visibility, default_branch, created_at
		FROM repositories
		WHERE id = $1 AND organization_id = $2
	`

	row := r.DB.QueryRowxContext(ctx, query, repositoryID, organizationID)

	var repo entity.Repository
	err := row.Scan(
		&repo.ID,
		&repo.OrganizationID,
		&repo.OwnerID,
		&repo.Name,
		&repo.Description,
		&repo.GitPath,
		&repo.OwnerLogin,
		&repo.Visibility,
		&repo.DefaultBranch,
		&repo.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

func (r *sqlxRepositoryRepository) CountByOrg(ctx context.Context, organizationID uuid.UUID) (int, error) {
	const query = `SELECT COUNT(*) FROM repositories WHERE organization_id = $1`
	var count int
	if err := r.DB.QueryRowxContext(ctx, query, organizationID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *sqlxRepositoryRepository) CountByOwner(ctx context.Context, ownerID uuid.UUID) (int, error) {
	const query = `SELECT COUNT(*) FROM repositories WHERE owner_id = $1`
	var count int
	if err := r.DB.QueryRowxContext(ctx, query, ownerID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
