package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func isJWTAuth(c echo.Context) bool {
	return c.Get(scopesContextKey) == nil
}

func hasScope(c echo.Context, scope string) bool {
	for _, s := range GetScopes(c) {
		if s == scope {
			return true
		}
	}
	return false
}

func scopeForbidden(c echo.Context, accepted string) error {
	c.Response().Header().Set("X-Accepted-OAuth-Scopes", accepted)
	c.Response().Header().Set("X-OAuth-Scopes", strings.Join(GetScopes(c), ", "))
	return echo.NewHTTPError(http.StatusForbidden, githubAuthError{
		Message:          "Resource not accessible by personal access token",
		DocumentationURL: githubDocsURL,
	})
}

func RequireScope(scope string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if isJWTAuth(c) {
				return next(c)
			}
			if hasScope(c, scope) {
				return next(c)
			}
			return scopeForbidden(c, scope)
		}
	}
}

func RequireAnyScope(scopes ...string) echo.MiddlewareFunc {
	accepted := strings.Join(scopes, ", ")
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if isJWTAuth(c) {
				return next(c)
			}
			for _, scope := range scopes {
				if hasScope(c, scope) {
					return next(c)
				}
			}
			return scopeForbidden(c, accepted)
		}
	}
}
