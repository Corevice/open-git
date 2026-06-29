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
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
)

type sqlxActionCacheRepository struct {
	db *sqlx.DB
}

var _ domainrepo.IActionCacheRepository = (*sqlxActionCacheRepository)(nil)

func NewActionCacheRepository(db *sqlx.DB) domainrepo.IActionCacheRepository {
	return &sqlxActionCacheRepository{db: db}
}

func (r *sqlxActionCacheRepository) GetByKey(ctx context.Context, orgID uuid.UUID, actionName, resolvedRef string) (*entity.ActionCacheEntry, error) {
	const query = `
		SELECT id, organization_id, action_name, resolved_ref, storage_path, cached_at
		FROM action_cache_entries
		WHERE organization_id = ? AND action_name = ? AND resolved_ref = ?
	`
	q := r.db.Rebind(query)

	var entry entity.ActionCacheEntry
	err := r.db.QueryRowContext(ctx, q, orgID, actionName, resolvedRef).Scan(
		&entry.ID,
		&entry.OrganizationID,
		&entry.ActionName,
		&entry.ResolvedRef,
		&entry.StoragePath,
		&entry.CachedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return &entry, nil
}

func (r *sqlxActionCacheRepository) Create(ctx context.Context, e *entity.ActionCacheEntry) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.CachedAt.IsZero() {
		e.CachedAt = time.Now().UTC()
	}

	const query = `
		INSERT INTO action_cache_entries (
			id, organization_id, action_name, resolved_ref, storage_path, cached_at
		) VALUES (
			:id, :organization_id, :action_name, :resolved_ref, :storage_path, :cached_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, map[string]any{
		"id":              e.ID,
		"organization_id": e.OrganizationID,
		"action_name":     e.ActionName,
		"resolved_ref":    e.ResolvedRef,
		"storage_path":    e.StoragePath,
		"cached_at":       e.CachedAt,
	})
	return dbErrors.MapDBError(err)
}

func (r *sqlxActionCacheRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM action_cache_entries WHERE id = ?`
	q := r.db.Rebind(query)

	_, err := r.db.ExecContext(ctx, q, id)
	return dbErrors.MapDBError(err)
}
