package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/handler"
	orgUC "github.com/open-git/backend/internal/usecase/org"
)

type mockOrgRepo struct {
	byLogin   map[string]*domain.Organization
	byUserID  map[int64][]*domain.Organization
}

func (m *mockOrgRepo) GetByLogin(_ context.Context, login string) (*domain.Organization, error) {
	if m.byLogin == nil {
		return nil, nil
	}
	return m.byLogin[login], nil
}

func (m *mockOrgRepo) ListByUserID(_ context.Context, userID int64) ([]*domain.Organization, error) {
	if m.byUserID == nil {
		return []*domain.Organization{}, nil
	}
	return m.byUserID[userID], nil
}

func (m *mockOrgRepo) GetMemberRole(_ context.Context, _, _ int64) (string, error) {
	return "", nil
}

func newOrgHandlerEcho(t *testing.T, orgs *mockOrgRepo, auth echo.MiddlewareFunc) *echo.Echo {
	t.Helper()

	e := echo.New()
	h := handler.NewOrgHandler(
		orgUC.NewGetOrgUsecase(orgs),
		orgUC.NewListUserOrgsUsecase(orgs),
		nil,
		nil,
		nil,
	)

	g := e.Group("")
	h.RegisterRoutes(g, auth)
	return e
}

func TestGetOrgNotFound(t *testing.T) {
	orgs := &mockOrgRepo{byLogin: map[string]*domain.Organization{}}
	e := newOrgHandlerEcho(t, orgs, func(next echo.HandlerFunc) echo.HandlerFunc {
		return next
	})

	req := httptest.NewRequest(http.MethodGet, "/orgs/missing", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["message"] != "Not Found" {
		t.Fatalf("message = %q, want Not Found", resp["message"])
	}
}

func TestGetOrgOK(t *testing.T) {
	orgs := &mockOrgRepo{
		byLogin: map[string]*domain.Organization{
			"acme": {ID: 1, Login: "acme", Name: "Acme Corp"},
		},
	}
	e := newOrgHandlerEcho(t, orgs, func(next echo.HandlerFunc) echo.HandlerFunc {
		return next
	})

	req := httptest.NewRequest(http.MethodGet, "/orgs/acme", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["login"] != "acme" {
		t.Fatalf("login = %v, want acme", resp["login"])
	}
	if resp["type"] != "Organization" {
		t.Fatalf("type = %v, want Organization", resp["type"])
	}
}

func TestListUserOrgsOK(t *testing.T) {
	orgs := &mockOrgRepo{
		byUserID: map[int64][]*domain.Organization{
			7: {
				{ID: 1, Login: "acme", Name: "Acme Corp"},
				{ID: 2, Login: "beta", Name: "Beta Inc"},
			},
		},
	}
	auth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("user_id", int64(7))
			return next(c)
		}
	}
	e := newOrgHandlerEcho(t, orgs, auth)

	req := httptest.NewRequest(http.MethodGet, "/user/orgs", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("expected 2 orgs, got %d", len(resp))
	}
}

func TestListUserOrgsUnauth(t *testing.T) {
	orgs := &mockOrgRepo{}
	e := newOrgHandlerEcho(t, orgs, func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "Bad credentials"})
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/user/orgs", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
