package entity_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

func TestWebhookDeliveryStatusConstants(t *testing.T) {
	if entity.StatusPending != "pending" {
		t.Fatalf("StatusPending = %q, want pending", entity.StatusPending)
	}
	if entity.StatusSuccess != "success" {
		t.Fatalf("StatusSuccess = %q, want success", entity.StatusSuccess)
	}
	if entity.StatusFailed != "failed" {
		t.Fatalf("StatusFailed = %q, want failed", entity.StatusFailed)
	}
}

func TestWebhookDeliveryZeroValue(t *testing.T) {
	d := entity.WebhookDelivery{}

	if d.ID != uuid.Nil {
		t.Fatalf("ID = %v, want uuid.Nil", d.ID)
	}
	if d.WebhookID != uuid.Nil {
		t.Fatalf("WebhookID = %v, want uuid.Nil", d.WebhookID)
	}
	if d.OrganizationID != uuid.Nil {
		t.Fatalf("OrganizationID = %v, want uuid.Nil", d.OrganizationID)
	}
	if d.Event != "" {
		t.Fatalf("Event = %q, want empty string", d.Event)
	}
	if d.Status != "" {
		t.Fatalf("Status = %q, want empty string", d.Status)
	}
	if d.StatusCode != nil {
		t.Fatalf("StatusCode = %v, want nil", d.StatusCode)
	}
	if d.RequestHeaders != nil {
		t.Fatalf("RequestHeaders = %v, want nil", d.RequestHeaders)
	}
	if d.RequestBody != "" {
		t.Fatalf("RequestBody = %q, want empty string", d.RequestBody)
	}
	if d.ResponseHeaders != nil {
		t.Fatalf("ResponseHeaders = %v, want nil", d.ResponseHeaders)
	}
	if d.ResponseBody != nil {
		t.Fatalf("ResponseBody = %v, want nil", d.ResponseBody)
	}
	if d.DurationMs != nil {
		t.Fatalf("DurationMs = %v, want nil", d.DurationMs)
	}
	if d.Attempt != 0 {
		t.Fatalf("Attempt = %d, want 0", d.Attempt)
	}
	if d.Redelivery {
		t.Fatalf("Redelivery = true, want false")
	}
	if d.ParentDeliveryID != nil {
		t.Fatalf("ParentDeliveryID = %v, want nil", d.ParentDeliveryID)
	}
	if d.DeliveredAt != nil {
		t.Fatalf("DeliveredAt = %v, want nil", d.DeliveredAt)
	}
	if !d.CreatedAt.Equal(time.Time{}) {
		t.Fatalf("CreatedAt = %v, want zero time", d.CreatedAt)
	}
}
