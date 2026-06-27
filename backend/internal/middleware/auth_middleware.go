package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
)

const (
	userIDContextKey = "user_id"
	scopesContextKey = "scopes"
)

type jwtClaims struct {
	UserID int64 `json:"sub"`
	jwt.StandardClaims
}

type patTokenLookup interface {
	FindByTokenHash(ctx context.Context, tokenHash string) (*domain.AccessToken, error)
}

func AuthMiddleware(tokens patTokenLookup) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			raw, ok := bearerToken(c.Request().Header.Get("Authorization"))
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "missing authorization token"})
			}

			if userID, scopes, ok := parseJWTAuth(raw); ok {
				c.Set(userIDContextKey, userID)
				c.Set(scopesContextKey, scopes)
				return next(c)
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

func OptionalAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			raw, ok := bearerToken(c.Request().Header.Get("Authorization"))
			if ok {
				if userID, scopes, ok := parseJWTAuth(raw); ok {
					c.Set(userIDContextKey, userID)
					c.Set(scopesContextKey, scopes)
				}
			}
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
	if userID == 0 {
		return 0, echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
	}
	return userID, nil
}

func UserIDFromContext(c echo.Context) int64 {
	v := c.Get(userIDContextKey)
	if v == nil {
		return 0
	}
	userID, ok := v.(int64)
	if !ok {
		return 0
	}
	return userID
}

func GetUserUUID(c echo.Context) (uuid.UUID, error) {
	userID, err := GetUserID(c)
	if err != nil {
		return uuid.Nil, err
	}
	return Int64ToUUID(userID), nil
}

func UserUUIDFromContext(c echo.Context) uuid.UUID {
	return Int64ToUUID(UserIDFromContext(c))
}

func Int64ToUUID(id int64) uuid.UUID {
	if id == 0 {
		return uuid.Nil
	}
	var u uuid.UUID
	binary.BigEndian.PutUint64(u[8:], uint64(id))
	return u
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

func parseJWTAuth(raw string) (int64, []string, bool) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return 0, nil, false
	}

	token, err := jwt.ParseWithClaims(raw, &jwtClaims{}, func(_ *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return 0, nil, false
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || claims.UserID == 0 {
		return 0, nil, false
	}

	return claims.UserID, nil, true
}
