package middleware

import (
	"net/http"

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
			return echo.NewHTTPError(http.StatusForbidden, map[string]string{
				"message": "Missing scope: " + scope,
			})
		}
	}
}
