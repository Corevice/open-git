package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/middleware"
	workflowusecase "github.com/open-git/backend/internal/usecase/workflow"
)

type WorkflowDispatchHandler struct {
	triggerUC   *workflowusecase.TriggerWorkflowUsecase
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error)
}

func NewWorkflowDispatchHandler(
	triggerUC *workflowusecase.TriggerWorkflowUsecase,
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error),
) *WorkflowDispatchHandler {
	return &WorkflowDispatchHandler{
		triggerUC:   triggerUC,
		resolveRepo: resolveRepo,
	}
}

func (h *WorkflowDispatchHandler) RegisterRoutes(g *echo.Group, auth echo.MiddlewareFunc) {
	writeScope := middleware.RequireScope("write")
	g.POST("/repos/:owner/:repo/actions/workflows/:workflow_id/dispatches", h.DispatchWorkflow, auth, writeScope)
}

type dispatchWorkflowRequest struct {
	Ref    string            `json:"ref"`
	Inputs map[string]string `json:"inputs"`
}

func (h *WorkflowDispatchHandler) DispatchWorkflow(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	if _, err := strconv.ParseInt(c.Param("workflow_id"), 10, 64); err != nil {
		return RespondGitHubError(c, http.StatusUnprocessableEntity, "Invalid workflow id", nil)
	}

	var req dispatchWorkflowRequest
	if err := c.Bind(&req); err != nil {
		return RespondGitHubError(c, http.StatusUnprocessableEntity, "Invalid request body", nil)
	}

	if strings.TrimSpace(req.Ref) == "" {
		return RespondGitHubError(c, http.StatusUnprocessableEntity, "Invalid inputs", []GitHubFieldError{
			{Resource: "workflow_dispatch", Field: "ref", Code: "invalid"},
		})
	}

	actorID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	inputs := req.Inputs
	if inputs == nil {
		inputs = map[string]string{}
	}

	headBranch := strings.TrimPrefix(strings.TrimSpace(req.Ref), "refs/heads/")

	out, err := h.triggerUC.Execute(c.Request().Context(), workflowusecase.TriggerWorkflowInput{
		RepositoryID: repo.ID,
		OrgID:        repo.OrganizationID,
		ActorID:      actorID,
		Event:        "workflow_dispatch",
		HeadBranch:   headBranch,
		Ref:          req.Ref,
		Inputs:       inputs,
	})
	if err != nil {
		if errors.Is(err, domain.ErrValidation) {
			return RespondGitHubError(c, http.StatusUnprocessableEntity, err.Error(), nil)
		}
		return err
	}

	if out == nil || out.RunID == uuid.Nil {
		return RespondGitHubError(c, http.StatusUnprocessableEntity, "Workflow not found", nil)
	}

	return c.NoContent(http.StatusNoContent)
}
