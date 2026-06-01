package observability

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests handled, partitioned by method, path and status.",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency distribution in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal, httpRequestDurationSeconds)
}

// EchoPrometheusMiddleware records request count and latency for every request
// passing through the Echo router.
func EchoPrometheusMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()
		err := next(c)

		path := c.Path()
		if path == "" {
			path = c.Request().URL.Path
		}
		method := c.Request().Method
		status := c.Response().Status

		httpRequestsTotal.WithLabelValues(method, path, strconv.Itoa(status)).Inc()
		httpRequestDurationSeconds.WithLabelValues(method, path).Observe(time.Since(start).Seconds())

		return err
	}
}

// RegisterMetricsRoute mounts the Prometheus scrape endpoint at GET /metrics.
func RegisterMetricsRoute(e *echo.Echo) {
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
}
