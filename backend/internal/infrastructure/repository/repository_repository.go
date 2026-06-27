package repository

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
)

var ErrInvalidDiskPath = errors.New("invalid disk path")

type sqlxRepositoryRepository struct {
	*sqlx.DB
}

func NewRepositoryRepository(db *sqlx.DB) *sqlxRepositoryRepository {
	return &sqlxRepositoryRepository{DB: db}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRepositoryFull(scanner rowScanner) (*entity.Repository, error) {
	var repo entity.Repository
	var isEmpty int
	err := scanner.Scan(
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

func scanRepositoryMetadata(scanner rowScanner) (*entity.Repository, error) {
	var repo entity.Repository
	var isEmpty int
	err := scanner.Scan(
		&repo.ID,
		&repo.OrganizationID,
		&repo.OwnerID,
		&repo.Name,
		&repo.Visibility,
		&repo.DefaultBranch,
		&repo.Description,
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

func validateDiskPath(diskPath string) error {
	if diskPath == "" {
		return nil
	}
	if strings.Contains(diskPath, "..") {
		return ErrInvalidDiskPath
	}
	if !filepath.IsAbs(diskPath) {
		return ErrInvalidDiskPath
	}
	if filepath.Clean(diskPath) != diskPath {
		return ErrInvalidDiskPath
	}
	return nil
}

func canViewRepository(repo *entity.Repository, requestUserID uuid.UUID) bool {
	switch repo.Visibility {
	case entity.VisibilityPublic:
		return true
	case entity.VisibilityPrivate, entity.VisibilityInternal:
		return requestUserID != uuid.Nil && requestUserID == repo.OwnerID
	default:
		return false
	}
}

func execUpdateOne(ctx context.Context, db *sqlx.DB, query string, args ...any) error {
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
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
	return scanRepositoryFull(row)
}

// GetByOwnerLoginAndName resolves a repository by owner login and name.
// disk_path is intentionally omitted. Visibility is enforced here: private and
// internal repositories are only returned when requestUserID matches the owner.
func (r *sqlxRepositoryRepository) GetByOwnerLoginAndName(ctx context.Context, ownerLogin, name string, requestUserID uuid.UUID) (*entity.Repository, error) {
	const query = `
		SELECT r.id, r.organization_id, r.owner_id, r.name, r.visibility, r.default_branch, r.description, r.is_empty, r.created_at
		FROM repositories r
		JOIN users u ON r.owner_id = u.id
		WHERE u.login = $1 AND r.name = $2
	`

	row := r.DB.QueryRowxContext(ctx, query, ownerLogin, name)
	repo, err := scanRepositoryMetadata(row)
	if err != nil || repo == nil {
		return repo, err
	}
	if !canViewRepository(repo, requestUserID) {
		return nil, nil
	}
	return repo, nil
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
		repo, err := scanRepositoryFull(rows)
		if err != nil {
			return nil, err
		}
		repos = append(repos, repo)
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
	if err := validateDiskPath(diskPath); err != nil {
		return err
	}
	const query = `UPDATE repositories SET disk_path = $1 WHERE id = $2`
	return execUpdateOne(ctx, r.DB, query, diskPath, id)
}

func (r *sqlxRepositoryRepository) SetIsEmpty(ctx context.Context, id uuid.UUID, isEmpty bool) error {
	isEmptyInt := 0
	if isEmpty {
		isEmptyInt = 1
	}
	const query = `UPDATE repositories SET is_empty = $1 WHERE id = $2`
	return execUpdateOne(ctx, r.DB, query, isEmptyInt, id)
}

func (r *sqlxRepositoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM repositories WHERE id = $1`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}
