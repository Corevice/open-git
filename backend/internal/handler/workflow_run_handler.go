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
	"github.com/open-git/backend/internal/middleware"
	workflowusecase "github.com/open-git/backend/internal/usecase/workflow"
)

type WorkflowRunHandler struct {
	listRunsUC  *workflowusecase.ListRunsUsecase
	getRunUC    *workflowusecase.GetRunUsecase
	cancelRunUC *workflowusecase.CancelRunUsecase
	rerunUC     *workflowusecase.RerunRunUsecase
	listJobsUC  *workflowusecase.ListJobsUsecase
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error)
}

func NewWorkflowRunHandler(
	listRunsUC *workflowusecase.ListRunsUsecase,
	getRunUC *workflowusecase.GetRunUsecase,
	cancelRunUC *workflowusecase.CancelRunUsecase,
	rerunUC *workflowusecase.RerunRunUsecase,
	listJobsUC *workflowusecase.ListJobsUsecase,
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error),
) *WorkflowRunHandler {
	return &WorkflowRunHandler{
		listRunsUC:  listRunsUC,
		getRunUC:    getRunUC,
		cancelRunUC: cancelRunUC,
		rerunUC:     rerunUC,
		listJobsUC:  listJobsUC,
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

	actor, err := middleware.GetActor(c)
	if err != nil {
		return err
	}

	page, perPage, err := middleware.ParsePaginationParams(c)
	if err != nil {
		return err
	}

	output, err := h.listRunsUC.Execute(c.Request().Context(), workflowusecase.ListRunsInput{
		OrganizationID: actor.OrganizationID,
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
		runs = append(runs, toWorkflowRunResponse(run, c.Param("owner"), c.Param("repo"), c.Request().Host))
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

	actor, err := middleware.GetActor(c)
	if err != nil {
		return err
	}

	runID, err := uuid.Parse(c.Param("run_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid run_id")
	}

	run, err := h.getRunUC.Execute(c.Request().Context(), workflowusecase.GetRunInput{
		OrganizationID: actor.OrganizationID,
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

	return c.JSON(http.StatusOK, toWorkflowRunResponse(run, c.Param("owner"), c.Param("repo"), c.Request().Host))
}

func (h *WorkflowRunHandler) CancelRun(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	actor, err := middleware.GetActor(c)
	if err != nil {
		return err
	}

	runID, err := uuid.Parse(c.Param("run_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid run_id")
	}

	err = h.cancelRunUC.Execute(c.Request().Context(), workflowusecase.CancelRunInput{
		OrganizationID: actor.OrganizationID,
		RepositoryID:   repo.ID,
		RunID:          runID,
		ActorID:        actor.UserID,
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

	actor, err := middleware.GetActor(c)
	if err != nil {
		return err
	}

	runID, err := uuid.Parse(c.Param("run_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid run_id")
	}

	_, err = h.rerunUC.Execute(c.Request().Context(), workflowusecase.RerunRunInput{
		OrganizationID: actor.OrganizationID,
		RepositoryID:   repo.ID,
		RunID:          runID,
		ActorID:        actor.UserID,
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

	actor, err := middleware.GetActor(c)
	if err != nil {
		return err
	}

	runID, err := uuid.Parse(c.Param("run_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid run_id")
	}

	jobs, err := h.listJobsUC.Execute(c.Request().Context(), workflowusecase.ListJobsInput{
		OrganizationID: actor.OrganizationID,
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

func toWorkflowRunResponse(run *entity.WorkflowRun, owner, repo, host string) workflowRunResponse {
	var conclusion *string
	if run.Conclusion != "" {
		conclusion = &run.Conclusion
	}

	return workflowRunResponse{
		ID:         middleware.UUIDToInt64(run.ID),
		NodeID:     NodeID("WorkflowRun", run.ID.String()),
		Name:       run.Workflow,
		HeadSHA:    run.HeadSHA,
		Status:     run.Status,
		Conclusion: conclusion,
		HTMLURL:    workflowRunHTMLURL(host, owner, repo, run.ID),
		CreatedAt:  run.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  run.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toWorkflowJobResponse(job *workflowusecase.WorkflowJob) workflowJobResponse {
	var conclusion *string
	if job.Conclusion != "" {
		conclusion = &job.Conclusion
	}

	resp := workflowJobResponse{
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

func workflowRunHTMLURL(host, owner, repo string, runID uuid.UUID) string {
	if host == "" {
		return "/repos/" + owner + "/" + repo + "/actions/runs/" + runID.String()
	}
	return "https://" + host + "/repos/" + owner + "/" + repo + "/actions/runs/" + runID.String()
}
