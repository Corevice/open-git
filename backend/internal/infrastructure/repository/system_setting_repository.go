package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
)

type sqlxSystemSettingRepository struct {
	db *sqlx.DB
}

var _ domainrepo.SystemSettingRepository = (*sqlxSystemSettingRepository)(nil)

func NewSystemSettingRepository(db *sqlx.DB) domainrepo.SystemSettingRepository {
	return &sqlxSystemSettingRepository{db: db}
}

type systemSettingRow struct {
	Key       string    `db:"key"`
	Value     []byte    `db:"value"`
	UpdatedBy string    `db:"updated_by"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (r *sqlxSystemSettingRepository) Get(ctx context.Context, key string) (*entity.SystemSetting, error) {
	query := `
		SELECT key, value, updated_by, updated_at
		FROM system_settings
		WHERE key = ?
	`
	query = r.db.Rebind(query)

	var row systemSettingRow
	err := r.db.GetContext(ctx, &row, query, key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}

	var value map[string]any
	if len(row.Value) > 0 {
		if err := json.Unmarshal(row.Value, &value); err != nil {
			return nil, err
		}
	}

	updatedBy, err := uuid.Parse(row.UpdatedBy)
	if err != nil {
		return nil, err
	}

	return &entity.SystemSetting{
		Key:       row.Key,
		Value:     value,
		UpdatedBy: updatedBy,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *sqlxSystemSettingRepository) Set(ctx context.Context, setting *entity.SystemSetting) error {
	valueJSON, err := json.Marshal(setting.Value)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if setting.UpdatedAt.IsZero() {
		setting.UpdatedAt = now
	}

	const query = `
		INSERT INTO system_settings (key, value, updated_by, updated_at)
		VALUES (:key, :value, :updated_by, :updated_at)
		ON CONFLICT (key) DO UPDATE SET
			value = EXCLUDED.value,
			updated_by = EXCLUDED.updated_by,
			updated_at = EXCLUDED.updated_at
	`

	_, err = r.db.NamedExecContext(ctx, query, map[string]any{
		"key":        setting.Key,
		"value":      string(valueJSON),
		"updated_by": setting.UpdatedBy.String(),
		"updated_at": setting.UpdatedAt,
	})
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	return nil
}
