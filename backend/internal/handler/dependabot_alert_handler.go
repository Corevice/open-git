package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/middleware"
	securityusecase "github.com/open-git/backend/internal/usecase/security"
)

type ListDependabotAlertsExecutor interface {
	Execute(ctx context.Context, input securityusecase.ListDependabotAlertsInput) (*securityusecase.ListDependabotAlertsOutput, error)
}

type UpdateDependabotAlertExecutor interface {
	Execute(ctx context.Context, input securityusecase.UpdateDependabotAlertInput) (*entity.DependabotAlert, error)
}

type DependabotAlertHandler struct {
	memberships domainrepo.IMembershipRepository
	listUC      ListDependabotAlertsExecutor
	updateUC    UpdateDependabotAlertExecutor
	alertRepo   domainrepo.IDependabotAlertRepository
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error)
}

func NewDependabotAlertHandler(
	memberships domainrepo.IMembershipRepository,
	listUC ListDependabotAlertsExecutor,
	updateUC UpdateDependabotAlertExecutor,
	alertRepo domainrepo.IDependabotAlertRepository,
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error),
) *DependabotAlertHandler {
	return &DependabotAlertHandler{
		memberships: memberships,
		listUC:      listUC,
		updateUC:    updateUC,
		alertRepo:   alertRepo,
		resolveRepo: resolveRepo,
	}
}

func (h *DependabotAlertHandler) RegisterRoutes(g *echo.Group, authMiddleware echo.MiddlewareFunc) {
	g.GET("/repos/:owner/:repo/dependabot/alerts", h.List, authMiddleware)
	g.GET("/repos/:owner/:repo/dependabot/alerts/:alert_number", h.Get, authMiddleware)
	g.PATCH("/repos/:owner/:repo/dependabot/alerts/:alert_number", h.Patch, authMiddleware)
}

type patchDependabotAlertRequest struct {
	State           entity.DependabotAlertState `json:"state"`
	DismissedReason *entity.DismissedReason     `json:"dismissed_reason"`
}

type dependabotAlertResponse struct {
	Number          int                         `json:"number"`
	ManifestPath    string                      `json:"manifest_path"`
	State           entity.DependabotAlertState `json:"state"`
	DismissedReason *entity.DismissedReason     `json:"dismissed_reason,omitempty"`
}

func toDependabotAlertResponse(a *entity.DependabotAlert) dependabotAlertResponse {
	return dependabotAlertResponse{
		Number:          a.AlertNumber,
		ManifestPath:    a.ManifestPath,
		State:           a.State,
		DismissedReason: a.DismissedReason,
	}
}

func (h *DependabotAlertHandler) List(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	page, perPage, err := middleware.ParsePaginationParams(c)
	if err != nil {
		return err
	}

	output, err := h.listUC.Execute(c.Request().Context(), securityusecase.ListDependabotAlertsInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		State:          c.QueryParam("state"),
		Page:           page,
		PerPage:        perPage,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to list dependabot alerts"})
	}

	if link := middleware.BuildLinkHeader(c.Request().URL.Path, page, perPage, output.Total); link != "" {
		c.Response().Header().Set("Link", link)
	}

	resp := make([]dependabotAlertResponse, 0, len(output.Alerts))
	for _, alert := range output.Alerts {
		resp = append(resp, toDependabotAlertResponse(alert))
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *DependabotAlertHandler) Get(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	alertNumber, err := strconv.Atoi(c.Param("alert_number"))
	if err != nil || alertNumber < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid alert number"})
	}

	alert, err := h.alertRepo.GetByAlertNumber(c.Request().Context(), repo.OrganizationID, repo.ID, alertNumber)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to get dependabot alert"})
	}
	if alert == nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}

	return c.JSON(http.StatusOK, toDependabotAlertResponse(alert))
}

func (h *DependabotAlertHandler) Patch(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	alertNumber, err := strconv.Atoi(c.Param("alert_number"))
	if err != nil || alertNumber < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid alert number"})
	}

	var req patchDependabotAlertRequest
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

	alert, err := h.updateUC.Execute(c.Request().Context(), securityusecase.UpdateDependabotAlertInput{
		OrganizationID:  repo.OrganizationID,
		RepositoryID:    repo.ID,
		AlertNumber:     alertNumber,
		State:           req.State,
		DismissedReason: req.DismissedReason,
	})
	if err != nil {
		if errors.Is(err, securityusecase.ErrInvalidDependabotTransition) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": err.Error()})
		}
		if errors.Is(err, apperror.ErrValidation) {
			return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": err.Error()})
		}
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to update dependabot alert"})
	}

	return c.JSON(http.StatusOK, toDependabotAlertResponse(alert))
}
