package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

const (
	TypeWebhookDeliver = "webhook:deliver"
)

type WebhookDeliveryPayload struct {
	DeliveryID     string `json:"delivery_id"`
	OrganizationID string `json:"organization_id"`
	ContentType    string `json:"content_type"`
	HookID         string `json:"hook_id"`
	Event          string `json:"event"`
	Body           []byte `json:"body"`
	Attempt        int    `json:"attempt"`
}

func NewAsynqClient(addr string) *asynq.Client {
	return asynq.NewClient(asynq.RedisClientOpt{Addr: addr})
}

func NewAsynqServer(addr string, concurrency int) *asynq.Server {
	return asynq.NewServer(
		asynq.RedisClientOpt{Addr: addr},
		asynq.Config{Concurrency: concurrency},
	)
}

func EnqueueWebhookDelivery(ctx context.Context, client *asynq.Client, payload WebhookDeliveryPayload) (*asynq.TaskInfo, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal webhook delivery payload: %w", err)
	}
	task := asynq.NewTask(TypeWebhookDeliver, data)
	return client.EnqueueContext(ctx, task, asynq.MaxRetry(5))
}
