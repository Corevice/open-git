package repository

import (
	"context"

	"github.com/open-git/backend/internal/domain/entity"
)

type SystemSettingRepository interface {
	Get(ctx context.Context, key string) (*entity.SystemSetting, error)
	Set(ctx context.Context, setting *entity.SystemSetting) error
}
