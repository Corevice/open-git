package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

const (
	rateLimitLimitHeader     = "X-RateLimit-Limit"
	rateLimitRemainingHeader = "X-RateLimit-Remaining"
	rateLimitUsedHeader      = "X-RateLimit-Used"
	rateLimitResetHeader     = "X-RateLimit-Reset"
)

type tokenBucket struct {
	limiter *rate.Limiter
	limit   int
	mu      sync.Mutex
	resetAt time.Time
}

// RateLimitMiddleware enforces an in-memory token bucket and sets X-RateLimit-* headers.
func RateLimitMiddleware(limit int) echo.MiddlewareFunc {
	if limit < 1 {
		limit = 60
	}
	bucket := &tokenBucket{
		limiter: rate.NewLimiter(rate.Limit(limit), limit),
		limit:   limit,
		resetAt: time.Now().Add(time.Hour),
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			bucket.mu.Lock()
			if time.Now().After(bucket.resetAt) {
				bucket.resetAt = time.Now().Add(time.Hour)
				bucket.limiter = rate.NewLimiter(rate.Limit(bucket.limit), bucket.limit)
			}

			remaining := int(bucket.limiter.Tokens())
			if remaining < 0 {
				remaining = 0
			}

			if !bucket.limiter.Allow() {
				remaining = 0
				reset := bucket.resetAt.Unix()
				bucket.mu.Unlock()
				c.Response().Header().Set(rateLimitLimitHeader, strconv.Itoa(bucket.limit))
				c.Response().Header().Set(rateLimitRemainingHeader, "0")
				c.Response().Header().Set(rateLimitUsedHeader, strconv.Itoa(bucket.limit))
				c.Response().Header().Set(rateLimitResetHeader, strconv.FormatInt(reset, 10))
				return echo.NewHTTPError(http.StatusForbidden, map[string]string{
					"message": "API rate limit exceeded",
				})
			}

			remaining = int(bucket.limiter.Tokens())
			if remaining < 0 {
				remaining = 0
			}
			reset := bucket.resetAt.Unix()
			bucket.mu.Unlock()

			c.Response().Header().Set(rateLimitLimitHeader, strconv.Itoa(bucket.limit))
			c.Response().Header().Set(rateLimitRemainingHeader, strconv.Itoa(remaining))
			c.Response().Header().Set(rateLimitUsedHeader, strconv.Itoa(bucket.limit-remaining))
			c.Response().Header().Set(rateLimitResetHeader, strconv.FormatInt(reset, 10))
			return next(c)
		}
	}
}
