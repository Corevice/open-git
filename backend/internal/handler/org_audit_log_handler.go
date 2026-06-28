package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/middleware"
	orgUC "github.com/open-git/backend/internal/usecase/org"
	securityusecase "github.com/open-git/backend/internal/usecase/security"
)

type OrgAuditLogHandler struct {
	getOrg      *orgUC.GetOrgUsecase
	memberships domainrepo.IMembershipRepository
	searchUC    *securityusecase.SearchAuditLogsUsecase
	exportUC    *securityusecase.ExportAuditLogsUsecase
}

func NewOrgAuditLogHandler(
	getOrg *orgUC.GetOrgUsecase,
	memberships domainrepo.IMembershipRepository,
	searchUC *securityusecase.SearchAuditLogsUsecase,
	exportUC *securityusecase.ExportAuditLogsUsecase,
) *OrgAuditLogHandler {
	return &OrgAuditLogHandler{
		getOrg:      getOrg,
		memberships: memberships,
		searchUC:    searchUC,
		exportUC:    exportUC,
	}
}

func (h *OrgAuditLogHandler) RegisterRoutes(g *echo.Group, authMiddleware echo.MiddlewareFunc) {
	g.GET("/orgs/:org/audit-log", h.Search, authMiddleware)
	g.GET("/orgs/:org/audit-log/export", h.Export, authMiddleware)
}

type exportAuditLogsResponse struct {
	JobID string `json:"job_id"`
}

func (h *OrgAuditLogHandler) Search(c echo.Context) error {
	orgUUID, err := h.resolveOrgAndAuthorize(c)
	if err != nil {
		return err
	}

	after, before, err := parseAuditLogDateRange(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": err.Error()})
	}

	page, perPage, err := middleware.ParsePaginationParams(c)
	if err != nil {
		return err
	}

	output, err := h.searchUC.Execute(c.Request().Context(), securityusecase.SearchAuditLogsInput{
		OrganizationID: orgUUID,
		Phrase:         c.QueryParam("phrase"),
		Action:         c.QueryParam("include"),
		After:          after,
		Before:         before,
		Page:           page,
		PerPage:        perPage,
	})
	if err != nil {
		if errors.Is(err, securityusecase.ErrDateRangeExceeded) || errors.Is(err, apperror.ErrValidation) {
			return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": err.Error()})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to search audit logs"})
	}

	if link := middleware.BuildLinkHeader(c.Request().URL.Path, page, perPage, output.Total); link != "" {
		c.Response().Header().Set("Link", link)
	}

	resp := make([]auditLogEntryResponse, 0, len(output.Logs))
	for _, log := range output.Logs {
		resp = append(resp, toAuditLogResponse(log))
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *OrgAuditLogHandler) Export(c echo.Context) error {
	orgUUID, err := h.resolveOrgAndAuthorize(c)
	if err != nil {
		return err
	}

	after, before, err := parseAuditLogDateRange(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": err.Error()})
	}

	format := c.QueryParam("format")
	if format == "" {
		format = "csv"
	}
	if format != "csv" && format != "json" {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "format must be csv or json"})
	}

	output, err := h.exportUC.Execute(c.Request().Context(), securityusecase.ExportAuditLogsInput{
		OrganizationID: orgUUID,
		ActorID:        middleware.UserUUIDFromContext(c),
		Format:         format,
		Phrase:         c.QueryParam("phrase"),
		Action:         c.QueryParam("include"),
		After:          after,
		Before:         before,
	})
	if err != nil {
		if errors.Is(err, securityusecase.ErrDateRangeExceeded) || errors.Is(err, apperror.ErrValidation) {
			return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": err.Error()})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to enqueue audit log export"})
	}

	return c.JSON(http.StatusAccepted, exportAuditLogsResponse{
		JobID: output.JobID.String(),
	})
}

func (h *OrgAuditLogHandler) resolveOrgAndAuthorize(c echo.Context) (orgUUID uuid.UUID, err error) {
	userUUID, err := middleware.GetUserUUID(c)
	if err != nil {
		return uuid.Nil, err
	}

	org, err := h.getOrg.Execute(c.Request().Context(), c.Param("org"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return uuid.Nil, echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return uuid.Nil, echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to get organization"})
	}

	orgUUID = middleware.Int64ToUUID(org.ID)
	role, err := h.memberships.GetRole(c.Request().Context(), orgUUID, userUUID)
	if err != nil {
		return uuid.Nil, echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to check membership"})
	}
	if role != entity.RoleAdmin && role != entity.RoleOwner {
		return uuid.Nil, echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "Forbidden"})
	}

	return orgUUID, nil
}

func parseAuditLogDateRange(c echo.Context) (after, before *time.Time, err error) {
	if raw := c.QueryParam("after"); raw != "" {
		parsed, parseErr := time.Parse(time.RFC3339, raw)
		if parseErr != nil {
			return nil, nil, parseErr
		}
		after = &parsed
	}
	if raw := c.QueryParam("before"); raw != "" {
		parsed, parseErr := time.Parse(time.RFC3339, raw)
		if parseErr != nil {
			return nil, nil, parseErr
		}
		before = &parsed
	}
	return after, before, nil
}
