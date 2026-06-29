package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IActionCacheRepository interface {
	GetByKey(ctx context.Context, orgID uuid.UUID, actionName, resolvedRef string) (*entity.ActionCacheEntry, error)
	Create(ctx context.Context, e *entity.ActionCacheEntry) error
	Delete(ctx context.Context, id uuid.UUID) error
}
