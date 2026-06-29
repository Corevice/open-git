package middleware

import (
	"fmt"
	"net"
	"strings"

	"github.com/labstack/echo/v4"
)

// SetupProxyTrust configures Echo to extract client IPs from X-Forwarded-For
// when the request originates from a trusted proxy CIDR.
func SetupProxyTrust(e *echo.Echo, trustedCIDRs string) error {
	if strings.TrimSpace(trustedCIDRs) == "" {
		return nil
	}

	parts := strings.Split(trustedCIDRs, ",")
	trustOpts := make([]echo.TrustOption, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		_, ipNet, err := net.ParseCIDR(part)
		if err != nil {
			return fmt.Errorf("parse trusted proxy CIDR %q: %w", part, err)
		}
		trustOpts = append(trustOpts, echo.TrustIPRange(ipNet))
	}

	if len(trustOpts) > 0 {
		e.IPExtractor = echo.ExtractIPFromXFFHeader(trustOpts...)
	}

	return nil
}
