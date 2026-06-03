package observability

import (
	"net/http"
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
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}

// EchoPrometheusMiddleware records request count and latency for each route.
func EchoPrometheusMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
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

		path := c.Path()
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

// RegisterMetricsRoute registers the /metrics endpoint on the given Echo instance.
func RegisterMetricsRoute(e *echo.Echo) {
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
}
