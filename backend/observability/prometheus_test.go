package observability_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/open-git/backend/observability"
	dto "github.com/prometheus/client_model/go"
)

func TestMiddlewareSkipsMetricsPath(t *testing.T) {
	gatherer := observability.InitTestMetrics(t)

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

	metricFamilies, err := gatherer.Gather()
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
	t.Parallel()

	e := echo.New()
	mw := observability.EchoPrometheusMiddleware(func(c echo.Context) error { return nil })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := mw(c); err != nil {
		t.Fatalf("middleware returned error: %v", err)
	}
}

func TestNewMetricsHandlerRequiresAuthToken(t *testing.T) {
	e := echo.New()
	e.GET("/metrics", observability.NewMetricsHandler(""))

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
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
	observability.InitTestMetrics(t)

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
	observability.InitTestMetrics(t)

	e := echo.New()
	observability.RegisterMetricsRoute(e, "/metrics", "secret")

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer secret")
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
	gatherer := observability.InitTestMetrics(t)

	observability.ObserveGitOperation("clone", "https")

	metric := findMetricSample(t, gatherer, "git_operations_total", map[string]string{
		"type":     "clone",
		"protocol": "https",
	})
	if metric == nil {
		t.Fatal("expected git_operations_total sample")
	}
	if got := metric.GetCounter().GetValue(); got != 1 {
		t.Fatalf("counter value = %v, want 1", got)
	}
}

func TestObserveWorkflowRun(t *testing.T) {
	gatherer := observability.InitTestMetrics(t)

	observability.ObserveWorkflowRun("success")

	metric := findMetricSample(t, gatherer, "workflow_runs_total", map[string]string{
		"status": "success",
	})
	if metric == nil {
		t.Fatal("expected workflow_runs_total sample")
	}
	if got := metric.GetCounter().GetValue(); got != 1 {
		t.Fatalf("counter value = %v, want 1", got)
	}
}

func TestObserveDBQuery(t *testing.T) {
	gatherer := observability.InitTestMetrics(t)

	observability.ObserveDBQuery("select_users", 0.25)

	metric := findMetricSample(t, gatherer, "db_query_duration_seconds", map[string]string{
		"query_name": "select_users",
	})
	if metric == nil {
		t.Fatal("expected db_query_duration_seconds sample")
	}
	if got := metric.GetHistogram().GetSampleCount(); got != 1 {
		t.Fatalf("histogram sample count = %v, want 1", got)
	}
}

func TestObserveDBQueryUnknownName(t *testing.T) {
	gatherer := observability.InitTestMetrics(t)

	observability.ObserveDBQuery("drop_table", 0.1)

	metric := findMetricSample(t, gatherer, "db_query_duration_seconds", map[string]string{
		"query_name": "other",
	})
	if metric == nil {
		t.Fatal("expected db_query_duration_seconds sample with query_name=other")
	}
	if got := metric.GetHistogram().GetSampleCount(); got != 1 {
		t.Fatalf("histogram sample count = %v, want 1", got)
	}
}

func findMetricSample(t *testing.T, gatherer interface {
	Gather() ([]*dto.MetricFamily, error)
}, name string, labels map[string]string) *dto.Metric {
	t.Helper()

	metricFamilies, err := gatherer.Gather()
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
