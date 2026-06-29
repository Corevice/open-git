package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

type authRateLimitBucket struct {
	mu      sync.Mutex
	count   int
	resetAt time.Time
}

// AuthRateLimitMiddleware limits authentication attempts per IP within a sliding window.
func AuthRateLimitMiddleware(maxAttempts int, window time.Duration) echo.MiddlewareFunc {
	if maxAttempts < 1 {
		maxAttempts = 10
	}
	if window <= 0 {
		window = 15 * time.Minute
	}

	var buckets sync.Map

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()

			val, _ := buckets.LoadOrStore(ip, &authRateLimitBucket{
				resetAt: time.Now().Add(window),
			})
			bucket := val.(*authRateLimitBucket)

			bucket.mu.Lock()
			now := time.Now()
			if now.After(bucket.resetAt) {
				bucket.count = 0
				bucket.resetAt = now.Add(window)
			}

			bucket.count++
			if bucket.count > maxAttempts {
				retryAfter := int(time.Until(bucket.resetAt).Seconds())
				if retryAfter < 1 {
					retryAfter = 1
				}
				bucket.mu.Unlock()
				c.Response().Header().Set("Retry-After", strconv.Itoa(retryAfter))
				return echo.NewHTTPError(http.StatusTooManyRequests, map[string]string{
					"message": "Too many authentication attempts",
				})
			}
			bucket.mu.Unlock()

			return next(c)
		}
	}
}
