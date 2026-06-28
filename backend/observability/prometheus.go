package observability

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// maxPrometheusLabelLen caps label values to stay within Prometheus' recommended
// 64-character limit and avoid unbounded cardinality from long user input.
const maxPrometheusLabelLen = 64

var allowedDBQueryNames = map[string]struct{}{
	"select_users": {},
}

var (
	metricsRegistry *prometheus.Registry
	metricsGatherer prometheus.Gatherer

	httpRequestsTotal *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	gitOperationsTotal *prometheus.CounterVec
	workflowRunsTotal *prometheus.CounterVec
	dbQueryDuration *prometheus.HistogramVec
)

func init() {
	metricsRegistry = prometheus.NewRegistry()
	metricsGatherer = metricsRegistry
	initCollectors(metricsRegistry)
}

func initCollectors(reg prometheus.Registerer) {
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "githost_http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "githost_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	gitOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "git_operations_total",
			Help: "Total git operations.",
		},
		[]string{"type", "protocol"},
	)

	workflowRunsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "workflow_runs_total",
			Help: "Total workflow runs.",
		},
		[]string{"status"},
	)

	dbQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "DB query duration.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"query_name"},
	)

	reg.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		gitOperationsTotal,
		workflowRunsTotal,
		dbQueryDuration,
	)
}

// InitTestMetrics reinitializes collectors on a fresh registry for isolated tests.
func InitTestMetrics(t *testing.T) prometheus.Gatherer {
	t.Helper()

	reg := prometheus.NewRegistry()
	initCollectors(reg)
	metricsRegistry = reg
	metricsGatherer = reg

	t.Cleanup(func() {
		reg := prometheus.NewRegistry()
		initCollectors(reg)
		metricsRegistry = reg
		metricsGatherer = reg
	})

	return reg
}

// MetricsGatherer returns the active Prometheus gatherer for this package.
func MetricsGatherer() prometheus.Gatherer {
	return metricsGatherer
}

func sanitizePrometheusLabel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}

	var b strings.Builder
	runeCount := 0
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
		runeCount++
		if runeCount >= maxPrometheusLabelLen {
			break
		}
	}

	if b.Len() == 0 {
		return "unknown"
	}
	return b.String()
}

func sanitizeDBQueryName(name string) string {
	sanitized := sanitizePrometheusLabel(name)
	if _, ok := allowedDBQueryNames[sanitized]; ok {
		return sanitized
	}
	return "other"
}

// ObserveGitOperation increments the git operations counter.
func ObserveGitOperation(opType, protocol string) {
	gitOperationsTotal.WithLabelValues(
		sanitizePrometheusLabel(opType),
		sanitizePrometheusLabel(protocol),
	).Inc()
}

// ObserveWorkflowRun increments the workflow runs counter.
func ObserveWorkflowRun(status string) {
	workflowRunsTotal.WithLabelValues(
		sanitizePrometheusLabel(status),
	).Inc()
}

// ObserveDBQuery records a DB query duration observation.
func ObserveDBQuery(queryName string, duration float64) {
	dbQueryDuration.WithLabelValues(sanitizeDBQueryName(queryName)).Observe(duration)
}

// EchoPrometheusMiddleware records request count and latency for each route.
func EchoPrometheusMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		path := c.Path()
		if path == "" {
			path = c.Request().URL.Path
		}
		if path == "/metrics" || strings.HasPrefix(path, "/metrics") {
			return next(c)
		}

		start := time.Now()
		err := next(c)
		duration := time.Since(start).Seconds()

		status := c.Response().Status
		if err != nil {
			if he, ok := err.(*echo.HTTPError); ok {
				status = he.Code
			} else {
				status = http.StatusInternalServerError
			}
		}

		path = c.Path()
		if path == "" {
			path = c.Request().URL.Path
		}

		httpRequestsTotal.WithLabelValues(
			c.Request().Method,
			path,
			strconv.Itoa(status),
		).Inc()

		httpRequestDuration.WithLabelValues(
			c.Request().Method,
			path,
		).Observe(duration)

		return err
	}
}

// NewMetricsHandler returns an Echo handler that exposes Prometheus metrics.
func NewMetricsHandler(authToken string) echo.HandlerFunc {
	return func(c echo.Context) error {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("metrics handler panic recovered: %v", r)
			}
		}()

		if authToken == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}

		auth := c.Request().Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		if token != authToken {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}

		return echo.WrapHandler(promhttp.HandlerFor(metricsGatherer, promhttp.HandlerOpts{}))(c)
	}
}

// RegisterMetricsRoute registers the metrics endpoint on the given Echo instance.
func RegisterMetricsRoute(e *echo.Echo, path, authToken string) {
	e.GET(path, NewMetricsHandler(authToken))
}
