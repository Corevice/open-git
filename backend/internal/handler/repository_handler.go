package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/middleware"
	repo "github.com/open-git/backend/internal/repository"
	repoUC "github.com/open-git/backend/internal/usecase/repository"
)

type RepositoryHandler struct {
	create *repoUC.CreateRepositoryUsecase
	get    *repoUC.GetRepositoryUsecase
	repos  repo.IRepositoryRepository
}

func NewRepositoryHandler(
	create *repoUC.CreateRepositoryUsecase,
	get *repoUC.GetRepositoryUsecase,
	repos repo.IRepositoryRepository,
) *RepositoryHandler {
	return &RepositoryHandler{
		create: create,
		get:    get,
		repos:  repos,
	}
}

func (h *RepositoryHandler) RegisterRoutes(g *echo.Group, authMiddleware echo.MiddlewareFunc) {
	repoScope := middleware.RequireScope("repo")
	g.POST("/user/repos", h.CreateRepository, authMiddleware, repoScope)
	g.GET("/repos/:owner/:repo", h.GetRepository, middleware.OptionalAuth())
	g.PATCH("/repos/:owner/:repo", h.UpdateVisibility, authMiddleware, repoScope)
	g.DELETE("/repos/:owner/:repo", h.DeleteRepository, authMiddleware, repoScope)
}

type createRepositoryRequest struct {
	Name        string `json:"name"`
	Private     bool   `json:"private"`
	Description string `json:"description"`
}

type updateVisibilityRequest struct {
	Private *bool `json:"private"`
}

type repositoryOwnerResponse struct {
	Login string `json:"login"`
	ID    int64  `json:"id"`
}

type repositoryResponse struct {
	ID            int64                   `json:"id"`
	Name          string                  `json:"name"`
	FullName      string                  `json:"full_name"`
	Private       bool                    `json:"private"`
	Description   string                  `json:"description"`
	DefaultBranch string                  `json:"default_branch"`
	Owner         repositoryOwnerResponse `json:"owner"`
}

func (h *RepositoryHandler) CreateRepository(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	var req createRepositoryRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "invalid request"})
	}

	repository, err := h.create.Execute(c.Request().Context(), repoUC.CreateRepositoryInput{
		OwnerID:        userID,
		OrganizationID: userID,
		Name:           req.Name,
		Private:        req.Private,
		Description:    req.Description,
	})
	if err != nil {
		if errors.Is(err, repoUC.ErrDuplicateName) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "Repository name already exists"})
		}
		if errors.Is(err, repoUC.ErrInvalidName) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": err.Error()})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to create repository"})
	}

	return c.JSON(http.StatusCreated, toRepositoryResponse(repository))
}

func (h *RepositoryHandler) GetRepository(c echo.Context) error {
	requestUserID := middleware.UserIDFromContext(c)

	repository, err := h.get.Execute(c.Request().Context(), repoUC.GetRepositoryInput{
		RequestUserID: requestUserID,
		OwnerLogin:    c.Param("owner"),
		Name:          c.Param("repo"),
	})
	if err != nil {
		if errors.Is(err, repoUC.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to get repository"})
	}

	return c.JSON(http.StatusOK, toRepositoryResponse(repository))
}

func (h *RepositoryHandler) UpdateVisibility(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	repository, err := h.resolveOwnedRepository(c, userID)
	if err != nil {
		return err
	}

	var req updateVisibilityRequest
	if err := c.Bind(&req); err != nil || req.Private == nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "invalid request"})
	}

	visibility := domain.VisibilityPublic
	if *req.Private {
		visibility = domain.VisibilityPrivate
	}

	if err := h.repos.UpdateVisibility(c.Request().Context(), repository.ID, visibility); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to update repository"})
	}

	repository.Visibility = visibility
	return c.JSON(http.StatusOK, toRepositoryResponse(repository))
}

func (h *RepositoryHandler) DeleteRepository(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	repository, err := h.resolveOwnedRepository(c, userID)
	if err != nil {
		return err
	}

	if err := h.repos.Delete(c.Request().Context(), repository.ID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to delete repository"})
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *RepositoryHandler) resolveOwnedRepository(c echo.Context, userID int64) (*domain.Repository, error) {
	repository, err := h.get.Execute(c.Request().Context(), repoUC.GetRepositoryInput{
		RequestUserID: userID,
		OwnerLogin:    c.Param("owner"),
		Name:          c.Param("repo"),
	})
	if err != nil {
		if errors.Is(err, repoUC.ErrNotFound) {
			return nil, echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return nil, echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to get repository"})
	}
	if repository.OwnerID != userID {
		return nil, echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}
	return repository, nil
}

func toRepositoryResponse(r *domain.Repository) repositoryResponse {
	ownerLogin := r.OwnerLogin
	if ownerLogin == "" {
		ownerLogin = "unknown"
	}
	return repositoryResponse{
		ID:            r.ID,
		Name:          r.Name,
		FullName:      ownerLogin + "/" + r.Name,
		Private:       r.Visibility == domain.VisibilityPrivate,
		Description:   r.Description,
		DefaultBranch: r.DefaultBranch,
		Owner: repositoryOwnerResponse{
			Login: ownerLogin,
			ID:    r.OwnerID,
		},
	}
}
