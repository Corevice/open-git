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
		INSERT INTO repositories (id, organization_id, owner_id, name, visibility, default_branch, description, disk_path, is_empty, created_at)
		VALUES (:id, :organization_id, :owner_id, :name, :visibility, :default_branch, :description, :disk_path, :is_empty, :created_at)
	`

	isEmpty := 0
	if repo.IsEmpty {
		isEmpty = 1
	}

	_, err := r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":              repo.ID,
		"organization_id": repo.OrganizationID,
		"owner_id":        repo.OwnerID,
		"name":            repo.Name,
		"visibility":      repo.Visibility,
		"default_branch":  repo.DefaultBranch,
		"description":     repo.Description,
		"disk_path":       repo.DiskPath,
		"is_empty":        isEmpty,
		"created_at":      repo.CreatedAt,
	})
	return err
}

func (r *sqlxRepositoryRepository) GetByOwnerAndName(ctx context.Context, ownerID uuid.UUID, name string) (*entity.Repository, error) {
	const query = `
		SELECT id, organization_id, owner_id, name, visibility, default_branch, description, disk_path, is_empty, created_at
		FROM repositories
		WHERE owner_id = $1 AND name = $2
	`

	row := r.DB.QueryRowxContext(ctx, query, ownerID, name)

	var repo entity.Repository
	var isEmpty int
	err := row.Scan(
		&repo.ID,
		&repo.OrganizationID,
		&repo.OwnerID,
		&repo.Name,
		&repo.Visibility,
		&repo.DefaultBranch,
		&repo.Description,
		&repo.DiskPath,
		&isEmpty,
		&repo.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	repo.IsEmpty = isEmpty != 0
	return &repo, nil
}

func (r *sqlxRepositoryRepository) GetByOwnerLoginAndName(ctx context.Context, ownerLogin, name string) (*entity.Repository, error) {
	const query = `
		SELECT r.id, r.organization_id, r.owner_id, r.name, r.visibility, r.default_branch, r.description, r.disk_path, r.is_empty, r.created_at
		FROM repositories r
		JOIN users u ON r.owner_id = u.id
		WHERE u.login = $1 AND r.name = $2
	`

	row := r.DB.QueryRowxContext(ctx, query, ownerLogin, name)

	var repo entity.Repository
	var isEmpty int
	err := row.Scan(
		&repo.ID,
		&repo.OrganizationID,
		&repo.OwnerID,
		&repo.Name,
		&repo.Visibility,
		&repo.DefaultBranch,
		&repo.Description,
		&repo.DiskPath,
		&isEmpty,
		&repo.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	repo.IsEmpty = isEmpty != 0
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
		SELECT id, organization_id, owner_id, name, visibility, default_branch, description, disk_path, is_empty, created_at
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
		var isEmpty int
		if err := rows.Scan(
			&repo.ID,
			&repo.OrganizationID,
			&repo.OwnerID,
			&repo.Name,
			&repo.Visibility,
			&repo.DefaultBranch,
			&repo.Description,
			&repo.DiskPath,
			&isEmpty,
			&repo.CreatedAt,
		); err != nil {
			return nil, err
		}
		repo.IsEmpty = isEmpty != 0
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

func (r *sqlxRepositoryRepository) UpdateDiskPath(ctx context.Context, id uuid.UUID, diskPath string) error {
	const query = `UPDATE repositories SET disk_path = $1 WHERE id = $2`
	_, err := r.DB.ExecContext(ctx, query, diskPath, id)
	return err
}

func (r *sqlxRepositoryRepository) SetIsEmpty(ctx context.Context, id uuid.UUID, isEmpty bool) error {
	isEmptyInt := 0
	if isEmpty {
		isEmptyInt = 1
	}
	const query = `UPDATE repositories SET is_empty = $1 WHERE id = $2`
	_, err := r.DB.ExecContext(ctx, query, isEmptyInt, id)
	return err
}

func (r *sqlxRepositoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM repositories WHERE id = $1`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}
