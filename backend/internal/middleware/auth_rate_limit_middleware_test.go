package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/middleware"
)

func newAuthRateLimitEcho(maxAttempts int, window time.Duration) *echo.Echo {
	e := echo.New()
	e.Use(middleware.AuthRateLimitMiddleware(maxAttempts, window))
	e.POST("/", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	return e
}

func authRateLimitRequest(e *echo.Echo) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(echo.HeaderXRealIP, "203.0.113.10")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestAuthRateLimitAllowsUnderLimit(t *testing.T) {
	e := newAuthRateLimitEcho(3, time.Minute)

	for i := 0; i < 3; i++ {
		rec := authRateLimitRequest(e)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}
}

func TestAuthRateLimitBlocksOnLimitPlusOne(t *testing.T) {
	e := newAuthRateLimitEcho(3, time.Minute)

	for i := 0; i < 3; i++ {
		rec := authRateLimitRequest(e)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	rec := authRateLimitRequest(e)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}

	retryAfter := rec.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Fatal("expected Retry-After header")
	}
	seconds, err := strconv.Atoi(retryAfter)
	if err != nil {
		t.Fatalf("parse Retry-After: %v", err)
	}
	if seconds <= 0 {
		t.Fatalf("expected Retry-After > 0, got %d", seconds)
	}
}

func TestAuthRateLimitResetsAfterWindow(t *testing.T) {
	e := newAuthRateLimitEcho(2, 50*time.Millisecond)

	for i := 0; i < 2; i++ {
		rec := authRateLimitRequest(e)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	blocked := authRateLimitRequest(e)
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 before reset, got %d", blocked.Code)
	}

	time.Sleep(60 * time.Millisecond)

	rec := authRateLimitRequest(e)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 after window reset, got %d", rec.Code)
	}
}
