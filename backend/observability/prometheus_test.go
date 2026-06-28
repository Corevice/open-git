package observability_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/open-git/backend/observability"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
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

func TestEchoPrometheusMiddleware(t *testing.T) {
	e := echo.New()
	mw := observability.EchoPrometheusMiddleware(func(c echo.Context) error { return nil })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := mw(c); err != nil {
		t.Fatalf("middleware returned error: %v", err)
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

func TestRegisterMetricsRoute(t *testing.T) {
	e := echo.New()
	observability.RegisterMetricsRoute(e, "/metrics", "")

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

func TestObserveGitOperation(t *testing.T) {
	const orgID = "550e8400-e29b-41d4-a716-446655440000"
	observability.ObserveGitOperation("clone", "https", orgID)
	observability.ObserveGitOperation("clone", "https", "not-a-uuid")

	metric := findCounterSample(t, "git_operations_total", map[string]string{
		"type":            "clone",
		"protocol":        "https",
		"organization_id": orgID,
	})
	if metric == nil || metric.GetCounter().GetValue() < 1 {
		t.Fatal("expected git_operations_total sample for valid org ID")
	}

	invalidMetric := findCounterSample(t, "git_operations_total", map[string]string{
		"type":            "clone",
		"protocol":        "https",
		"organization_id": "invalid",
	})
	if invalidMetric == nil || invalidMetric.GetCounter().GetValue() < 1 {
		t.Fatal("expected git_operations_total sample with organization_id=invalid")
	}
}

func TestObserveWorkflowRun(t *testing.T) {
	const orgID = "550e8400-e29b-41d4-a716-446655440000"
	observability.ObserveWorkflowRun("success", orgID)

	metric := findCounterSample(t, "workflow_runs_total", map[string]string{
		"status":          "success",
		"organization_id": orgID,
	})
	if metric == nil || metric.GetCounter().GetValue() < 1 {
		t.Fatal("expected workflow_runs_total sample")
	}
}

func TestObserveDBQuery(t *testing.T) {
	observability.ObserveDBQuery("select_users", 0.25)

	metric := findHistogramSample(t, "db_query_duration_seconds", map[string]string{
		"query_name": "select_users",
	})
	if metric == nil || metric.GetHistogram().GetSampleCount() < 1 {
		t.Fatal("expected db_query_duration_seconds sample")
	}
}

func findCounterSample(t *testing.T, name string, labels map[string]string) *dto.Metric {
	t.Helper()

	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	for _, mf := range metricFamilies {
		if mf.GetName() != name {
			continue
		}
		for _, metric := range mf.GetMetric() {
			if labelsMatch(metric.GetLabel(), labels) {
				return metric
			}
		}
	}
	return nil
}

func findHistogramSample(t *testing.T, name string, labels map[string]string) *dto.Metric {
	t.Helper()

	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	for _, mf := range metricFamilies {
		if mf.GetName() != name {
			continue
		}
		for _, metric := range mf.GetMetric() {
			if labelsMatch(metric.GetLabel(), labels) {
				return metric
			}
		}
	}
	return nil
}

func labelsMatch(actual []*dto.LabelPair, expected map[string]string) bool {
	if len(actual) != len(expected) {
		return false
	}
	for _, label := range actual {
		want, ok := expected[label.GetName()]
		if !ok || label.GetValue() != want {
			return false
		}
	}
	return true
}
