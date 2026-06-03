package repository

import (
	"context"

	"github.com/Corevice/open-git/backend/internal/domain/entity"
	"github.com/google/uuid"
)

type IBranchProtectionRepository interface {
	GetForRef(ctx context.Context, repositoryID uuid.UUID, ref string) (*entity.BranchProtection, error)
}
