package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/handler"
)

func TestGetRateLimit(t *testing.T) {
	e := echo.New()
	h := handler.NewRateLimitHandler()
	e.GET("/rate_limit", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/rate_limit", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	resources, ok := body["resources"].(map[string]any)
	if !ok {
		t.Fatal("resources key missing or invalid")
	}
	core, ok := resources["core"].(map[string]any)
	if !ok {
		t.Fatal("resources.core missing or invalid")
	}
	if _, ok := core["limit"]; !ok {
		t.Fatal("resources.core.limit missing")
	}
}
