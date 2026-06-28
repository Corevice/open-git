package webhook_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	webhookusecase "github.com/open-git/backend/internal/usecase/webhook"
)

func TestRedeliverWebhookUsecaseCreatesRedelivery(t *testing.T) {
	orgID := uuid.New()
	webhookID := uuid.New()
	originalID := uuid.New()

	webhookRepo := newMockWebhookRepo(&entity.Webhook{
		ID:             webhookID,
		OrganizationID: orgID,
		URL:            "https://example.com/hook",
		ContentType:    entity.ContentTypeJSON,
		Events:         []string{"push"},
		Active:         true,
	})
	deliveryRepo := newMockWebhookDeliveryRepo(map[uuid.UUID]*entity.WebhookDelivery{
		originalID: {
			ID:             originalID,
			WebhookID:      webhookID,
			OrganizationID: orgID,
			Event:          "push",
			Status:         entity.StatusSuccess,
			RequestBody:    `{"ref":"refs/heads/main"}`,
		},
	})
	enqueuer := &mockWebhookDeliveryEnqueuer{}
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
