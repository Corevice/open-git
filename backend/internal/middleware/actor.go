package middleware

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type Actor struct {
	UserID         uuid.UUID
	OrganizationID uuid.UUID
}

const actorContextKey = "actor"

func SetActor(c echo.Context, a Actor) {
	c.Set(actorContextKey, a)
}

func GetActor(c echo.Context) (*Actor, error) {
	v := c.Get(actorContextKey)
	if v == nil {
		return nil, echo.ErrUnauthorized
	}
	a, ok := v.(Actor)
	if !ok {
		return nil, echo.ErrUnauthorized
	}
	return &a, nil
}
