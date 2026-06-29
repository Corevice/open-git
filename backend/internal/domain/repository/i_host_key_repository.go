package repository

import (
	"context"

	"github.com/open-git/backend/internal/domain/entity"
)

type IHostKeyRepository interface {
	Create(ctx context.Context, key *entity.HostKey) error
	FindByAlgorithm(ctx context.Context, algo string) (*entity.HostKey, error)
}
