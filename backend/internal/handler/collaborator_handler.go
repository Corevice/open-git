package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/middleware"
	repo "github.com/open-git/backend/internal/repository"
)

type CollaboratorHandler struct {
	resolver     GitRepositoryResolver
	repos        repo.IRepositoryRepository
	collaborators repo.IRepositoryCollaboratorRepository
	users        domainrepo.IUserRepository
}

func NewCollaboratorHandler(
	resolver GitRepositoryResolver,
	repos repo.IRepositoryRepository,
	collaborators repo.IRepositoryCollaboratorRepository,
	users domainrepo.IUserRepository,
) *CollaboratorHandler {
	return &CollaboratorHandler{
		resolver:     resolver,
		repos:        repos,
		collaborators: collaborators,
		users:        users,
	}
}

func (h *CollaboratorHandler) RegisterRoutes(g *echo.Group, authMiddleware echo.MiddlewareFunc) {
	g.GET("/repos/:owner/:repo/collaborators", h.List, authMiddleware)
	g.PUT("/repos/:owner/:repo/collaborators/:username", h.Add, authMiddleware)
	g.DELETE("/repos/:owner/:repo/collaborators/:username", h.Remove, authMiddleware)
	g.GET("/repos/:owner/:repo/collaborators/:username/permission", h.GetPermission, authMiddleware)
}

type collaboratorPermissionsResponse struct {
	Pull  bool `json:"pull"`
	Push  bool `json:"push"`
	Admin bool `json:"admin"`
}

type collaboratorListItemResponse struct {
	Login       string                          `json:"login"`
	Permissions collaboratorPermissionsResponse `json:"permissions"`
}

type addCollaboratorRequest struct {
	Permission string `json:"permission"`
}

type collaboratorUserResponse struct {
	Login string `json:"login"`
}

type collaboratorPermissionResponse struct {
	Permission string                   `json:"permission"`
	User       collaboratorUserResponse `json:"user"`
	RoleName   string                   `json:"role_name"`
}

func (h *CollaboratorHandler) List(c echo.Context) error {
	repository, err := h.getRepository(c)
	if err != nil {
		return err
	}
	if err := h.ensureOwner(c, repository); err != nil {
		return err
	}

	ctx := c.Request().Context()
	collaborators, err := h.collaborators.ListCollaborators(ctx, repository.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to list collaborators"})
	}

	resp := make([]collaboratorListItemResponse, 0, len(collaborators))
	for _, collab := range collaborators {
		user, err := h.users.GetByID(ctx, collab.UserID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to resolve collaborator"})
		}
		resp = append(resp, collaboratorListItemResponse{
			Login:       user.Login,
			Permissions: permissionsForCollaborator(collab.Permission),
		})
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *CollaboratorHandler) Add(c echo.Context) error {
	repository, err := h.getRepository(c)
	if err != nil {
		return err
	}
	if err := h.ensureOwner(c, repository); err != nil {
		return err
	}

	var req addCollaboratorRequest
	if err := c.Bind(&req); err != nil && !errors.Is(err, io.EOF) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "invalid request"})
	}
	if req.Permission == "" {
		req.Permission = entity.CollaboratorPermWrite
	}
	if !isValidCollaboratorPermission(req.Permission) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{"message": "invalid permission"})
	}

	ctx := c.Request().Context()
	user, err := h.users.GetByLogin(ctx, c.Param("username"))
	if errors.Is(err, domain.ErrNotFound) || user == nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to resolve user"})
	}

	if err := h.collaborators.AddCollaborator(ctx, repository.ID, user.ID, req.Permission); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to add collaborator"})
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *CollaboratorHandler) Remove(c echo.Context) error {
	repository, err := h.getRepository(c)
	if err != nil {
		return err
	}
	if err := h.ensureOwner(c, repository); err != nil {
		return err
	}

	ctx := c.Request().Context()
	user, err := h.users.GetByLogin(ctx, c.Param("username"))
	if errors.Is(err, domain.ErrNotFound) || user == nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to resolve user"})
	}

	if err := h.collaborators.RemoveCollaborator(ctx, repository.ID, user.ID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to remove collaborator"})
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *CollaboratorHandler) GetPermission(c echo.Context) error {
	repository, err := h.getRepository(c)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()
	user, err := h.users.GetByLogin(ctx, c.Param("username"))
	if errors.Is(err, domain.ErrNotFound) || user == nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to resolve user"})
	}

	permission, err := h.collaborators.GetPermission(ctx, repository.ID, user.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to get permission"})
	}
	roleName := permission
	if permission == "" {
		permission = "none"
		roleName = "none"
	}

	return c.JSON(http.StatusOK, collaboratorPermissionResponse{
		Permission: permission,
		User:       collaboratorUserResponse{Login: user.Login},
		RoleName:   roleName,
	})
}

func (h *CollaboratorHandler) getRepository(c echo.Context) (*entity.Repository, error) {
	repository, err := h.repos.GetByOwnerLoginAndName(c.Request().Context(), c.Param("owner"), c.Param("repo"))
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to get repository"})
	}
	if repository == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}
	return repository, nil
}

func (h *CollaboratorHandler) ensureOwner(c echo.Context, repository *entity.Repository) error {
	requestUserID := middleware.UserUUIDFromContext(c)
	if repository.OwnerID != requestUserID {
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "Forbidden"})
	}
	return nil
}

func isValidCollaboratorPermission(permission string) bool {
	return permission == entity.CollaboratorPermRead ||
		permission == entity.CollaboratorPermWrite ||
		permission == entity.CollaboratorPermAdmin
}

func permissionsForCollaborator(permission string) collaboratorPermissionsResponse {
	return collaboratorPermissionsResponse{
		Pull:  permission == entity.CollaboratorPermRead || permission == entity.CollaboratorPermWrite || permission == entity.CollaboratorPermAdmin,
		Push:  permission == entity.CollaboratorPermWrite || permission == entity.CollaboratorPermAdmin,
		Admin: permission == entity.CollaboratorPermAdmin,
	}
}
