package repository

import (
	"context"

	"github.com/open-git/backend/internal/domain/entity"
)

type IUserPreferencesRepository interface {
	GetByUserID(ctx context.Context, userID int64) (*entity.UserPreferences, error)
	Upsert(ctx context.Context, prefs *entity.UserPreferences) error
}
