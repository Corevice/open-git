package webhook

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/queue"
)

type PingWebhookUsecase struct {
	webhookRepo  domainrepo.IWebhookRepository
	deliveryRepo domainrepo.IWebhookDeliveryRepository
	enqueuer     WebhookDeliveryEnqueuer
}

func NewPingWebhookUsecase(
	webhookRepo domainrepo.IWebhookRepository,
	deliveryRepo domainrepo.IWebhookDeliveryRepository,
	client *asynq.Client,
) *PingWebhookUsecase {
	return NewPingWebhookUsecaseWithEnqueuer(webhookRepo, deliveryRepo, newAsynqWebhookDeliveryEnqueuer(client))
}

func NewPingWebhookUsecaseWithEnqueuer(
	webhookRepo domainrepo.IWebhookRepository,
	deliveryRepo domainrepo.IWebhookDeliveryRepository,
	enqueuer WebhookDeliveryEnqueuer,
) *PingWebhookUsecase {
	return &PingWebhookUsecase{
		webhookRepo:  webhookRepo,
		deliveryRepo: deliveryRepo,
		enqueuer:     enqueuer,
	}
}

func (uc *PingWebhookUsecase) Execute(ctx context.Context, webhookID, orgID uuid.UUID) (*entity.WebhookDelivery, error) {
	webhook, err := uc.webhookRepo.GetByID(ctx, webhookID, orgID)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(map[string]any{
		"zen":     "Design for failure.",
		"hook_id": webhookID.String(),
		"hook": map[string]any{
			"type":   "Repository",
			"id":     webhookID.String(),
			"name":   "web",
			"active": webhook.Active,
			"events": webhook.Events,
			"config": map[string]string{
				"url":          webhook.URL,
				"content_type": webhook.ContentType,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	deliveryID := uuid.New()
	delivery := &entity.WebhookDelivery{
		ID:             deliveryID,
		WebhookID:      webhookID,
		OrganizationID: orgID,
		Event:          "ping",
		Status:         entity.StatusPending,
		RequestBody:    string(body),
		Attempt:        1,
		CreatedAt:      time.Now().UTC(),
	}

	if err := uc.deliveryRepo.Create(ctx, delivery); err != nil {
		return nil, err
	}

	if err := uc.enqueuer.EnqueueDelivery(ctx, queue.WebhookDeliveryPayload{
		DeliveryID:     deliveryID.String(),
		OrganizationID: orgID.String(),
		ContentType:    webhook.ContentType,
		HookID:         webhookID.String(),
		Event:          "ping",
		Body:           body,
		Attempt:        1,
	}); err != nil {
		return nil, err
	}

	return delivery, nil
}
