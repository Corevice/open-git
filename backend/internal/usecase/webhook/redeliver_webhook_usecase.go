package webhook

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/queue"
)

type WebhookDeliveryEnqueuer interface {
	EnqueueDelivery(ctx context.Context, payload queue.WebhookDeliveryPayload) error
}

type asynqWebhookDeliveryEnqueuer struct {
	client *asynq.Client
}

func newAsynqWebhookDeliveryEnqueuer(client *asynq.Client) WebhookDeliveryEnqueuer {
	return &asynqWebhookDeliveryEnqueuer{client: client}
}

func (e *asynqWebhookDeliveryEnqueuer) EnqueueDelivery(ctx context.Context, payload queue.WebhookDeliveryPayload) error {
	_, err := queue.EnqueueWebhookDelivery(ctx, e.client, payload)
	return err
}

type RedeliverWebhookUsecase struct {
	webhookRepo  domainrepo.IWebhookRepository
	deliveryRepo domainrepo.IWebhookDeliveryRepository
	enqueuer     WebhookDeliveryEnqueuer
}

func NewRedeliverWebhookUsecase(
	webhookRepo domainrepo.IWebhookRepository,
	deliveryRepo domainrepo.IWebhookDeliveryRepository,
	client *asynq.Client,
) *RedeliverWebhookUsecase {
	return NewRedeliverWebhookUsecaseWithEnqueuer(webhookRepo, deliveryRepo, newAsynqWebhookDeliveryEnqueuer(client))
}

func NewRedeliverWebhookUsecaseWithEnqueuer(
	webhookRepo domainrepo.IWebhookRepository,
	deliveryRepo domainrepo.IWebhookDeliveryRepository,
	enqueuer WebhookDeliveryEnqueuer,
) *RedeliverWebhookUsecase {
	return &RedeliverWebhookUsecase{
		webhookRepo:  webhookRepo,
		deliveryRepo: deliveryRepo,
		enqueuer:     enqueuer,
	}
}

func (uc *RedeliverWebhookUsecase) Execute(
	ctx context.Context,
	deliveryID, webhookID, orgID uuid.UUID,
) (*entity.WebhookDelivery, error) {
	webhook, err := uc.webhookRepo.GetByID(ctx, webhookID, orgID)
	if err != nil {
		return nil, err
	}

	original, err := uc.deliveryRepo.GetByID(ctx, deliveryID, webhookID, orgID)
	if err != nil {
		return nil, err
	}

	newID := uuid.New()
	parentID := original.ID
	delivery := &entity.WebhookDelivery{
		ID:               newID,
		WebhookID:        webhookID,
		OrganizationID:   orgID,
		Event:            original.Event,
		Status:           entity.StatusPending,
		RequestBody:      original.RequestBody,
		RequestHeaders:   original.RequestHeaders,
		Attempt:          1,
		Redelivery:       true,
		ParentDeliveryID: &parentID,
		CreatedAt:        time.Now().UTC(),
	}

	if err := uc.deliveryRepo.Create(ctx, delivery); err != nil {
		return nil, err
	}

	if err := uc.enqueuer.EnqueueDelivery(ctx, queue.WebhookDeliveryPayload{
		DeliveryID:     newID.String(),
		OrganizationID: orgID.String(),
		ContentType:    webhook.ContentType,
		HookID:         webhookID.String(),
		Event:          original.Event,
		Body:           []byte(original.RequestBody),
		Attempt:        1,
	}); err != nil {
		return nil, err
	}

	return delivery, nil
}
