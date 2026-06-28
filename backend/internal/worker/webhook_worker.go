package worker

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/queue"
)

const (
	MaxWebhookRetries        = 5
	httpDeliveryTimeout      = 10 * time.Second
	maxResponseBodyBytes     = 64 * 1024
	signatureHeader          = "X-Hub-Signature-256"
	eventHeader              = "X-GitHub-Event"
	deliveryHeader           = "X-GitHub-Delivery"
	hookIDHeader             = "X-GitHub-Hook-ID"
	userAgentHeader          = "User-Agent"
	userAgentValue           = "OpenGit-Hookshot/1.0"
	deliveryStatusSuccess    = "success"
	deliveryStatusFailed     = "failed"
	deliveryStatusSSRFBlocked = "ssrf_blocked"
)

var lookupHost = net.LookupHost

type WebhookSecretDecrypter func(encrypted []byte) (string, error)

type WebhookWorker struct {
	httpClient    *http.Client
	deliveryRepo  repository.IWebhookDeliveryRepository
	webhookRepo   repository.IWebhookRepository
	decryptSecret WebhookSecretDecrypter
}

func NewWebhookWorker(
	deliveryRepo repository.IWebhookDeliveryRepository,
	webhookRepo repository.IWebhookRepository,
) *WebhookWorker {
	return &WebhookWorker{
		httpClient:    &http.Client{Timeout: httpDeliveryTimeout},
		deliveryRepo:  deliveryRepo,
		webhookRepo:   webhookRepo,
		decryptSecret: identityWebhookSecretDecrypter,
	}
}

func (w *WebhookWorker) WithSecretDecrypter(d WebhookSecretDecrypter) *WebhookWorker {
	w.decryptSecret = d
	return w
}

