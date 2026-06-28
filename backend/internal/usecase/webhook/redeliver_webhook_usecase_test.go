package webhook_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/queue"
	webhookusecase "github.com/open-git/backend/internal/usecase/webhook"
)

type mockRedeliverDeliveryRepo struct {
	byID    map[uuid.UUID]*entity.WebhookDelivery
	created *entity.WebhookDelivery
}

func (m *mockRedeliverDeliveryRepo) Create(_ context.Context, delivery *entity.WebhookDelivery) error {
	copyDelivery := *delivery
	m.created = &copyDelivery
	if m.byID == nil {
		m.byID = map[uuid.UUID]*entity.WebhookDelivery{}
	}
	m.byID[delivery.ID] = &copyDelivery
	return nil
}

func (m *mockRedeliverDeliveryRepo) GetByID(_ context.Context, deliveryID, webhookID, orgID uuid.UUID) (*entity.WebhookDelivery, error) {
	delivery, ok := m.byID[deliveryID]
	if !ok || delivery.WebhookID != webhookID || delivery.OrganizationID != orgID {
		return nil, apperror.ErrNotFound
	}
	copyDelivery := *delivery
	return &copyDelivery, nil
}

func (m *mockRedeliverDeliveryRepo) ListByWebhook(context.Context, uuid.UUID, uuid.UUID, int, int) ([]*entity.WebhookDelivery, int64, error) {
	return nil, 0, nil
}

func (m *mockRedeliverDeliveryRepo) UpdateStatus(context.Context, uuid.UUID, string, *int, map[string][]string, *string, *int, time.Time) error {
	return nil
}

var _ domainrepo.IWebhookDeliveryRepository = (*mockRedeliverDeliveryRepo)(nil)

type mockRedeliverWebhookRepo struct {
	webhook *entity.Webhook
}

func (m *mockRedeliverWebhookRepo) Create(context.Context, *entity.Webhook) error { return nil }

func (m *mockRedeliverWebhookRepo) GetByID(_ context.Context, id, orgID uuid.UUID) (*entity.Webhook, error) {
	if m.webhook == nil || m.webhook.ID != id || m.webhook.OrganizationID != orgID {
		return nil, apperror.ErrNotFound
	}
	copyHook := *m.webhook
	return &copyHook, nil
}

func (m *mockRedeliverWebhookRepo) ListByRepo(context.Context, uuid.UUID, uuid.UUID, int, int) ([]*entity.Webhook, int64, error) {
	return nil, 0, nil
}

func (m *mockRedeliverWebhookRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Webhook, int64, error) {
	return nil, 0, nil
}

func (m *mockRedeliverWebhookRepo) Update(context.Context, *entity.Webhook) error { return nil }

func (m *mockRedeliverWebhookRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func (m *mockRedeliverWebhookRepo) ListActiveByRepoAndEvent(context.Context, uuid.UUID, uuid.UUID, string) ([]*entity.Webhook, error) {
	return nil, nil
}

var _ domainrepo.IWebhookRepository = (*mockRedeliverWebhookRepo)(nil)

type mockRedeliverEnqueuer struct {
	payload queue.WebhookDeliveryPayload
}

func (m *mockRedeliverEnqueuer) EnqueueDelivery(_ context.Context, payload queue.WebhookDeliveryPayload) error {
	m.payload = payload
	return nil
}

func TestRedeliverWebhookUsecaseCreatesRedelivery(t *testing.T) {
	orgID := uuid.New()
	webhookID := uuid.New()
	originalID := uuid.New()

	webhookRepo := &mockRedeliverWebhookRepo{
		webhook: &entity.Webhook{
			ID:             webhookID,
			OrganizationID: orgID,
			URL:            "https://example.com/hook",
			ContentType:    entity.ContentTypeJSON,
			Events:         []string{"push"},
			Active:         true,
		},
	}
	deliveryRepo := &mockRedeliverDeliveryRepo{
		byID: map[uuid.UUID]*entity.WebhookDelivery{
			originalID: {
				ID:             originalID,
				WebhookID:      webhookID,
				OrganizationID: orgID,
				Event:          "push",
				Status:         entity.StatusSuccess,
				RequestBody:    `{"ref":"refs/heads/main"}`,
			},
		},
	}
	enqueuer := &mockRedeliverEnqueuer{}
	uc := webhookusecase.NewRedeliverWebhookUsecaseWithEnqueuer(webhookRepo, deliveryRepo, enqueuer)

	created, err := uc.Execute(context.Background(), originalID, webhookID, orgID)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !created.Redelivery {
		t.Fatalf("expected redelivery=true")
	}
	if created.ParentDeliveryID == nil || *created.ParentDeliveryID != originalID {
		t.Fatalf("parent_delivery_id = %v, want %s", created.ParentDeliveryID, originalID)
	}
	if created.ID == originalID {
		t.Fatalf("expected new delivery id")
	}
	if deliveryRepo.created == nil {
		t.Fatalf("expected delivery to be persisted")
	}
	if enqueuer.payload.DeliveryID != created.ID.String() {
		t.Fatalf("enqueued delivery_id = %q, want %q", enqueuer.payload.DeliveryID, created.ID.String())
	}
	if string(enqueuer.payload.Body) != `{"ref":"refs/heads/main"}` {
		t.Fatalf("enqueued body = %q", string(enqueuer.payload.Body))
	}
}
