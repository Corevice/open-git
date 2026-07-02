package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/middleware"
	orgUC "github.com/open-git/backend/internal/usecase/org"
	securityusecase "github.com/open-git/backend/internal/usecase/security"
)

const roleSecurityManager = "security_manager"

type ListAdvisoriesExecutor interface {
	Execute(ctx context.Context, input securityusecase.ListAdvisoriesInput) (*securityusecase.ListAdvisoriesOutput, error)
}

type GetAdvisoryExecutor interface {
	Execute(ctx context.Context, input securityusecase.GetAdvisoryInput) (*entity.SecurityAdvisory, error)
}

type UpdateAdvisoryStateExecutor interface {
	Execute(ctx context.Context, input securityusecase.UpdateAdvisoryStateInput) (*entity.SecurityAdvisory, error)
}

type SecurityAdvisoryHandler struct {
	access        *RepoAccess
	getOrg        *orgUC.GetOrgUsecase
	memberships   domainrepo.IMembershipRepository
	listUC        ListAdvisoriesExecutor
	getUC         GetAdvisoryExecutor
	updateUC      UpdateAdvisoryStateExecutor
	resolveRepo   func(c echo.Context, owner, repo string) (*entity.Repository, error)
}

func NewSecurityAdvisoryHandler(
	getOrg *orgUC.GetOrgUsecase,
	memberships domainrepo.IMembershipRepository,
	listUC ListAdvisoriesExecutor,
	getUC GetAdvisoryExecutor,
	updateUC UpdateAdvisoryStateExecutor,
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error),
) *SecurityAdvisoryHandler {
	return &SecurityAdvisoryHandler{
		getOrg:      getOrg,
		memberships: memberships,
		listUC:      listUC,
		getUC:       getUC,
		updateUC:    updateUC,
		resolveRepo: resolveRepo,
	}
}

func (h *SecurityAdvisoryHandler) SetAccess(a *RepoAccess) { h.access = a }

func (h *SecurityAdvisoryHandler) RegisterRoutes(g *echo.Group, authMiddleware echo.MiddlewareFunc) {
	g.GET("/orgs/:org/security-advisories", h.ListOrgAdvisories, authMiddleware)
	g.GET("/repos/:owner/:repo/security-advisories/:ghsa_id", h.GetRepoAdvisory, authMiddleware)
	g.PATCH("/repos/:owner/:repo/security-advisories/:ghsa_id", h.PatchRepoAdvisory, authMiddleware)
}

type patchAdvisoryRequest struct {
	State           entity.AdvisoryState    `json:"state"`
	DismissedReason *entity.DismissedReason `json:"dismissed_reason"`
}

type securityAdvisoryResponse struct {
	GHSAPID          string                  `json:"ghsa_id"`
	CVEID            string                  `json:"cve_id,omitempty"`
	Severity         entity.AdvisorySeverity `json:"severity"`
	Summary          string                  `json:"summary"`
	Description      string                  `json:"description"`
	AffectedPackage  string                  `json:"affected_package"`
	AffectedVersions string                  `json:"affected_versions"`
	PatchedVersions  string                  `json:"patched_versions"`
	State            entity.AdvisoryState    `json:"state"`
	DismissedReason  *entity.DismissedReason `json:"dismissed_reason,omitempty"`
}

func toSecurityAdvisoryResponse(a *entity.SecurityAdvisory) securityAdvisoryResponse {
	return securityAdvisoryResponse{
		GHSAPID:          a.GHSAPID,
		CVEID:            a.CVEID,
		Severity:         a.Severity,
		Summary:          a.Summary,
		Description:      a.Description,
		AffectedPackage:  a.AffectedPackage,
		AffectedVersions: a.AffectedVersions,
		PatchedVersions:  a.PatchedVersions,
		State:            a.State,
		DismissedReason:  a.DismissedReason,
	}
}

func (h *SecurityAdvisoryHandler) ListOrgAdvisories(c echo.Context) error {
	org, err := h.getOrg.Execute(c.Request().Context(), c.Param("org"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to get organization"})
	}

	page, perPage, err := middleware.ParsePaginationParams(c)
	if err != nil {
		return err
	}

	orgUUID := middleware.Int64ToUUID(org.ID)
	if err := h.access.EnsureOrgMember(c, orgUUID); err != nil {
		return err
	}
	output, err := h.listUC.Execute(c.Request().Context(), securityusecase.ListAdvisoriesInput{
		OrganizationID: orgUUID,
		State:          c.QueryParam("state"),
		Severity:       c.QueryParam("severity"),
		Page:           page,
		PerPage:        perPage,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to list security advisories"})
	}

	if link := middleware.BuildLinkHeader(c.Request().URL.Path, page, perPage, output.Total); link != "" {
		c.Response().Header().Set("Link", link)
	}

	resp := make([]securityAdvisoryResponse, 0, len(output.Advisories))
	for _, advisory := range output.Advisories {
		resp = append(resp, toSecurityAdvisoryResponse(advisory))
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *SecurityAdvisoryHandler) GetRepoAdvisory(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	advisory, err := h.getUC.Execute(c.Request().Context(), securityusecase.GetAdvisoryInput{
		OrganizationID: repo.OrganizationID,
		GHSAPID:        c.Param("ghsa_id"),
	})
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to get security advisory"})
	}

	return c.JSON(http.StatusOK, toSecurityAdvisoryResponse(advisory))
}

func (h *SecurityAdvisoryHandler) PatchRepoAdvisory(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	var req patchAdvisoryRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid request body"})
	}

	userUUID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	role, err := h.memberships.GetRole(c.Request().Context(), repo.OrganizationID, userUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to check membership"})
	}
	if role != entity.RoleAdmin && role != roleSecurityManager {
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "Forbidden"})
	}

	advisory, err := h.updateUC.Execute(c.Request().Context(), securityusecase.UpdateAdvisoryStateInput{
		OrganizationID:  repo.OrganizationID,
		GHSAPID:         c.Param("ghsa_id"),
		State:           req.State,
		DismissedReason: req.DismissedReason,
	})
	if err != nil {
		if errors.Is(err, securityusecase.ErrInvalidTransition) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": err.Error()})
		}
		if errors.Is(err, apperror.ErrValidation) {
			return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": err.Error()})
		}
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to update security advisory"})
	}

	return c.JSON(http.StatusOK, toSecurityAdvisoryResponse(advisory))
}
