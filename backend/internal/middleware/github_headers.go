package middleware

import (
	"github.com/labstack/echo/v4"
)

const (
	githubMediaTypeHeader = "X-GitHub-Media-Type"
	githubMediaTypeValue  = "github.v3; format=json"
	defaultContentType    = "application/json; charset=utf-8"
)

// GitHubCommonHeadersMiddleware sets GitHub-compatible response headers on /api/v3 routes.
func GitHubCommonHeadersMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if err := next(c); err != nil {
				return err
			}

			if c.Response().Header().Get(echo.HeaderContentType) == "" {
				c.Response().Header().Set(echo.HeaderContentType, defaultContentType)
			}
			if c.Response().Header().Get(githubMediaTypeHeader) == "" {
				c.Response().Header().Set(githubMediaTypeHeader, githubMediaTypeValue)
			}

			return nil
		}
	}
}
