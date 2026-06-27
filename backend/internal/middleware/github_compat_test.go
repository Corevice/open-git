package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/middleware"
)

func TestGitHubCompatHeadersPresent(t *testing.T) {
	e := echo.New()
	e.Use(middleware.GitHubCompatHeaders())
	e.GET("/", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("X-GitHub-Media-Type"); got != "github.v3; format=json" {
		t.Fatalf("expected X-GitHub-Media-Type=github.v3; format=json, got %q", got)
	}
}

func TestGitHubCompatHeadersWithScopes(t *testing.T) {
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middleware.SetAuthContext(c, 1, []string{"read", "repo"})
			return next(c)
		}
	})
	e.Use(middleware.GitHubCompatHeaders())
	e.GET("/", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("X-OAuth-Scopes"); got != "read,repo" {
		t.Fatalf("expected X-OAuth-Scopes=read,repo, got %q", got)
	}
}
