package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/middleware"
)

func TestRequireScopeRepoMissing(t *testing.T) {
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middleware.SetAuthContext(c, 1, []string{"read"})
			return next(c)
		}
	})
	e.Use(middleware.RequireScope("repo"))
	e.GET("/", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if got := rec.Header().Get("X-Accepted-OAuth-Scopes"); got != "repo" {
		t.Fatalf("X-Accepted-OAuth-Scopes = %q, want repo", got)
	}
	if got := rec.Header().Get("X-OAuth-Scopes"); got != "read" {
		t.Fatalf("X-OAuth-Scopes = %q, want read", got)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["message"] != "Resource not accessible by personal access token" {
		t.Fatalf("unexpected message: %q", body["message"])
	}
	if body["documentation_url"] != "https://docs.github.com/rest" {
		t.Fatalf("documentation_url = %q", body["documentation_url"])
	}
}

func TestRequireScopeRepoPresent(t *testing.T) {
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middleware.SetAuthContext(c, 1, []string{"read", "repo"})
			return next(c)
		}
	})
	e.Use(middleware.RequireScope("repo"))
	e.GET("/", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
