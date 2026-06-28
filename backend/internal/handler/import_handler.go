package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/middleware"
	repo "github.com/open-git/backend/internal/repository"
	importUC "github.com/open-git/backend/internal/usecase/import"
)

type ImportHandler struct {
	create  *importUC.CreateImportJobUsecase
	get     *importUC.GetImportJobUsecase
	list    *importUC.ListImportJobsUsecase
	cancel  *importUC.CancelImportJobUsecase
	retry   *importUC.RetryImportJobUsecase
	orgs    repo.IOrganizationRepository
	members repo.IMembershipRepository
}

func NewImportHandler(
	create *importUC.CreateImportJobUsecase,
	get *importUC.GetImportJobUsecase,
	list *importUC.ListImportJobsUsecase,
	cancel *importUC.CancelImportJobUsecase,
	retry *importUC.RetryImportJobUsecase,
	orgs repo.IOrganizationRepository,
	members repo.IMembershipRepository,
) *ImportHandler {
	return &ImportHandler{
		create:  create,
		get:     get,
		list:    list,
		cancel:  cancel,
		retry:   retry,
		orgs:    orgs,
		members: members,
	}
}

func (h *ImportHandler) RegisterRoutes(g *echo.Group, authMiddleware echo.MiddlewareFunc) {
	imp := g.Group("/orgs/:org/imports", authMiddleware)
	imp.POST("", h.Create)
	imp.GET("", h.List)
	imp.GET("/:job_id", h.Get)
	imp.POST("/:job_id/cancel", h.Cancel)
	imp.POST("/:job_id/retry", h.Retry)
}

type createImportRequest struct {
	SourceURL   string   `json:"source_url"`
	TargetName  string   `json:"target_name"`
	Include     []string `json:"include"`
	GitHubToken string   `json:"github_token"`
}

type importJobResponse struct {
	JobID     string                 `json:"job_id"`
	Status    string                 `json:"status"`
	Phase     string                 `json:"phase,omitempty"`
	Progress  entity.ImportProgress  `json:"progress,omitempty"`
	Error     *string                `json:"error"`
	CreatedAt string                 `json:"created_at,omitempty"`
	UpdatedAt string                 `json:"updated_at,omitempty"`
}

func (h *ImportHandler) Create(c echo.Context) error {
	callerID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	orgID, err := h.resolveOrg(c)
	if err != nil {
		return err
	}

	var req createImportRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid request"})
	}

	job, err := h.create.Execute(c.Request().Context(), importUC.CreateImportJobInput{
		OrganizationID: orgID,
		CallerID:       callerID,
		SourceURL:      req.SourceURL,
		TargetName:     req.TargetName,
		Include:        req.Include,
		GitHubToken:    req.GitHubToken,
	})
	if err != nil {
		if errors.Is(err, importUC.ErrInvalidSourceURL) {
			return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": err.Error()})
		}
		if errors.Is(err, importUC.ErrForbidden) {
			return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "Forbidden"})
		}
		if errors.Is(err, importUC.ErrTargetNameConflict) {
			return echo.NewHTTPError(http.StatusConflict, map[string]string{"message": err.Error()})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to create import job"})
	}

	return c.JSON(http.StatusAccepted, map[string]string{
		"job_id": job.ID.String(),
		"status": string(job.Status),
	})
}

func (h *ImportHandler) Get(c echo.Context) error {
	orgID, err := h.resolveOrg(c)
	if err != nil {
		return err
	}
	if err := h.requireOrgReadAccess(c, orgID); err != nil {
		return err
	}

	jobID, err := uuid.Parse(c.Param("job_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid job_id"})
	}

	job, err := h.get.Execute(c.Request().Context(), importUC.GetImportJobInput{
		OrganizationID: orgID,
		JobID:          jobID,
	})
	if err != nil {
		if errors.Is(err, importUC.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to get import job"})
	}

	return c.JSON(http.StatusOK, toImportJobResponse(job))
}

func (h *ImportHandler) List(c echo.Context) error {
	orgID, err := h.resolveOrg(c)
	if err != nil {
		return err
	}
	if err := h.requireOrgReadAccess(c, orgID); err != nil {
		return err
	}

	page, perPage, err := middleware.ParsePaginationParams(c)
	if err != nil {
		return err
	}

	output, err := h.list.Execute(c.Request().Context(), importUC.ListImportJobsInput{
		OrganizationID: orgID,
		Page:           page,
		PerPage:        perPage,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to list import jobs"})
	}

	if link := middleware.BuildLinkHeader(c.Request().URL.Path, page, perPage, output.Total); link != "" {
		c.Response().Header().Set("Link", link)
	}

	resp := make([]importJobResponse, 0, len(output.Jobs))
	for _, job := range output.Jobs {
		resp = append(resp, toImportJobResponse(job))
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *ImportHandler) Cancel(c echo.Context) error {
	callerID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	orgID, err := h.resolveOrg(c)
	if err != nil {
		return err
	}

	jobID, err := uuid.Parse(c.Param("job_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid job_id"})
	}

	_, err = h.cancel.Execute(c.Request().Context(), importUC.CancelImportJobInput{
		OrganizationID: orgID,
		JobID:          jobID,
		CallerID:       callerID,
	})
	if err != nil {
		if errors.Is(err, importUC.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		if errors.Is(err, importUC.ErrForbidden) {
			return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "Forbidden"})
		}
		if errors.Is(err, importUC.ErrInvalidTransition) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": err.Error()})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to cancel import job"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "cancelled"})
}

func (h *ImportHandler) Retry(c echo.Context) error {
	callerID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	orgID, err := h.resolveOrg(c)
	if err != nil {
		return err
	}

	jobID, err := uuid.Parse(c.Param("job_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid job_id"})
	}

	_, err = h.retry.Execute(c.Request().Context(), importUC.RetryImportJobInput{
		OrganizationID: orgID,
		JobID:          jobID,
		CallerID:       callerID,
	})
	if err != nil {
		if errors.Is(err, importUC.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		if errors.Is(err, importUC.ErrForbidden) {
			return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "Forbidden"})
		}
		if errors.Is(err, importUC.ErrInvalidTransition) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": err.Error()})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to retry import job"})
	}

	return c.JSON(http.StatusAccepted, map[string]string{"status": "queued"})
}

func (h *ImportHandler) resolveOrg(c echo.Context) (uuid.UUID, error) {
	org, err := h.orgs.GetByLogin(c.Request().Context(), c.Param("org"))
	if err != nil {
		return uuid.Nil, echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to get organization"})
	}
	if org == nil {
		return uuid.Nil, echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}
	return middleware.Int64ToUUID(org.ID), nil
}

func (h *ImportHandler) requireOrgReadAccess(c echo.Context, orgID uuid.UUID) error {
	userUUID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}
	ok, err := h.members.HasReadAccess(c.Request().Context(), userUUID, orgID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to check membership"})
	}
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}
	return nil
}

func toImportJobResponse(job *entity.ImportJob) importJobResponse {
	resp := importJobResponse{
		JobID:     job.ID.String(),
		Status:    string(job.Status),
		Phase:     string(job.Phase),
		Progress:  job.Progress,
		Error:     job.Error,
		CreatedAt: job.CreatedAt.Format(time.RFC3339),
		UpdatedAt: job.UpdatedAt.Format(time.RFC3339),
	}
	if resp.Progress == nil {
		resp.Progress = entity.ImportProgress{}
	}
	return resp
}
