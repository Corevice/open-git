package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/middleware"
	mcpusecase "github.com/open-git/backend/internal/usecase/mcp"
)

type MCPVerificationHandler struct {
	runUC         *mcpusecase.RunVerificationUsecase
	getLatestUC   *mcpusecase.GetLatestVerificationUsecase
	listHistoryUC *mcpusecase.ListVerificationHistoryUsecase
	getJobUC      *mcpusecase.GetJobStatusUsecase
	deleteUC      *mcpusecase.DeleteVerificationUsecase
}

func NewMCPVerificationHandler(
	runUC *mcpusecase.RunVerificationUsecase,
	getLatestUC *mcpusecase.GetLatestVerificationUsecase,
	listHistoryUC *mcpusecase.ListVerificationHistoryUsecase,
	getJobUC *mcpusecase.GetJobStatusUsecase,
	deleteUC *mcpusecase.DeleteVerificationUsecase,
) *MCPVerificationHandler {
	return &MCPVerificationHandler{
		runUC:         runUC,
		getLatestUC:   getLatestUC,
		listHistoryUC: listHistoryUC,
		getJobUC:      getJobUC,
		deleteUC:      deleteUC,
	}
}

func (h *MCPVerificationHandler) RegisterRoutes(g *echo.Group, auth echo.MiddlewareFunc) {
	adminScope := middleware.RequireScope("admin")

	g.POST("/mcp/verification/run", h.RunVerification, auth, adminScope)
	g.GET("/mcp/verification/jobs/:job_id", h.GetJobStatus, auth)
	g.GET("/mcp/verification/latest", h.GetLatest, auth)
	g.GET("/mcp/verification/history", h.GetHistory, auth)
	g.DELETE("/mcp/verification/runs/:run_id", h.DeleteRun, auth, adminScope)
}

type runVerificationRequest struct {
	Repository string   `json:"repository"`
	Targets    []string `json:"targets"`
}

type runVerificationResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

type jobStatusResponse struct {
	JobID    string  `json:"job_id"`
	Status   string  `json:"status"`
	Progress float64 `json:"progress"`
}

type latestVerificationResponse struct {
	RunID         string          `json:"run_id"`
	Repository    string          `json:"repository"`
	OverallStatus *string         `json:"overall_status,omitempty"`
	ExecutedAt    time.Time       `json:"executed_at"`
	Checks        []checkResponse `json:"checks"`
}

type checkResponse struct {
	ID         string          `json:"id"`
	Category   string          `json:"category"`
	Status     string          `json:"status"`
	Expected   json.RawMessage `json:"expected"`
	Actual     json.RawMessage `json:"actual"`
	Error      *string         `json:"error"`
	DurationMS int             `json:"duration_ms"`
}

