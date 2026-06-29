package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/handler"
	repoUC "github.com/open-git/backend/internal/usecase/repository"
)

type mockListAuditLogsUsecase struct {
	logs       []*entity.AuditLog
	total      int
	lastAction string
}

func (m *mockListAuditLogsUsecase) Create(context.Context, *entity.AuditLog) error {
	return nil
}

func (m *mockListAuditLogsUsecase) List(_ context.Context, _ uuid.UUID, action string, _, _ int) ([]*entity.AuditLog, int, error) {
	m.lastAction = action
	return m.logs, m.total, nil
}

func (m *mockListAuditLogsUsecase) InsertAuditLog(context.Context, uuid.UUID, uuid.UUID, string, string, uuid.UUID, json.RawMessage) error {
	return nil
}

type mockAuditOrgRepo struct {
	roles map[int64]map[int64]string
}

func (m *mockAuditOrgRepo) GetByLogin(context.Context, string) (*domain.Organization, error) {
	return nil, nil
}

func (m *mockAuditOrgRepo) ListByUserID(context.Context, int64) ([]*domain.Organization, error) {
	return nil, nil
}

func (m *mockAuditOrgRepo) GetMemberRole(_ context.Context, orgID, userID int64) (string, error) {
	if m.roles == nil {
		return "", nil
	}
	if userRoles, ok := m.roles[orgID]; ok {
		return userRoles[userID], nil
	}
	return "", nil
}

func newAuditLogHandlerEcho(
	t *testing.T,
	repos *listMockRepositoryRepo,
	orgs *mockAuditOrgRepo,
	auditRepo *mockListAuditLogsUsecase,
	auth echo.MiddlewareFunc,
) *echo.Echo {
	t.Helper()

	memberships := &listMockMembershipRepo{}
	listRepos := repoUC.NewListRepositoriesUsecase(repos, memberships, nil)
	create := repoUC.NewCreateRepositoryUsecase(repos)
	get := repoUC.NewGetRepositoryUsecase(repos, nil, memberships)
	listAuditLogs := repoUC.NewListAuditLogsUsecase(auditRepo)

	h := handler.NewRepositoryHandler(create, get, listRepos, repos, orgs, &listMockAuditLogRepo{}, listAuditLogs)

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g, auth)
	return e
}

func testAuditRepository() *entity.Repository {
	return &entity.Repository{
		ID:             uuid.New(),
		OrganizationID: listTestOrgUUID,
		OwnerID:        listTestUserUUID,
		Name:           "demo",
		OwnerLogin:     "acme",
		Visibility:     entity.VisibilityPublic,
		DefaultBranch:  "main",
	}
}

func TestGetAuditLog_OK(t *testing.T) {
	logID := uuid.New()
	testRepo := testAuditRepository()
	repos := &listMockRepositoryRepo{
		byLoginName: map[string]*entity.Repository{
			repoLoginKey("acme", "demo"): testRepo,
		},
	}
	auditRepo := &mockListAuditLogsUsecase{
		logs: []*entity.AuditLog{{
			ID:         logID,
			ActorLogin: "alice",
			Action:     "repo.create",
			TargetType: "Repository",
			TargetID:   "123",
			CreatedAt:  time.Date(2025, 1, 15, 9, 24, 0, 0, time.UTC),
			Metadata:   map[string]any{"name": "demo"},
		}},
		total: 1,
	}
	orgs := &mockAuditOrgRepo{
		roles: map[int64]map[int64]string{
			listTestOrgID: {listTestUserID: "admin"},
		},
	}
	e := newAuditLogHandlerEcho(t, repos, orgs, auditRepo, authMiddleware(listTestUserID))

	req := httptest.NewRequest(http.MethodGet, "/repos/acme/demo/audit-log", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(resp))
	}

	entry := resp[0]
	for _, key := range []string{"id", "actor_login", "action", "target_type", "target_id", "created_at", "metadata"} {
		if _, ok := entry[key]; !ok {
			t.Fatalf("missing key %q in response: %v", key, entry)
		}
	}
	if entry["actor_login"] != "alice" {
		t.Fatalf("actor_login = %v, want alice", entry["actor_login"])
	}
	if entry["action"] != "repo.create" {
		t.Fatalf("action = %v, want repo.create", entry["action"])
	}
}

func TestGetAuditLog_Forbidden(t *testing.T) {
	testRepo := testAuditRepository()
	repos := &listMockRepositoryRepo{
		byLoginName: map[string]*entity.Repository{
			repoLoginKey("acme", "demo"): testRepo,
		},
	}
	auditRepo := &mockListAuditLogsUsecase{}
	orgs := &mockAuditOrgRepo{
		roles: map[int64]map[int64]string{
			listTestOrgID: {listTestUserID: "member"},
		},
	}
	e := newAuditLogHandlerEcho(t, repos, orgs, auditRepo, authMiddleware(listTestUserID))

	req := httptest.NewRequest(http.MethodGet, "/repos/acme/demo/audit-log", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestGetAuditLog_ActionFilter(t *testing.T) {
	testRepo := testAuditRepository()
	repos := &listMockRepositoryRepo{
		byLoginName: map[string]*entity.Repository{
			repoLoginKey("acme", "demo"): testRepo,
		},
	}
	auditRepo := &mockListAuditLogsUsecase{
		logs:  []*entity.AuditLog{},
		total: 0,
	}
	orgs := &mockAuditOrgRepo{
		roles: map[int64]map[int64]string{
			listTestOrgID: {listTestUserID: "owner"},
		},
	}
	e := newAuditLogHandlerEcho(t, repos, orgs, auditRepo, authMiddleware(listTestUserID))

	req := httptest.NewRequest(http.MethodGet, "/repos/acme/demo/audit-log?action=repo.delete", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if auditRepo.lastAction != "repo.delete" {
		t.Fatalf("lastAction = %q, want repo.delete", auditRepo.lastAction)
	}
}
