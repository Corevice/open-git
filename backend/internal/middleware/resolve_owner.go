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

type OrgMembershipLookup interface {
	GetMemberRole(ctx context.Context, orgID, userID int64) (string, error)
}

func ResolveOwner(orgs OrgByLoginLookup, memberships OrgMembershipLookup) echo.MiddlewareFunc {
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
				return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to resolve organization"})
			}
			if org == nil {
				return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
			}

			userID := UserIDFromContext(c)
			if userID != 0 && memberships != nil {
				role, err := memberships.GetMemberRole(c.Request().Context(), org.ID, userID)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to check membership"})
				}
				if role == "" {
					return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
				}
			}

			// Populate the organization scope on the request context so that
			// GetActor (defined in auth_middleware.go) reports the resolved owner.
			c.Set(organizationIDContextKey, Int64ToUUID(org.ID))
			return next(c)
		}
	}
}