type historyRunResponse struct {
	RunID         string    `json:"run_id"`
	Repository    string    `json:"repository"`
	Status        string    `json:"status"`
	OverallStatus *string   `json:"overall_status,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func (h *MCPVerificationHandler) RunVerification(c echo.Context) error {
	var req runVerificationRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid request body"})
	}

	orgID, err := h.resolveOrgID(c)
	if err != nil {
		return err
	}

	run, err := h.runUC.Execute(
		c.Request().Context(),
		orgID,
		middleware.UserUUIDFromContext(c),
		mcpusecase.RunVerificationInput{
			RepositoryFullName: req.Repository,
			Targets:            req.Targets,
		},
	)
	if err != nil {
		if errors.Is(err, mcpusecase.ErrMCPRunConflict) {
			return echo.NewHTTPError(http.StatusConflict, map[string]string{"message": err.Error()})
		}
		if errors.Is(err, mcpusecase.ErrMCPPlanLimitExceeded) {
			return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": err.Error()})
		}
		if isMCPValidationError(err) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": err.Error()})
		}
		return err
	}

	return c.JSON(http.StatusAccepted, runVerificationResponse{
		JobID:  run.ID.String(),
		Status: string(run.Status),
	})
}

func (h *MCPVerificationHandler) GetJobStatus(c echo.Context) error {
	jobID, err := uuid.Parse(c.Param("job_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid job_id"})
	}

	orgID, err := h.resolveOrgID(c)
	if err != nil {
		return err
	}

	run, progress, err := h.getJobUC.Execute(c.Request().Context(), jobID, orgID)
	if err != nil {
		return err
	}
	if run == nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}

	return c.JSON(http.StatusOK, jobStatusResponse{
		JobID:    run.ID.String(),
		Status:   string(run.Status),
		Progress: progress,
	})
}

func (h *MCPVerificationHandler) GetLatest(c echo.Context) error {
	orgID, err := h.resolveOrgID(c)
	if err != nil {
		return err
	}

	run, checks, err := h.getLatestUC.Execute(c.Request().Context(), orgID)
	if err != nil {
		return err
	}
	if run == nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}

	return c.JSON(http.StatusOK, toLatestVerificationResponse(run, checks))
}

func (h *MCPVerificationHandler) GetHistory(c echo.Context) error {
	orgID, err := h.resolveOrgID(c)
	if err != nil {
		return err
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	if perPage > 100 {
		perPage = 100
	}

	runs, total, err := h.listHistoryUC.Execute(c.Request().Context(), orgID, page, perPage)
	if err != nil {
		return err
	}

	setPaginationHeaders(c, page, perPage, int(total))

	responses := make([]historyRunResponse, 0, len(runs))
	for _, run := range runs {
		responses = append(responses, toHistoryRunResponse(run))
	}
	return c.JSON(http.StatusOK, responses)
}

func (h *MCPVerificationHandler) DeleteRun(c echo.Context) error {
	runID, err := uuid.Parse(c.Param("run_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid run_id"})
	}

	orgID, err := h.resolveOrgID(c)
	if err != nil {
		return err
	}

	err = h.deleteUC.Execute(
		c.Request().Context(),
		orgID,
		middleware.UserUUIDFromContext(c),
		runID,
	)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *MCPVerificationHandler) resolveOrgID(c echo.Context) (uuid.UUID, error) {
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

func isMCPValidationError(err error) bool {
	if errors.Is(err, apperror.ErrValidation) {
		return true
	}
	return strings.Contains(err.Error(), "repository is required")
}

func toLatestVerificationResponse(run *entity.MCPVerificationRun, checks []*entity.MCPVerificationCheck) latestVerificationResponse {
	executedAt := run.CreatedAt
	if run.FinishedAt != nil {
		executedAt = *run.FinishedAt
	}

	resp := latestVerificationResponse{
		RunID:      run.ID.String(),
		Repository: run.RepositoryFullName,
		ExecutedAt: executedAt,
		Checks:     make([]checkResponse, 0, len(checks)),
	}
	if run.OverallStatus != nil {
		status := string(*run.OverallStatus)
		resp.OverallStatus = &status
	}
	for _, check := range checks {
		resp.Checks = append(resp.Checks, toCheckResponse(check))
	}
	return resp
}

func toHistoryRunResponse(run *entity.MCPVerificationRun) historyRunResponse {
	resp := historyRunResponse{
		RunID:      run.ID.String(),
		Repository: run.RepositoryFullName,
		Status:     string(run.Status),
		CreatedAt:  run.CreatedAt,
	}
	if run.OverallStatus != nil {
		status := string(*run.OverallStatus)
		resp.OverallStatus = &status
	}
	return resp
}

func toCheckResponse(check *entity.MCPVerificationCheck) checkResponse {
	return checkResponse{
		ID:         check.CheckID,
		Category:   string(check.Category),
		Status:     string(check.Status),
		Expected:   check.Expected,
		Actual:     check.Actual,
		Error:      check.Error,
		DurationMS: check.DurationMS,
	}
}
