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

type sqlxCommentRepository struct {
	*sqlx.DB
}

var _ domainrepo.ICommentRepository = (*sqlxCommentRepository)(nil)

func NewCommentRepository(db *sqlx.DB) *sqlxCommentRepository {
	return &sqlxCommentRepository{DB: db}
}

func (r *sqlxCommentRepository) Create(ctx context.Context, comment *entity.Comment) error {
	if comment.ID == uuid.Nil {
		comment.ID = uuid.New()
	}
	now := time.Now().UTC()

	const query = `
		INSERT INTO comments (id, organization_id, issue_id, author_id, body, created_at, updated_at)
		VALUES (:id, :organization_id, :issue_id, :author_id, :body, :created_at, :updated_at)
	`

	_, err := r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":              comment.ID,
		"organization_id": comment.OrganizationID,
		"issue_id":        comment.IssueID,
		"author_id":       comment.AuthorID,
		"body":            comment.Body,
		"created_at":      now,
		"updated_at":      now,
	})
	return err
}

func (r *sqlxCommentRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Comment, error) {
	const query = `
		SELECT c.id, c.issue_id, c.organization_id, c.author_id, COALESCE(u.login, '') AS author_login,
			c.body, c.created_at, c.updated_at
		FROM comments c
		LEFT JOIN users u ON c.author_id = u.id
		WHERE c.id = $1
	`

	row := r.DB.QueryRowxContext(ctx, query, id)

	var comment entity.Comment
	err := row.Scan(
		&comment.ID,
		&comment.IssueID,
		&comment.OrganizationID,
		&comment.AuthorID,
		&comment.AuthorLogin,
		&comment.Body,
		&comment.CreatedAt,
		&comment.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

func (r *sqlxCommentRepository) ListByIssue(ctx context.Context, issueID uuid.UUID, page, perPage int) ([]*entity.Comment, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	offset := (page - 1) * perPage

	const countQuery = `SELECT COUNT(*) FROM comments WHERE issue_id = $1`
	var total int
	if err := r.DB.QueryRowxContext(ctx, countQuery, issueID).Scan(&total); err != nil {
		return nil, 0, err
	}

	const query = `
		SELECT c.id, c.issue_id, c.organization_id, c.author_id, COALESCE(u.login, '') AS author_login,
			c.body, c.created_at, c.updated_at
		FROM comments c
		LEFT JOIN users u ON c.author_id = u.id
		WHERE c.issue_id = $1
		ORDER BY c.created_at ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.DB.QueryxContext(ctx, query, issueID, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	comments := make([]*entity.Comment, 0)
	for rows.Next() {
		var comment entity.Comment
		if err := rows.Scan(
			&comment.ID,
			&comment.IssueID,
			&comment.OrganizationID,
			&comment.AuthorID,
			&comment.AuthorLogin,
			&comment.Body,
			&comment.CreatedAt,
			&comment.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		comments = append(comments, &comment)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return comments, total, nil
}

func (r *sqlxCommentRepository) Update(ctx context.Context, comment *entity.Comment) error {
	const query = `
		UPDATE comments
		SET body = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := r.DB.ExecContext(ctx, query, comment.Body, time.Now().UTC(), comment.ID)
	return err
}

func (r *sqlxCommentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM comments WHERE id = $1`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}
