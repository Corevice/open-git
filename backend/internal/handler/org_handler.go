package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/middleware"
	orgUC "github.com/open-git/backend/internal/usecase/org"
)

type OrgHandler struct {
	getOrg       *orgUC.GetOrgUsecase
	listUserOrgs *orgUC.ListUserOrgsUsecase
}

func NewOrgHandler(
	getOrg *orgUC.GetOrgUsecase,
	listUserOrgs *orgUC.ListUserOrgsUsecase,
) *OrgHandler {
	return &OrgHandler{
		getOrg:       getOrg,
		listUserOrgs: listUserOrgs,
	}
}

type orgResponse struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
	Type  string `json:"type"`
}

func (h *OrgHandler) RegisterRoutes(g *echo.Group, authMiddleware echo.MiddlewareFunc) {
	g.GET("/orgs/:org", h.GetOrg, middleware.OptionalAuth())
	g.GET("/user/orgs", h.ListUserOrgs, authMiddleware)
}

func (h *OrgHandler) GetOrg(c echo.Context) error {
	org, err := h.getOrg.Execute(c.Request().Context(), c.Param("org"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return RespondGitHubError(c, http.StatusNotFound, "Not Found", nil)
		}
		return RespondGitHubError(c, http.StatusInternalServerError, "Internal Server Error", nil)
	}
	if org == nil {
		return RespondGitHubError(c, http.StatusNotFound, "Not Found", nil)
	}

	return RespondGitHubOK(c, toOrgResponse(org))
}

func (h *OrgHandler) ListUserOrgs(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	orgs, err := h.listUserOrgs.Execute(c.Request().Context(), userID)
	if err != nil {
		return RespondGitHubError(c, http.StatusInternalServerError, "Internal Server Error", nil)
	}

	resp := make([]orgResponse, 0, len(orgs))
	for _, org := range orgs {
		resp = append(resp, toOrgResponse(org))
	}
	return RespondGitHubOK(c, resp)
}

func toOrgResponse(o *domain.Organization) orgResponse {
	return orgResponse{
		ID:    o.ID,
		Login: o.Login,
		Name:  o.Name,
		Type:  "Organization",
	}
}