func (w *WebhookWorker) HandleWebhookDeliver(ctx context.Context, task *asynq.Task) error {
	var payload queue.WebhookDeliveryPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal webhook payload: %w", err)
	}

	deliveryID, err := uuid.Parse(payload.DeliveryID)
	if err != nil {
		return fmt.Errorf("parse delivery_id: %w", err)
	}
	orgID, err := uuid.Parse(payload.OrganizationID)
	if err != nil {
		return fmt.Errorf("parse organization_id: %w", err)
	}
	hookID, err := uuid.Parse(payload.HookID)
	if err != nil {
		return fmt.Errorf("parse hook_id: %w", err)
	}

	webhook, err := w.webhookRepo.GetByID(ctx, hookID, orgID)
	if err != nil {
		return fmt.Errorf("get webhook: %w", err)
	}

	parsedURL, err := url.Parse(webhook.URL)
	if err != nil {
		return fmt.Errorf("parse webhook url: %w", err)
	}

	host := parsedURL.Hostname()
	if host == "" {
		return fmt.Errorf("webhook url missing host")
	}

	addrs, err := lookupHost(host)
	if err != nil {
		w.recordBlockedDelivery(ctx, deliveryID, hookID, orgID, payload, deliveryStatusFailed, nil, nil)
		return nil
	}
	for _, addr := range addrs {
		if isPrivateIP(net.ParseIP(addr)) {
			w.recordBlockedDelivery(ctx, deliveryID, hookID, orgID, payload, deliveryStatusSSRFBlocked, nil, nil)
			return nil
		}
	}

	secret, err := w.decryptSecret(webhook.SecretEncrypted)
	if err != nil {
		return fmt.Errorf("decrypt webhook secret: %w", err)
	}

	contentType := resolveContentType(payload.ContentType)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewReader(payload.Body))
	if err != nil {
		return fmt.Errorf("build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set(eventHeader, payload.Event)
	req.Header.Set(deliveryHeader, payload.DeliveryID)
	req.Header.Set(hookIDHeader, payload.HookID)
	req.Header.Set(userAgentHeader, userAgentValue)
	if secret != "" {
		req.Header.Set(signatureHeader, computeSignature(secret, payload.Body))
	}

	requestHeaders := req.Header.Clone()
	if err := w.deliveryRepo.Create(ctx, &entity.WebhookDelivery{
		ID:             deliveryID,
		WebhookID:      hookID,
		OrganizationID: orgID,
		Event:          payload.Event,
		Status:         entity.StatusPending,
		RequestHeaders: requestHeaders,
		RequestBody:    string(payload.Body),
		Attempt:        payload.Attempt,
		CreatedAt:      time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("create delivery record: %w", err)
	}

	start := time.Now()
	resp, sendErr := w.httpClient.Do(req)
	durationMs := int(time.Since(start).Milliseconds())
	deliveredAt := time.Now().UTC()

	if sendErr != nil {
		w.updateDeliveryResult(ctx, deliveryID, deliveryStatusFailed, nil, nil, nil, &durationMs, deliveredAt)
		return w.deliveryRetryError(ctx, sendErr)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	responseBodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
	responseBody := string(responseBodyBytes)
	statusCode := resp.StatusCode
	responseHeaders := resp.Header.Clone()

	status := deliveryStatusSuccess
	if statusCode < 200 || statusCode >= 300 {
		status = deliveryStatusFailed
	}

	w.updateDeliveryResult(ctx, deliveryID, status, &statusCode, responseHeaders, &responseBody, &durationMs, deliveredAt)

	if status == deliveryStatusFailed {
		return w.deliveryRetryError(ctx, fmt.Errorf("non-2xx response: %d", statusCode))
	}
	return nil
}

func (w *WebhookWorker) recordBlockedDelivery(
	ctx context.Context,
	deliveryID, hookID, orgID uuid.UUID,
	payload queue.WebhookDeliveryPayload,
	status string,
	statusCode *int,
	durationMs *int,
) {
	_ = w.deliveryRepo.Create(ctx, &entity.WebhookDelivery{
		ID:             deliveryID,
		WebhookID:      hookID,
		OrganizationID: orgID,
		Event:          payload.Event,
		Status:         status,
		StatusCode:     statusCode,
		RequestBody:    string(payload.Body),
		DurationMs:     durationMs,
		Attempt:        payload.Attempt,
		CreatedAt:      time.Now().UTC(),
	})
}

func (w *WebhookWorker) updateDeliveryResult(
	ctx context.Context,
	deliveryID uuid.UUID,
	status string,
	statusCode *int,
	responseHeaders map[string][]string,
	responseBody *string,
	durationMs *int,
	deliveredAt time.Time,
) {
	_ = w.deliveryRepo.UpdateStatus(
		ctx,
		deliveryID,
		status,
		statusCode,
		responseHeaders,
		responseBody,
		durationMs,
		deliveredAt,
	)
}

func (w *WebhookWorker) deliveryRetryError(ctx context.Context, deliveryErr error) error {
	retried, _ := asynq.GetRetryCount(ctx)
	if retried < MaxWebhookRetries-1 {
		return fmt.Errorf("webhook delivery failed (attempt %d): %w", retried+1, deliveryErr)
	}
	return fmt.Errorf("webhook delivery exhausted retries: %w: %w", deliveryErr, asynq.SkipRetry)
}

func resolveContentType(contentType string) string {
	switch contentType {
	case "json":
		return "application/json"
	case "form":
		return "application/x-www-form-urlencoded"
	default:
		if contentType != "" {
			return contentType
		}
		return "application/json"
	}
}

func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip4 := ip.To4(); ip4 != nil {
		switch {
		case ip4[0] == 127:
			return true
		case ip4[0] == 10:
			return true
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return true
		case ip4[0] == 192 && ip4[1] == 168:
			return true
		case ip4[0] == 169 && ip4[1] == 254:
			return true
		default:
			return false
		}
	}
	if len(ip) == net.IPv6len && ip.To4() == nil {
		if ip.IsLoopback() {
			return true
		}
		if ip[0]&0xfe == 0xfc {
			return true
		}
	}
	return false
}

func computeSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func identityWebhookSecretDecrypter(encrypted []byte) (string, error) {
	return string(encrypted), nil
}
