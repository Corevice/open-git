package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IOrganizationRepository interface {
	Create(ctx context.Context, org *entity.Organization) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Organization, error)
	GetByLogin(ctx context.Context, login string) (*entity.Organization, error)
	List(ctx context.Context, page, perPage int) ([]*entity.Organization, error)
	Update(ctx context.Context, org *entity.Organization) error
	Delete(ctx context.Context, id uuid.UUID) error
}
