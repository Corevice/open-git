package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
)

func GitHubCompatHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			c.Response().Header().Set("X-GitHub-Media-Type", "github.v3; format=json")
			scopes := GetScopes(c)
			if len(scopes) > 0 {
				c.Response().Header().Set("X-OAuth-Scopes", strings.Join(scopes, ","))
			}
			return err
		}
	}
}
