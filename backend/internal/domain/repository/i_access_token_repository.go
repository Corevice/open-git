package repository

import (
	"context"

	"github.com/Corevice/open-git/backend/internal/domain/entity"
	"github.com/google/uuid"
)

type IAccessTokenRepository interface {
	Create(ctx context.Context, token *entity.AccessToken) error
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.AccessToken, error)
	FindByTokenHash(ctx context.Context, tokenHash string) (*entity.AccessToken, error)
	Revoke(ctx context.Context, tokenID, userID uuid.UUID) error
}
