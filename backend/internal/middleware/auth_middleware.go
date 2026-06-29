package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/repository"
)

const (
	userIDContextKey      = "user_id"
	userUUIDContextKey    = "user_uuid"
	scopesContextKey      = "scopes"
	unauthorizedMessage   = "unauthorized"
)

func AuthMiddleware(tokens repository.IAccessTokenRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			raw, ok := bearerToken(c.Request().Header.Get("Authorization"))
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": unauthorizedMessage})
			}

			tokenHash := hashToken(raw)
			record, err := tokens.FindByTokenHash(c.Request().Context(), tokenHash)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": unauthorizedMessage})
			}
			if record == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": unauthorizedMessage})
			}
			if record.RevokedAt != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": unauthorizedMessage})
			}
			if record.ExpiresAt != nil && !record.ExpiresAt.After(time.Now().UTC()) {
				return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": unauthorizedMessage})
			}

			c.Set(userIDContextKey, record.UserID)
			c.Set(scopesContextKey, record.Scopes)
			SetUserUUID(c, record.UserUUID)
			return next(c)
		}
	}
}

func GetUserID(c echo.Context) (int64, error) {
	v := c.Get(userIDContextKey)
	if v == nil {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": unauthorizedMessage})
	}
	userID, ok := v.(int64)
	if !ok {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": unauthorizedMessage})
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

func SetUserUUID(c echo.Context, id uuid.UUID) {
	c.Set(userUUIDContextKey, id)
}

func GetUserUUID(c echo.Context) (uuid.UUID, error) {
	v := c.Get(userUUIDContextKey)
	if v == nil {
		return uuid.Nil, echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": unauthorizedMessage})
	}
	userUUID, ok := v.(uuid.UUID)
	if !ok {
		return uuid.Nil, echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": unauthorizedMessage})
	}
	return userUUID, nil
}

func bearerToken(header string) (string, bool) {
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
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
