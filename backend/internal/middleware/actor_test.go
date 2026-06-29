package middleware_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/middleware"
)

type mockOrgByLoginLookup struct {
	orgs map[string]*domain.Organization
	err  error
}

func (m *mockOrgByLoginLookup) GetByLogin(_ context.Context, login string) (*domain.Organization, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.orgs == nil {
		return nil, nil
	}
	return m.orgs[login], nil
}

type mockOrgMembershipLookup struct {
	roles map[string]string
	err   error
}

func membershipRoleKey(orgID, userID int64) string {
	return fmt.Sprintf("%d:%d", orgID, userID)
}

func (m *mockOrgMembershipLookup) GetMemberRole(_ context.Context, orgID, userID int64) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if m.roles == nil {
		return "member", nil
	}
	role, ok := m.roles[membershipRoleKey(orgID, userID)]
	if !ok {
		return "", nil
	}
	return role, nil
}

func TestGetActorAbsent(t *testing.T) {
	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		actor, err := middleware.GetActor(c)
		if err == nil {
			t.Fatal("expected error when actor absent")
		}
		if err != echo.ErrUnauthorized {
			t.Fatalf("expected echo.ErrUnauthorized, got %v", err)
		}
		if actor != nil {
			t.Fatal("expected nil actor")
		}
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestGetActorSet(t *testing.T) {
	orgID := uuid.New()
	want := middleware.Actor{
		UserID:         42,
		OrganizationID: orgID,
	}

	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middleware.SetActor(c, want)
			return next(c)
		}
	})
	e.GET("/", func(c echo.Context) error {
		actor, err := middleware.GetActor(c)
		if err != nil {
			t.Fatalf("GetActor: %v", err)
		}
		if actor.UserID != want.UserID {
			t.Fatalf("UserID: got %v want %v", actor.UserID, want.UserID)
		}
		if actor.OrganizationID != want.OrganizationID {
			t.Fatalf("OrganizationID: got %v want %v", actor.OrganizationID, want.OrganizationID)
		}
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestResolveOwnerUnknown(t *testing.T) {
	e := echo.New()
	e.Use(middleware.ResolveOwner(&mockOrgByLoginLookup{}, &mockOrgMembershipLookup{}))
	e.GET("/repos/:owner/:repo/issues", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/repos/unknown-org/some-repo/issues", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestResolveOwnerDBError(t *testing.T) {
	e := echo.New()
	e.Use(middleware.ResolveOwner(
		&mockOrgByLoginLookup{err: errors.New("db connection failed")},
		&mockOrgMembershipLookup{},
	))
	e.GET("/repos/:owner/:repo/issues", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/repos/acme/some-repo/issues", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "failed to resolve organization") {
		t.Fatalf("expected sanitized error body, got %q", body)
	}
	if strings.Contains(body, "db connection failed") {
		t.Fatalf("internal error leaked in body: %q", body)
	}
}

func TestResolveOwnerMembershipDBError(t *testing.T) {
	orgID := int64(7)
	orgs := &mockOrgByLoginLookup{
		orgs: map[string]*domain.Organization{
			"acme": {ID: orgID, Login: "acme", Name: "Acme"},
		},
	}
	memberships := &mockOrgMembershipLookup{err: errors.New("db query failed")}

	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middleware.SetAuthContext(c, 99, nil)
			return next(c)
		}
	})
	e.Use(middleware.ResolveOwner(orgs, memberships))
	e.GET("/repos/:owner/:repo/issues", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/repos/acme/my-repo/issues", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestResolveOwnerNonMember(t *testing.T) {
	orgID := int64(7)
	userID := int64(99)
	orgs := &mockOrgByLoginLookup{
		orgs: map[string]*domain.Organization{
			"acme": {ID: orgID, Login: "acme", Name: "Acme"},
		},
	}
	memberships := &mockOrgMembershipLookup{
		roles: map[string]string{},
	}

	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middleware.SetAuthContext(c, userID, nil)
			return next(c)
		}
	})
	e.Use(middleware.ResolveOwner(orgs, memberships))
	e.GET("/repos/:owner/:repo/issues", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/repos/acme/my-repo/issues", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestResolveOwnerKnown(t *testing.T) {
	orgID := int64(7)
	userID := int64(99)
	orgUUID := middleware.Int64ToUUID(orgID)

	orgs := &mockOrgByLoginLookup{
		orgs: map[string]*domain.Organization{
			"acme": {ID: orgID, Login: "acme", Name: "Acme"},
		},
	}
	memberships := &mockOrgMembershipLookup{
		roles: map[string]string{
			membershipRoleKey(orgID, userID): "member",
		},
	}

	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middleware.SetAuthContext(c, userID, nil)
			return next(c)
		}
	})
	e.Use(middleware.ResolveOwner(orgs, memberships))
	e.GET("/repos/:owner/:repo/issues", func(c echo.Context) error {
		actor, err := middleware.GetActor(c)
		if err != nil {
			t.Fatalf("GetActor: %v", err)
		}
		if actor.UserID != userID {
			t.Fatalf("UserID: got %v want %v", actor.UserID, userID)
		}
		if actor.OrganizationID != orgUUID {
			t.Fatalf("OrganizationID: got %v want %v", actor.OrganizationID, orgUUID)
		}
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/repos/acme/my-repo/issues", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestResolveOwnerOrgParam(t *testing.T) {
	orgID := int64(3)
	orgUUID := middleware.Int64ToUUID(orgID)

	orgs := &mockOrgByLoginLookup{
		orgs: map[string]*domain.Organization{
			"my-org": {ID: orgID, Login: "my-org", Name: "My Org"},
		},
	}

	e := echo.New()
	e.Use(middleware.ResolveOwner(orgs, nil))
	e.GET("/orgs/:org/repos", func(c echo.Context) error {
		actor, err := middleware.GetActor(c)
		if err != nil {
			t.Fatalf("GetActor: %v", err)
		}
		if actor.OrganizationID != orgUUID {
			t.Fatalf("OrganizationID: got %v want %v", actor.OrganizationID, orgUUID)
		}
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/orgs/my-org/repos", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
