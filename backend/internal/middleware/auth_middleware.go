package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/Corevice/open-git/backend/internal/repository"
)

const (
	userIDContextKey = "user_id"
	scopesContextKey = "scopes"
)

func AuthMiddleware(tokens repository.IAccessTokenRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			raw, ok := bearerToken(c.Request().Header.Get("Authorization"))
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "missing authorization token"})
			}

			tokenHash := hashToken(raw)
			record, err := tokens.FindByTokenHash(c.Request().Context(), tokenHash)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "invalid authorization token"})
			}
			if record == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "invalid authorization token"})
			}
			if record.RevokedAt != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "token has been revoked"})
			}
			if record.ExpiresAt != nil && !record.ExpiresAt.After(time.Now().UTC()) {
				return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "token has expired"})
			}

			c.Set(userIDContextKey, record.UserID)
			c.Set(scopesContextKey, record.Scopes)
			return next(c)
		}
	}
}

func GetUserID(c echo.Context) (int64, error) {
	v := c.Get(userIDContextKey)
	if v == nil {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
	}
	userID, ok := v.(int64)
	if !ok {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
	}
	return userID, nil
}

func GetScopes(c echo.Context) []string {
	v := c.Get(scopesContextKey)
	if v == nil {
		return nil
	}
	scopes, ok := v.([]string)
	if !ok {
		return nil
	}
	return scopes
}

func SetAuthContext(c echo.Context, userID int64, scopes []string) {
	c.Set(userIDContextKey, userID)
	c.Set(scopesContextKey, scopes)
}

func bearerToken(header string) (string, bool) {
	return BearerToken(header)
}

// BearerToken extracts the bearer token from an Authorization header value.
func BearerToken(header string) (string, bool) {
	if header == "" {
		return "", false
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	token := strings.TrimSpace(header[len(prefix):])
	if token == "" {
		return "", false
	}
	return token, true
}

func hashToken(raw string) string {
	return HashToken(raw)
}

// HashToken returns the SHA-256 hex digest of raw.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
