package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/middleware"
	repo "github.com/open-git/backend/internal/repository"
	repoUC "github.com/open-git/backend/internal/usecase/repository"
)

type RepositoryHandler struct {
	create    *repoUC.CreateRepositoryUsecase
	get       *repoUC.GetRepositoryUsecase
	listRepos *repoUC.ListRepositoriesUsecase
	repos     repo.IRepositoryRepository
	orgs      repo.IOrganizationRepository
	auditLog  repo.IAuditLogRepository
}

func NewRepositoryHandler(
	create *repoUC.CreateRepositoryUsecase,
	get *repoUC.GetRepositoryUsecase,
	listRepos *repoUC.ListRepositoriesUsecase,
	repos repo.IRepositoryRepository,
	orgs repo.IOrganizationRepository,
	auditLog repo.IAuditLogRepository,
) *RepositoryHandler {
	return &RepositoryHandler{
		create:    create,
		get:       get,
		listRepos: listRepos,
		repos:     repos,
		orgs:      orgs,
		auditLog:  auditLog,
	}
}

func (h *RepositoryHandler) RegisterRoutes(g *echo.Group, authMiddleware echo.MiddlewareFunc) {
	repoScope := middleware.RequireScope("repo")
	g.GET("/user/repos", h.List, authMiddleware)
	g.POST("/user/repos", h.CreateRepository, authMiddleware, repoScope)
	g.GET("/orgs/:org/repos", h.ListOrg, authMiddleware)
	g.POST("/orgs/:org/repos", h.CreateForOrg, authMiddleware, repoScope)
	g.GET("/repos/:owner/:repo", h.GetRepository, middleware.OptionalAuth())
	g.PATCH("/repos/:owner/:repo", h.UpdateVisibility, authMiddleware, repoScope)
	g.DELETE("/repos/:owner/:repo", h.DeleteRepository, authMiddleware, repoScope)
}

type createRepositoryRequest struct {
	Name              string `json:"name"`
	Private           bool   `json:"private"`
	Description       string `json:"description"`
	AutoInit          bool   `json:"auto_init"`
	GitignoreTemplate string `json:"gitignore_template"`
	LicenseTemplate   string `json:"license_template"`
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

func (h *RepositoryHandler) List(c echo.Context) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	page, perPage, err := middleware.ParsePaginationParams(c)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()
	repositories, err := h.listRepos.Execute(ctx, repoUC.ListRepositoriesInput{
		OwnerID:       userID,
		RequestUserID: userID,
		Page:          page,
		PerPage:       perPage,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to list repositories"})
	}

	total, err := h.repos.CountByOwner(ctx, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to count repositories"})
	}

	if link := middleware.BuildLinkHeader(c.Request().URL.Path, page, perPage, total); link != "" {
		c.Response().Header().Set("Link", link)
	}

	resp := make([]repositoryResponse, 0, len(repositories))
	for _, r := range repositories {
		resp = append(resp, toRepositoryResponse(r))
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *RepositoryHandler) ListOrg(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	page, perPage, err := middleware.ParsePaginationParams(c)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()
	org, err := h.orgs.GetByLogin(ctx, c.Param("org"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to get organization"})
	}
	if org == nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}

	role, err := h.orgs.GetMemberRole(ctx, org.ID, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to check membership"})
	}
	if role == "" {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}

	orgUUID := middleware.Int64ToUUID(org.ID)
	repositories, err := h.listRepos.Execute(ctx, repoUC.ListRepositoriesInput{
		OrganizationID: orgUUID,
		RequestUserID:  middleware.Int64ToUUID(userID),
		Page:           page,
		PerPage:        perPage,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to list repositories"})
	}

	total, err := h.repos.CountByOrg(ctx, orgUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to count repositories"})
	}

	if link := middleware.BuildLinkHeader(c.Request().URL.Path, page, perPage, total); link != "" {
		c.Response().Header().Set("Link", link)
	}

	resp := make([]repositoryResponse, 0, len(repositories))
	for _, r := range repositories {
		resp = append(resp, toRepositoryResponse(r))
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *RepositoryHandler) CreateForOrg(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}
	userUUID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()
	org, err := h.orgs.GetByLogin(ctx, c.Param("org"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to get organization"})
	}
	if org == nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}

	role, err := h.orgs.GetMemberRole(ctx, org.ID, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to check membership"})
	}
	if role != "admin" && role != "owner" {
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "Forbidden"})
	}

	var req createRepositoryRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "invalid request"})
	}

	orgUUID := middleware.Int64ToUUID(org.ID)
	repository, err := h.create.Execute(ctx, repoUC.CreateRepositoryInput{
		OwnerID:           userUUID,
		OrganizationID:    orgUUID,
		Name:              req.Name,
		Private:           req.Private,
		Description:       req.Description,
		AutoInit:          req.AutoInit,
		GitIgnoreTemplate: req.GitignoreTemplate,
		LicenseTemplate:   req.LicenseTemplate,
		OwnerLogin:        org.Login,
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

	if err := h.recordAudit(ctx, repository.OrganizationID, userUUID, "repo.create", "Repository", repository.ID, map[string]any{"name": repository.Name}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to record audit log"})
	}

	return c.JSON(http.StatusCreated, toRepositoryResponse(repository))
}

func (h *RepositoryHandler) CreateRepository(c echo.Context) error {
	userID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	var req createRepositoryRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "invalid request"})
	}

	ctx := c.Request().Context()
	repository, err := h.create.Execute(ctx, repoUC.CreateRepositoryInput{
		OwnerID:           userID,
		OrganizationID:    userID,
		Name:              req.Name,
		Private:           req.Private,
		Description:       req.Description,
		AutoInit:          req.AutoInit,
		GitIgnoreTemplate: req.GitignoreTemplate,
		LicenseTemplate:   req.LicenseTemplate,
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

	if err := h.recordAudit(ctx, repository.OrganizationID, userID, "repo.create", "Repository", repository.ID, map[string]any{"name": repository.Name}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to record audit log"})
	}

	return c.JSON(http.StatusCreated, toRepositoryResponse(repository))
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

	return c.JSON(http.StatusOK, toRepositoryResponse(repository))
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
	return c.JSON(http.StatusOK, toRepositoryResponse(repository))
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

	ctx := c.Request().Context()
	if err := h.recordAudit(ctx, repository.OrganizationID, userID, "repo.delete", "Repository", repository.ID, nil); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to record audit log"})
	}

	if err := h.repos.Delete(ctx, repository.ID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to delete repository"})
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *RepositoryHandler) recordAudit(
	ctx context.Context,
	orgID, actorID uuid.UUID,
	action, targetType string,
	targetID uuid.UUID,
	metadata map[string]any,
) error {
	if h.auditLog == nil {
		return nil
	}
	return h.auditLog.Record(ctx, orgID, actorID, action, targetType, targetID, metadata)
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

func toRepositoryResponse(r *entity.Repository) repositoryResponse {
	ownerLogin := r.OwnerLogin
	return repositoryResponse{
		ID:            r.ID.String(),
		Name:          r.Name,
		FullName:      ownerLogin + "/" + r.Name,
		Private:       r.Visibility == entity.VisibilityPrivate,
		Description:   r.Description,
		DefaultBranch: r.DefaultBranch,
		Owner: repositoryOwnerResponse{
			Login: ownerLogin,
			ID:    r.OwnerID.String(),
		},
	}
}
