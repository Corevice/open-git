package middleware

import (
	"runtime/debug"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/logger"
)

func RequestLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)

			status := c.Response().Status
			requestID := c.Response().Header().Get(echo.HeaderXRequestID)
			latencyMS := time.Since(start).Milliseconds()

			event := logger.Get()
			switch {
			case status >= 500:
				event.Error().
					Str("request_id", requestID).
					Str("http_method", c.Request().Method).
					Str("path", c.Request().URL.Path).
					Int("status", status).
					Int64("latency_ms", latencyMS).
					Msg("request completed")
			case status >= 400:
				event.Warn().
					Str("request_id", requestID).
					Str("http_method", c.Request().Method).
					Str("path", c.Request().URL.Path).
					Int("status", status).
					Int64("latency_ms", latencyMS).
					Msg("request completed")
			default:
				event.Info().
					Str("request_id", requestID).
					Str("http_method", c.Request().Method).
					Str("path", c.Request().URL.Path).
					Int("status", status).
					Int64("latency_ms", latencyMS).
					Msg("request completed")
			}

			return err
		}
	}
}

func StructuredRecover() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					requestID := c.Response().Header().Get(echo.HeaderXRequestID)
					logger.Get().Error().
						Str("request_id", requestID).
						Str("stack_trace", string(debug.Stack())).
						Msg("panic recovered")
					_ = c.Error(echo.NewHTTPError(500, "internal server error"))
				}
			}()
			return next(c)
		}
	}
}
