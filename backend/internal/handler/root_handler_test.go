package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/handler"
)

func TestGetRoot(t *testing.T) {
	e := echo.New()
	h := handler.NewRootHandler()
	e.GET("/", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "git.example.com"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := body["current_user_url"]; !ok {
		t.Fatal("current_user_url key missing")
	}
}
