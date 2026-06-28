package middleware

import (
	"github.com/labstack/echo/v4"
)

const (
	hstsHeaderValue         = "max-age=31536000; includeSubDomains"
	contentTypeOptionsValue = "nosniff"
	frameOptionsValue       = "DENY"
	xssProtectionValue      = "1; mode=block"
	referrerPolicyValue     = "strict-origin-when-cross-origin"
)

// SecurityHeaders sets standard HTTP security headers on every response.
func SecurityHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			res := c.Response().Header()
			res.Set(echo.HeaderStrictTransportSecurity, hstsHeaderValue)
			res.Set(echo.HeaderXContentTypeOptions, contentTypeOptionsValue)
			res.Set(echo.HeaderXFrameOptions, frameOptionsValue)
			res.Set(echo.HeaderXXSSProtection, xssProtectionValue)
			res.Set(echo.HeaderReferrerPolicy, referrerPolicyValue)

			return next(c)
		}
	}
}
