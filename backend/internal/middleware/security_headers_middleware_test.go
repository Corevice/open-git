package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/middleware"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	e := echo.New()
	e.Use(middleware.SecurityHeadersMiddleware())
	e.GET("/", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	cases := []struct {
		header string
		want   string
	}{
		{"X-Frame-Options", "DENY"},
		{"X-Content-Type-Options", "nosniff"},
		{"Content-Security-Policy", "default-src 'self'"},
		{"Strict-Transport-Security", "max-age=31536000; includeSubDomains"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
		{"Permissions-Policy", "camera=(), microphone=(), geolocation=()"},
	}

	for _, tc := range cases {
		t.Run(tc.header, func(t *testing.T) {
			if got := rec.Header().Get(tc.header); got != tc.want {
				t.Fatalf("%s = %q, want %q", tc.header, got, tc.want)
			}
		})
	}
}

func TestSecurityHeadersMiddlewarePreservesExistingHeaders(t *testing.T) {
	e := echo.New()
	e.Use(middleware.SecurityHeadersMiddleware())
	e.GET("/", func(c echo.Context) error {
		c.Response().Header().Set("X-Frame-Options", "SAMEORIGIN")
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Frame-Options"); got != "SAMEORIGIN" {
		t.Fatalf("X-Frame-Options = %q, want handler value preserved", got)
	}
}
