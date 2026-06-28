package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/handler"
	"github.com/open-git/backend/internal/middleware"
)

var (
	secretTestOrgID  = uuid.MustParse("00000000-0000-0000-0000-000000000030")
	secretTestRepoID = uuid.MustParse("00000000-0000-0000-0000-000000000031")
	secretTestUserID = int64(31)
	secretTestActor  = uuid.MustParse("00000000-0000-0000-0000-000000000031")
)

type mockActionSecretRepo struct {
	secrets []*entity.ActionSecret
}

func (m *mockActionSecretRepo) List(_ context.Context, _, _ uuid.UUID) ([]*entity.ActionSecret, error) {
	return m.secrets, nil
}

func (m *mockActionSecretRepo) GetByName(_ context.Context, _, _ uuid.UUID, name string) (*entity.ActionSecret, error) {
	for _, secret := range m.secrets {
		if secret.Name == name {
			return secret, nil
		}
	}
	return nil, apperror.ErrNotFound
}

func (m *mockActionSecretRepo) Upsert(_ context.Context, secret *entity.ActionSecret) error {
	for i, existing := range m.secrets {
		if existing.Name == secret.Name {
			copySecret := *secret
			if existing.CreatedAt.IsZero() {
				copySecret.CreatedAt = time.Now().UTC()
			} else {
				copySecret.CreatedAt = existing.CreatedAt
			}
			m.secrets[i] = &copySecret
			return nil
		}
	}
	copySecret := *secret
	if copySecret.CreatedAt.IsZero() {
		copySecret.CreatedAt = time.Now().UTC()
	}
	m.secrets = append(m.secrets, &copySecret)
	return nil
}

func (m *mockActionSecretRepo) Delete(_ context.Context, _, _ uuid.UUID, name string) error {
	for i, secret := range m.secrets {
		if secret.Name == name {
			m.secrets = append(m.secrets[:i], m.secrets[i+1:]...)
			return nil
		}
	}
	return apperror.ErrNotFound
}

type mockSecretAuditRepo struct {
	records []secretAuditRecord
}

type secretAuditRecord struct {
	orgID      uuid.UUID
	actorID    uuid.UUID
	action     string
	targetType string
	targetID   uuid.UUID
	metadata   map[string]any
}

func (m *mockSecretAuditRepo) Record(_ context.Context, orgID, actorID uuid.UUID, action, targetType string, targetID uuid.UUID, metadata map[string]any) error {
	m.records = append(m.records, secretAuditRecord{
		orgID:      orgID,
		actorID:    actorID,
		action:     action,
		targetType: targetType,
		targetID:   targetID,
		metadata:   metadata,
	})
	return nil
}

func secretTestRepo() *entity.Repository {
	return &entity.Repository{
		ID:             secretTestRepoID,
		OrganizationID: secretTestOrgID,
		OwnerLogin:     "alice",
		Name:           "demo",
	}
}

func secretTestAuth(scopes ...string) echo.MiddlewareFunc {
	if len(scopes) == 0 {
		scopes = []string{"read", "write", "admin"}
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middleware.SetAuthContext(c, secretTestUserID, scopes)
			return next(c)
		}
	}
}

func newActionSecretHandlerEcho(t *testing.T, secretRepo *mockActionSecretRepo, auditRepo *mockSecretAuditRepo, auth echo.MiddlewareFunc) *echo.Echo {
	t.Helper()

	if secretRepo == nil {
		secretRepo = &mockActionSecretRepo{}
	}
	if auditRepo == nil {
		auditRepo = &mockSecretAuditRepo{}
	}
	if auth == nil {
		auth = secretTestAuth()
	}

	h := handler.NewActionSecretHandler(
		secretRepo,
		auditRepo,
		func(_ echo.Context, _, _ string) (*entity.Repository, error) {
			return secretTestRepo(), nil
		},
	)

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g, auth)
	return e
}

