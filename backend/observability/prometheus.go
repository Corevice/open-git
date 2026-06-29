package observability

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

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
)

func init() {
	if err := prometheus.Register(httpRequestsTotal); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
			log.Printf("metrics: already registered, reusing existing collector: %v", err)
		}
	}
	if err := prometheus.Register(httpRequestDuration); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
			log.Printf("metrics: already registered, reusing existing collector: %v", err)
		}
	}
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
