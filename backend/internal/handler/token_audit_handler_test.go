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

func (m *tokenAuditMockAuditRepo) List(context.Context, uuid.UUID, string, int, int) ([]*entity.AuditLog, int, error) {
	return nil, 0, nil
}

func (m *tokenAuditMockAuditRepo) InsertAuditLog(context.Context, uuid.UUID, uuid.UUID, string, string, uuid.UUID, json.RawMessage) error {
	return nil
}

var _ domainrepo.IAuditLogRepository = (*tokenAuditMockAuditRepo)(nil)

type tokenAuditMockUserLookup struct {
	user *entity.User
	err  error
}

func (m *tokenAuditMockUserLookup) GetByID(_ context.Context, _ uuid.UUID) (*entity.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.user, nil
}

type tokenAuditUserLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
}

func newTokenAuditEcho(t *testing.T, auditRepo *tokenAuditMockAuditRepo, userID int64, userLookup tokenAuditUserLookup) *echo.Echo {
	t.Helper()

	tokenRepo := &tokenAuditMockTokenRepo{}
	issueUC := authUC.NewIssuePATUsecase(tokenRepo)
	revokeUC := authUC.NewRevokePATUsecase(tokenRepo)
	if userLookup == nil {
		userLookup = &tokenAuditMockUserLookup{
			user: &entity.User{
				ID:    middleware.Int64ToUUID(userID),
				Login: "octocat",
			},
		}
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
	tokens.DELETE("/:id", h.Revoke)
	return e
}

func TestTokenCreate_RecordsAudit(t *testing.T) {
	const userID int64 = 42
	auditRepo := &tokenAuditMockAuditRepo{}
	e := newTokenAuditEcho(t, auditRepo, userID, nil)

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
	if auditRepo.logs[0].OrganizationID != middleware.Int64ToUUID(userID) {
		t.Fatalf("organizationID = %v, want personal org %v", auditRepo.logs[0].OrganizationID, middleware.Int64ToUUID(userID))
	}
	if auditRepo.logs[0].TargetType != "token" {
		t.Fatalf("targetType = %q, want token", auditRepo.logs[0].TargetType)
	}
	if auditRepo.logs[0].TargetID != "1" {
		t.Fatalf("targetID = %q, want 1", auditRepo.logs[0].TargetID)
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
	e := newTokenAuditEcho(t, auditRepo, userID, nil)

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

func TestTokenRevoke_RecordsAudit(t *testing.T) {
	const userID int64 = 42
	auditRepo := &tokenAuditMockAuditRepo{}
	e := newTokenAuditEcho(t, auditRepo, userID, nil)

	body := `{"note":"test","scopes":["read"]}`
	createReq := httptest.NewRequest(http.MethodPost, "/user/tokens", bytes.NewReader([]byte(body)))
	createReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	createRec := httptest.NewRecorder()
	e.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d", createRec.Code, http.StatusCreated)
	}
	auditRepo.calls = 0
	auditRepo.logs = nil

	revokeReq := httptest.NewRequest(http.MethodDelete, "/user/tokens/1", nil)
	revokeRec := httptest.NewRecorder()
	e.ServeHTTP(revokeRec, revokeReq)

	if revokeRec.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d, want %d", revokeRec.Code, http.StatusNoContent)
	}
	if auditRepo.calls != 1 {
		t.Fatalf("audit Create calls = %d, want 1", auditRepo.calls)
	}
	if len(auditRepo.logs) != 1 {
		t.Fatalf("audit logs = %d, want 1", len(auditRepo.logs))
	}
	if auditRepo.logs[0].Action != "token.revoke" {
		t.Fatalf("action = %q, want token.revoke", auditRepo.logs[0].Action)
	}
	if auditRepo.logs[0].OrganizationID != middleware.Int64ToUUID(userID) {
		t.Fatalf("organizationID = %v, want personal org %v", auditRepo.logs[0].OrganizationID, middleware.Int64ToUUID(userID))
	}
	if auditRepo.logs[0].TargetID != "1" {
		t.Fatalf("targetID = %q, want 1", auditRepo.logs[0].TargetID)
	}
	if auditRepo.logs[0].Metadata["actor_login"] != "octocat" {
		t.Fatalf("actor_login = %v, want octocat", auditRepo.logs[0].Metadata["actor_login"])
	}
}

func TestTokenCreate_OmitsActorLoginWhenUserLookupFails(t *testing.T) {
	const userID int64 = 42
	auditRepo := &tokenAuditMockAuditRepo{}
	userLookup := &tokenAuditMockUserLookup{err: errors.New("user not found")}
	e := newTokenAuditEcho(t, auditRepo, userID, userLookup)

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
	if _, ok := auditRepo.logs[0].Metadata["actor_login"]; ok {
		t.Fatalf("actor_login should be omitted when user lookup fails, got %v", auditRepo.logs[0].Metadata["actor_login"])
	}
}
