package worker

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/queue"
)

type mockWebhookDeliveryRepo struct {
	created   []*entity.WebhookDelivery
	updated   []deliveryStatusUpdate
	createErr error
	updateErr error
}

type deliveryStatusUpdate struct {
	deliveryID      uuid.UUID
	status          string
	statusCode      *int
	responseHeaders map[string][]string
	responseBody    *string
	durationMs      *int
	deliveredAt     time.Time
}

func (m *mockWebhookDeliveryRepo) Create(_ context.Context, delivery *entity.WebhookDelivery) error {
	if m.createErr != nil {
		return m.createErr
	}
	copyDelivery := *delivery
	m.created = append(m.created, &copyDelivery)
	return nil
}

func (m *mockWebhookDeliveryRepo) GetByID(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*entity.WebhookDelivery, error) {
	return nil, nil
}

func (m *mockWebhookDeliveryRepo) ListByWebhook(context.Context, uuid.UUID, uuid.UUID, int, int) ([]*entity.WebhookDelivery, int64, error) {
	return nil, 0, nil
}

func (m *mockWebhookDeliveryRepo) UpdateStatus(
	_ context.Context,
	deliveryID uuid.UUID,
	status string,
	statusCode *int,
	responseHeaders map[string][]string,
	responseBody *string,
	durationMs *int,
	deliveredAt time.Time,
) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updated = append(m.updated, deliveryStatusUpdate{
		deliveryID:      deliveryID,
		status:          status,
		statusCode:      statusCode,
		responseHeaders: responseHeaders,
		responseBody:    responseBody,
		durationMs:      durationMs,
		deliveredAt:     deliveredAt,
	})
	return nil
}

type mockWebhookRepo struct {
	webhook *entity.Webhook
	err     error
}

func (m *mockWebhookRepo) Create(context.Context, *entity.Webhook) error { return nil }
func (m *mockWebhookRepo) GetByID(_ context.Context, _, _ uuid.UUID) (*entity.Webhook, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.webhook, nil
}
func (m *mockWebhookRepo) ListByRepo(context.Context, uuid.UUID, uuid.UUID, int, int) ([]*entity.Webhook, int64, error) {
	return nil, 0, nil
}
func (m *mockWebhookRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Webhook, int64, error) {
	return nil, 0, nil
}
func (m *mockWebhookRepo) Update(context.Context, *entity.Webhook) error { return nil }
func (m *mockWebhookRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}
func (m *mockWebhookRepo) ListActiveByRepoAndEvent(context.Context, uuid.UUID, uuid.UUID, string) ([]*entity.Webhook, error) {
	return nil, nil
}

func usePublicDNSLookup(t *testing.T) {
	t.Helper()
	original := lookupHost
	lookupHost = func(string) ([]string, error) {
		return []string{"8.8.8.8"}, nil
	}
	t.Cleanup(func() { lookupHost = original })
}

func newTestWorker(serverURL string, secret []byte, deliveryRepo *mockWebhookDeliveryRepo) *WebhookWorker {
	hookID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	orgID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	webhookRepo := &mockWebhookRepo{
		webhook: &entity.Webhook{
			ID:              hookID,
			OrganizationID:  orgID,
			URL:             serverURL,
			ContentType:     entity.ContentTypeJSON,
			SecretEncrypted: secret,
		},
	}
	return NewWebhookWorker(deliveryRepo, webhookRepo)
}

func marshalPayload(payload queue.WebhookDeliveryPayload) []byte {
	data, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return data
}

