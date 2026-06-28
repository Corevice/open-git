package middleware

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type Actor struct {
	UserID         int64
	OrganizationID uuid.UUID
}

const actorContextKey = "actor"

func SetActor(c echo.Context, a Actor) {
	c.Set(actorContextKey, a)
}

func GetActor(c echo.Context) (*Actor, error) {
	v := c.Get(actorContextKey)
	if v != nil {
		if a, ok := v.(Actor); ok {
			return &a, nil
		}
	}

	userID, err := GetUserID(c)
	if err != nil {
		return nil, err
	}

	userUUID := Int64ToUUID(userID)
	return &Actor{
		UserID:         userID,
		OrganizationID: userUUID,
	}, nil
}
