package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/entity"
)

type sqlxReviewCommentRepository struct {
	db *sqlx.DB
}

var _ domainrepo.IReviewCommentRepository = (*sqlxReviewCommentRepository)(nil)

func NewReviewCommentRepository(db *sqlx.DB) domainrepo.IReviewCommentRepository {
	return &sqlxReviewCommentRepository{db: db}
}

func (r *sqlxReviewCommentRepository) Create(ctx context.Context, c *entity.ReviewComment) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	now := time.Now().UTC()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}

	const query = `
		INSERT INTO pull_request_review_comments (
			id, review_id, pull_request_id, author_id, path, diff_hunk, line, side,
			body, in_reply_to_id, resolved, created_at, updated_at
		) VALUES (
			:id, :review_id, :pull_request_id, :author_id, :path, :diff_hunk, :line, :side,
			:body, :in_reply_to_id, :resolved, :created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, map[string]any{
		"id":              c.ID,
		"review_id":       c.ReviewID,
		"pull_request_id": c.PullRequestID,
		"author_id":       c.AuthorID,
		"path":            c.Path,
		"diff_hunk":       c.DiffHunk,
		"line":            c.Line,
		"side":            c.Side,
		"body":            c.Body,
		"in_reply_to_id":  c.InReplyToID,
		"resolved":        boolToInt(c.Resolved),
		"created_at":      c.CreatedAt,
		"updated_at":      c.UpdatedAt,
	})
	return err
}

func (r *sqlxReviewCommentRepository) ListByPR(ctx context.Context, prID uuid.UUID) ([]*entity.ReviewComment, error) {
	const query = `
		SELECT id, pull_request_id, author_id, review_id, path, diff_hunk, body, line, side,
			in_reply_to_id, resolved, created_at, updated_at
		FROM pull_request_review_comments
		WHERE pull_request_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryxContext(ctx, query, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanReviewComments(rows)
}

func (r *sqlxReviewCommentRepository) ListByReview(ctx context.Context, reviewID uuid.UUID) ([]*entity.ReviewComment, error) {
	const query = `
		SELECT id, pull_request_id, author_id, review_id, path, diff_hunk, body, line, side,
			in_reply_to_id, resolved, created_at, updated_at
		FROM pull_request_review_comments
		WHERE review_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryxContext(ctx, query, reviewID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanReviewComments(rows)
}

func scanReviewComments(rows *sqlx.Rows) ([]*entity.ReviewComment, error) {
	comments := make([]*entity.ReviewComment, 0)
	for rows.Next() {
		var (
			comment    entity.ReviewComment
			reviewID   sqlNullUUID
			inReplyTo  sqlNullUUID
			resolved   int
		)

		if err := rows.Scan(
			&comment.ID,
			&comment.PullRequestID,
			&comment.AuthorID,
			&reviewID,
			&comment.Path,
			&comment.DiffHunk,
			&comment.Body,
			&comment.Line,
			&comment.Side,
			&inReplyTo,
			&resolved,
			&comment.CreatedAt,
			&comment.UpdatedAt,
		); err != nil {
			return nil, err
		}

		comment.ReviewID = reviewID.toPtr()
		comment.InReplyToID = inReplyTo.toPtr()
		comment.Resolved = resolved != 0
		comments = append(comments, &comment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return comments, nil
}

type sqlNullUUID struct {
	valid bool
	value uuid.UUID
}

func (n *sqlNullUUID) Scan(src any) error {
	if src == nil {
		n.valid = false
		return nil
	}
	switch v := src.(type) {
	case string:
		if v == "" {
			n.valid = false
			return nil
		}
		id, err := uuid.Parse(v)
		if err != nil {
			return err
		}
		n.value = id
		n.valid = true
		return nil
	case []byte:
		if len(v) == 0 {
			n.valid = false
			return nil
		}
		id, err := uuid.Parse(string(v))
		if err != nil {
			return err
		}
		n.value = id
		n.valid = true
		return nil
	default:
		return nil
	}
}

func (n sqlNullUUID) toPtr() *uuid.UUID {
	if !n.valid {
		return nil
	}
	v := n.value
	return &v
}