func TestHMACSignature(t *testing.T) {
	usePublicDNSLookup(t)
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

	deliveryRepo := &mockWebhookDeliveryRepo{}
	worker := newTestWorker(server.URL, []byte(secret), deliveryRepo)

	payload := queue.WebhookDeliveryPayload{
		DeliveryID:     "33333333-3333-3333-3333-333333333333",
		OrganizationID: "22222222-2222-2222-2222-222222222222",
		ContentType:    "json",
		HookID:         "11111111-1111-1111-1111-111111111111",
		Event:          "push",
		Body:           body,
		Attempt:        1,
	}
	task := asynq.NewTask(queue.TypeWebhookDeliver, marshalPayload(payload))
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
	usePublicDNSLookup(t)
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	deliveryRepo := &mockWebhookDeliveryRepo{}
	worker := newTestWorker(server.URL, []byte("secret"), deliveryRepo)

	payload := queue.WebhookDeliveryPayload{
		DeliveryID:     "44444444-4444-4444-4444-444444444444",
		OrganizationID: "22222222-2222-2222-2222-222222222222",
		ContentType:    "json",
		HookID:         "11111111-1111-1111-1111-111111111111",
		Event:          "push",
		Body:           []byte(`{}`),
		Attempt:        1,
	}
	task := asynq.NewTask(queue.TypeWebhookDeliver, marshalPayload(payload))

	if err := worker.HandleWebhookDeliver(context.Background(), task); err == nil {
		t.Fatal("expected error from HandleWebhookDeliver on 500, got nil")
	}
	if hits == 0 {
		t.Fatal("expected webhook endpoint to be hit at least once")
	}
	if len(deliveryRepo.updated) != 1 {
		t.Fatalf("expected one UpdateStatus call, got %d", len(deliveryRepo.updated))
	}
	if deliveryRepo.updated[0].status != deliveryStatusFailed {
		t.Fatalf("status = %q, want %q", deliveryRepo.updated[0].status, deliveryStatusFailed)
	}
}

func TestSSRFBlocked(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	originalLookup := lookupHost
	lookupHost = func(host string) ([]string, error) {
		return []string{"10.0.0.1"}, nil
	}
	defer func() { lookupHost = originalLookup }()

	deliveryRepo := &mockWebhookDeliveryRepo{}
	worker := newTestWorker(server.URL, nil, deliveryRepo)

	payload := queue.WebhookDeliveryPayload{
		DeliveryID:     "55555555-5555-5555-5555-555555555555",
		OrganizationID: "22222222-2222-2222-2222-222222222222",
		ContentType:    "json",
		HookID:         "11111111-1111-1111-1111-111111111111",
		Event:          "push",
		Body:           []byte(`{}`),
		Attempt:        1,
	}
	task := asynq.NewTask(queue.TypeWebhookDeliver, marshalPayload(payload))

	if err := worker.HandleWebhookDeliver(context.Background(), task); err != nil {
		t.Fatalf("expected nil error for SSRF block, got %v", err)
	}
	if atomic.LoadInt32(&hits) != 0 {
		t.Fatal("expected no HTTP call for SSRF blocked delivery")
	}
	if len(deliveryRepo.created) != 1 {
		t.Fatalf("expected one Create call, got %d", len(deliveryRepo.created))
	}
	if deliveryRepo.created[0].Status != deliveryStatusSSRFBlocked {
		t.Fatalf("status = %q, want %q", deliveryRepo.created[0].Status, deliveryStatusSSRFBlocked)
	}
}

