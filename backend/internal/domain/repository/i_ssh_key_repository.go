package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type ISSHKeyRepository interface {
	FindByPublicKey(ctx context.Context, publicKey string) (*entity.SSHKey, error)
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
}
