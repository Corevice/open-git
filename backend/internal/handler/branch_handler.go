package handler

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"

	infragit "github.com/open-git/backend/internal/infrastructure/git"
	"github.com/open-git/backend/internal/middleware"
	repo "github.com/open-git/backend/internal/repository"
)

// BranchHandler serves branch listing and ref management endpoints.
type BranchHandler struct {
	access      *RepoAccess
	resolver    GitRepositoryResolver
	repos       repo.IRepositoryRepository
	memberships GitMembershipAccess
}

func NewBranchHandler(resolver GitRepositoryResolver, repos repo.IRepositoryRepository, memberships GitMembershipAccess) *BranchHandler {
	return &BranchHandler{
		resolver:    resolver,
		repos:       repos,
		memberships: memberships,
	}
}

func (h *BranchHandler) SetAccess(a *RepoAccess) { h.access = a }

func (h *BranchHandler) RegisterRoutes(g *echo.Group, authMiddleware echo.MiddlewareFunc) {
	g.GET("/repos/:owner/:repo/branches", h.ListBranches, middleware.OptionalAuth())
	g.GET("/repos/:owner/:repo/branches/:branch", h.GetBranch, middleware.OptionalAuth())
	g.POST("/repos/:owner/:repo/git/refs", h.CreateRef, authMiddleware)
	g.DELETE("/repos/:owner/:repo/git/refs/heads/:branch", h.DeleteRef, authMiddleware)
}

type createRefRequest struct {
	Ref string `json:"ref"`
	SHA string `json:"sha"`
}

func (h *BranchHandler) ListBranches(c echo.Context) error {
	branches, err := h.fetchBranches(c)
	if err != nil {
		return err
	}
	if branches == nil {
		branches = []map[string]any{}
	}
	return c.JSON(http.StatusOK, branches)
}

func (h *BranchHandler) GetBranch(c echo.Context) error {
	branches, err := h.fetchBranches(c)
	if err != nil {
		return err
	}

	branchName := c.Param("branch")
	for _, b := range branches {
		if name, ok := b["name"].(string); ok && name == branchName {
			return c.JSON(http.StatusOK, b)
		}
	}
	return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
}

func (h *BranchHandler) fetchBranches(c echo.Context) ([]map[string]any, error) {
	resolved, err := h.resolver.Resolve(c.Request().Context(), c.Param("owner"), c.Param("repo"))
	if err != nil {
		return nil, err
	}

	if err := h.access.EnsureReadGit(c, resolved); err != nil {
		return nil, err
	}

	raw, err := infragit.GetBranches(resolved.DiskPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []map[string]any{}, nil
		}
		return nil, echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}

	out := make([]map[string]any, 0, len(raw))
	for _, b := range raw {
		out = append(out, map[string]any{
			"name": b.Name,
			"commit": map[string]string{
				"sha": b.CommitSHA,
			},
		})
	}
	return out, nil
}

func (h *BranchHandler) CreateRef(c echo.Context) error {
	var req createRefRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid request body"})
	}

	name := strings.TrimPrefix(req.Ref, "refs/heads/")
	if name == req.Ref {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "ref must start with refs/heads/"})
	}

	resolved, err := h.resolver.Resolve(c.Request().Context(), c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}
	if err := h.ensureWriteAccess(c, resolved); err != nil {
		return err
	}

	if err := infragit.CreateBranch(resolved.DiskPath, name, req.SHA); err != nil {
		if errors.Is(err, infragit.ErrRefAlreadyExists) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "reference already exists"})
		}
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"ref": req.Ref,
		"object": map[string]string{
			"type": "commit",
			"sha":  req.SHA,
		},
	})
}

func (h *BranchHandler) DeleteRef(c echo.Context) error {
	resolved, err := h.resolver.Resolve(c.Request().Context(), c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}
	if err := h.ensureWriteAccess(c, resolved); err != nil {
		return err
	}

	repoMeta, err := h.repos.GetByOwnerLoginAndName(c.Request().Context(), c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}
	if repoMeta == nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}

	branch := c.Param("branch")
	if branch == repoMeta.DefaultBranch {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "cannot delete default branch"})
	}

	if err := infragit.DeleteBranch(resolved.DiskPath, branch); err != nil {
		if errors.Is(err, infragit.ErrPathNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *BranchHandler) ensureWriteAccess(c echo.Context, resolved *ResolvedGitRepository) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}
	if resolved.OwnerID != 0 && resolved.OwnerID == userID {
		return nil
	}
	if h.memberships == nil {
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "write access required"})
	}
	ok, err := h.memberships.HasWriteAccess(c.Request().Context(), userID, resolved.OrganizationID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to check permissions"})
	}
	if !ok {
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "write access required"})
	}
	return nil
}
