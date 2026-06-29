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
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/handler"
	"github.com/open-git/backend/internal/middleware"
	orgUC "github.com/open-git/backend/internal/usecase/org"
	securityusecase "github.com/open-git/backend/internal/usecase/security"
)

type orgAuditSearchRepo struct {
	logs []*entity.AuditLog
}

func (m *orgAuditSearchRepo) Search(_ context.Context, _ domainrepo.AuditLogSearchInput) ([]*entity.AuditLog, int, error) {
	return m.logs, len(m.logs), nil
}

type orgAuditMembershipRepo struct {
	roles map[int64]map[int64]string
}

func (m *orgAuditMembershipRepo) Add(context.Context, *entity.Membership) error {
	return nil
}

func (m *orgAuditMembershipRepo) GetRole(_ context.Context, orgID, userID uuid.UUID) (string, error) {
	if m.roles == nil {
		return "", nil
	}
	orgKey := middleware.UUIDToInt64(orgID)
	userKey := middleware.UUIDToInt64(userID)
	if userRoles, ok := m.roles[orgKey]; ok {
		return userRoles[userKey], nil
	}
	return "", nil
}

func (m *orgAuditMembershipRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Membership, error) {
	return nil, nil
}

func (m *orgAuditMembershipRepo) UpdateRole(context.Context, uuid.UUID, uuid.UUID, string) error {
	return nil
}

func (m *orgAuditMembershipRepo) Remove(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

type orgAuditExportEnqueuer struct{}

func (orgAuditExportEnqueuer) EnqueueAuditLogExport(context.Context, securityusecase.AuditLogExportPayload) error {
	return nil
}

func newOrgAuditLogHandlerEcho(
	t *testing.T,
	orgs *mockOrgRepo,
	memberships *orgAuditMembershipRepo,
	searchRepo *orgAuditSearchRepo,
	auth echo.MiddlewareFunc,
) *echo.Echo {
	t.Helper()

	searchUC := securityusecase.NewSearchAuditLogsUsecase(searchRepo)
	exportUC := securityusecase.NewExportAuditLogsUsecaseWithDeps(orgAuditExportEnqueuer{})

	h := handler.NewOrgAuditLogHandler(
		orgUC.NewGetOrgUsecase(orgs),
		memberships,
		searchUC,
		exportUC,
	)

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g, auth)
	return e
}

func TestOrgAuditLogSearch_OK(t *testing.T) {
	logID := uuid.New()
	orgs := &mockOrgRepo{
		byLogin: map[string]*domain.Organization{
			"acme": {ID: listTestOrgID, Login: "acme"},
		},
	}
	memberships := &orgAuditMembershipRepo{
		roles: map[int64]map[int64]string{
			listTestOrgID: {listTestUserID: entity.RoleAdmin},
		},
	}
	searchRepo := &orgAuditSearchRepo{
		logs: []*entity.AuditLog{{
			ID:         logID,
			ActorLogin: "alice",
			Action:     "member.add",
			TargetType: "User",
			TargetID:   "user-1",
			CreatedAt:  time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
		}},
	}
	e := newOrgAuditLogHandlerEcho(t, orgs, memberships, searchRepo, authMiddleware(listTestUserID))

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/audit-log", nil)
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
	if resp[0]["action"] != "member.add" {
		t.Fatalf("action = %v, want member.add", resp[0]["action"])
	}
}

func TestOrgAuditLogSearch_Forbidden(t *testing.T) {
	orgs := &mockOrgRepo{
		byLogin: map[string]*domain.Organization{
			"acme": {ID: listTestOrgID, Login: "acme"},
		},
	}
	memberships := &orgAuditMembershipRepo{
		roles: map[int64]map[int64]string{
			listTestOrgID: {listTestUserID: entity.RoleMember},
		},
	}
	e := newOrgAuditLogHandlerEcho(t, orgs, memberships, &orgAuditSearchRepo{}, authMiddleware(listTestUserID))

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme/audit-log", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestOrgAuditLogSearch_DateRangeExceeded(t *testing.T) {
	orgs := &mockOrgRepo{
		byLogin: map[string]*domain.Organization{
			"acme": {ID: listTestOrgID, Login: "acme"},
		},
	}
	memberships := &orgAuditMembershipRepo{
		roles: map[int64]map[int64]string{
			listTestOrgID: {listTestUserID: entity.RoleAdmin},
		},
	}
	e := newOrgAuditLogHandlerEcho(t, orgs, memberships, &orgAuditSearchRepo{}, authMiddleware(listTestUserID))

	after := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	before := after.Add(91 * 24 * time.Hour)
	url := "/orgs/acme/audit-log?after=" + after.Format(time.RFC3339) + "&before=" + before.Format(time.RFC3339)

	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestOrgAuditLogExport_Accepted(t *testing.T) {
	orgs := &mockOrgRepo{
		byLogin: map[string]*domain.Organization{
			"acme": {ID: listTestOrgID, Login: "acme"},
		},
	}
	memberships := &orgAuditMembershipRepo{
		roles: map[int64]map[int64]string{
			listTestOrgID: {listTestUserID: entity.RoleAdmin},
		},
	}
	e := newOrgAuditLogHandlerEcho(t, orgs, memberships, &orgAuditSearchRepo{}, authMiddleware(listTestUserID))

	after := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	before := after.Add(7 * 24 * time.Hour)
	url := "/orgs/acme/audit-log/export?format=json&after=" + after.Format(time.RFC3339) + "&before=" + before.Format(time.RFC3339)

	req := httptest.NewRequest(http.MethodPost, url, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusAccepted, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := resp["job_id"]; !ok {
		t.Fatalf("expected job_id in response, got %v", resp)
	}
}

func TestOrgAuditLogExport_RedisNotConfigured(t *testing.T) {
	orgs := &mockOrgRepo{
		byLogin: map[string]*domain.Organization{
			"acme": {ID: listTestOrgID, Login: "acme"},
		},
	}
	memberships := &orgAuditMembershipRepo{
		roles: map[int64]map[int64]string{
			listTestOrgID: {listTestUserID: entity.RoleAdmin},
		},
	}
	searchUC := securityusecase.NewSearchAuditLogsUsecase(&orgAuditSearchRepo{})
	exportUC := securityusecase.NewExportAuditLogsUsecase(nil)
	h := handler.NewOrgAuditLogHandler(
		orgUC.NewGetOrgUsecase(orgs),
		memberships,
		searchUC,
		exportUC,
	)
	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g, authMiddleware(listTestUserID))

	req := httptest.NewRequest(http.MethodPost, "/orgs/acme/audit-log/export?format=csv", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
}
