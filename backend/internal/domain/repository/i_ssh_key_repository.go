package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type ISSHKeyRepository interface {
	Create(ctx context.Context, key *entity.SSHKey) error
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.SSHKey, error)
	FindByID(ctx context.Context, id uuid.UUID) (*entity.SSHKey, error)
	FindByUserFingerprint(ctx context.Context, userID uuid.UUID, fp string) (*entity.SSHKey, error)
	FindByPublicKey(ctx context.Context, publicKey string) (*entity.SSHKey, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID, t time.Time) error
}
