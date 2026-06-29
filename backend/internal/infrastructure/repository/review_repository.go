package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/entity"
)

type sqlxReviewRepository struct {
	db *sqlx.DB
}

var _ domainrepo.IReviewRepository = (*sqlxReviewRepository)(nil)

func NewReviewRepository(db *sqlx.DB) domainrepo.IReviewRepository {
	return &sqlxReviewRepository{db: db}
}

func (r *sqlxReviewRepository) Create(ctx context.Context, review *entity.Review) error {
	if review.ID == uuid.Nil {
		review.ID = uuid.New()
	}
	now := time.Now().UTC()
	if review.CreatedAt.IsZero() {
		review.CreatedAt = now
	}

	var orgID uuid.UUID
	const orgQuery = `SELECT organization_id FROM pull_requests WHERE id = $1`
	if err := r.db.QueryRowxContext(ctx, orgQuery, review.PullRequestID).Scan(&orgID); err != nil {
		return err
	}

	const query = `
		INSERT INTO reviews (
			id, organization_id, pull_request_id, reviewer_id, state, body, commit_sha, submitted_at
		) VALUES (
			:id, :organization_id, :pull_request_id, :reviewer_id, :state, :body, :commit_sha, :submitted_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, map[string]any{
		"id":              review.ID,
		"organization_id": orgID,
		"pull_request_id": review.PullRequestID,
		"reviewer_id":     review.ReviewerID,
		"state":           review.State,
		"body":            review.Body,
		"commit_sha":      review.CommitSHA,
		"submitted_at":    review.SubmittedAt,
	})
	return err
}

func (r *sqlxReviewRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Review, error) {
	const query = `
		SELECT id, pull_request_id, reviewer_id, state, body, commit_sha, submitted_at
		FROM reviews
		WHERE id = $1
	`

	row := r.db.QueryRowxContext(ctx, query, id)
	review, err := scanReviewRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return review, nil
}

func (r *sqlxReviewRepository) ListByPR(ctx context.Context, prID uuid.UUID) ([]*entity.Review, error) {
	const query = `
		SELECT id, pull_request_id, reviewer_id, state, body, commit_sha, submitted_at
		FROM reviews
		WHERE pull_request_id = $1
		ORDER BY submitted_at ASC
	`

	rows, err := r.db.QueryxContext(ctx, query, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews := make([]*entity.Review, 0)
	for rows.Next() {
		review, err := scanReviewRow(rows)
		if err != nil {
			return nil, err
		}
		reviews = append(reviews, review)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return reviews, nil
}

func (r *sqlxReviewRepository) CountSatisfiedReviews(ctx context.Context, prID uuid.UUID) (int, error) {
	const query = `
		SELECT COUNT(DISTINCT reviewer_id)
		FROM reviews
		WHERE pull_request_id = $1 AND state = 'APPROVED' AND submitted_at IS NOT NULL
	`

	var count int
	if err := r.db.QueryRowxContext(ctx, query, prID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *sqlxReviewRepository) HasBlockingReviews(ctx context.Context, prID uuid.UUID) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1
			FROM reviews
			WHERE pull_request_id = $1 AND state = 'CHANGES_REQUESTED' AND submitted_at IS NOT NULL
		)
	`

	var exists bool
	if err := r.db.QueryRowxContext(ctx, query, prID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

type reviewScanner interface {
	Scan(dest ...any) error
}

func scanReviewRow(scanner reviewScanner) (*entity.Review, error) {
	var (
		review      entity.Review
		submittedAt sql.NullTime
	)

	if err := scanner.Scan(
		&review.ID,
		&review.PullRequestID,
		&review.ReviewerID,
		&review.State,
		&review.Body,
		&review.CommitSHA,
		&submittedAt,
	); err != nil {
		return nil, err
	}

	if submittedAt.Valid {
		t := submittedAt.Time
		review.SubmittedAt = &t
		review.CreatedAt = t
	}

	return &review, nil
}
