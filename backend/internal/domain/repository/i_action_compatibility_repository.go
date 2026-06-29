package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IActionCompatibilityRepository interface {
	UpsertResult(ctx context.Context, r *entity.ActionCompatibilityResult) error
	ListResults(ctx context.Context, orgID uuid.UUID, repoID *uuid.UUID) ([]*entity.ActionCompatibilityResult, error)
	GetResult(ctx context.Context, orgID uuid.UUID, actionName, actionVersion string) (*entity.ActionCompatibilityResult, error)
}
