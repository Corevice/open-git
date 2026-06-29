package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/middleware"
	orgUC "github.com/open-git/backend/internal/usecase/org"
)

type OrgHandler struct {
	getOrg       *orgUC.GetOrgUsecase
	listUserOrgs *orgUC.ListUserOrgsUsecase
	createOrg    *orgUC.CreateOrgUsecase
	updateOrg    *orgUC.UpdateOrgUsecase
	deleteOrg    *orgUC.DeleteOrgUsecase
	inviteMember *orgUC.InviteMemberUsecase
	removeMember *orgUC.RemoveMemberUsecase
	memberships  domainrepo.IMembershipRepository
	users        domainrepo.IUserRepository
}

func NewOrgHandler(
	getOrg *orgUC.GetOrgUsecase,
	listUserOrgs *orgUC.ListUserOrgsUsecase,
	createOrg *orgUC.CreateOrgUsecase,
	updateOrg *orgUC.UpdateOrgUsecase,
	deleteOrg *orgUC.DeleteOrgUsecase,
	inviteMember *orgUC.InviteMemberUsecase,
	removeMember *orgUC.RemoveMemberUsecase,
	memberships domainrepo.IMembershipRepository,
	users domainrepo.IUserRepository,
) *OrgHandler {
	return &OrgHandler{
		getOrg:       getOrg,
		listUserOrgs: listUserOrgs,
		createOrg:    createOrg,
		updateOrg:    updateOrg,
		deleteOrg:    deleteOrg,
		inviteMember: inviteMember,
		removeMember: removeMember,
		memberships:  memberships,
		users:        users,
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

type updateOrgRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type memberResponse struct {
	Login string `json:"login"`
	Role  string `json:"role"`
	ID    string `json:"id"`
}

type updateMembershipRequest struct {
	Role string `json:"role"`
}

func (h *OrgHandler) RegisterRoutes(g *echo.Group, authMiddleware echo.MiddlewareFunc) {
	g.POST("/orgs", h.CreateOrg, authMiddleware)
	g.GET("/orgs/:org", h.GetOrg, middleware.OptionalAuth())
	g.PATCH("/orgs/:org", h.UpdateOrg, authMiddleware)
	g.DELETE("/orgs/:org", h.DeleteOrg, authMiddleware)
	g.GET("/user/orgs", h.ListUserOrgs, authMiddleware)
	g.GET("/orgs/:org/members", h.ListOrgMembers, authMiddleware)
	g.PUT("/orgs/:org/memberships/:username", h.UpdateMembership, authMiddleware)
	g.DELETE("/orgs/:org/members/:username", h.RemoveMember, authMiddleware)
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

func (h *OrgHandler) UpdateOrg(c echo.Context) error {
	callerID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	org, err := h.getOrg.Execute(c.Request().Context(), c.Param("org"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return RespondGitHubError(c, http.StatusNotFound, "Not Found", nil)
		}
		return RespondGitHubError(c, http.StatusInternalServerError, "Internal Server Error", nil)
	}

	var req updateOrgRequest
	if err := c.Bind(&req); err != nil {
		return RespondGitHubError(c, http.StatusUnprocessableEntity, "Validation Failed", nil)
	}

	updated, err := h.updateOrg.Execute(c.Request().Context(), orgUC.UpdateOrgInput{
		OrgID:       middleware.Int64ToUUID(org.ID),
		CallerID:    callerID,
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return RespondGitHubError(c, http.StatusForbidden, "Forbidden", nil)
		}
		if errors.Is(err, domain.ErrNotFound) {
			return RespondGitHubError(c, http.StatusNotFound, "Not Found", nil)
		}
		return RespondGitHubError(c, http.StatusInternalServerError, "Internal Server Error", nil)
	}

	return RespondGitHubOK(c, toEntityOrgResponse(updated))
}

func (h *OrgHandler) DeleteOrg(c echo.Context) error {
	callerID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	org, err := h.getOrg.Execute(c.Request().Context(), c.Param("org"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return RespondGitHubError(c, http.StatusNotFound, "Not Found", nil)
		}
		return RespondGitHubError(c, http.StatusInternalServerError, "Internal Server Error", nil)
	}

	err = h.deleteOrg.Execute(c.Request().Context(), orgUC.DeleteOrgInput{
		OrgID:    middleware.Int64ToUUID(org.ID),
		CallerID: callerID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return RespondGitHubError(c, http.StatusForbidden, "Forbidden", nil)
		}
		if errors.Is(err, domain.ErrNotFound) {
			return RespondGitHubError(c, http.StatusNotFound, "Not Found", nil)
		}
		return RespondGitHubError(c, http.StatusInternalServerError, "Internal Server Error", nil)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *OrgHandler) ListOrgMembers(c echo.Context) error {
	callerID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	org, err := h.getOrg.Execute(c.Request().Context(), c.Param("org"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return RespondGitHubError(c, http.StatusNotFound, "Not Found", nil)
		}
		return RespondGitHubError(c, http.StatusInternalServerError, "Internal Server Error", nil)
	}

	orgID := middleware.Int64ToUUID(org.ID)
	if _, err := h.memberships.GetRole(c.Request().Context(), orgID, callerID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return RespondGitHubError(c, http.StatusNotFound, "Not Found", nil)
		}
		return RespondGitHubError(c, http.StatusInternalServerError, "Internal Server Error", nil)
	}

	memberships, err := h.memberships.ListByOrg(c.Request().Context(), orgID, 1, 100)
	if err != nil {
		return RespondGitHubError(c, http.StatusInternalServerError, "Internal Server Error", nil)
	}

	resp := make([]memberResponse, 0, len(memberships))
	for _, membership := range memberships {
		user, err := h.users.GetByID(c.Request().Context(), membership.UserID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				continue
			}
			return RespondGitHubError(c, http.StatusInternalServerError, "Internal Server Error", nil)
		}
		resp = append(resp, memberResponse{
			Login: user.Login,
			Role:  membership.Role,
			ID:    user.ID.String(),
		})
	}

	return RespondGitHubOK(c, resp)
}

func (h *OrgHandler) UpdateMembership(c echo.Context) error {
	callerID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	org, err := h.getOrg.Execute(c.Request().Context(), c.Param("org"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return RespondGitHubError(c, http.StatusNotFound, "Not Found", nil)
		}
		return RespondGitHubError(c, http.StatusInternalServerError, "Internal Server Error", nil)
	}

	targetUser, err := h.users.GetByLogin(c.Request().Context(), c.Param("username"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return RespondGitHubError(c, http.StatusNotFound, "Not Found", nil)
		}
		return RespondGitHubError(c, http.StatusInternalServerError, "Internal Server Error", nil)
	}

	var req updateMembershipRequest
	if err := c.Bind(&req); err != nil {
		return RespondGitHubError(c, http.StatusUnprocessableEntity, "Validation Failed", nil)
	}

	err = h.inviteMember.Execute(c.Request().Context(), orgUC.InviteMemberInput{
		OrgID:        middleware.Int64ToUUID(org.ID),
		CallerID:     callerID,
		TargetUserID: targetUser.ID,
		Role:         req.Role,
	})
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return RespondGitHubError(c, http.StatusForbidden, "Forbidden", nil)
		}
		if errors.Is(err, orgUC.ErrLastOwner) {
			return RespondGitHubError(c, http.StatusUnprocessableEntity, err.Error(), nil)
		}
		if errors.Is(err, domain.ErrValidation) {
			return RespondGitHubError(c, http.StatusUnprocessableEntity, err.Error(), nil)
		}
		return RespondGitHubError(c, http.StatusInternalServerError, "Internal Server Error", nil)
	}

	return RespondGitHubOK(c, memberResponse{
		Login: targetUser.Login,
		Role:  req.Role,
		ID:    targetUser.ID.String(),
	})
}

