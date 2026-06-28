package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/handler"
	"github.com/open-git/backend/internal/infrastructure/crypto"
	"github.com/open-git/backend/internal/middleware"
	webhookusecase "github.com/open-git/backend/internal/usecase/webhook"
)

var (
	webhookTestOrgID   = uuid.MustParse("00000000-0000-0000-0000-000000000042")
	webhookTestRepoID  = uuid.MustParse("00000000-0000-0000-0000-000000000099")
	webhookTestHookID  = uuid.MustParse("00000000-0000-0000-0000-000000000201")
	webhookTestUserID  = int64(7)
	otherOrgID         = uuid.MustParse("00000000-0000-0000-0000-000000000999")
)

type handlerWebhookRepo struct {
	webhooks map[uuid.UUID]*entity.Webhook
}

func (m *handlerWebhookRepo) Create(_ context.Context, webhook *entity.Webhook) error {
	if m.webhooks == nil {
		m.webhooks = map[uuid.UUID]*entity.Webhook{}
	}
	copyHook := *webhook
	m.webhooks[webhook.ID] = &copyHook
	return nil
}

func (m *handlerWebhookRepo) GetByID(_ context.Context, id, orgID uuid.UUID) (*entity.Webhook, error) {
	hook, ok := m.webhooks[id]
	if !ok || hook.OrganizationID != orgID {
		return nil, apperror.ErrNotFound
	}
	copyHook := *hook
	return &copyHook, nil
}

func (m *handlerWebhookRepo) ListByRepo(_ context.Context, orgID, repoID uuid.UUID, _, _ int) ([]*entity.Webhook, int64, error) {
	var hooks []*entity.Webhook
	for _, hook := range m.webhooks {
		if hook.OrganizationID == orgID && hook.RepositoryID != nil && *hook.RepositoryID == repoID {
			copyHook := *hook
			hooks = append(hooks, &copyHook)
		}
	}
	return hooks, int64(len(hooks)), nil
}

func (m *handlerWebhookRepo) ListByOrg(_ context.Context, orgID uuid.UUID, _, _ int) ([]*entity.Webhook, int64, error) {
	var hooks []*entity.Webhook
	for _, hook := range m.webhooks {
		if hook.OrganizationID == orgID && hook.RepositoryID == nil {
			copyHook := *hook
			hooks = append(hooks, &copyHook)
		}
	}
	return hooks, int64(len(hooks)), nil
}

func (m *handlerWebhookRepo) Update(_ context.Context, webhook *entity.Webhook) error {
	if m.webhooks == nil {
		return apperror.ErrNotFound
	}
	if _, ok := m.webhooks[webhook.ID]; !ok {
		return apperror.ErrNotFound
	}
	copyHook := *webhook
	m.webhooks[webhook.ID] = &copyHook
	return nil
}

func (m *handlerWebhookRepo) Delete(_ context.Context, id, orgID uuid.UUID) error {
	hook, ok := m.webhooks[id]
	if !ok || hook.OrganizationID != orgID {
		return apperror.ErrNotFound
	}
	delete(m.webhooks, id)
	return nil
}

func (m *handlerWebhookRepo) ListActiveByRepoAndEvent(_ context.Context, _, _ uuid.UUID, _ string) ([]*entity.Webhook, error) {
	return nil, nil
}

type handlerWebhookAuditRepo struct{}

func (handlerWebhookAuditRepo) InsertAuditLog(context.Context, uuid.UUID, uuid.UUID, string, string, uuid.UUID, json.RawMessage) error {
	return nil
}

func webhookTestRepo() *entity.Repository {
	return &entity.Repository{
		ID:             webhookTestRepoID,
		OrganizationID: webhookTestOrgID,
		OwnerLogin:     "alice",
		Name:           "demo",
	}
}

func webhookAdminAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		middleware.SetAuthContext(c, webhookTestUserID, []string{"write", "admin"})
		return next(c)
	}
}

