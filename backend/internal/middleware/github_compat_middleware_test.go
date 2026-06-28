package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/middleware"
)

func TestGitHubCompatHeaders(t *testing.T) {
	e := echo.New()
	g := e.Group("", middleware.GitHubCompatHeaders())
	g.GET("/", func(c echo.Context) error {
		middleware.SetAuthContext(c, 1, []string{"repo", "read:org"})
		return c.NoContent(http.StatusOK)
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
	if got := rec.Header().Get("X-OAuth-Scopes"); got != "repo, read:org" {
		t.Fatalf("X-OAuth-Scopes = %q, want %q", got, "repo, read:org")
	}
	if got := rec.Header().Get("X-GitHub-Api-Version-Selected"); got != "2022-11-28" {
		t.Fatalf("X-GitHub-Api-Version-Selected = %q, want %q", got, "2022-11-28")
	}
}

func TestGitHubCompatUnknownApiVersion(t *testing.T) {
	e := echo.New()
	g := e.Group("", middleware.GitHubCompatHeaders())
	g.GET("/", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-GitHub-Api-Version", "2099-01-01")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
