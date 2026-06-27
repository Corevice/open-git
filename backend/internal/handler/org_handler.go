package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/middleware"
	orgUC "github.com/open-git/backend/internal/usecase/org"
)

type OrgHandler struct {
	getOrg       *orgUC.GetOrgUsecase
	listUserOrgs *orgUC.ListUserOrgsUsecase
	createOrg    *orgUC.CreateOrgUsecase
}

func NewOrgHandler(
	getOrg *orgUC.GetOrgUsecase,
	listUserOrgs *orgUC.ListUserOrgsUsecase,
	createOrg *orgUC.CreateOrgUsecase,
) *OrgHandler {
	return &OrgHandler{
		getOrg:       getOrg,
		listUserOrgs: listUserOrgs,
		createOrg:    createOrg,
	}
}

type orgResponse struct {
	ID          int64  `json:"id"`
	Login       string `json:"login"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

type createOrgRequest struct {
	Login       string `json:"login"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *OrgHandler) RegisterRoutes(g *echo.Group, authMiddleware echo.MiddlewareFunc) {
	g.POST("/orgs", h.CreateOrg, authMiddleware)
	g.GET("/orgs/:org", h.GetOrg, middleware.OptionalAuth())
	g.GET("/user/orgs", h.ListUserOrgs, authMiddleware)
}

func (h *OrgHandler) CreateOrg(c echo.Context) error {
	creatorID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	var req createOrgRequest
	if err := c.Bind(&req); err != nil {
		return RespondGitHubError(c, http.StatusUnprocessableEntity, "Validation Failed", nil)
	}

	org, err := h.createOrg.Execute(c.Request().Context(), orgUC.CreateOrgInput{
		CreatorID:   creatorID,
		Login:       req.Login,
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		if errors.Is(err, orgUC.ErrDuplicateLogin) || errors.Is(err, orgUC.ErrReservedLogin) {
			return RespondGitHubError(c, http.StatusUnprocessableEntity, err.Error(), nil)
		}
		return RespondGitHubError(c, http.StatusUnprocessableEntity, err.Error(), nil)
	}

	return c.JSON(http.StatusCreated, toEntityOrgResponse(org))
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

func toEntityOrgResponse(o *entity.Organization) orgResponse {
	return orgResponse{
		ID:          middleware.UUIDToInt64(o.ID),
		Login:       o.Login,
		Name:        o.Name,
		Description: o.Description,
		Type:        "Organization",
	}
}
