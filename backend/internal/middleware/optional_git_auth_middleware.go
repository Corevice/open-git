package middleware

import (
	"github.com/labstack/echo/v4"
)

func OptionalGitAuth(tokens ITokenLookup) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			record, err := lookupBasicAuthToken(c, tokens)
			if err == nil && record != nil {
				SetAuthContext(c, record.UserID, record.Scopes)
			}
			return next(c)
		}
	}
}
