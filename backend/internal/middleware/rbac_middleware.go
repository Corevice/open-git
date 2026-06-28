package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func RequireScope(scope string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			for _, s := range GetScopes(c) {
				if s == scope {
					return next(c)
				}
			}
			c.Response().Header().Set("X-Accepted-OAuth-Scopes", scope)
			c.Response().Header().Set("X-OAuth-Scopes", strings.Join(GetScopes(c), ", "))
			return echo.NewHTTPError(http.StatusForbidden, githubAuthError{
				Message:          "Resource not accessible by personal access token",
				DocumentationURL: githubDocsURL,
			})
		}
	}
}
