package handler

import (
	"errors"
	"net/http"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/middleware"
	repo "github.com/open-git/backend/internal/repository"
	repoUC "github.com/open-git/backend/internal/usecase/repository"
)

type RepositoryHandler struct {
	create *repoUC.CreateRepositoryUsecase
	list   *repoUC.ListRepositoriesUsecase
	get    *repoUC.GetRepositoryUsecase
	repos  repo.IRepositoryRepository
	users  repo.IUserRepository
}

func NewRepositoryHandler(
	create *repoUC.CreateRepositoryUsecase,
	list *repoUC.ListRepositoriesUsecase,
	get *repoUC.GetRepositoryUsecase,
	repos repo.IRepositoryRepository,
	users repo.IUserRepository,
) *RepositoryHandler {
	return &RepositoryHandler{
		create: create,
		list:   list,
		get:    get,
		repos:  repos,
		users:  users,
	}
}

func (h *RepositoryHandler) RegisterRoutes(g *echo.Group, authMiddleware echo.MiddlewareFunc) {
	repoScope := middleware.RequireScope("repo")
	g.POST("/user/repos", h.CreateRepository, authMiddleware, repoScope)
	g.GET("/user/repos", h.ListRepositories, authMiddleware)
	g.GET("/users/:owner/repos", h.ListOwnerRepos, middleware.OptionalAuth())
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
	ID    string `json:"id"`
}

type repositoryResponse struct {
	ID            string                  `json:"id"`
	Name          string                  `json:"name"`
	FullName      string                  `json:"full_name"`
	Private       bool                    `json:"private"`
	Description   string                  `json:"description"`
	DefaultBranch string                  `json:"default_branch"`
	Owner         repositoryOwnerResponse `json:"owner"`
}

func (h *RepositoryHandler) CreateRepository(c echo.Context) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	intUserID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	user, err := h.users.GetByID(c.Request().Context(), intUserID)
	if err != nil || user == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to resolve user"})
	}

	var req createRepositoryRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "invalid request"})
	}

	repository, err := h.create.Execute(c.Request().Context(), repoUC.CreateRepositoryInput{
		OwnerID:        userID,
		OwnerLogin:     user.Login,
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

	return c.JSON(http.StatusCreated, toRepositoryResponse(repository, user.Login))
}

func (h *RepositoryHandler) ListRepositories(c echo.Context) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	intUserID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	user, err := h.users.GetByID(c.Request().Context(), intUserID)
	if err != nil || user == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to resolve user"})
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))

	repositories, total, err := h.list.Execute(c.Request().Context(), repoUC.ListRepositoriesInput{
		RequestUserID: userID,
		OwnerLogin:    user.Login,
		Page:          page,
		PerPage:       perPage,
	})
	if err != nil {
		if errors.Is(err, repoUC.ErrOwnerNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to list repositories"})
	}

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	setPaginationHeaders(c, page, perPage, total)

	return c.JSON(http.StatusOK, toRepositoryResponses(repositories, user.Login))
}

func (h *RepositoryHandler) ListOwnerRepos(c echo.Context) error {
	requestUserID := middleware.UserUUIDFromContext(c)
	ownerLogin := c.Param("owner")

	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))

	repositories, total, err := h.list.Execute(c.Request().Context(), repoUC.ListRepositoriesInput{
		RequestUserID: requestUserID,
		OwnerLogin:    ownerLogin,
		Page:          page,
		PerPage:       perPage,
	})
	if err != nil {
		if errors.Is(err, repoUC.ErrOwnerNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to list repositories"})
	}

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	setPaginationHeaders(c, page, perPage, total)

	return c.JSON(http.StatusOK, toRepositoryResponses(repositories, ownerLogin))
}

func (h *RepositoryHandler) GetRepository(c echo.Context) error {
	requestUserID := middleware.UserUUIDFromContext(c)

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

	return c.JSON(http.StatusOK, toRepositoryResponse(repository, c.Param("owner")))
}

func (h *RepositoryHandler) UpdateVisibility(c echo.Context) error {
	userID, err := middleware.GetUserUUID(c)
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

	visibility := entity.VisibilityPublic
	if *req.Private {
		visibility = entity.VisibilityPrivate
	}

	if err := h.repos.UpdateVisibility(c.Request().Context(), repository.ID, visibility); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to update repository"})
	}

	repository.Visibility = visibility
	return c.JSON(http.StatusOK, toRepositoryResponse(repository, c.Param("owner")))
}

func (h *RepositoryHandler) DeleteRepository(c echo.Context) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	repository, err := h.resolveOwnedRepository(c, userID)
	if err != nil {
		return err
	}

	diskPath := repository.DiskPath

	if err := h.repos.Delete(c.Request().Context(), repository.ID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to delete repository"})
	}

	if diskPath != "" {
		_ = os.RemoveAll(diskPath)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *RepositoryHandler) resolveOwnedRepository(c echo.Context, userID uuid.UUID) (*entity.Repository, error) {
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

func toRepositoryResponses(repositories []*entity.Repository, ownerLogin string) []repositoryResponse {
	responses := make([]repositoryResponse, 0, len(repositories))
	for _, r := range repositories {
		responses = append(responses, toRepositoryResponse(r, ownerLogin))
	}
	return responses
}

func toRepositoryResponse(r *entity.Repository, ownerLogin string) repositoryResponse {
	if ownerLogin == "" {
		ownerLogin = "unknown"
	}
	return repositoryResponse{
		ID:            r.ID.String(),
		Name:          r.Name,
		FullName:      ownerLogin + "/" + r.Name,
		Private:       r.Visibility == entity.VisibilityPrivate,
		Description:   "",
		DefaultBranch: r.DefaultBranch,
		Owner: repositoryOwnerResponse{
			Login: ownerLogin,
			ID:    r.OwnerID.String(),
		},
	}
}
