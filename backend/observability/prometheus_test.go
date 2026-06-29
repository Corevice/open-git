package observability_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/open-git/backend/observability"
	"github.com/prometheus/client_golang/prometheus"
)

func TestMiddlewareSkipsMetricsPath(t *testing.T) {
	e := echo.New()
	e.Use(observability.EchoPrometheusMiddleware)
	e.GET("/metrics", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	for _, mf := range metricFamilies {
		if mf.GetName() != "githost_http_requests_total" {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, label := range m.GetLabel() {
				if label.GetName() == "path" && label.GetValue() == "/metrics" {
					t.Fatal("found githost_http_requests_total sample with path=/metrics")
				}
			}
		}
	}
}

func TestNewMetricsHandlerNoAuth(t *testing.T) {
	e := echo.New()
	e.GET("/metrics", observability.NewMetricsHandler(""))

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/plain") {
		t.Fatalf("Content-Type = %q, want containing text/plain", rec.Header().Get("Content-Type"))
	}
}

func TestNewMetricsHandlerWrongToken(t *testing.T) {
	e := echo.New()
	e.GET("/metrics", observability.NewMetricsHandler("secret"))

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestNewMetricsHandlerCorrectToken(t *testing.T) {
	e := echo.New()
	e.GET("/metrics", observability.NewMetricsHandler("secret"))

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
