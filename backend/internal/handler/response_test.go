package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/handler"
)

func TestRespondOK(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.RespondOK(c, map[string]string{"key": "val"}); err != nil {
		t.Fatalf("RespondOK: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["data"]["key"] != "val" {
		t.Fatalf("data.key = %q, want %q", body["data"]["key"], "val")
	}
}

func TestRespondError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	const (
		statusCode = http.StatusNotFound
		code       = handler.CodeNotFound
		message    = "resource not found"
		requestID  = "req_123"
	)

	if err := handler.RespondError(c, statusCode, code, message, requestID); err != nil {
		t.Fatalf("RespondError: %v", err)
	}

	if rec.Code != statusCode {
		t.Fatalf("status = %d, want %d", rec.Code, statusCode)
	}

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Error.Code != code {
		t.Fatalf("error.code = %q, want %q", body.Error.Code, code)
	}
	if body.Error.Message != message {
		t.Fatalf("error.message = %q, want %q", body.Error.Message, message)
	}
	if body.RequestID != requestID {
		t.Fatalf("request_id = %q, want %q", body.RequestID, requestID)
	}
}
