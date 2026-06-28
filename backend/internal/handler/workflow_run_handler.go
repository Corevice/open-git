package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/middleware"
	workflowusecase "github.com/open-git/backend/internal/usecase/workflow"
)

const streamJobLogsPollInterval = 500 * time.Millisecond

// JobLogChunk is an append-only log segment indexed by offset for SSE resume.
type JobLogChunk struct {
	Offset int64
	Chunk  string
}

// IJobLogRepository provides offset-based log chunk reads for job log streaming.
type IJobLogRepository interface {
	ListByJobIDFromOffset(ctx context.Context, jobID uuid.UUID, offset int64, limit int) ([]*JobLogChunk, error)
}

type WorkflowRunHandler struct {
	listRunsUC  *workflowusecase.ListRunsUsecase
	getRunUC    *workflowusecase.GetRunUsecase
	cancelRunUC *workflowusecase.CancelRunUsecase
	rerunUC     *workflowusecase.RerunRunUsecase
	listJobsUC  *workflowusecase.ListJobsUsecase
	logRepo     IJobLogRepository
	jobRepo     domainrepo.IWorkflowJobRepository
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error)
}

func NewWorkflowRunHandler(
	listRunsUC *workflowusecase.ListRunsUsecase,
	getRunUC *workflowusecase.GetRunUsecase,
	cancelRunUC *workflowusecase.CancelRunUsecase,
	rerunUC *workflowusecase.RerunRunUsecase,
	listJobsUC *workflowusecase.ListJobsUsecase,
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error),
	logRepo IJobLogRepository,
	jobRepo domainrepo.IWorkflowJobRepository,
) *WorkflowRunHandler {
	return &WorkflowRunHandler{
		listRunsUC:  listRunsUC,
		getRunUC:    getRunUC,
		cancelRunUC: cancelRunUC,
		rerunUC:     rerunUC,
		listJobsUC:  listJobsUC,
		logRepo:     logRepo,
		jobRepo:     jobRepo,
		resolveRepo: resolveRepo,
	}
}

func (h *WorkflowRunHandler) RegisterRoutes(g *echo.Group, auth echo.MiddlewareFunc) {
	readScope := middleware.RequireScope("read")
	writeScope := middleware.RequireScope("write")

	g.GET("/repos/:owner/:repo/actions/runs", h.ListRuns, auth, readScope)
	g.GET("/repos/:owner/:repo/actions/runs/:run_id", h.GetRun, auth, readScope)
	g.POST("/repos/:owner/:repo/actions/runs/:run_id/cancel", h.CancelRun, auth, writeScope)
	g.POST("/repos/:owner/:repo/actions/runs/:run_id/rerun", h.RerunRun, auth, writeScope)
	g.GET("/repos/:owner/:repo/actions/runs/:run_id/jobs", h.ListJobs, auth, readScope)
	g.GET("/repos/:owner/:repo/actions/jobs/:job_id/logs/stream", h.StreamJobLogs, auth, readScope)
}

type listWorkflowRunsResponse struct {
	TotalCount   int                    `json:"total_count"`
	WorkflowRuns []workflowRunResponse  `json:"workflow_runs"`
}

