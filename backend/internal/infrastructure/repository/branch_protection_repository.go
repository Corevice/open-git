package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/entity"
)

type sqlxBranchProtectionRepository struct {
	db *sqlx.DB
}

var _ domainrepo.IBranchProtectionRepository = (*sqlxBranchProtectionRepository)(nil)

func NewBranchProtectionRepository(db *sqlx.DB) domainrepo.IBranchProtectionRepository {
	return &sqlxBranchProtectionRepository{db: db}
}

func (r *sqlxBranchProtectionRepository) GetByBranch(ctx context.Context, repoID uuid.UUID, branch string) (*entity.BranchProtection, error) {
	const query = `
		SELECT id, repository_id, pattern, required_reviews, required_checks
		FROM branch_protections
		WHERE repository_id = $1 AND pattern = $2
	`

	row := r.db.QueryRowxContext(ctx, query, repoID, branch)
	bp, err := scanBranchProtectionRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return bp, nil
}

func (r *sqlxBranchProtectionRepository) Upsert(ctx context.Context, bp *entity.BranchProtection) error {
	if bp.ID == uuid.Nil {
		bp.ID = uuid.New()
	}

	var orgID uuid.UUID
	const orgQuery = `SELECT organization_id FROM repositories WHERE id = $1`
	if err := r.db.QueryRowxContext(ctx, orgQuery, bp.RepositoryID).Scan(&orgID); err != nil {
		return err
	}

	checksJSON, err := json.Marshal(bp.RequiredStatusChecks)
	if err != nil {
		return err
	}
	if bp.RequiredStatusChecks == nil {
		checksJSON = []byte("[]")
	}

	const query = `
		INSERT OR REPLACE INTO branch_protections (
			id, organization_id, repository_id, pattern, required_reviews, required_checks, created_at
		) VALUES (
			:id, :organization_id, :repository_id, :pattern, :required_reviews, :required_checks, :created_at
		)
	`

	_, err = r.db.NamedExecContext(ctx, query, map[string]any{
		"id":               bp.ID,
		"organization_id":  orgID,
		"repository_id":    bp.RepositoryID,
		"pattern":          bp.Pattern,
		"required_reviews": bp.RequiredApprovingReviews,
		"required_checks":  string(checksJSON),
		"created_at":       time.Now().UTC(),
	})
	return err
}

type branchProtectionScanner interface {
	Scan(dest ...any) error
}

func scanBranchProtectionRow(scanner branchProtectionScanner) (*entity.BranchProtection, error) {
	var (
		bp         entity.BranchProtection
		checksJSON string
	)

	if err := scanner.Scan(
		&bp.ID,
		&bp.RepositoryID,
		&bp.Pattern,
		&bp.RequiredApprovingReviews,
		&checksJSON,
	); err != nil {
		return nil, err
	}

	if checksJSON != "" {
		if err := json.Unmarshal([]byte(checksJSON), &bp.RequiredStatusChecks); err != nil {
			return nil, err
		}
	}
	if bp.RequiredStatusChecks == nil {
		bp.RequiredStatusChecks = []string{}
	}

	return &bp, nil
}
