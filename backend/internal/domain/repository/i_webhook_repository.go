package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IWebhookRepository interface {
	Create(ctx context.Context, webhook *entity.Webhook) error
	GetByID(ctx context.Context, id uuid.UUID, orgID uuid.UUID) (*entity.Webhook, error)
	ListByRepo(ctx context.Context, orgID, repoID uuid.UUID, page, perPage int) ([]*entity.Webhook, int64, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID, page, perPage int) ([]*entity.Webhook, int64, error)
	Update(ctx context.Context, webhook *entity.Webhook) error
	Delete(ctx context.Context, id uuid.UUID, orgID uuid.UUID) error
	ListActiveByRepoAndEvent(ctx context.Context, orgID, repoID uuid.UUID, event string) ([]*entity.Webhook, error)
}
