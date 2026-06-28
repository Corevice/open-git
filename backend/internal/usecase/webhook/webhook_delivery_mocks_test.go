package webhook_test

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/queue"
)

type mockWebhookDeliveryRepo struct {
	byID    map[uuid.UUID]*entity.WebhookDelivery
	created *entity.WebhookDelivery
}

func newMockWebhookDeliveryRepo(byID map[uuid.UUID]*entity.WebhookDelivery) *mockWebhookDeliveryRepo {
	return &mockWebhookDeliveryRepo{byID: byID}
}

func (m *mockWebhookDeliveryRepo) Create(_ context.Context, delivery *entity.WebhookDelivery) error {
	copyDelivery := *delivery
	m.created = &copyDelivery
	if m.byID == nil {
		m.byID = map[uuid.UUID]*entity.WebhookDelivery{}
	}
	m.byID[delivery.ID] = &copyDelivery
	return nil
}

func (m *mockWebhookDeliveryRepo) GetByID(_ context.Context, deliveryID, webhookID, orgID uuid.UUID) (*entity.WebhookDelivery, error) {
	delivery, ok := m.byID[deliveryID]
	if !ok || delivery.WebhookID != webhookID || delivery.OrganizationID != orgID {
		return nil, apperror.ErrNotFound
	}
	copyDelivery := *delivery
	return &copyDelivery, nil
}

func (m *mockWebhookDeliveryRepo) ListByWebhook(context.Context, uuid.UUID, uuid.UUID, int, int) ([]*entity.WebhookDelivery, int64, error) {
	return nil, 0, nil
}

func (m *mockWebhookDeliveryRepo) UpdateStatus(context.Context, uuid.UUID, string, *int, map[string][]string, *string, *int, time.Time) error {
	return nil
}

var _ domainrepo.IWebhookDeliveryRepository = (*mockWebhookDeliveryRepo)(nil)

type mockWebhookRepo struct {
	webhook *entity.Webhook
}

func newMockWebhookRepo(webhook *entity.Webhook) *mockWebhookRepo {
	return &mockWebhookRepo{webhook: webhook}
}

func (m *mockWebhookRepo) Create(context.Context, *entity.Webhook) error { return nil }

func (m *mockWebhookRepo) GetByID(_ context.Context, id, orgID uuid.UUID) (*entity.Webhook, error) {
	if m.webhook == nil || m.webhook.ID != id || m.webhook.OrganizationID != orgID {
		return nil, apperror.ErrNotFound
	}
	copyHook := *m.webhook
	return &copyHook, nil
}

func (m *mockWebhookRepo) ListByRepo(context.Context, uuid.UUID, uuid.UUID, int, int) ([]*entity.Webhook, int64, error) {
	return nil, 0, nil
}

func (m *mockWebhookRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Webhook, int64, error) {
	return nil, 0, nil
}

func (m *mockWebhookRepo) Update(context.Context, *entity.Webhook) error { return nil }

func (m *mockWebhookRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func (m *mockWebhookRepo) ListActiveByRepoAndEvent(context.Context, uuid.UUID, uuid.UUID, string) ([]*entity.Webhook, error) {
	return nil, nil
}

var _ domainrepo.IWebhookRepository = (*mockWebhookRepo)(nil)

type mockWebhookDeliveryEnqueuer struct {
	payload queue.WebhookDeliveryPayload
}

func (m *mockWebhookDeliveryEnqueuer) EnqueueDelivery(_ context.Context, payload queue.WebhookDeliveryPayload) error {
	m.payload = payload
	return nil
}
