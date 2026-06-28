package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

const (
	rateLimitLimitHeader     = "X-RateLimit-Limit"
	rateLimitRemainingHeader = "X-RateLimit-Remaining"
	rateLimitResetHeader     = "X-RateLimit-Reset"
	retryAfterHeader         = "Retry-After"
	bucketEvictionInterval   = 5 * time.Minute
)

type tokenBucket struct {
	limiter *rate.Limiter
	limit   int
	mu      sync.Mutex
	resetAt time.Time
}

type rateLimitStore struct {
	mu      sync.RWMutex
	buckets map[string]*tokenBucket
}

func newRateLimitStore() *rateLimitStore {
	store := &rateLimitStore{
		buckets: make(map[string]*tokenBucket),
	}
	go store.cleanupExpiredBuckets()
	return store
}

func (s *rateLimitStore) cleanupExpiredBuckets() {
	ticker := time.NewTicker(bucketEvictionInterval)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for key, bucket := range s.buckets {
			bucket.mu.Lock()
			expired := now.After(bucket.resetAt.Add(time.Hour))
			bucket.mu.Unlock()
			if expired {
				delete(s.buckets, key)
			}
		}
		s.mu.Unlock()
	}
}

func (s *rateLimitStore) getBucket(key string, limit int) *tokenBucket {
	s.mu.RLock()
	bucket, ok := s.buckets[key]
	s.mu.RUnlock()
	if ok {
		return bucket
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if bucket, ok = s.buckets[key]; ok {
		return bucket
	}
	bucket = &tokenBucket{
		limiter: rate.NewLimiter(rate.Limit(limit), limit),
		limit:   limit,
		resetAt: time.Now().Add(time.Hour),
	}
	s.buckets[key] = bucket
	return bucket
}

func rateLimitKeyAndLimit(c echo.Context, authenticatedLimit, unauthenticatedLimit int) (string, int) {
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader != "" {
		var rawToken string
		switch {
		case strings.HasPrefix(authHeader, "token "):
			rawToken = strings.TrimPrefix(authHeader, "token ")
		case strings.HasPrefix(authHeader, "Bearer "):
			rawToken = strings.TrimPrefix(authHeader, "Bearer ")
		default:
			return "ip:" + c.RealIP(), unauthenticatedLimit
		}
		sum := sha256.Sum256([]byte(rawToken))
		return "token:" + hex.EncodeToString(sum[:]), authenticatedLimit
	}
	return "ip:" + c.RealIP(), unauthenticatedLimit
}

// RateLimitMiddleware enforces per-token or per-IP in-memory token buckets
// and sets X-RateLimit-* headers.
func RateLimitMiddleware(authenticatedLimit, unauthenticatedLimit int) echo.MiddlewareFunc {
	if authenticatedLimit < 1 {
		authenticatedLimit = 5000
	}
	if unauthenticatedLimit < 1 {
		unauthenticatedLimit = 60
	}
	store := newRateLimitStore()

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key, limit := rateLimitKeyAndLimit(c, authenticatedLimit, unauthenticatedLimit)
			bucket := store.getBucket(key, limit)

			bucket.mu.Lock()
			if time.Now().After(bucket.resetAt) {
				bucket.resetAt = time.Now().Add(time.Hour)
				bucket.limiter = rate.NewLimiter(rate.Limit(bucket.limit), bucket.limit)
			}

			if !bucket.limiter.Allow() {
				reset := bucket.resetAt.Unix()
				bucket.mu.Unlock()
				c.Response().Header().Set(rateLimitLimitHeader, strconv.Itoa(bucket.limit))
				c.Response().Header().Set(rateLimitRemainingHeader, "0")
				c.Response().Header().Set(rateLimitResetHeader, strconv.FormatInt(reset, 10))
				c.Response().Header().Set(retryAfterHeader, strconv.FormatInt(reset, 10))
				return echo.NewHTTPError(http.StatusForbidden, map[string]string{
					"message": "API rate limit exceeded",
				})
			}

			remaining := int(bucket.limiter.Tokens())
			if remaining < 0 {
				remaining = 0
			}
			reset := bucket.resetAt.Unix()
			bucket.mu.Unlock()

			c.Response().Header().Set(rateLimitLimitHeader, strconv.Itoa(bucket.limit))
			c.Response().Header().Set(rateLimitRemainingHeader, strconv.Itoa(remaining))
			c.Response().Header().Set(rateLimitResetHeader, strconv.FormatInt(reset, 10))
			return next(c)
		}
	}
}
