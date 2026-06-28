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

const milestoneNextNumberMaxRetries = 5

type sqlxMilestoneRepository struct {
	*sqlx.DB
}

var _ domainrepo.IMilestoneRepository = (*sqlxMilestoneRepository)(nil)

func NewMilestoneRepository(db *sqlx.DB) *sqlxMilestoneRepository {
	return &sqlxMilestoneRepository{DB: db}
}

func (r *sqlxMilestoneRepository) Create(ctx context.Context, milestone *entity.Milestone) error {
	if milestone.ID == uuid.Nil {
		milestone.ID = uuid.New()
	}
	now := time.Now().UTC()

	const query = `
		INSERT INTO milestones (
			id, organization_id, repository_id, number, title, description, state, due_on,
			open_issues, closed_issues, created_at, updated_at
		)
		VALUES (
			:id, :organization_id, :repository_id, :number, :title, :description, :state, :due_on,
			:open_issues, :closed_issues, :created_at, :updated_at
		)
	`

	_, err := r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":              milestone.ID,
		"organization_id": milestone.OrganizationID,
		"repository_id":   milestone.RepositoryID,
		"number":          milestone.Number,
		"title":           milestone.Title,
		"description":     milestone.Description,
		"state":           milestone.State,
		"due_on":          milestone.DueOn,
		"open_issues":     milestone.OpenIssues,
		"closed_issues":   milestone.ClosedIssues,
		"created_at":      now,
		"updated_at":      now,
	})
	return err
}

func (r *sqlxMilestoneRepository) GetByNumber(ctx context.Context, repoID uuid.UUID, number int) (*entity.Milestone, error) {
	const query = `
		SELECT id, repository_id, organization_id, number, title, description, state, due_on,
			open_issues, closed_issues, created_at, updated_at, closed_at
		FROM milestones
		WHERE repository_id = $1 AND number = $2
	`

	row := r.DB.QueryRowxContext(ctx, query, repoID, number)

	var (
		milestone entity.Milestone
		dueOn     sql.NullTime
		closedAt  sql.NullTime
	)
	err := row.Scan(
		&milestone.ID,
		&milestone.RepositoryID,
		&milestone.OrganizationID,
		&milestone.Number,
		&milestone.Title,
		&milestone.Description,
		&milestone.State,
		&dueOn,
		&milestone.OpenIssues,
		&milestone.ClosedIssues,
		&milestone.CreatedAt,
		&milestone.UpdatedAt,
		&closedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if dueOn.Valid {
		t := dueOn.Time
		milestone.DueOn = &t
	}
	if closedAt.Valid {
		t := closedAt.Time
		milestone.ClosedAt = &t
	}
	return &milestone, nil
}

func (r *sqlxMilestoneRepository) ListByRepo(ctx context.Context, repoID uuid.UUID, state string, page, perPage int) ([]*entity.Milestone, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	offset := (page - 1) * perPage

	countQuery := `SELECT COUNT(*) FROM milestones WHERE repository_id = $1`
	countArgs := []any{repoID}
	if state != "" {
		countQuery += ` AND state = $2`
		countArgs = append(countArgs, state)
	}

	var total int
	if err := r.DB.QueryRowxContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, repository_id, organization_id, number, title, description, state, due_on,
			open_issues, closed_issues, created_at, updated_at, closed_at
		FROM milestones
		WHERE repository_id = $1
	`
	args := []any{repoID}
	if state != "" {
		query += ` AND state = $2`
		args = append(args, state)
	}
	query += ` ORDER BY number ASC LIMIT $` + itoa(len(args)+1) + ` OFFSET $` + itoa(len(args)+2)
	args = append(args, perPage, offset)

	rows, err := r.DB.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	milestones := make([]*entity.Milestone, 0)
	for rows.Next() {
		var (
			milestone entity.Milestone
			dueOn     sql.NullTime
			closedAt  sql.NullTime
		)
		if err := rows.Scan(
			&milestone.ID,
			&milestone.RepositoryID,
			&milestone.OrganizationID,
			&milestone.Number,
			&milestone.Title,
			&milestone.Description,
			&milestone.State,
			&dueOn,
			&milestone.OpenIssues,
			&milestone.ClosedIssues,
			&milestone.CreatedAt,
			&milestone.UpdatedAt,
			&closedAt,
		); err != nil {
			return nil, 0, err
		}
		if dueOn.Valid {
			t := dueOn.Time
			milestone.DueOn = &t
		}
		if closedAt.Valid {
			t := closedAt.Time
			milestone.ClosedAt = &t
		}
		milestones = append(milestones, &milestone)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return milestones, total, nil
}

func (r *sqlxMilestoneRepository) Update(ctx context.Context, milestone *entity.Milestone) error {
	now := time.Now().UTC()

	const query = `
		UPDATE milestones
		SET title = $1, description = $2, state = $3, due_on = $4, updated_at = $5, closed_at = $6
		WHERE id = $7
	`

	var closedAt *time.Time
	if milestone.State == "closed" {
		if milestone.ClosedAt != nil {
			closedAt = milestone.ClosedAt
		} else {
			closedAt = &now
		}
	}

	_, err := r.DB.ExecContext(
		ctx,
		query,
		milestone.Title,
		milestone.Description,
		milestone.State,
		milestone.DueOn,
		now,
		closedAt,
		milestone.ID,
	)
	return err
}

func (r *sqlxMilestoneRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM milestones WHERE id = $1`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

func (r *sqlxMilestoneRepository) NextNumber(ctx context.Context, repoID uuid.UUID) (int, error) {
	const query = `SELECT COALESCE(MAX(number), 0) + 1 FROM milestones WHERE repository_id = $1`

	var lastErr error
	for attempt := 0; attempt < milestoneNextNumberMaxRetries; attempt++ {
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
	return 0, errors.New("failed to allocate milestone number")
}

func (r *sqlxMilestoneRepository) IncrOpenCount(ctx context.Context, id uuid.UUID) error {
	const query = `UPDATE milestones SET open_issues = open_issues + 1 WHERE id = $1`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

func (r *sqlxMilestoneRepository) DecrOpenCount(ctx context.Context, id uuid.UUID) error {
	const query = `UPDATE milestones SET open_issues = open_issues - 1 WHERE id = $1`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}
