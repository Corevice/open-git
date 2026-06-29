package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/queue"
)

type TaskEnqueuer interface {
	EnqueueContext(ctx context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

type TriggerWebhooksInput struct {
	OrgID   uuid.UUID
	RepoID  uuid.UUID
	Event   string
	Payload []byte
}

type TriggerWebhooksUsecase struct {
	webhookRepo  domainrepo.IWebhookRepository
	deliveryRepo domainrepo.IWebhookDeliveryRepository
	client       TaskEnqueuer
}

func NewTriggerWebhooksUsecase(
	webhookRepo domainrepo.IWebhookRepository,
	deliveryRepo domainrepo.IWebhookDeliveryRepository,
	client TaskEnqueuer,
) *TriggerWebhooksUsecase {
	return &TriggerWebhooksUsecase{
		webhookRepo:  webhookRepo,
		deliveryRepo: deliveryRepo,
		client:       client,
	}
}

func (uc *TriggerWebhooksUsecase) Execute(ctx context.Context, input TriggerWebhooksInput) error {
	hooks, err := uc.webhookRepo.ListActiveByRepoAndEvent(ctx, input.OrgID, input.RepoID, input.Event)
	if err != nil {
		return fmt.Errorf("list active webhooks: %w", err)
	}
	if len(hooks) == 0 {
		return nil
	}

	for _, hook := range hooks {
		deliveryID := uuid.New()

		if err := uc.deliveryRepo.Create(ctx, &entity.WebhookDelivery{
			ID:             deliveryID,
			WebhookID:      hook.ID,
			OrganizationID: input.OrgID,
			Event:          input.Event,
			Status:         entity.StatusPending,
			RequestBody:    string(input.Payload),
			Attempt:        0,
			CreatedAt:      time.Now().UTC(),
		}); err != nil {
			return fmt.Errorf("create webhook delivery: %w", err)
		}

		taskPayload := queue.WebhookDeliveryPayload{
			DeliveryID:     deliveryID.String(),
			HookID:         hook.ID.String(),
			OrganizationID: input.OrgID.String(),
			Event:          input.Event,
			Body:           input.Payload,
			ContentType:    hook.ContentType,
			Attempt:        0,
		}
		body, err := json.Marshal(taskPayload)
		if err != nil {
			return fmt.Errorf("marshal task payload: %w", err)
		}
		task := asynq.NewTask(queue.TypeWebhookDeliver, body)
		if _, err := uc.client.EnqueueContext(
			ctx,
			task,
			asynq.MaxRetry(5),
		); err != nil {
			return fmt.Errorf("enqueue webhook delivery: %w", err)
		}
	}

	return nil
}