func TestListSecrets_ExcludesEncryptedValue(t *testing.T) {
	now := time.Now().UTC()
	secretRepo := &mockActionSecretRepo{
		secrets: []*entity.ActionSecret{
			{
				Name:           "MY_SECRET",
				EncryptedValue: "super-secret-value",
				CreatedAt:      now,
				UpdatedAt:      now,
			},
		},
	}

	e := newActionSecretHandlerEcho(t, secretRepo, nil, secretTestAuth("read"))
	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/actions/secrets", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if strings.Contains(body, "encrypted_value") {
		t.Fatalf("response must not contain encrypted_value, got %s", body)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	secrets, ok := resp["secrets"].([]any)
	if !ok || len(secrets) != 1 {
		t.Fatalf("expected 1 secret in response, got %#v", resp["secrets"])
	}
}

func TestPutSecret_InvalidName_Returns422(t *testing.T) {
	e := newActionSecretHandlerEcho(t, &mockActionSecretRepo{}, &mockSecretAuditRepo{}, secretTestAuth("admin"))

	body := bytes.NewBufferString(`{"encrypted_value":"value"}`)
	req := httptest.NewRequest(http.MethodPut, "/repos/alice/demo/actions/secrets/github_token", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPutSecret_GithubPrefix_Returns422(t *testing.T) {
	e := newActionSecretHandlerEcho(t, &mockActionSecretRepo{}, &mockSecretAuditRepo{}, secretTestAuth("admin"))

	body := bytes.NewBufferString(`{"encrypted_value":"value"}`)
	req := httptest.NewRequest(http.MethodPut, "/repos/alice/demo/actions/secrets/GITHUB_TOKEN", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPutSecret_ValidName_Returns201(t *testing.T) {
	auditRepo := &mockSecretAuditRepo{}
	e := newActionSecretHandlerEcho(t, &mockActionSecretRepo{}, auditRepo, secretTestAuth("admin"))

	body := bytes.NewBufferString(`{"encrypted_value":"encrypted-payload"}`)
	req := httptest.NewRequest(http.MethodPut, "/repos/alice/demo/actions/secrets/MY_SECRET", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	if len(auditRepo.records) != 1 || auditRepo.records[0].action != "secret.create" {
		t.Fatalf("expected secret.create audit record, got %#v", auditRepo.records)
	}
	if auditRepo.records[0].actorID != secretTestActor {
		t.Fatalf("expected actor_id from JWT, got %s", auditRepo.records[0].actorID)
	}
}

func TestDeleteSecret_AdminOnly(t *testing.T) {
	now := time.Now().UTC()
	secretRepo := &mockActionSecretRepo{
		secrets: []*entity.ActionSecret{
			{Name: "MY_SECRET", CreatedAt: now, UpdatedAt: now},
		},
	}
	e := newActionSecretHandlerEcho(t, secretRepo, &mockSecretAuditRepo{}, secretTestAuth("read", "write"))

	req := httptest.NewRequest(http.MethodDelete, "/repos/alice/demo/actions/secrets/MY_SECRET", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestDeleteSecret_RecordsAuditLog(t *testing.T) {
	now := time.Now().UTC()
	secretRepo := &mockActionSecretRepo{
		secrets: []*entity.ActionSecret{
			{Name: "MY_SECRET", CreatedAt: now, UpdatedAt: now},
		},
	}
	auditRepo := &mockSecretAuditRepo{}
	e := newActionSecretHandlerEcho(t, secretRepo, auditRepo, secretTestAuth("admin"))

	req := httptest.NewRequest(http.MethodDelete, "/repos/alice/demo/actions/secrets/MY_SECRET", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d body=%s", rec.Code, rec.Body.String())
	}
	if len(auditRepo.records) != 1 {
		t.Fatalf("expected 1 audit record, got %d", len(auditRepo.records))
	}
	if auditRepo.records[0].action != "secret.delete" {
		t.Fatalf("expected action secret.delete, got %q", auditRepo.records[0].action)
	}
	if auditRepo.records[0].actorID != secretTestActor {
		t.Fatalf("expected actor_id from JWT, got %s", auditRepo.records[0].actorID)
	}
}
