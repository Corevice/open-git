package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/handler"
	"github.com/open-git/backend/internal/middleware"
	authUC "github.com/open-git/backend/internal/usecase/auth"
)

type tokenAuditMockTokenRepo struct {
	created []*domain.AccessToken
}

func (m *tokenAuditMockTokenRepo) Create(_ context.Context, token *domain.AccessToken) error {
	m.created = append(m.created, token)
	token.ID = int64(len(m.created))
	return nil
}

func (m *tokenAuditMockTokenRepo) ListByUserID(_ context.Context, _ int64) ([]*domain.AccessToken, error) {
	return m.created, nil
}

func (m *tokenAuditMockTokenRepo) Revoke(_ context.Context, _, _ int64) error {
	return nil
}

func (m *tokenAuditMockTokenRepo) FindByTokenHash(_ context.Context, _ string) (*domain.AccessToken, error) {
	return nil, nil
}

type tokenAuditMockAuditRepo struct {
	logs  []*entity.AuditLog
	err   error
	calls int
}

func (m *tokenAuditMockAuditRepo) Create(_ context.Context, log *entity.AuditLog) error {
	m.calls++
	if m.err != nil {
		return m.err
	}
	copied := *log
	m.logs = append(m.logs, &copied)
	return nil
}

var _ domainrepo.IAuditLogRepository = (*tokenAuditMockAuditRepo)(nil)

type tokenAuditMockUserLookup struct {
	user *entity.User
}

func (m *tokenAuditMockUserLookup) GetByID(_ context.Context, _ uuid.UUID) (*entity.User, error) {
	return m.user, nil
}

func newTokenAuditEcho(t *testing.T, auditRepo *tokenAuditMockAuditRepo, userID int64) *echo.Echo {
	t.Helper()

	tokenRepo := &tokenAuditMockTokenRepo{}
	issueUC := authUC.NewIssuePATUsecase(tokenRepo)
	revokeUC := authUC.NewRevokePATUsecase(tokenRepo)
	userLookup := &tokenAuditMockUserLookup{
		user: &entity.User{
			ID:    middleware.Int64ToUUID(userID),
			Login: "octocat",
		},
	}
	h := handler.NewTokenHandler(tokenRepo, issueUC, revokeUC, auditRepo, userLookup)

	e := echo.New()
	auth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middleware.SetAuthContext(c, userID, nil)
			return next(c)
		}
	}

	tokens := e.Group("/user/tokens", auth)
	tokens.POST("", h.Create)
	return e
}

func TestTokenCreate_RecordsAudit(t *testing.T) {
	const userID int64 = 42
	auditRepo := &tokenAuditMockAuditRepo{}
	e := newTokenAuditEcho(t, auditRepo, userID)

	body := `{"note":"test","scopes":["read"]}`
	req := httptest.NewRequest(http.MethodPost, "/user/tokens", bytes.NewReader([]byte(body)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	if auditRepo.calls != 1 {
		t.Fatalf("audit Create calls = %d, want 1", auditRepo.calls)
	}
	if len(auditRepo.logs) != 1 {
		t.Fatalf("audit logs = %d, want 1", len(auditRepo.logs))
	}
	if auditRepo.logs[0].Action != "token.create" {
		t.Fatalf("action = %q, want token.create", auditRepo.logs[0].Action)
	}
	if auditRepo.logs[0].TargetType != "token" {
		t.Fatalf("targetType = %q, want token", auditRepo.logs[0].TargetType)
	}
	if auditRepo.logs[0].Metadata["actor_login"] != "octocat" {
		t.Fatalf("actor_login = %v, want octocat", auditRepo.logs[0].Metadata["actor_login"])
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	token, ok := resp["token"].(string)
	if !ok || token == "" {
		t.Fatalf("expected token in response, got %#v", resp["token"])
	}
}

func TestTokenCreate_AuditFailureDoesNotReturn500(t *testing.T) {
	const userID int64 = 42
	auditRepo := &tokenAuditMockAuditRepo{err: errors.New("audit write failed")}
	e := newTokenAuditEcho(t, auditRepo, userID)

	body := `{"note":"test","scopes":["read"]}`
	req := httptest.NewRequest(http.MethodPost, "/user/tokens", bytes.NewReader([]byte(body)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if auditRepo.calls != 1 {
		t.Fatalf("audit Create calls = %d, want 1", auditRepo.calls)
	}
}
