package worker

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hibiken/asynq"

	"github.com/Corevice/open-git/backend/internal/infrastructure/queue"
)

func TestHMACSignature(t *testing.T) {
	const secret = "shh-its-a-secret"
	body := []byte(`{"event":"push","repository":"acme/widgets"}`)

	type capturedRequest struct {
		signature   string
		event       string
		contentType string
		body        []byte
	}
	got := capturedRequest{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.signature = r.Header.Get("X-Hub-Signature-256")
		got.event = r.Header.Get("X-GitHub-Event")
		got.contentType = r.Header.Get("Content-Type")
		got.body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	payload := WebhookDeliveryPayload{
		WebhookID: "11111111-1111-1111-1111-111111111111",
		URL:       server.URL,
		Secret:    secret,
		Event:     "push",
		Body:      body,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	worker := NewWebhookWorker(nil)
	task := asynq.NewTask(queue.TypeWebhookDeliver, data)
	if err := worker.HandleWebhookDeliver(context.Background(), task); err != nil {
		t.Fatalf("HandleWebhookDeliver returned error: %v", err)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	want := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if got.signature != want {
		t.Fatalf("X-Hub-Signature-256 mismatch:\n got  %q\n want %q", got.signature, want)
	}
	if got.event != "push" {
		t.Fatalf("X-GitHub-Event = %q, want %q", got.event, "push")
	}
	if got.contentType != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got.contentType)
	}
	if !bytes.Equal(got.body, body) {
		t.Fatalf("body mismatch:\n got  %q\n want %q", got.body, body)
	}
}

func TestRetryOn500(t *testing.T) {
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	payload := WebhookDeliveryPayload{
		WebhookID: "22222222-2222-2222-2222-222222222222",
		URL:       server.URL,
		Secret:    "secret",
		Event:     "push",
		Body:      []byte(`{}`),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	worker := NewWebhookWorker(nil)
	task := asynq.NewTask(queue.TypeWebhookDeliver, data)

	if err := worker.HandleWebhookDeliver(context.Background(), task); err == nil {
		t.Fatal("expected error from HandleWebhookDeliver on 500, got nil")
	}
	if hits == 0 {
		t.Fatal("expected webhook endpoint to be hit at least once")
	}
}
