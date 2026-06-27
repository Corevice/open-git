package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
)

type sqlxOrganizationRepository struct {
	*sqlx.DB
}

func NewOrganizationRepository(db *sqlx.DB) *sqlxOrganizationRepository {
	return &sqlxOrganizationRepository{DB: db}
}

func (r *sqlxOrganizationRepository) Create(ctx context.Context, org *entity.Organization) error {
	if org.ID == uuid.Nil {
		org.ID = uuid.New()
	}
	if org.CreatedAt.IsZero() {
		org.CreatedAt = time.Now().UTC()
	}
	if org.PlanTier == "" {
		org.PlanTier = entity.PlanFree
	}

	const query = `
		INSERT INTO organizations (id, login, name, description, plan_tier, created_at)
		VALUES (:id, :login, :name, :description, :plan_tier, :created_at)
	`

	_, err := r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":          org.ID,
		"login":       org.Login,
		"name":        org.Name,
		"description": org.Description,
		"plan_tier":   org.PlanTier,
		"created_at":  org.CreatedAt,
	})
	return dbErrors.MapDBError(err)
}

func (r *sqlxOrganizationRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Organization, error) {
	return r.getOne(ctx, `SELECT id, login, name, description, plan_tier, created_at FROM organizations WHERE id = ?`, id)
}

func (r *sqlxOrganizationRepository) GetByLogin(ctx context.Context, login string) (*entity.Organization, error) {
	return r.getOne(ctx, `SELECT id, login, name, description, plan_tier, created_at FROM organizations WHERE login = ?`, login)
}

func (r *sqlxOrganizationRepository) getOne(ctx context.Context, query string, arg any) (*entity.Organization, error) {
	query = r.DB.Rebind(query)
	row := r.DB.QueryRowxContext(ctx, query, arg)

	var org entity.Organization
	err := row.Scan(&org.ID, &org.Login, &org.Name, &org.Description, &org.PlanTier, &org.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return &org, nil
}

func (r *sqlxOrganizationRepository) List(ctx context.Context, page, perPage int) ([]*entity.Organization, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	offset := (page - 1) * perPage

	query := `
		SELECT id, login, name, description, plan_tier, created_at
		FROM organizations
		ORDER BY created_at ASC
		LIMIT ? OFFSET ?
	`
	query = r.DB.Rebind(query)

	rows, err := r.DB.QueryxContext(ctx, query, perPage, offset)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	var orgs []*entity.Organization
	for rows.Next() {
		var org entity.Organization
		if err := rows.Scan(&org.ID, &org.Login, &org.Name, &org.Description, &org.PlanTier, &org.CreatedAt); err != nil {
			return nil, dbErrors.MapDBError(err)
		}
		orgs = append(orgs, &org)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return orgs, nil
}

func (r *sqlxOrganizationRepository) Update(ctx context.Context, org *entity.Organization) error {
	const query = `
		UPDATE organizations
		SET login = :login, name = :name, plan_tier = :plan_tier
		WHERE id = :id
	`

	_, err := r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":        org.ID,
		"login":     org.Login,
		"name":      org.Name,
		"plan_tier": org.PlanTier,
	})
	return dbErrors.MapDBError(err)
}

func (r *sqlxOrganizationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM organizations WHERE id = ?`
	query = r.DB.Rebind(query)
	_, err := r.DB.ExecContext(ctx, query, id)
	return dbErrors.MapDBError(err)
}