type workflowRunResponse struct {
	ID          int64   `json:"id"`
	NodeID      string  `json:"node_id"`
	Name        string  `json:"name"`
	HeadBranch  string  `json:"head_branch"`
	HeadSHA     string  `json:"head_sha"`
	RunNumber   int     `json:"run_number"`
	Event       string  `json:"event"`
	Status      string  `json:"status"`
	Conclusion  *string `json:"conclusion"`
	WorkflowID  int64   `json:"workflow_id"`
	HTMLURL     string  `json:"html_url"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type listWorkflowJobsResponse struct {
	TotalCount int                   `json:"total_count"`
	Jobs       []workflowJobResponse `json:"jobs"`
}

type workflowJobResponse struct {
	ID         int64   `json:"id"`
	RunID      int64   `json:"run_id"`
	NodeID     string  `json:"node_id"`
	Name       string  `json:"name"`
	Status     string  `json:"status"`
	Conclusion *string `json:"conclusion"`
	StartedAt  *string `json:"started_at"`
	CompletedAt *string `json:"completed_at"`
}

func (h *WorkflowRunHandler) ListRuns(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	page, perPage, err := middleware.ParsePaginationParams(c)
	if err != nil {
		return err
	}

	output, err := h.listRunsUC.Execute(c.Request().Context(), workflowusecase.ListRunsInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		Status:         c.QueryParam("status"),
		Branch:         c.QueryParam("branch"),
		Event:          c.QueryParam("event"),
		Actor:          c.QueryParam("actor"),
		Page:           page,
		PerPage:        perPage,
	})
	if err != nil {
		return err
	}

	setPaginationHeaders(c, output.Page, output.PerPage, output.Total)

	runs := make([]workflowRunResponse, 0, len(output.Runs))
	for _, run := range output.Runs {
		runs = append(runs, toWorkflowRunResponse(run, c.Scheme(), c.Param("owner"), c.Param("repo"), c.Request().Host))
	}

	return c.JSON(http.StatusOK, listWorkflowRunsResponse{
		TotalCount:   output.Total,
		WorkflowRuns: runs,
	})
}

func (h *WorkflowRunHandler) GetRun(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	runID, err := uuid.Parse(c.Param("run_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid run_id")
	}

	run, err := h.getRunUC.Execute(c.Request().Context(), workflowusecase.GetRunInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		RunID:          runID,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return err
	}
	if run == nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}

	return c.JSON(http.StatusOK, toWorkflowRunResponse(run, c.Scheme(), c.Param("owner"), c.Param("repo"), c.Request().Host))
}

func (h *WorkflowRunHandler) CancelRun(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	actorID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	runID, err := uuid.Parse(c.Param("run_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid run_id")
	}

	err = h.cancelRunUC.Execute(c.Request().Context(), workflowusecase.CancelRunInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		RunID:          runID,
		ActorID:        actorID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return RespondGitHubError(c, http.StatusConflict, "Run cannot be cancelled", nil)
		}
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return err
	}

	return c.NoContent(http.StatusAccepted)
}

func (h *WorkflowRunHandler) RerunRun(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	actorID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	runID, err := uuid.Parse(c.Param("run_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid run_id")
	}

	_, err = h.rerunUC.Execute(c.Request().Context(), workflowusecase.RerunRunInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		RunID:          runID,
		ActorID:        actorID,
	})
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return err
	}

	return c.NoContent(http.StatusAccepted)
}

func (h *WorkflowRunHandler) ListJobs(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	runID, err := uuid.Parse(c.Param("run_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid run_id")
	}

	jobs, err := h.listJobsUC.Execute(c.Request().Context(), workflowusecase.ListJobsInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		RunID:          runID,
	})
	if err != nil {
		return err
	}

	responses := make([]workflowJobResponse, 0, len(jobs))
	for _, job := range jobs {
		responses = append(responses, toWorkflowJobResponse(job))
	}

	return c.JSON(http.StatusOK, listWorkflowJobsResponse{
		TotalCount: len(responses),
		Jobs:       responses,
	})
}

func toWorkflowRunResponse(run *entity.WorkflowRun, scheme, owner, repo, host string) workflowRunResponse {
	var conclusion *string
	if run.Conclusion != "" {
		conclusion = &run.Conclusion
	}

	return workflowRunResponse{
		ID:         middleware.UUIDToInt64(run.ID),
		NodeID:     NodeID("WorkflowRun", run.ID.String()),
		Name:       run.Workflow,
		HeadBranch: run.HeadBranch,
		HeadSHA:    run.HeadSHA,
		RunNumber:  run.RunNumber,
		Event:      run.Event,
		Status:     run.Status,
		Conclusion: conclusion,
		WorkflowID: middleware.UUIDToInt64(run.WorkflowID),
		HTMLURL:    workflowRunHTMLURL(scheme, host, owner, repo, run.ID),
		CreatedAt:  run.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  run.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func workflowRunHTMLURL(scheme, host, owner, repo string, runID uuid.UUID) string {
	if host == "" {
		return "/repos/" + owner + "/" + repo + "/actions/runs/" + runID.String()
	}
	if scheme == "" {
		scheme = "https"
	}
	return scheme + "://" + host + "/repos/" + owner + "/" + repo + "/actions/runs/" + runID.String()
}

func (h *WorkflowRunHandler) StreamJobLogs(c echo.Context) error {
	if h.logRepo == nil || h.jobRepo == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "log streaming not configured")
	}

	jobID, err := uuid.Parse(c.Param("job_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job_id")
	}

	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	job, err := h.jobRepo.GetByID(c.Request().Context(), jobID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return err
	}
	if job == nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}
	if job.OrganizationID != repo.OrganizationID || job.RepositoryID != repo.ID {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}

	offset, err := parseStreamOffset(c.QueryParam("offset"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid offset")
	}

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return echo.NewHTTPError(http.StatusInternalServerError, "streaming not supported")
	}

	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("X-Accel-Buffering", "no")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	ctx := c.Request().Context()
	currentOffset := offset

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		chunks, err := h.logRepo.ListByJobIDFromOffset(ctx, jobID, currentOffset, 100)
		if err != nil {
			return err
		}

		if len(chunks) > 0 {
			for _, chunk := range chunks {
				if err := writeStreamJobLogSSE(c.Response(), flusher, chunk.Chunk); err != nil {
					return err
				}
				if chunk.Offset >= currentOffset {
					currentOffset = chunk.Offset + 1
				}
			}
			continue
		}

		job, err = h.jobRepo.GetByID(ctx, jobID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) || errors.Is(err, apperror.ErrNotFound) {
				return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
			}
			return err
		}
		if job != nil && isTerminalStreamJobStatus(job.Status, job.Conclusion) {
			if err := writeStreamJobDoneSSE(c.Response(), flusher); err != nil {
				return err
			}
			return nil
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(streamJobLogsPollInterval):
		}
	}
}

func parseStreamOffset(raw string) (int64, error) {
	if raw == "" {
		return 0, nil
	}
	offset, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || offset < 0 {
		return 0, fmt.Errorf("invalid offset")
	}
	return offset, nil
}

func isTerminalStreamJobStatus(status, conclusion string) bool {
	switch status {
	case entity.WorkflowJobStatusCompleted,
		entity.WorkflowJobStatusFailed,
		entity.WorkflowJobStatusCancelled,
		"failure":
		return true
	}
	switch conclusion {
	case entity.WorkflowJobConclusionSuccess,
		entity.WorkflowJobConclusionFailure,
		entity.WorkflowJobConclusionCancelled:
		return true
	default:
		return false
	}
}

func writeStreamJobLogSSE(w http.ResponseWriter, flusher http.Flusher, chunk string) error {
	if _, err := fmt.Fprintf(w, "data: %s\n\n", chunk); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

func writeStreamJobDoneSSE(w http.ResponseWriter, flusher http.Flusher) error {
	if _, err := fmt.Fprint(w, "data: {\"event\":\"done\"}\n\n"); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
