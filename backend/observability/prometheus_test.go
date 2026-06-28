package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestEchoPrometheusMiddleware(t *testing.T) {
	e := echo.New()
	mw := EchoPrometheusMiddleware(func(c echo.Context) error { return nil })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := mw(c); err != nil {
		t.Fatalf("middleware returned error: %v", err)
	}
}

func TestRegisterMetricsRoute(t *testing.T) {
	e := echo.New()
	RegisterMetricsRoute(e)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
