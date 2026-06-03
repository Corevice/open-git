package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/Corevice/open-git/backend/internal/domain/entity"
)

type IRepositoryRepository interface {
	Create(ctx context.Context, repo *entity.Repository) error
	GetByOwnerAndName(ctx context.Context, ownerID uuid.UUID, name string) (*entity.Repository, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID, page, perPage int) ([]*entity.Repository, error)
	UpdateVisibility(ctx context.Context, id uuid.UUID, visibility string) error
	Delete(ctx context.Context, id uuid.UUID) error
}
