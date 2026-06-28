package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IWebhookDeliveryRepository interface {
	Create(ctx context.Context, delivery *entity.WebhookDelivery) error
	GetByID(ctx context.Context, deliveryID, webhookID, orgID uuid.UUID) (*entity.WebhookDelivery, error)
	ListByWebhook(ctx context.Context, webhookID, orgID uuid.UUID, page, perPage int) ([]*entity.WebhookDelivery, int64, error)
	UpdateStatus(
		ctx context.Context,
		deliveryID uuid.UUID,
		status string,
		statusCode *int,
		responseHeaders map[string][]string,
		responseBody *string,
		durationMs *int,
		deliveredAt time.Time,
	) error
}
