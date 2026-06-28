package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/middleware"
)

func TestRateLimitMiddlewareIndependentTokenBuckets(t *testing.T) {
	e := echo.New()
	g := e.Group("", middleware.RateLimitMiddleware(10, 60))
	g.GET("/", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.Header.Set("Authorization", "token tok1")
	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, req1)

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("Authorization", "token tok2")
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)

	if rec1.Code != http.StatusOK || rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d and %d", rec1.Code, rec2.Code)
	}

	remaining1, err := strconv.Atoi(rec1.Header().Get("X-RateLimit-Remaining"))
	if err != nil {
		t.Fatalf("parse remaining1: %v", err)
	}
	remaining2, err := strconv.Atoi(rec2.Header().Get("X-RateLimit-Remaining"))
	if err != nil {
		t.Fatalf("parse remaining2: %v", err)
	}

	if remaining1 != 9 {
		t.Fatalf("token1 remaining = %d, want 9", remaining1)
	}
	if remaining2 != 9 {
		t.Fatalf("token2 remaining = %d, want 9 (independent bucket)", remaining2)
	}
}

func TestRateLimitMiddlewareUnauthenticatedLimit(t *testing.T) {
	e := echo.New()
	g := e.Group("", middleware.RateLimitMiddleware(5000, 60))
	g.GET("/", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-RateLimit-Limit"); got != "60" {
		t.Fatalf("X-RateLimit-Limit = %q, want 60", got)
	}
}

func TestRateLimitMiddlewareExceededReturns403WithRetryAfter(t *testing.T) {
	e := echo.New()
	g := e.Group("", middleware.RateLimitMiddleware(1, 1))
	g.GET("/", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.Header.Set("Authorization", "token single-token")
	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, req1)

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("Authorization", "token single-token")
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)

	if rec1.Code != http.StatusOK {
		t.Fatalf("first request expected 200, got %d", rec1.Code)
	}
	if rec2.Code != http.StatusForbidden {
		t.Fatalf("second request expected 403, got %d", rec2.Code)
	}
	if got := rec2.Header().Get("Retry-After"); got == "" {
		t.Fatal("expected Retry-After header")
	}
	if got := rec2.Header().Get("X-RateLimit-Remaining"); got != "0" {
		t.Fatalf("X-RateLimit-Remaining = %q, want 0", got)
	}
}
