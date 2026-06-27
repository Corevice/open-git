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
	userUC "github.com/open-git/backend/internal/usecase/user"
)

type mockUserRepo struct {
	byID    map[int64]*domain.User
	byLogin map[string]*domain.User
}

func (m *mockUserRepo) Create(_ context.Context, _ *domain.User) error {
	return nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id int64) (*domain.User, error) {
	if m.byID == nil {
		return nil, nil
	}
	return m.byID[id], nil
}

func (m *mockUserRepo) GetByLogin(_ context.Context, login string) (*domain.User, error) {
	if m.byLogin == nil {
		return nil, nil
	}
	return m.byLogin[login], nil
}

func (m *mockUserRepo) GetByEmail(_ context.Context, _ string) (*domain.User, error) {
	return nil, nil
}

func newUserHandlerEcho(t *testing.T, users *mockUserRepo, auth echo.MiddlewareFunc) *echo.Echo {
	t.Helper()

	e := echo.New()
	h := handler.NewUserHandler(
		userUC.NewGetCurrentUserUsecase(users),
		userUC.NewGetUserByLoginUsecase(users),
		userUC.NewUpdateUserUsecase(nil),
	)

	g := e.Group("")
	h.RegisterRoutes(g, auth)
	return e
}

func TestGetCurrentUserUnauthorized(t *testing.T) {
	users := &mockUserRepo{}
	e := newUserHandlerEcho(t, users, func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "Bad credentials"})
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/user", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestGetCurrentUserOK(t *testing.T) {
	users := &mockUserRepo{
		byID: map[int64]*domain.User{
			42: {ID: 42, Login: "alice", Email: "alice@example.com"},
		},
	}
	auth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("user_id", int64(42))
			return next(c)
		}
	}
	e := newUserHandlerEcho(t, users, auth)

	req := httptest.NewRequest(http.MethodGet, "/user", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["login"] != "alice" {
		t.Fatalf("login = %v, want alice", resp["login"])
	}
	if resp["type"] != "User" {
		t.Fatalf("type = %v, want User", resp["type"])
	}
}

func TestGetUserByLoginNotFound(t *testing.T) {
	users := &mockUserRepo{byLogin: map[string]*domain.User{}}
	e := newUserHandlerEcho(t, users, func(next echo.HandlerFunc) echo.HandlerFunc {
		return next
	})

	req := httptest.NewRequest(http.MethodGet, "/users/ghost", nil)
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

func TestGetUserByLoginOK(t *testing.T) {
	users := &mockUserRepo{
		byLogin: map[string]*domain.User{
			"alice": {ID: 1, Login: "alice", Email: "alice@example.com"},
		},
	}
	e := newUserHandlerEcho(t, users, func(next echo.HandlerFunc) echo.HandlerFunc {
		return next
	})

	req := httptest.NewRequest(http.MethodGet, "/users/alice", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["login"] != "alice" {
		t.Fatalf("login = %v, want alice", resp["login"])
	}
	if _, ok := resp["email"]; ok {
		t.Fatalf("email should not be exposed for unauthenticated request, got %v", resp["email"])
	}
}
