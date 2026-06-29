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

type sqlxLabelRepository struct {
	*sqlx.DB
}

var _ domainrepo.ILabelRepository = (*sqlxLabelRepository)(nil)

func NewLabelRepository(db *sqlx.DB) *sqlxLabelRepository {
	return &sqlxLabelRepository{DB: db}
}

func (r *sqlxLabelRepository) Create(ctx context.Context, label *entity.Label) error {
	if label.ID == uuid.Nil {
		label.ID = uuid.New()
	}
	now := time.Now().UTC()

	const query = `
		INSERT INTO labels (id, organization_id, repository_id, name, color, description, created_at)
		VALUES (:id, :organization_id, :repository_id, :name, :color, :description, :created_at)
	`

	_, err := r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":              label.ID,
		"organization_id": label.OrganizationID,
		"repository_id":   label.RepositoryID,
		"name":            label.Name,
		"color":           label.Color,
		"description":     label.Description,
		"created_at":      now,
	})
	return err
}

func (r *sqlxLabelRepository) GetByName(ctx context.Context, repoID uuid.UUID, name string) (*entity.Label, error) {
	const query = `
		SELECT id, repository_id, organization_id, name, color, description, created_at
		FROM labels
		WHERE repository_id = $1 AND name = $2
	`

	row := r.DB.QueryRowxContext(ctx, query, repoID, name)

	var label entity.Label
	err := row.Scan(
		&label.ID,
		&label.RepositoryID,
		&label.OrganizationID,
		&label.Name,
		&label.Color,
		&label.Description,
		&label.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &label, nil
}

func (r *sqlxLabelRepository) ListByRepo(ctx context.Context, repoID uuid.UUID, page, perPage int) ([]*entity.Label, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	offset := (page - 1) * perPage

	const countQuery = `SELECT COUNT(*) FROM labels WHERE repository_id = $1`
	var total int
	if err := r.DB.QueryRowxContext(ctx, countQuery, repoID).Scan(&total); err != nil {
		return nil, 0, err
	}

	const query = `
		SELECT id, repository_id, organization_id, name, color, description, created_at
		FROM labels
		WHERE repository_id = $1
		ORDER BY name ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.DB.QueryxContext(ctx, query, repoID, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	labels := make([]*entity.Label, 0)
	for rows.Next() {
		var label entity.Label
		if err := rows.Scan(
			&label.ID,
			&label.RepositoryID,
			&label.OrganizationID,
			&label.Name,
			&label.Color,
			&label.Description,
			&label.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		labels = append(labels, &label)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return labels, total, nil
}

func (r *sqlxLabelRepository) Update(ctx context.Context, label *entity.Label) error {
	const query = `
		UPDATE labels
		SET name = $1, color = $2, description = $3
		WHERE id = $4
	`

	_, err := r.DB.ExecContext(ctx, query, label.Name, label.Color, label.Description, label.ID)
	return err
}

func (r *sqlxLabelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM labels WHERE id = $1`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

func (r *sqlxLabelRepository) AddToIssue(ctx context.Context, repoID uuid.UUID, issueNumber int, labelID uuid.UUID) error {
	const query = `
		INSERT INTO issue_labels (issue_id, label_id)
		SELECT i.id, $3
		FROM issues i
		WHERE i.repository_id = $1 AND i.number = $2
	`

	_, err := r.DB.ExecContext(ctx, query, repoID, issueNumber, labelID)
	return err
}

func (r *sqlxLabelRepository) RemoveFromIssue(ctx context.Context, repoID uuid.UUID, issueNumber int, labelID uuid.UUID) error {
	const query = `
		DELETE FROM issue_labels
		WHERE issue_id = (
			SELECT id FROM issues WHERE repository_id = $1 AND number = $2
		) AND label_id = $3
	`

	_, err := r.DB.ExecContext(ctx, query, repoID, issueNumber, labelID)
	return err
}
