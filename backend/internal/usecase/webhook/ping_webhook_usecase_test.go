package webhook_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	webhookusecase "github.com/open-git/backend/internal/usecase/webhook"
)

func TestPingWebhookUsecaseCreatesPingDelivery(t *testing.T) {
	orgID := uuid.New()
	webhookID := uuid.New()

	webhookRepo := newMockWebhookRepo(&entity.Webhook{
		ID:             webhookID,
		OrganizationID: orgID,
		URL:            "https://example.com/hook",
		ContentType:    entity.ContentTypeJSON,
		Events:         []string{"push"},
		Active:         true,
	})
	deliveryRepo := newMockWebhookDeliveryRepo(nil)
	enqueuer := &mockWebhookDeliveryEnqueuer{}
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

	hook, ok := body["hook"].(map[string]any)
	if !ok {
		t.Fatalf("hook payload missing")
	}
	config, ok := hook["config"].(map[string]any)
	if !ok {
		t.Fatalf("hook config missing")
	}
	if _, hasSecret := config["secret"]; hasSecret {
		t.Fatalf("ping payload must not include secret field")
	}
}