func newWebhookHandlerEcho(t *testing.T, repo *handlerWebhookRepo) *echo.Echo {
	t.Helper()

	encryptor := crypto.NewSecretEncryptor(bytes.Repeat([]byte{0x22}, 32))
	createUC := webhookusecase.NewCreateWebhookUsecase(repo, handlerWebhookAuditRepo{}, encryptor)
	listUC := webhookusecase.NewListWebhooksUsecase(repo)
	getUC := webhookusecase.NewGetWebhookUsecase(repo)
	updateUC := webhookusecase.NewUpdateWebhookUsecase(repo, handlerWebhookAuditRepo{}, encryptor)
	deleteUC := webhookusecase.NewDeleteWebhookUsecase(repo, handlerWebhookAuditRepo{})

	h := handler.NewWebhookHandler(
		createUC,
		listUC,
		getUC,
		updateUC,
		deleteUC,
		func(_ echo.Context, _, _ string) (*entity.Repository, error) {
			return webhookTestRepo(), nil
		},
	)

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g, webhookAdminAuth)
	return e
}

func TestCreateHookRejectsFTPURL(t *testing.T) {
	e := newWebhookHandlerEcho(t, &handlerWebhookRepo{})

	body := bytes.NewBufferString(`{
		"name": "web",
		"active": true,
		"events": ["push"],
		"config": {
			"url": "ftp://example.com/hook",
			"content_type": "json",
			"secret": "s3cr3t"
		}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/repos/alice/demo/hooks", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

func TestCreateHookReturns201WithoutSecret(t *testing.T) {
	e := newWebhookHandlerEcho(t, &handlerWebhookRepo{})

	body := bytes.NewBufferString(`{
		"name": "web",
		"active": true,
		"events": ["push"],
		"config": {
			"url": "https://example.com/hook",
			"content_type": "json",
			"secret": "s3cr3t"
		}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/repos/alice/demo/hooks", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["name"] != "web" {
		t.Fatalf("name = %v, want web", resp["name"])
	}
	if resp["active"] != true {
		t.Fatalf("active = %v, want true", resp["active"])
	}
	events, ok := resp["events"].([]any)
	if !ok || len(events) != 1 || events[0] != "push" {
		t.Fatalf("events = %v, want [push]", resp["events"])
	}
	config, ok := resp["config"].(map[string]any)
	if !ok {
		t.Fatalf("config missing: %v", resp["config"])
	}
	if config["url"] != "https://example.com/hook" {
		t.Fatalf("config.url = %v", config["url"])
	}
	if config["content_type"] != "json" {
		t.Fatalf("config.content_type = %v", config["content_type"])
	}
	if secret, exists := config["secret"]; exists && secret != "" {
		t.Fatalf("secret should be empty or omitted, got %v", secret)
	}
}

func TestGetHookUnknownReturns404(t *testing.T) {
	e := newWebhookHandlerEcho(t, &handlerWebhookRepo{})

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/hooks/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestDeleteHookReturns204(t *testing.T) {
	repo := &handlerWebhookRepo{
		webhooks: map[uuid.UUID]*entity.Webhook{
			webhookTestHookID: {
				ID:             webhookTestHookID,
				OrganizationID: webhookTestOrgID,
				RepositoryID:   &webhookTestRepoID,
				URL:            "https://example.com/hook",
				ContentType:    entity.ContentTypeJSON,
				Events:         []string{"push"},
				Active:         true,
			},
		},
	}
	e := newWebhookHandlerEcho(t, repo)

	req := httptest.NewRequest(http.MethodDelete, "/repos/alice/demo/hooks/"+webhookTestHookID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
}

func TestGetHookCrossOrgReturns404(t *testing.T) {
	repo := &handlerWebhookRepo{
		webhooks: map[uuid.UUID]*entity.Webhook{
			webhookTestHookID: {
				ID:             webhookTestHookID,
				OrganizationID: otherOrgID,
				RepositoryID:   &webhookTestRepoID,
				URL:            "https://example.com/hook",
				ContentType:    entity.ContentTypeJSON,
				Events:         []string{"push"},
				Active:         true,
			},
		},
	}
	e := newWebhookHandlerEcho(t, repo)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/hooks/"+webhookTestHookID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}
