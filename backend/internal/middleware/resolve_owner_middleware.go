package middleware

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
)

type OrgByLoginLookup interface {
	GetByLogin(ctx context.Context, login string) (*domain.Organization, error)
}

func ResolveOwner(orgs OrgByLoginLookup) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			login := c.Param("owner")
			if login == "" {
				login = c.Param("org")
			}
			if login == "" {
				return next(c)
			}

			org, err := orgs.GetByLogin(c.Request().Context(), login)
			if err != nil {
				return err
			}
			if org == nil {
				return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
			}

			SetActor(c, Actor{
				UserID:         UserIDFromContext(c),
				OrganizationID: Int64ToUUID(org.ID),
			})
			return next(c)
		}
	}
}
