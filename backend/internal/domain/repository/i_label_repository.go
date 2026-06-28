package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type ILabelRepository interface {
	GetByName(ctx context.Context, repoID uuid.UUID, name string) (*entity.Label, error)
}
