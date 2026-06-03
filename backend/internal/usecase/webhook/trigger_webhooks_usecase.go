package webhook

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/Corevice/open-git/backend/internal/infrastructure/queue"
	"github.com/Corevice/open-git/backend/internal/worker"
)

type Webhook struct {
	ID     uuid.UUID
	URL    string
	Secret string
	Events []string
}

type WebhookRepository interface {
	ListActiveByRepoAndEvent(ctx context.Context, repoID uuid.UUID, event string) ([]Webhook, error)
}

type TriggerWebhooksUsecase struct {
	webhookRepo WebhookRepository
	client      *asynq.Client
}

func NewTriggerWebhooksUsecase(webhookRepo WebhookRepository, client *asynq.Client) *TriggerWebhooksUsecase {
	return &TriggerWebhooksUsecase{
		webhookRepo: webhookRepo,
		client:      client,
	}
}

func (uc *TriggerWebhooksUsecase) TriggerWebhooks(
	ctx context.Context,
	repoID uuid.UUID,
	event string,
	payload []byte,
) error {
	hooks, err := uc.webhookRepo.ListActiveByRepoAndEvent(ctx, repoID, event)
	if err != nil {
		return fmt.Errorf("list active webhooks: %w", err)
	}

	for _, h := range hooks {
		taskPayload := worker.WebhookDeliveryPayload{
			WebhookID: h.ID.String(),
			URL:       h.URL,
			Secret:    h.Secret,
			Event:     event,
			Body:      payload,
		}
		body, err := json.Marshal(taskPayload)
		if err != nil {
			return fmt.Errorf("marshal task payload: %w", err)
		}
		task := asynq.NewTask(queue.TypeWebhookDeliver, body)
		if _, err := uc.client.EnqueueContext(
			ctx,
			task,
			asynq.MaxRetry(worker.MaxWebhookRetries),
		); err != nil {
			return fmt.Errorf("enqueue webhook delivery: %w", err)
		}
	}

	return nil
}
