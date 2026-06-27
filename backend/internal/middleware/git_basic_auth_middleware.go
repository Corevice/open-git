package middleware

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
)

const wwwAuthenticateHeader = `Basic realm="OpenGit"`

type ITokenLookup interface {
	FindByTokenHash(ctx context.Context, hash string) (*domain.AccessToken, error)
}

func GitBasicAuthMiddleware(tokens ITokenLookup) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			record, err := lookupBasicAuthToken(c, tokens)
			if err != nil {
				c.Response().Header().Set("WWW-Authenticate", wwwAuthenticateHeader)
				return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
			}

			SetAuthContext(c, record.UserID, record.Scopes)
			return next(c)
		}
	}
}

func lookupBasicAuthToken(c echo.Context, tokens ITokenLookup) (*domain.AccessToken, error) {
	pat, ok := basicAuthPAT(c.Request().Header.Get("Authorization"))
	if !ok {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
	}

	tokenHash := hashToken(pat)
	record, err := tokens.FindByTokenHash(c.Request().Context(), tokenHash)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
	}
	if record == nil {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
	}
	if record.RevokedAt != nil {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
	}
	if record.ExpiresAt != nil && !record.ExpiresAt.After(time.Now().UTC()) {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
	}

	return record, nil
}

func basicAuthPAT(header string) (string, bool) {
	if header == "" {
		return "", false
	}
	const prefix = "Basic "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(header[len(prefix):]))
	if err != nil {
		return "", false
	}

	_, pat, ok := strings.Cut(string(decoded), ":")
	if !ok || pat == "" {
		return "", false
	}
	return pat, true
}
