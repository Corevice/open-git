package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/middleware"
)

// ActionsDispatchHandler serves manual workflow dispatch:
// POST /repos/:owner/:repo/actions/workflows/:workflow/dispatches {"ref": "main"}
type ActionsDispatchHandler struct {
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error)
	// canWrite reports whether the requester may trigger CI on the repository
	// (owner, org member with write, or collaborator — same policy as push).
	canWrite func(c echo.Context, repo *entity.Repository) bool
	// resolveSHA resolves a ref/branch to a commit SHA in the repo.
	resolveSHA func(ctx context.Context, diskPath, ref string) (string, error)
	// actorLogin returns the requesting user's login for run attribution.
	actorLogin func(c echo.Context) string
	dispatch   func(ctx context.Context, repo *entity.Repository, workflowFile, branch, sha, actor string) (*entity.WorkflowRun, error)
}

func NewActionsDispatchHandler(
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error),
	canWrite func(c echo.Context, repo *entity.Repository) bool,
	resolveSHA func(ctx context.Context, diskPath, ref string) (string, error),
	actorLogin func(c echo.Context) string,
	dispatch func(ctx context.Context, repo *entity.Repository, workflowFile, branch, sha, actor string) (*entity.WorkflowRun, error),
) *ActionsDispatchHandler {
	return &ActionsDispatchHandler{
		resolveRepo: resolveRepo,
		canWrite:    canWrite,
		resolveSHA:  resolveSHA,
		actorLogin:  actorLogin,
		dispatch:    dispatch,
	}
}

func (h *ActionsDispatchHandler) RegisterRoutes(g *echo.Group, auth echo.MiddlewareFunc) {
	writeScope := middleware.RequireScope("write")
	g.POST("/repos/:owner/:repo/actions/workflows/:workflow/dispatches", h.Dispatch, auth, writeScope)
}

type workflowDispatchRequest struct {
	Ref string `json:"ref"`
}

func (h *ActionsDispatchHandler) Dispatch(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}
	if !h.canWrite(c, repo) {
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "Forbidden"})
	}

	var req workflowDispatchRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "invalid request body"})
	}
	ref := req.Ref
	if ref == "" {
		ref = repo.DefaultBranch
	}
	if ref == "" {
		ref = "main"
	}

	ctx := c.Request().Context()
	sha, err := h.resolveSHA(ctx, repo.GitPath, ref)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "unknown ref"})
	}

	run, err := h.dispatch(ctx, repo, c.Param("workflow"), ref, sha, h.actorLogin(c))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"id":         middleware.UUIDToInt64(run.ID),
		"run_number": run.RunNumber,
		"status":     run.Status,
	})
}
