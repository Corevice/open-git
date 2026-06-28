package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IBranchProtectionRepository interface {
	GetForRef(ctx context.Context, repoID uuid.UUID, ref string) (*entity.BranchProtection, error)
}
