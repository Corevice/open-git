package repository

import (
	"context"
	"database/sql"
	"errors"
	"os"
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

func scanRepository(scanner rowScanner, includeDiskPath bool) (*entity.Repository, error) {
	var repo entity.Repository
	var isEmpty int
	dest := []any{
		&repo.ID,
		&repo.OrganizationID,
		&repo.OwnerID,
		&repo.Name,
		&repo.Visibility,
		&repo.DefaultBranch,
		&repo.Description,
	}
	if includeDiskPath {
		dest = append(dest, &repo.DiskPath)
	}
	dest = append(dest, &isEmpty, &repo.CreatedAt)

	err := scanner.Scan(dest...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	repo.IsEmpty = isEmpty != 0
	return &repo, nil
}

func gitStorageBaseDir() string {
	if baseDir := os.Getenv("GIT_STORAGE_PATH"); baseDir != "" {
		return baseDir
	}
	return "/data/repos"
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
	cleaned := filepath.Clean(diskPath)
	if cleaned != diskPath {
		return ErrInvalidDiskPath
	}
	baseAbs, err := filepath.Abs(gitStorageBaseDir())
	if err != nil {
		return ErrInvalidDiskPath
	}
	pathAbs, err := filepath.Abs(cleaned)
	if err != nil {
		return ErrInvalidDiskPath
	}
	rel, err := filepath.Rel(baseAbs, pathAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ErrInvalidDiskPath
	}
	return nil
}

func (r *sqlxRepositoryRepository) isRepositoryCollaborator(ctx context.Context, repositoryID, userID uuid.UUID) (bool, error) {
	const query = `
		SELECT COUNT(*)
		FROM repository_collaborators
		WHERE repository_id = $1 AND user_id = $2
	`
	var count int
	if err := r.DB.GetContext(ctx, &count, query, repositoryID, userID); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *sqlxRepositoryRepository) hasOrganizationMembership(ctx context.Context, organizationID, userID uuid.UUID) (bool, error) {
	const query = `
		SELECT COUNT(*)
		FROM memberships
		WHERE organization_id = $1 AND user_id = $2
	`
	var count int
	if err := r.DB.GetContext(ctx, &count, query, organizationID, userID); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *sqlxRepositoryRepository) canViewRepository(ctx context.Context, repo *entity.Repository, requestUserID uuid.UUID) (bool, error) {
	switch repo.Visibility {
	case entity.VisibilityPublic:
		return true, nil
	case entity.VisibilityPrivate, entity.VisibilityInternal:
		if requestUserID == uuid.Nil {
			return false, nil
		}
		if requestUserID == repo.OwnerID {
			return true, nil
		}
		isCollaborator, err := r.isRepositoryCollaborator(ctx, repo.ID, requestUserID)
		if err != nil {
			return false, err
		}
		if isCollaborator {
			return true, nil
		}
		if repo.Visibility == entity.VisibilityInternal {
			return r.hasOrganizationMembership(ctx, repo.OrganizationID, requestUserID)
		}
		return false, nil
	default:
		return false, nil
	}
}

func (r *sqlxRepositoryRepository) authorizeRepositoryWrite(ctx context.Context, requestUserID, repositoryID uuid.UUID) error {
	if requestUserID == uuid.Nil {
		return domain.ErrForbidden
	}

	const query = `
		SELECT owner_id
		FROM repositories
		WHERE id = $1
	`
	var ownerID uuid.UUID
	if err := r.DB.GetContext(ctx, &ownerID, query, repositoryID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	if requestUserID == ownerID {
		return nil
	}

	const collaboratorQuery = `
		SELECT COUNT(*)
		FROM repository_collaborators
		WHERE repository_id = $1 AND user_id = $2 AND permission IN ('write', 'admin')
	`
	var count int
	if err := r.DB.GetContext(ctx, &count, collaboratorQuery, repositoryID, requestUserID); err != nil {
		return err
	}
	if count == 0 {
		return domain.ErrForbidden
	}
	return nil
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
	return scanRepository(row, true)
}

// GetByOwnerLoginAndName resolves a repository by owner login and name.
// Visibility is enforced here: private/internal repositories require owner,
// collaborator, or (for internal) organization membership access.
func (r *sqlxRepositoryRepository) GetByOwnerLoginAndName(ctx context.Context, ownerLogin, name string, requestUserID uuid.UUID) (*entity.Repository, error) {
	const query = `
		SELECT r.id, r.organization_id, r.owner_id, r.name, r.visibility, r.default_branch, r.description, r.disk_path, r.is_empty, r.created_at
		FROM repositories r
		JOIN users u ON r.owner_id = u.id
		WHERE u.login = $1 AND r.name = $2
	`

	row := r.DB.QueryRowxContext(ctx, query, ownerLogin, name)
	repo, err := scanRepository(row, true)
	if err != nil || repo == nil {
		return repo, err
	}
	canView, err := r.canViewRepository(ctx, repo, requestUserID)
	if err != nil {
		return nil, err
	}
	if !canView {
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
		repo, err := scanRepository(rows, true)
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

func (r *sqlxRepositoryRepository) UpdateDiskPath(ctx context.Context, requestUserID, id uuid.UUID, diskPath string) error {
	if err := r.authorizeRepositoryWrite(ctx, requestUserID, id); err != nil {
		return err
	}
	if err := validateDiskPath(diskPath); err != nil {
		return err
	}
	const query = `UPDATE repositories SET disk_path = $1 WHERE id = $2`
	return execUpdateOne(ctx, r.DB, query, diskPath, id)
}

func (r *sqlxRepositoryRepository) SetIsEmpty(ctx context.Context, requestUserID, id uuid.UUID, isEmpty bool) error {
	if err := r.authorizeRepositoryWrite(ctx, requestUserID, id); err != nil {
		return err
	}
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
