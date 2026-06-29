package middleware

import (
	"io"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/logger"
)

var appLogger = slog.New(newJSONHandler(os.Stdout, slog.LevelInfo))

func newJSONHandler(w io.Writer, level slog.Level) *slog.JSONHandler {
	return slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: lowerLogLevel,
	})
}

func lowerLogLevel(_ []string, attr slog.Attr) slog.Attr {
	if attr.Key == slog.LevelKey {
		level, ok := attr.Value.Any().(slog.Level)
		if ok {
			return slog.String("level", strings.ToLower(level.String()))
		}
	}
	return attr
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func InitLogging(level string) {
	InitLoggingWithOutput(os.Stdout, level)
}

func InitLoggingWithOutput(w io.Writer, level string) {
	appLogger = slog.New(newJSONHandler(w, parseLogLevel(level)))
}

func Log() *slog.Logger {
	return appLogger
}

func RequestLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)

			status := c.Response().Status
			attrs := []any{
				"request_id", c.Response().Header().Get(echo.HeaderXRequestID),
				"http_method", c.Request().Method,
				"path", c.Request().URL.Path,
				"status", status,
				"latency_ms", time.Since(start).Milliseconds(),
			}
			if Log().Enabled(c.Request().Context(), slog.LevelDebug) {
				attrs = append(attrs, "request_headers", logger.MaskHeaders(c.Request().Header))
			}

			switch {
			case status >= 500:
				Log().Error("request completed", attrs...)
			case status >= 400:
				Log().Warn("request completed", attrs...)
			default:
				Log().Info("request completed", attrs...)
			}

			return err
		}
	}
}

func StructuredRecover() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					requestID := c.Response().Header().Get(echo.HeaderXRequestID)
					Log().Error("panic recovered",
						"request_id", requestID,
						"stack_trace", string(debug.Stack()),
					)
					err = echo.NewHTTPError(500, "internal server error")
				}
			}()
			return next(c)
		}
	}
}
