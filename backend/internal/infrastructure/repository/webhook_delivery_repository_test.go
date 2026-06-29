package repository_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func newWebhookDeliveryTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db := newWebhookTestDB(t)

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS webhook_deliveries (
			id TEXT PRIMARY KEY,
			webhook_id TEXT NOT NULL,
			organization_id TEXT NOT NULL,
			event TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			status_code INTEGER,
			request_headers TEXT NOT NULL DEFAULT '{}',
			request_body TEXT NOT NULL DEFAULT '',
			response_headers TEXT,
			response_body TEXT,
			duration_ms INTEGER,
			attempt INTEGER NOT NULL DEFAULT 1,
			redelivery INTEGER NOT NULL DEFAULT 0,
			parent_delivery_id TEXT,
			delivered_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("create webhook_deliveries table: %v", err)
	}

	return db
}

func TestWebhookDeliveryRepository_CreateGetByID(t *testing.T) {
	db := newWebhookDeliveryTestDB(t)
	repo := repository.NewWebhookDeliveryRepository(db)

	webhookID := uuid.New()
	orgID := uuid.New()
	delivery := &entity.WebhookDelivery{
		WebhookID:      webhookID,
		OrganizationID: orgID,
		Event:          "push",
		Status:         entity.StatusPending,
		RequestHeaders: map[string][]string{"X-Test": {"value"}},
		RequestBody:    `{"hello":"world"}`,
		Attempt:        1,
	}

	if err := repo.Create(context.Background(), delivery); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(context.Background(), delivery.ID, webhookID, orgID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected delivery, got nil")
	}
	if got.Event != delivery.Event || got.RequestBody != delivery.RequestBody {
		t.Fatalf("unexpected delivery: %+v", got)
	}
	if len(got.RequestHeaders["X-Test"]) != 1 || got.RequestHeaders["X-Test"][0] != "value" {
		t.Fatalf("unexpected request headers: %+v", got.RequestHeaders)
	}
}

func TestWebhookDeliveryRepository_GetByIDWrongOrg(t *testing.T) {
	db := newWebhookDeliveryTestDB(t)
	repo := repository.NewWebhookDeliveryRepository(db)

	webhookID := uuid.New()
	orgID := uuid.New()
	otherOrgID := uuid.New()

	delivery := &entity.WebhookDelivery{
		WebhookID:      webhookID,
		OrganizationID: orgID,
		Event:          "push",
		Status:         entity.StatusPending,
		Attempt:        1,
	}
	if err := repo.Create(context.Background(), delivery); err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err := repo.GetByID(context.Background(), delivery.ID, webhookID, otherOrgID)
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestWebhookDeliveryRepository_ListByWebhook(t *testing.T) {
	db := newWebhookDeliveryTestDB(t)
	repo := repository.NewWebhookDeliveryRepository(db)

	webhookID := uuid.New()
	orgID := uuid.New()

	for i := 0; i < 2; i++ {
		delivery := &entity.WebhookDelivery{
			WebhookID:      webhookID,
			OrganizationID: orgID,
			Event:          "push",
			Status:         entity.StatusPending,
			Attempt:        1,
		}
		if err := repo.Create(context.Background(), delivery); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	deliveries, total, err := repo.ListByWebhook(context.Background(), webhookID, orgID, 1, 10)
	if err != nil {
		t.Fatalf("ListByWebhook: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}
	if len(deliveries) != 2 {
		t.Fatalf("expected 2 deliveries, got %d", len(deliveries))
	}
}

func TestWebhookDeliveryRepository_UpdateStatusTruncatesResponseBody(t *testing.T) {
	db := newWebhookDeliveryTestDB(t)
	repo := repository.NewWebhookDeliveryRepository(db)

	webhookID := uuid.New()
	orgID := uuid.New()
	delivery := &entity.WebhookDelivery{
		WebhookID:      webhookID,
		OrganizationID: orgID,
		Event:          "push",
		Status:         entity.StatusPending,
		Attempt:        1,
	}
	if err := repo.Create(context.Background(), delivery); err != nil {
		t.Fatalf("Create: %v", err)
	}

	largeBody := strings.Repeat("x", 70*1024)
	statusCode := 200
	durationMs := 42
	deliveredAt := time.Now().UTC()

	if err := repo.UpdateStatus(
		context.Background(),
		delivery.ID,
		entity.StatusSuccess,
		&statusCode,
		map[string][]string{"Content-Type": {"text/plain"}},
		&largeBody,
		&durationMs,
		deliveredAt,
	); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	got, err := repo.GetByID(context.Background(), delivery.ID, webhookID, orgID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ResponseBody == nil {
		t.Fatal("expected response body, got nil")
	}
	if len(*got.ResponseBody) != 64*1024 {
		t.Fatalf("expected truncated body length 65536, got %d", len(*got.ResponseBody))
	}
	if got.Status != entity.StatusSuccess {
		t.Fatalf("unexpected status: %s", got.Status)
	}
}
