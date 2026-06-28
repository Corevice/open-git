package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/middleware"
	workflowusecase "github.com/open-git/backend/internal/usecase/workflow"
)

type WorkflowJobHandler struct {
	getJobUC    *workflowusecase.GetJobUsecase
	listStepsUC *workflowusecase.ListStepsUsecase
	logRepo     workflowusecase.JobLogRepository
	jobRepo     workflowusecase.WorkflowJobRepository
}

func NewWorkflowJobHandler(
	getJobUC *workflowusecase.GetJobUsecase,
	listStepsUC *workflowusecase.ListStepsUsecase,
	logRepo workflowusecase.JobLogRepository,
	jobRepo workflowusecase.WorkflowJobRepository,
) *WorkflowJobHandler {
	return &WorkflowJobHandler{
		getJobUC:    getJobUC,
		listStepsUC: listStepsUC,
		logRepo:     logRepo,
		jobRepo:     jobRepo,
	}
}

func (h *WorkflowJobHandler) RegisterRoutes(g *echo.Group, auth echo.MiddlewareFunc) {
	readScope := middleware.RequireScope("read")

	g.GET("/repos/:owner/:repo/actions/jobs/:job_id", h.GetJob, auth, readScope)
	g.GET("/repos/:owner/:repo/actions/jobs/:job_id/logs", h.GetLogs, auth, readScope)
	g.GET("/repos/:owner/:repo/actions/jobs/:job_id/logs/stream", h.StreamJobLogs, auth, readScope)
}

type workflowJobDetailResponse struct {
	ID          int64   `json:"id"`
	RunID       int64   `json:"run_id"`
	NodeID      string  `json:"node_id"`
	Name        string  `json:"name"`
	Status      string  `json:"status"`
	Conclusion  *string `json:"conclusion"`
	StartedAt   *string `json:"started_at"`
	CompletedAt *string `json:"completed_at"`
}

func (h *WorkflowJobHandler) GetJob(c echo.Context) error {
	actor, err := middleware.GetActor(c)
	if err != nil {
		return err
	}

	jobID, err := uuid.Parse(c.Param("job_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job_id")
	}

	job, err := h.getJobUC.Execute(c.Request().Context(), workflowusecase.GetJobInput{
		OrganizationID: actor.OrganizationID,
		JobID:          jobID,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return err
	}
	if job == nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}

	return c.JSON(http.StatusOK, toWorkflowJobDetailResponse(job))
}

func (h *WorkflowJobHandler) GetLogs(c echo.Context) error {
	jobID, err := uuid.Parse(c.Param("job_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job_id")
	}

	if err := h.ensureJobAccess(c, jobID); err != nil {
		return err
	}

	chunks, err := h.logRepo.ListChunks(c.Request().Context(), jobID, 0)
	if err != nil {
		return err
	}

	var builder strings.Builder
	for _, chunk := range chunks {
		builder.WriteString(chunk.Chunk)
	}

	return c.Blob(http.StatusOK, "text/plain; charset=utf-8", []byte(builder.String()))
}

func (h *WorkflowJobHandler) StreamJobLogs(c echo.Context) error {
	jobID, err := uuid.Parse(c.Param("job_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job_id")
	}

	if err := h.ensureJobAccess(c, jobID); err != nil {
		return err
	}

	offset, _ := strconv.ParseInt(c.QueryParam("offset"), 10, 64)

	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return echo.NewHTTPError(http.StatusInternalServerError, "streaming not supported")
	}

	chunks, err := h.logRepo.ListChunks(c.Request().Context(), jobID, offset)
	if err != nil {
		return err
	}

	for _, chunk := range chunks {
		if _, err := fmt.Fprintf(c.Response(), "data: %s\n\n", chunk.Chunk); err != nil {
			return err
		}
		flusher.Flush()
	}

	return nil
}

func (h *WorkflowJobHandler) ensureJobAccess(c echo.Context, jobID uuid.UUID) error {
	actor, err := middleware.GetActor(c)
	if err != nil {
		return err
	}

	job, err := h.jobRepo.GetByID(c.Request().Context(), actor.OrganizationID, jobID)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return err
	}
	if job == nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}
	return nil
}

func toWorkflowJobDetailResponse(job *workflowusecase.WorkflowJob) workflowJobDetailResponse {
	var conclusion *string
	if job.Conclusion != "" {
		conclusion = &job.Conclusion
	}

	resp := workflowJobDetailResponse{
		ID:     middleware.UUIDToInt64(job.ID),
		RunID:  middleware.UUIDToInt64(job.RunID),
		NodeID: NodeID("WorkflowJob", job.ID.String()),
		Name:   job.Name,
		Status: job.Status,
		Conclusion: conclusion,
	}
	if job.StartedAt != nil {
		started := job.StartedAt.UTC().Format(time.RFC3339)
		resp.StartedAt = &started
	}
	if job.CompletedAt != nil {
		completed := job.CompletedAt.UTC().Format(time.RFC3339)
		resp.CompletedAt = &completed
	}
	return resp
}
