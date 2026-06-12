package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/handler"
)

type pingDB interface {
	PingContext(ctx context.Context) error
}

type stubDB struct {
	err error
}

func (s stubDB) PingContext(ctx context.Context) error {
	return s.err
}

func newOperationalEcho(t *testing.T, db pingDB) *echo.Echo {
	t.Helper()

	e := echo.New()
	e.GET("/healthz", func(c echo.Context) error {
		return handler.RespondOK(c, map[string]string{"status": "ok"})
	})
	e.GET("/version", func(c echo.Context) error {
		return handler.RespondOK(c, map[string]string{
			"version":   "1.2.3",
			"commit":    "abc123",
			"buildTime": "2026-06-12T00:00:00Z",
		})
	})
	e.GET("/readyz", func(c echo.Context) error {
		if err := db.PingContext(c.Request().Context()); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]any{
				"data": map[string]string{"db": "down"},
			})
		}
		return handler.RespondOK(c, map[string]string{"db": "ok"})
	})
	return e
}

func TestHealthz(t *testing.T) {
	e := newOperationalEcho(t, stubDB{})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["data"]["status"] != "ok" {
		t.Fatalf("data.status = %q, want %q", body["data"]["status"], "ok")
	}
}

func TestVersion(t *testing.T) {
	e := newOperationalEcho(t, stubDB{})

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, key := range []string{"version", "commit", "buildTime"} {
		if body.Data[key] == "" {
			t.Fatalf("data.%s is empty", key)
		}
	}
}

func TestReadyzDBUp(t *testing.T) {
	e := newOperationalEcho(t, stubDB{})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["data"]["db"] != "ok" {
		t.Fatalf("data.db = %q, want %q", body["data"]["db"], "ok")
	}
}

func TestReadyzDBDown(t *testing.T) {
	e := newOperationalEcho(t, stubDB{err: errors.New("connection refused")})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	var body map[string]map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["data"]["db"] != "down" {
		t.Fatalf("data.db = %q, want %q", body["data"]["db"], "down")
	}
}
