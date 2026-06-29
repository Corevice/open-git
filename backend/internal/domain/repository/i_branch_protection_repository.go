package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IBranchProtectionRepository interface {
	GetByBranch(ctx context.Context, repoID uuid.UUID, branch string) (*entity.BranchProtection, error)
	Upsert(ctx context.Context, bp *entity.BranchProtection) error
}
