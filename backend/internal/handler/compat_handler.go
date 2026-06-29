package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/middleware"
	compatusecase "github.com/open-git/backend/internal/usecase/compat"
)

type CompatHandler struct {
	getReportUC  *compatusecase.GetReportUsecase
	triggerRunUC *compatusecase.TriggerRunUsecase
	compatRepo   domainrepo.ICompatRepository
}

func NewCompatHandler(
	getReportUC *compatusecase.GetReportUsecase,
	triggerRunUC *compatusecase.TriggerRunUsecase,
	compatRepo domainrepo.ICompatRepository,
) *CompatHandler {
	return &CompatHandler{
		getReportUC:  getReportUC,
		triggerRunUC: triggerRunUC,
		compatRepo:   compatRepo,
	}
}

func (h *CompatHandler) RegisterRoutes(g *echo.Group, auth echo.MiddlewareFunc) {
	adminScope := middleware.RequireScope("admin")
	g.GET("/internal/compat/report", h.GetReport, auth, adminScope)
	g.POST("/internal/compat/run", h.TriggerRun, auth, adminScope)
	g.GET("/internal/compat/run/:job_id", h.GetRunStatus, auth, adminScope)
}

func (h *CompatHandler) GetReport(c echo.Context) error {
	orgID, err := h.resolveOrgID(c)
	if err != nil {
		return err
	}

	report, err := h.getReportUC.Execute(c.Request().Context(), orgID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, report)
}

type triggerRunRequest struct {
	Suite  string   `json:"suite"`
	Filter []string `json:"filter"`
}

func (h *CompatHandler) TriggerRun(c echo.Context) error {
	var req triggerRunRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid request body"})
	}

	orgID, err := h.resolveOrgID(c)
	if err != nil {
		return err
	}

	triggeredBy := middleware.UserUUIDFromContext(c)
	run, err := h.triggerRunUC.Execute(
		c.Request().Context(),
		req.Suite,
		req.Filter,
		orgID,
		triggeredBy,
	)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusAccepted, map[string]string{
		"job_id": run.ID.String(),
		"status": "queued",
	})
}

func (h *CompatHandler) GetRunStatus(c echo.Context) error {
	jobID, err := uuid.Parse(c.Param("job_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid job_id"})
	}

	run, err := h.compatRepo.GetRun(c.Request().Context(), jobID)
	if err != nil {
		return err
	}
	if run == nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}

	resp := map[string]string{
		"job_id": run.ID.String(),
		"status": run.Status,
	}
	if run.Status == entity.CompatStatusCompleted || run.Status == entity.CompatStatusFailed {
		resp["report_id"] = run.ID.String()
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *CompatHandler) resolveOrgID(c echo.Context) (uuid.UUID, error) {
	if raw := c.QueryParam("organization_id"); raw != "" {
		orgID, err := uuid.Parse(raw)
		if err != nil {
			return uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid organization_id"})
		}
		return orgID, nil
	}

	userID, err := middleware.GetUserID(c)
	if err != nil {
		return uuid.Nil, err
	}
	return middleware.Int64ToUUID(userID), nil
}
