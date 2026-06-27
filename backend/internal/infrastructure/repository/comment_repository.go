package repository

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/database"
)

type CommentRepository struct {
	db *sqlx.DB
}

func NewCommentRepository(db *sqlx.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) Create(ctx context.Context, comment *entity.Comment) error {
	const query = `
		INSERT INTO comments (id, organization_id, issue_id, author_id, body, created_at)
		SELECT $1, organization_id, $2, $3, $4, $5
		FROM issues
		WHERE id = $2
	`
	result, err := database.SQLxExecutor(ctx, r.db).ExecContext(
		ctx,
		query,
		comment.ID,
		comment.IssueID,
		comment.AuthorID,
		comment.Body,
		time.Now().UTC(),
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return apperror.ErrNotFound
	}
	return nil
}
