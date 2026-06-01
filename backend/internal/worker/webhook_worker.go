package worker

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hibiken/asynq"
)

const (
	MaxWebhookRetries     = 5
	httpDeliveryTimeout   = 5 * time.Second
	signatureHeader       = "X-Hub-Signature-256"
	eventHeader           = "X-GitHub-Event"
	deliveryStatusSuccess = "success"
	deliveryStatusFailed  = "failed"
)

type WebhookDeliveryPayload struct {
	WebhookID string `json:"webhook_id"`
	URL       string `json:"url"`
	Secret    string `json:"secret"`
	Event     string `json:"event"`
	Body      []byte `json:"body"`
}

type WebhookWorker struct {
	httpClient *http.Client
	db         *sql.DB
}

func NewWebhookWorker(db *sql.DB) *WebhookWorker {
	return &WebhookWorker{
		httpClient: &http.Client{Timeout: httpDeliveryTimeout},
		db:         db,
	}
}

func (w *WebhookWorker) HandleWebhookDeliver(ctx context.Context, task *asynq.Task) error {
	var payload WebhookDeliveryPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal webhook payload: %w", err)
	}

	signature := computeSignature(payload.Secret, payload.Body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, payload.URL, bytes.NewReader(payload.Body))
	if err != nil {
		return fmt.Errorf("build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(signatureHeader, signature)
	if payload.Event != "" {
		req.Header.Set(eventHeader, payload.Event)
	}

	statusCode, deliveryErr := w.send(req)
	w.persistDeliveryStatus(ctx, payload.WebhookID, statusCode, deliveryErr)

	if deliveryErr == nil {
		return nil
	}

	retried, _ := asynq.GetRetryCount(ctx)
	if retried < MaxWebhookRetries-1 {
		return fmt.Errorf("webhook delivery failed (attempt %d): %w", retried+1, deliveryErr)
	}
	return fmt.Errorf("webhook delivery exhausted retries: %w: %w", deliveryErr, asynq.SkipRetry)
}

func (w *WebhookWorker) send(req *http.Request) (int, error) {
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("non-2xx response: %d", resp.StatusCode)
	}
	return resp.StatusCode, nil
}

func (w *WebhookWorker) persistDeliveryStatus(ctx context.Context, webhookID string, statusCode int, deliveryErr error) {
	if w.db == nil || webhookID == "" {
		return
	}
	status := deliveryStatusSuccess
	if deliveryErr != nil {
		status = deliveryStatusFailed
	}
	if statusCode != 0 {
		status = fmt.Sprintf("%s:%d", status, statusCode)
	}
	_, _ = w.db.ExecContext(
		ctx,
		`UPDATE webhooks SET last_delivery_status = $1, last_delivery_at = $2 WHERE id = $3`,
		status,
		time.Now().UTC(),
		webhookID,
	)
}

func computeSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
