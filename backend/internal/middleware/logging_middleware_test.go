package middleware_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/middleware"
)

func initTestLogger(buf *bytes.Buffer) {
	middleware.InitLoggingWithOutput(buf, "debug")
}

func TestRequestLogger_FieldsPresent(t *testing.T) {
	var buf bytes.Buffer
	initTestLogger(&buf)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)
	c.SetPath("/test")
	c.Response().Header().Set(echo.HeaderXRequestID, "req-test-123")

	handler := middleware.RequestLogger()(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	if err := handler(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log output: %v; raw=%q", err, buf.String())
	}

	for _, field := range []string{"request_id", "http_method", "path", "status", "latency_ms"} {
		if _, ok := entry[field]; !ok {
			t.Fatalf("expected field %q in log output: %s", field, buf.String())
		}
	}

	if entry["request_id"] != "req-test-123" {
		t.Fatalf("expected request_id req-test-123, got %v", entry["request_id"])
	}
	if entry["http_method"] != http.MethodGet {
		t.Fatalf("expected http_method GET, got %v", entry["http_method"])
	}
	if entry["path"] != "/test" {
		t.Fatalf("expected path /test, got %v", entry["path"])
	}
}

func TestRequestLogger_ErrorLevelOn5xx(t *testing.T) {
	var buf bytes.Buffer
	initTestLogger(&buf)

	req := httptest.NewRequest(http.MethodGet, "/fail", nil)
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)
	c.SetPath("/fail")

	handler := middleware.RequestLogger()(func(c echo.Context) error {
		return c.NoContent(http.StatusInternalServerError)
	})
	if err := handler(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"level":"error"`) {
		t.Fatalf("expected error level log, got: %s", output)
	}
}

func TestStructuredRecover_PanicReturns500(t *testing.T) {
	var buf bytes.Buffer
	initTestLogger(&buf)

	e := echo.New()
	e.Use(middleware.StructuredRecover())
	e.GET("/panic", func(c echo.Context) error {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}

	output := buf.String()
	if !strings.Contains(output, "stack_trace") {
		t.Fatalf("expected stack_trace in log output, got: %s", output)
	}
}