func (h *OrgHandler) RemoveMember(c echo.Context) error {
	callerID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	org, err := h.getOrg.Execute(c.Request().Context(), c.Param("org"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return RespondGitHubError(c, http.StatusNotFound, "Not Found", nil)
		}
		return RespondGitHubError(c, http.StatusInternalServerError, "Internal Server Error", nil)
	}

	targetUser, err := h.users.GetByLogin(c.Request().Context(), c.Param("username"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return RespondGitHubError(c, http.StatusNotFound, "Not Found", nil)
		}
		return RespondGitHubError(c, http.StatusInternalServerError, "Internal Server Error", nil)
	}

	err = h.removeMember.Execute(c.Request().Context(), orgUC.RemoveMemberInput{
		OrgID:        middleware.Int64ToUUID(org.ID),
		CallerID:     callerID,
		TargetUserID: targetUser.ID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return RespondGitHubError(c, http.StatusForbidden, "Forbidden", nil)
		}
		if errors.Is(err, orgUC.ErrLastOwner) {
			return RespondGitHubError(c, http.StatusUnprocessableEntity, err.Error(), nil)
		}
		if errors.Is(err, domain.ErrNotFound) {
			return RespondGitHubError(c, http.StatusNotFound, "Not Found", nil)
		}
		return RespondGitHubError(c, http.StatusInternalServerError, "Internal Server Error", nil)
	}

	return c.NoContent(http.StatusNoContent)
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
