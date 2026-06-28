package observability

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const maxPrometheusLabelLen = 64

var (
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
		[]string{"type", "protocol", "organization_id"},
	)

	workflowRunsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "workflow_runs_total",
			Help: "Total workflow runs.",
		},
		[]string{"status", "organization_id"},
	)

	dbQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "DB query duration.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"query_name"},
	)
)

func init() {
	registerCollector(httpRequestsTotal)
	registerCollector(httpRequestDuration)
	registerCollector(gitOperationsTotal)
	registerCollector(workflowRunsTotal)
	registerCollector(dbQueryDuration)
}

func registerCollector(collector prometheus.Collector) {
	if err := prometheus.Register(collector); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
			log.Printf("metrics: already registered, reusing existing collector: %v", err)
			return
		}
		panic(err)
	}
}

func sanitizePrometheusLabel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}

	var b strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
		if b.Len() >= maxPrometheusLabelLen {
			break
		}
	}

	if b.Len() == 0 {
		return "unknown"
	}
	return b.String()
}

func sanitizeOrgID(orgID string) string {
	if _, err := uuid.Parse(strings.TrimSpace(orgID)); err == nil {
		return strings.TrimSpace(orgID)
	}
	return "invalid"
}

// ObserveGitOperation increments the git operations counter.
func ObserveGitOperation(opType, protocol, orgID string) {
	gitOperationsTotal.WithLabelValues(
		sanitizePrometheusLabel(opType),
		sanitizePrometheusLabel(protocol),
		sanitizeOrgID(orgID),
	).Inc()
}

// ObserveWorkflowRun increments the workflow runs counter.
func ObserveWorkflowRun(status, orgID string) {
	workflowRunsTotal.WithLabelValues(
		sanitizePrometheusLabel(status),
		sanitizeOrgID(orgID),
	).Inc()
}

// ObserveDBQuery records a DB query duration observation.
func ObserveDBQuery(queryName string, duration float64) {
	dbQueryDuration.WithLabelValues(sanitizePrometheusLabel(queryName)).Observe(duration)
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

		if authToken != "" {
			auth := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}
			token := strings.TrimPrefix(auth, "Bearer ")
			if token != authToken {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}
		}

		return echo.WrapHandler(promhttp.Handler())(c)
	}
}

// RegisterMetricsRoute registers the metrics endpoint on the given Echo instance.
func RegisterMetricsRoute(e *echo.Echo, path, authToken string) {
	e.GET(path, NewMetricsHandler(authToken))
}
