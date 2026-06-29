package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const actorContextKey = "actor"

// Actor represents the authenticated principal making a request, scoped to an
// organization. It is populated by authentication middleware and consumed by
// handlers that need the acting user and organization context.
type Actor struct {
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	Scopes         []string
}

// SetActor stores the resolved actor on the request context.
func SetActor(c echo.Context, actor Actor) {
	c.Set(actorContextKey, actor)
}

// GetActor retrieves the authenticated actor from the request context. It
// returns 401 if no actor is present.
func GetActor(c echo.Context) (*Actor, error) {
	v := c.Get(actorContextKey)
	if v == nil {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
	}
	actor, ok := v.(Actor)
	if !ok {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
	}
	return &actor, nil
}

// OptionalAuth allows a request to proceed whether or not it is authenticated.
// When credentials are present, upstream auth middleware populates the context;
// otherwise the request continues unauthenticated.
func OptionalAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(c)
		}
	}
}

// UserIDFromContext returns the authenticated user's legacy integer ID, or 0 if
// the request is unauthenticated. Useful for optional-auth endpoints.
func UserIDFromContext(c echo.Context) int64 {
	v := c.Get(userIDContextKey)
	if v == nil {
		return 0
	}
	id, ok := v.(int64)
	if !ok {
		return 0
	}
	return id
}
