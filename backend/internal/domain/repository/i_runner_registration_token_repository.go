package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IRunnerRegistrationTokenRepository interface {
	Create(ctx context.Context, token *entity.RunnerRegistrationToken) error
	GetByTokenHash(ctx context.Context, hash string) (*entity.RunnerRegistrationToken, error)
	MarkUsed(ctx context.Context, id uuid.UUID, usedAt time.Time) error
}