func TestTimeout(t *testing.T) {
	usePublicDNSLookup(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(15 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	deliveryRepo := &mockWebhookDeliveryRepo{}
	worker := newTestWorker(server.URL, nil, deliveryRepo)

	payload := queue.WebhookDeliveryPayload{
		DeliveryID:     "66666666-6666-6666-6666-666666666666",
		OrganizationID: "22222222-2222-2222-2222-222222222222",
		ContentType:    "json",
		HookID:         "11111111-1111-1111-1111-111111111111",
		Event:          "push",
		Body:           []byte(`{}`),
		Attempt:        1,
	}
	task := asynq.NewTask(queue.TypeWebhookDeliver, marshalPayload(payload))

	start := time.Now()
	err := worker.HandleWebhookDeliver(context.Background(), task)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if elapsed > 11*time.Second {
		t.Fatalf("delivery took too long: %v", elapsed)
	}
	if len(deliveryRepo.updated) != 1 {
		t.Fatalf("expected one UpdateStatus call, got %d", len(deliveryRepo.updated))
	}
	if deliveryRepo.updated[0].status != deliveryStatusFailed {
		t.Fatalf("status = %q, want %q", deliveryRepo.updated[0].status, deliveryStatusFailed)
	}
}

func TestSuccessHeadersAndStatus(t *testing.T) {
	usePublicDNSLookup(t)
	deliveryID := "77777777-7777-7777-7777-777777777777"
	hookID := "11111111-1111-1111-1111-111111111111"

	var gotDeliveryHeader string
	var gotHookIDHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotDeliveryHeader = r.Header.Get("X-GitHub-Delivery")
		gotHookIDHeader = r.Header.Get("X-GitHub-Hook-ID")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	deliveryRepo := &mockWebhookDeliveryRepo{}
	worker := newTestWorker(server.URL, nil, deliveryRepo)

	payload := queue.WebhookDeliveryPayload{
		DeliveryID:     deliveryID,
		OrganizationID: "22222222-2222-2222-2222-222222222222",
		ContentType:    "json",
		HookID:         hookID,
		Event:          "push",
		Body:           []byte(`{}`),
		Attempt:        1,
	}
	task := asynq.NewTask(queue.TypeWebhookDeliver, marshalPayload(payload))

	if err := worker.HandleWebhookDeliver(context.Background(), task); err != nil {
		t.Fatalf("HandleWebhookDeliver returned error: %v", err)
	}
	if gotDeliveryHeader != deliveryID {
		t.Fatalf("X-GitHub-Delivery = %q, want %q", gotDeliveryHeader, deliveryID)
	}
	if gotHookIDHeader != hookID {
		t.Fatalf("X-GitHub-Hook-ID = %q, want %q", gotHookIDHeader, hookID)
	}
	if len(deliveryRepo.updated) != 1 {
		t.Fatalf("expected one UpdateStatus call, got %d", len(deliveryRepo.updated))
	}
	update := deliveryRepo.updated[0]
	if update.status != deliveryStatusSuccess {
		t.Fatalf("status = %q, want %q", update.status, deliveryStatusSuccess)
	}
	if update.statusCode == nil || *update.statusCode != http.StatusOK {
		t.Fatalf("status_code = %v, want %d", update.statusCode, http.StatusOK)
	}
}

func TestContentTypeForm(t *testing.T) {
	usePublicDNSLookup(t)
	hookID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	orgID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	var gotContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	deliveryRepo := &mockWebhookDeliveryRepo{}
	webhookRepo := &mockWebhookRepo{
		webhook: &entity.Webhook{
			ID:             hookID,
			OrganizationID: orgID,
			URL:            server.URL,
			ContentType:    entity.ContentTypeForm,
		},
	}
	worker := NewWebhookWorker(deliveryRepo, webhookRepo)

	payload := queue.WebhookDeliveryPayload{
		DeliveryID:     "88888888-8888-8888-8888-888888888888",
		OrganizationID: orgID.String(),
		ContentType:    "form",
		HookID:         hookID.String(),
		Event:          "push",
		Body:           []byte(`payload=1`),
		Attempt:        1,
	}
	task := asynq.NewTask(queue.TypeWebhookDeliver, marshalPayload(payload))

	if err := worker.HandleWebhookDeliver(context.Background(), task); err != nil {
		t.Fatalf("HandleWebhookDeliver returned error: %v", err)
	}
	if gotContentType != "application/x-www-form-urlencoded" {
		t.Fatalf("Content-Type = %q, want application/x-www-form-urlencoded", gotContentType)
	}
}

func TestNoSignatureWhenSecretEmpty(t *testing.T) {
	usePublicDNSLookup(t)
	var gotSignature string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSignature = r.Header.Get("X-Hub-Signature-256")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	deliveryRepo := &mockWebhookDeliveryRepo{}
	worker := newTestWorker(server.URL, nil, deliveryRepo)

	payload := queue.WebhookDeliveryPayload{
		DeliveryID:     "99999999-9999-9999-9999-999999999999",
		OrganizationID: "22222222-2222-2222-2222-222222222222",
		ContentType:    "json",
		HookID:         "11111111-1111-1111-1111-111111111111",
		Event:          "push",
		Body:           []byte(`{}`),
		Attempt:        1,
	}
	task := asynq.NewTask(queue.TypeWebhookDeliver, marshalPayload(payload))

	if err := worker.HandleWebhookDeliver(context.Background(), task); err != nil {
		t.Fatalf("HandleWebhookDeliver returned error: %v", err)
	}
	if gotSignature != "" {
		t.Fatalf("X-Hub-Signature-256 = %q, want empty", gotSignature)
	}
}

func TestIsPrivateIP(t *testing.T) {
	cases := []struct {
		ip      string
		private bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"169.254.1.1", true},
		{"8.8.8.8", false},
		{"::1", true},
		{"fc00::1", true},
	}
	for _, tc := range cases {
		got := isPrivateIP(netParseIP(t, tc.ip))
		if got != tc.private {
			t.Fatalf("isPrivateIP(%q) = %v, want %v", tc.ip, got, tc.private)
		}
	}
}

func netParseIP(t *testing.T, s string) net.IP {
	t.Helper()
	ip := net.ParseIP(s)
	if ip == nil {
		t.Fatalf("invalid ip %q", s)
	}
	return ip
}
