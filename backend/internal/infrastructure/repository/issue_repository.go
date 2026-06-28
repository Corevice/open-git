package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

const issueNextNumberMaxRetries = 5

type sqlxIssueRepository struct {
	*sqlx.DB
}

func NewIssueRepository(db *sqlx.DB) *sqlxIssueRepository {
	return &sqlxIssueRepository{DB: db}
}

func (r *sqlxIssueRepository) Create(ctx context.Context, issue *entity.Issue) error {
	if issue.ID == uuid.Nil {
		issue.ID = uuid.New()
	}

	const query = `
		INSERT INTO issues (id, organization_id, repository_id, number, title, body, state, author_id, created_at)
		VALUES (:id, :organization_id, :repository_id, :number, :title, :body, :state, :author_id, :created_at)
	`

	_, err := r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":              issue.ID,
		"organization_id": issue.OrganizationID,
		"repository_id":   issue.RepositoryID,
		"number":          issue.Number,
		"title":           issue.Title,
		"body":            issue.Body,
		"state":           issue.State,
		"author_id":       issue.AuthorID,
		"created_at":      time.Now().UTC(),
	})
	return err
}

func (r *sqlxIssueRepository) GetByNumber(ctx context.Context, repoID uuid.UUID, number int) (*entity.Issue, error) {
	const query = `
		SELECT id, organization_id, repository_id, number, title, body, state, author_id
		FROM issues
		WHERE repository_id = $1 AND number = $2
	`

	row := r.DB.QueryRowxContext(ctx, query, repoID, number)

	var issue entity.Issue
	err := row.Scan(
		&issue.ID,
		&issue.OrganizationID,
		&issue.RepositoryID,
		&issue.Number,
		&issue.Title,
		&issue.Body,
		&issue.State,
		&issue.AuthorID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &issue, nil
}

func (r *sqlxIssueRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Issue, error) {
	const query = `
		SELECT id, organization_id, repository_id, number, title, body, state, author_id
		FROM issues
		WHERE id = $1
	`

	row := r.DB.QueryRowxContext(ctx, query, id)

	var issue entity.Issue
	err := row.Scan(
		&issue.ID,
		&issue.OrganizationID,
		&issue.RepositoryID,
		&issue.Number,
		&issue.Title,
		&issue.Body,
		&issue.State,
		&issue.AuthorID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &issue, nil
}

func (r *sqlxIssueRepository) ListByRepo(ctx context.Context, filter domainrepo.ListIssuesFilter) ([]*entity.Issue, int, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage < 1 {
		perPage = 30
	}
	offset := (page - 1) * perPage

	countQuery := `
		SELECT COUNT(*)
		FROM issues
		WHERE repository_id = $1
	`
	countArgs := []any{filter.RepositoryID}
	countIdx := 2

	listQuery := `
		SELECT id, organization_id, repository_id, number, title, body, state, author_id
		FROM issues
		WHERE repository_id = $1
	`
	listArgs := []any{filter.RepositoryID}
	listIdx := 2

	if filter.State != "" {
		countQuery += " AND state = $" + itoa(countIdx)
		countArgs = append(countArgs, filter.State)
		countIdx++

		listQuery += " AND state = $" + itoa(listIdx)
		listArgs = append(listArgs, filter.State)
		listIdx++
	}
	if len(filter.Labels) > 0 {
		labelClause := " AND id IN (SELECT issue_id FROM issue_labels il JOIN labels l ON il.label_id = l.id WHERE l.name = $" + itoa(countIdx) + ")"
		countQuery += labelClause
		countArgs = append(countArgs, filter.Labels[0])
		countIdx++

		labelClause = " AND id IN (SELECT issue_id FROM issue_labels il JOIN labels l ON il.label_id = l.id WHERE l.name = $" + itoa(listIdx) + ")"
		listQuery += labelClause
		listArgs = append(listArgs, filter.Labels[0])
		listIdx++
	}

	var total int
	if err := r.DB.QueryRowxContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery += " ORDER BY number DESC LIMIT $" + itoa(listIdx) + " OFFSET $" + itoa(listIdx+1)
	listArgs = append(listArgs, perPage, offset)

	rows, err := r.DB.QueryxContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var issues []*entity.Issue
	for rows.Next() {
		var issue entity.Issue
		if err := rows.Scan(
			&issue.ID,
			&issue.OrganizationID,
			&issue.RepositoryID,
			&issue.Number,
			&issue.Title,
			&issue.Body,
			&issue.State,
			&issue.AuthorID,
		); err != nil {
			return nil, 0, err
		}
		issues = append(issues, &issue)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return issues, total, nil
}

func (r *sqlxIssueRepository) Update(ctx context.Context, issue *entity.Issue) error {
	const query = `
		UPDATE issues
		SET title = :title, body = :body, state = :state, author_id = :author_id
		WHERE id = :id
	`

	_, err := r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":        issue.ID,
		"title":     issue.Title,
		"body":      issue.Body,
		"state":     issue.State,
		"author_id": issue.AuthorID,
	})
	return err
}

func (r *sqlxIssueRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE issues SET state = 'deleted' WHERE id = $1`, id)
	return err
}

func (r *sqlxIssueRepository) Count(ctx context.Context, filter domainrepo.ListIssuesFilter) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM issues
		WHERE repository_id = $1
	`
	args := []any{filter.RepositoryID}
	idx := 2

	if filter.State != "" {
		query += " AND state = $" + itoa(idx)
		args = append(args, filter.State)
		idx++
	}
	if len(filter.Labels) > 0 {
		query += " AND id IN (SELECT issue_id FROM issue_labels il JOIN labels l ON il.label_id = l.id WHERE l.name = $" + itoa(idx) + ")"
		args = append(args, filter.Labels[0])
	}

	var total int
	if err := r.DB.QueryRowxContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// NextNumber allocates the next sequential issue number for the given repository.
// Wrapped in a transaction with retry on UNIQUE conflict to handle concurrent inserts.
func (r *sqlxIssueRepository) NextNumber(ctx context.Context, repoID uuid.UUID) (int, error) {
	const query = `SELECT COALESCE(MAX(number), 0) + 1 FROM issues WHERE repository_id = $1`

	var lastErr error
	for attempt := 0; attempt < issueNextNumberMaxRetries; attempt++ {
		tx, err := r.DB.BeginTxx(ctx, nil)
		if err != nil {
			return 0, err
		}

		var next int
		if err := tx.QueryRowxContext(ctx, query, repoID).Scan(&next); err != nil {
			_ = tx.Rollback()
			return 0, err
		}

		if err := tx.Commit(); err != nil {
			lastErr = err
			if isUniqueViolation(err) {
				continue
			}
			return 0, err
		}
		return next, nil
	}

	if lastErr != nil {
		return 0, lastErr
	}
	return 0, errors.New("failed to allocate issue number")
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "unique")
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}
