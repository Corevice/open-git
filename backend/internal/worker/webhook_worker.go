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
	"syscall"
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

// addrBlocked reports whether an IP must not be reached by webhook delivery.
// It is a package variable so tests (which deliver to httptest servers on
// loopback) can relax it; production uses isPrivateIP.
var addrBlocked = isPrivateIP

// WebhookDeliveryPayload is the legacy enqueue payload shape used by callers
// that build webhook:deliver tasks directly. New code should use
// queue.WebhookDeliveryPayload instead.
type WebhookDeliveryPayload struct {
	WebhookID string `json:"webhook_id"`
	URL       string `json:"url"`
	Secret    string `json:"secret"`
	Event     string `json:"event"`
	Body      []byte `json:"body"`
}

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
		httpClient:    newSSRFSafeHTTPClient(),
		deliveryRepo:  deliveryRepo,
		webhookRepo:   webhookRepo,
		decryptSecret: identityWebhookSecretDecrypter,
	}
}

// newSSRFSafeHTTPClient builds a client that rejects connections to private,
// loopback, link-local and unspecified addresses at every dial — which is what
// actually protects against DNS rebinding (the pre-flight lookup and the real
// connection resolve DNS independently) — and re-validates the host on every
// redirect hop.
func newSSRFSafeHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: httpDeliveryTimeout}
	safeControl := func(_, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return err
		}
		if addrBlocked(net.ParseIP(host)) {
			return fmt.Errorf("blocked connection to non-public address %s", host)
		}
		return nil
	}
	dialer.Control = safeControl
	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   httpDeliveryTimeout,
		ResponseHeaderTimeout: httpDeliveryTimeout,
	}
	return &http.Client{
		Timeout:   httpDeliveryTimeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			if host := req.URL.Hostname(); host != "" {
				if ip := net.ParseIP(host); ip != nil && addrBlocked(ip) {
					return fmt.Errorf("blocked redirect to non-public address %s", host)
				}
			}
			return nil
		},
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
		if recordErr := w.recordBlockedDelivery(ctx, deliveryID, hookID, orgID, payload, deliveryStatusFailed, nil, nil); recordErr != nil {
			// Best-effort audit; DNS failures must not retry.
		}
		return nil
	}
	for _, addr := range addrs {
		if addrBlocked(net.ParseIP(addr)) {
			if recordErr := w.recordBlockedDelivery(ctx, deliveryID, hookID, orgID, payload, deliveryStatusSSRFBlocked, nil, nil); recordErr != nil {
				// Best-effort audit; SSRF blocks must not retry.
			}
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
		if err := w.updateDeliveryResult(ctx, deliveryID, deliveryStatusFailed, nil, nil, nil, &durationMs, deliveredAt); err != nil {
			return fmt.Errorf("update failed delivery: %w", err)
		}
		return w.deliveryRetryError(ctx, sendErr)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	responseBodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
	if readErr != nil {
		if err := w.updateDeliveryResult(ctx, deliveryID, deliveryStatusFailed, nil, nil, nil, &durationMs, deliveredAt); err != nil {
			return fmt.Errorf("update delivery after read failure: %w", err)
		}
		return w.deliveryRetryError(ctx, readErr)
	}
	responseBody := string(responseBodyBytes)
	statusCode := resp.StatusCode
	responseHeaders := resp.Header.Clone()

	status := deliveryStatusSuccess
	if statusCode < 200 || statusCode >= 300 {
		status = deliveryStatusFailed
	}

	if err := w.updateDeliveryResult(ctx, deliveryID, status, &statusCode, responseHeaders, &responseBody, &durationMs, deliveredAt); err != nil {
		return fmt.Errorf("update delivery result: %w", err)
	}

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
) error {
	return w.deliveryRepo.Create(ctx, &entity.WebhookDelivery{
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
) error {
	return w.deliveryRepo.UpdateStatus(
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

// isPrivateIP reports whether ip is one a webhook must never reach: private,
// loopback, link-local, unique-local, unspecified (0.0.0.0/::), or carrier-grade
// NAT space. Covers both IPv4 and IPv6.
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true // unparseable → treat as unsafe
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		// Carrier-grade NAT (100.64.0.0/10) is not covered by IsPrivate.
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
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
