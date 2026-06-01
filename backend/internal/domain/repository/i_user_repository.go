package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IUserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	GetByLogin(ctx context.Context, login string) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
}
