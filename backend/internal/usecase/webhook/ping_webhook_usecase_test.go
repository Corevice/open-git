package webhook_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/queue"
	webhookusecase "github.com/open-git/backend/internal/usecase/webhook"
)

type mockPingDeliveryRepo struct {
	created *entity.WebhookDelivery
}

func (m *mockPingDeliveryRepo) Create(_ context.Context, delivery *entity.WebhookDelivery) error {
	copyDelivery := *delivery
	m.created = &copyDelivery
	return nil
}

func (m *mockPingDeliveryRepo) GetByID(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*entity.WebhookDelivery, error) {
	return nil, apperror.ErrNotFound
}

func (m *mockPingDeliveryRepo) ListByWebhook(context.Context, uuid.UUID, uuid.UUID, int, int) ([]*entity.WebhookDelivery, int64, error) {
	return nil, 0, nil
}

func (m *mockPingDeliveryRepo) UpdateStatus(context.Context, uuid.UUID, string, *int, map[string][]string, *string, *int, time.Time) error {
	return nil
}

var _ domainrepo.IWebhookDeliveryRepository = (*mockPingDeliveryRepo)(nil)

type mockPingWebhookRepo struct {
	webhook *entity.Webhook
}

func (m *mockPingWebhookRepo) Create(context.Context, *entity.Webhook) error { return nil }

func (m *mockPingWebhookRepo) GetByID(_ context.Context, id, orgID uuid.UUID) (*entity.Webhook, error) {
	if m.webhook == nil || m.webhook.ID != id || m.webhook.OrganizationID != orgID {
		return nil, apperror.ErrNotFound
	}
	copyHook := *m.webhook
	return &copyHook, nil
}

func (m *mockPingWebhookRepo) ListByRepo(context.Context, uuid.UUID, uuid.UUID, int, int) ([]*entity.Webhook, int64, error) {
	return nil, 0, nil
}

func (m *mockPingWebhookRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Webhook, int64, error) {
	return nil, 0, nil
}

func (m *mockPingWebhookRepo) Update(context.Context, *entity.Webhook) error { return nil }

func (m *mockPingWebhookRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func (m *mockPingWebhookRepo) ListActiveByRepoAndEvent(context.Context, uuid.UUID, uuid.UUID, string) ([]*entity.Webhook, error) {
	return nil, nil
}

var _ domainrepo.IWebhookRepository = (*mockPingWebhookRepo)(nil)

type mockPingEnqueuer struct {
	payload queue.WebhookDeliveryPayload
}

func (m *mockPingEnqueuer) EnqueueDelivery(_ context.Context, payload queue.WebhookDeliveryPayload) error {
	m.payload = payload
	return nil
}

func TestPingWebhookUsecaseCreatesPingDelivery(t *testing.T) {
	orgID := uuid.New()
	webhookID := uuid.New()

	webhookRepo := &mockPingWebhookRepo{
		webhook: &entity.Webhook{
			ID:             webhookID,
			OrganizationID: orgID,
			URL:            "https://example.com/hook",
			ContentType:    entity.ContentTypeJSON,
			Events:         []string{"push"},
			Active:         true,
		},
	}
	deliveryRepo := &mockPingDeliveryRepo{}
	enqueuer := &mockPingEnqueuer{}
	uc := webhookusecase.NewPingWebhookUsecaseWithEnqueuer(webhookRepo, deliveryRepo, enqueuer)

	created, err := uc.Execute(context.Background(), webhookID, orgID)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if created.Event != "ping" {
		t.Fatalf("event = %q, want ping", created.Event)
	}
	if deliveryRepo.created == nil {
		t.Fatalf("expected delivery to be persisted")
	}
	if enqueuer.payload.Event != "ping" {
		t.Fatalf("enqueued event = %q, want ping", enqueuer.payload.Event)
	}

	var body map[string]any
	if err := json.Unmarshal([]byte(created.RequestBody), &body); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if body["zen"] != "Design for failure." {
		t.Fatalf("zen = %v", body["zen"])
	}
	if body["hook_id"] != webhookID.String() {
		t.Fatalf("hook_id = %v, want %s", body["hook_id"], webhookID)
	}
}
