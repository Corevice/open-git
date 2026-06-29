package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/middleware"
)

func TestGitHubCommonHeadersMiddleware(t *testing.T) {
	e := echo.New()
	g := e.Group("", middleware.GitHubCommonHeadersMiddleware())
	g.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("X-GitHub-Media-Type"); got != "github.v3; format=json" {
		t.Fatalf("X-GitHub-Media-Type = %q, want %q", got, "github.v3; format=json")
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want %q", got, "application/json; charset=utf-8")
	}
}

func TestGitHubCommonHeadersMiddlewarePreservesExistingMediaType(t *testing.T) {
	e := echo.New()
	g := e.Group("", middleware.GitHubCommonHeadersMiddleware())
	g.GET("/", func(c echo.Context) error {
		c.Response().Header().Set("X-GitHub-Media-Type", "custom.v3; format=json")
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-GitHub-Media-Type"); got != "custom.v3; format=json" {
		t.Fatalf("X-GitHub-Media-Type = %q, want handler value preserved", got)
	}
}
