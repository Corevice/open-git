package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/entity"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
)

type sqlxUserPreferencesRepository struct {
	*sqlx.DB
}

var _ domainrepo.IUserPreferencesRepository = (*sqlxUserPreferencesRepository)(nil)

func NewUserPreferencesRepository(db *sqlx.DB) domainrepo.IUserPreferencesRepository {
	return &sqlxUserPreferencesRepository{DB: db}
}

func (r *sqlxUserPreferencesRepository) GetByUserID(ctx context.Context, userID int64) (*entity.UserPreferences, error) {
	query := r.DB.Rebind(`SELECT user_id, theme, updated_at FROM user_preferences WHERE user_id = ?`)

	row := r.DB.QueryRowxContext(ctx, query, userID)

	var prefs entity.UserPreferences
	var updatedAt sql.NullTime
	err := row.Scan(&prefs.UserID, &prefs.Theme, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	if updatedAt.Valid {
		prefs.UpdatedAt = updatedAt.Time
	}
	return &prefs, nil
}

func (r *sqlxUserPreferencesRepository) Upsert(ctx context.Context, prefs *entity.UserPreferences) error {
	query := r.DB.Rebind(`
		INSERT INTO user_preferences (user_id, theme, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id) DO UPDATE SET theme = excluded.theme, updated_at = CURRENT_TIMESTAMP
	`)

	_, err := r.DB.ExecContext(ctx, query, prefs.UserID, prefs.Theme)
	if err != nil {
		return dbErrors.MapDBError(err)
	}

	prefs.UpdatedAt = time.Now().UTC()
	return nil
}
