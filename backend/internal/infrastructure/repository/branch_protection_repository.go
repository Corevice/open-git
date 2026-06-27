package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/database"
)

type BranchProtectionRepository struct {
	db *sqlx.DB
}

func NewBranchProtectionRepository(db *sqlx.DB) *BranchProtectionRepository {
	return &BranchProtectionRepository{db: db}
}

func (r *BranchProtectionRepository) GetForRef(ctx context.Context, repoID uuid.UUID, ref string) (*entity.BranchProtection, error) {
	const query = `
		SELECT required_reviews, required_checks
		FROM branch_protections
		WHERE repository_id = $1 AND pattern = $2
		LIMIT 1
	`
	var requiredReviews int
	var requiredChecksRaw string
	err := database.SQLxExecutor(ctx, r.db).QueryRowxContext(ctx, query, repoID, ref).Scan(&requiredReviews, &requiredChecksRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var requiredChecks []string
	if requiredChecksRaw != "" {
		if err := json.Unmarshal([]byte(requiredChecksRaw), &requiredChecks); err != nil {
			return nil, err
		}
	}

	return &entity.BranchProtection{
		RequiredReviews: requiredReviews,
		RequiredChecks:  requiredChecks,
	}, nil
}

type GitBranchProtectionStore struct {
	db *sqlx.DB
}

func NewGitBranchProtectionStore(db *sqlx.DB) *GitBranchProtectionStore {
	return &GitBranchProtectionStore{db: db}
}

func (s *GitBranchProtectionStore) IsBranchProtected(ctx context.Context, repositoryID uuid.UUID, branch string) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1 FROM branch_protections
			WHERE repository_id = $1 AND pattern = $2
		)
	`
	var exists bool
	if err := database.SQLxExecutor(ctx, s.db).QueryRowxContext(ctx, query, repositoryID, branch).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}
