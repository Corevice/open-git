package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
)

func GitHubCompatHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if err := next(c); err != nil {
				return err
			}
			scopes := strings.Join(GetScopes(c), ", ")
			c.Response().Header().Set("X-GitHub-Media-Type", "github.v3; format=json")
			c.Response().Header().Set("X-OAuth-Scopes", scopes)
			c.Response().Header().Set("X-GitHub-Api-Version-Selected", "2022-11-28")
			return nil
		}
	}
}
