package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const actorContextKey = "actor"

type Actor struct {
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	Scopes         []string
}

func GetActor(c echo.Context) (*Actor, error) {
	v := c.Get(actorContextKey)
	if v == nil {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
	}
	actor, ok := v.(*Actor)
	if !ok {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
	}
	return actor, nil
}

func SetActor(c echo.Context, actor *Actor) {
	c.Set(actorContextKey, actor)
}

func UserIDFromContext(c echo.Context) int64 {
	v := c.Get(userIDContextKey)
	if v == nil {
		return 0
	}
	id, _ := v.(int64)
	return id
}

func OptionalAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(c)
		}
	}
}
