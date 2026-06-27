package repository

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/database"
)

type PullRequestRepository struct {
	db *sqlx.DB
}

func NewPullRequestRepository(db *sqlx.DB) *PullRequestRepository {
	return &PullRequestRepository{db: db}
}

func (r *PullRequestRepository) Create(ctx context.Context, pr *entity.PullRequest) error {
	if pr.AuthorID == uuid.Nil {
		return apperror.ErrValidation
	}

	const query = `
		INSERT INTO pull_requests (
			id, organization_id, repository_id, number, head_ref, base_ref, state, author_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := database.SQLxExecutor(ctx, r.db).ExecContext(
		ctx,
		query,
		pr.ID,
		pr.OrganizationID,
		pr.RepositoryID,
		pr.Number,
		pr.HeadRef,
		pr.BaseRef,
		pr.State,
		pr.AuthorID,
		time.Now().UTC(),
	)
	return err
}

func (r *PullRequestRepository) GetByNumber(ctx context.Context, repoID uuid.UUID, number int) (*entity.PullRequest, error) {
	const query = `
		SELECT id, organization_id, repository_id, number, head_ref, base_ref, state, author_id, merged_at
		FROM pull_requests
		WHERE repository_id = $1 AND number = $2
	`
	row := database.SQLxExecutor(ctx, r.db).QueryRowxContext(ctx, query, repoID, number)

	var pr entity.PullRequest
	var mergedAt sql.NullTime
	err := row.Scan(
		&pr.ID,
		&pr.OrganizationID,
		&pr.RepositoryID,
		&pr.Number,
		&pr.HeadRef,
		&pr.BaseRef,
		&pr.State,
		&pr.AuthorID,
		&mergedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if mergedAt.Valid {
		t := mergedAt.Time
		pr.MergedAt = &t
	}
	return &pr, nil
}

func validatePullRequestState(state string) error {
	if state == "" {
		return nil
	}
	switch state {
	case entity.PullRequestStateOpen, entity.PullRequestStateClosed, entity.PullRequestStateMerged:
		return nil
	default:
		return apperror.ErrValidation
	}
}

func (r *PullRequestRepository) ListByRepo(ctx context.Context, repoID uuid.UUID, state string, page, perPage int) ([]*entity.PullRequest, error) {
	if err := validatePullRequestState(state); err != nil {
		return nil, err
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	offset := (page - 1) * perPage

	query := `
		SELECT id, organization_id, repository_id, number, head_ref, base_ref, state, author_id, merged_at
		FROM pull_requests
		WHERE repository_id = $1
	`
	args := []any{repoID}
	idx := 2
	if state != "" {
		query += " AND state = $" + strconv.Itoa(idx)
		args = append(args, state)
		idx++
	}
	query += " ORDER BY number DESC LIMIT $" + strconv.Itoa(idx) + " OFFSET $" + strconv.Itoa(idx+1)
	args = append(args, perPage, offset)

	rows, err := database.SQLxExecutor(ctx, r.db).QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pulls []*entity.PullRequest
	for rows.Next() {
		var pr entity.PullRequest
		var mergedAt sql.NullTime
		if err := rows.Scan(
			&pr.ID,
			&pr.OrganizationID,
			&pr.RepositoryID,
			&pr.Number,
			&pr.HeadRef,
			&pr.BaseRef,
			&pr.State,
			&pr.AuthorID,
			&mergedAt,
		); err != nil {
			return nil, err
		}
		if mergedAt.Valid {
			t := mergedAt.Time
			pr.MergedAt = &t
		}
		pulls = append(pulls, &pr)
	}
	return pulls, rows.Err()
}

func (r *PullRequestRepository) UpdateState(ctx context.Context, id uuid.UUID, state string) error {
	const query = `UPDATE pull_requests SET state = $1 WHERE id = $2`
	_, err := database.SQLxExecutor(ctx, r.db).ExecContext(ctx, query, state, id)
	return err
}

func (r *PullRequestRepository) SetMerged(ctx context.Context, id uuid.UUID, mergedAt time.Time) error {
	const query = `UPDATE pull_requests SET state = $1, merged_at = $2 WHERE id = $3`
	_, err := database.SQLxExecutor(ctx, r.db).ExecContext(ctx, query, entity.PullRequestStateMerged, mergedAt, id)
	return err
}

func (r *PullRequestRepository) Update(ctx context.Context, pr *entity.PullRequest) error {
	const query = `
		UPDATE pull_requests
		SET state = $1, merged_at = $2, head_ref = $3, base_ref = $4
		WHERE id = $5
	`
	var mergedAt any
	if pr.MergedAt != nil {
		mergedAt = *pr.MergedAt
	}
	_, err := database.SQLxExecutor(ctx, r.db).ExecContext(ctx, query, pr.State, mergedAt, pr.HeadRef, pr.BaseRef, pr.ID)
	return err
}

func (r *PullRequestRepository) NextNumber(ctx context.Context, repoID uuid.UUID) (int, error) {
	const query = `SELECT COALESCE(MAX(number), 0) + 1 FROM pull_requests WHERE repository_id = $1`
	var next int
	if err := database.SQLxExecutor(ctx, r.db).QueryRowxContext(ctx, query, repoID).Scan(&next); err != nil {
		return 0, err
	}
	return next, nil
}
