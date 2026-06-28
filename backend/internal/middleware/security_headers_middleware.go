package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

const (
	securityHeaderXFrameOptions           = "X-Frame-Options"
	securityHeaderXFrameOptionsValue      = "DENY"
	securityHeaderXContentTypeOptions     = "X-Content-Type-Options"
	securityHeaderXContentTypeOptionsValue = "nosniff"
	securityHeaderCSP                     = "Content-Security-Policy"
	securityHeaderCSPValue                = "default-src 'self'"
	securityHeaderHSTS                    = "Strict-Transport-Security"
	securityHeaderHSTSValue               = "max-age=31536000; includeSubDomains"
	securityHeaderReferrerPolicy          = "Referrer-Policy"
	securityHeaderReferrerPolicyValue     = "strict-origin-when-cross-origin"
	securityHeaderPermissionsPolicy       = "Permissions-Policy"
	securityHeaderPermissionsPolicyValue  = "camera=(), microphone=(), geolocation=()"
)

// SecurityHeadersMiddleware sets standard security headers on every response.
func SecurityHeadersMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Before(func() {
				h := c.Response().Header()
				setSecurityHeaderIfEmpty(h, securityHeaderXFrameOptions, securityHeaderXFrameOptionsValue)
				setSecurityHeaderIfEmpty(h, securityHeaderXContentTypeOptions, securityHeaderXContentTypeOptionsValue)
				setSecurityHeaderIfEmpty(h, securityHeaderCSP, securityHeaderCSPValue)
				setSecurityHeaderIfEmpty(h, securityHeaderHSTS, securityHeaderHSTSValue)
				setSecurityHeaderIfEmpty(h, securityHeaderReferrerPolicy, securityHeaderReferrerPolicyValue)
				setSecurityHeaderIfEmpty(h, securityHeaderPermissionsPolicy, securityHeaderPermissionsPolicyValue)
			})
			return next(c)
		}
	}
}

func setSecurityHeaderIfEmpty(h http.Header, key, value string) {
	if h.Get(key) == "" {
		h.Set(key, value)
	}
}
