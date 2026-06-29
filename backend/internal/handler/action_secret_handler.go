package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/middleware"
)

type IActionSecretRepository interface {
	List(ctx context.Context, orgID, repoID uuid.UUID) ([]*entity.ActionSecret, error)
	GetByName(ctx context.Context, orgID, repoID uuid.UUID, name string) (*entity.ActionSecret, error)
	Upsert(ctx context.Context, secret *entity.ActionSecret) error
	Delete(ctx context.Context, orgID, repoID uuid.UUID, name string) error
}

type IAuditLogRepository interface {
	Record(ctx context.Context, orgID, actorID uuid.UUID, action, targetType string, targetID uuid.UUID, metadata map[string]any) error
}

type ActionSecretHandler struct {
	secretRepo  IActionSecretRepository
	auditRepo   IAuditLogRepository
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error)
}

func NewActionSecretHandler(
	secretRepo IActionSecretRepository,
	auditRepo IAuditLogRepository,
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error),
) *ActionSecretHandler {
	return &ActionSecretHandler{
		secretRepo:  secretRepo,
		auditRepo:   auditRepo,
		resolveRepo: resolveRepo,
	}
}

func (h *ActionSecretHandler) RegisterRoutes(g *echo.Group, auth echo.MiddlewareFunc) {
	readScope := middleware.RequireScope("read")
	adminScope := middleware.RequireScope("admin")

	g.GET("/repos/:owner/:repo/actions/secrets", h.ListSecrets, auth, readScope)
	g.PUT("/repos/:owner/:repo/actions/secrets/:name", h.PutSecret, auth, adminScope)
	g.DELETE("/repos/:owner/:repo/actions/secrets/:name", h.DeleteSecret, auth, adminScope)
}

type listSecretsResponse struct {
	TotalCount int                    `json:"total_count"`
	Secrets    []actionSecretResponse `json:"secrets"`
}

type actionSecretResponse struct {
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type putSecretRequest struct {
	EncryptedValue string `json:"encrypted_value"`
}

func (h *ActionSecretHandler) ListSecrets(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	secrets, err := h.secretRepo.List(c.Request().Context(), repo.OrganizationID, repo.ID)
	if err != nil {
		return err
	}

	responses := make([]actionSecretResponse, 0, len(secrets))
	for _, secret := range secrets {
		responses = append(responses, toActionSecretResponse(secret))
	}

	return c.JSON(http.StatusOK, listSecretsResponse{
		TotalCount: len(responses),
		Secrets:    responses,
	})
}

func (h *ActionSecretHandler) PutSecret(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	name := c.Param("name")
	secret := entity.ActionSecret{Name: name}
	if err := secret.Validate(); err != nil {
		return RespondGitHubError(c, http.StatusUnprocessableEntity, err.Error(), []GitHubFieldError{
			{Resource: "secret", Field: "name", Code: "invalid"},
		})
	}

	var req putSecretRequest
	if err := c.Bind(&req); err != nil {
		return RespondGitHubError(c, http.StatusUnprocessableEntity, "Invalid request body", nil)
	}

	actorID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()
	action := "secret.create"
	if existing, err := h.secretRepo.GetByName(ctx, repo.OrganizationID, repo.ID, name); err == nil && existing != nil {
		action = "secret.update"
	} else if err != nil && !errors.Is(err, apperror.ErrNotFound) {
		return err
	}

	now := time.Now().UTC()
	toSave := &entity.ActionSecret{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repo.ID,
		Name:           name,
		EncryptedValue: req.EncryptedValue,
		UpdatedAt:      now,
	}
	if action == "secret.create" {
		toSave.CreatedAt = now
	}

	if err := h.secretRepo.Upsert(ctx, toSave); err != nil {
		return err
	}

	if h.auditRepo != nil {
		_ = h.auditRepo.Record(ctx, repo.OrganizationID, actorID, action, "action_secret", repo.ID, map[string]any{
			"name": name,
		})
	}

	return c.NoContent(http.StatusCreated)
}

func (h *ActionSecretHandler) DeleteSecret(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	name := c.Param("name")
	actorID, err := middleware.GetUserUUID(c)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()
	if err := h.secretRepo.Delete(ctx, repo.OrganizationID, repo.ID, name); err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return err
	}

	if h.auditRepo != nil {
		_ = h.auditRepo.Record(ctx, repo.OrganizationID, actorID, "secret.delete", "action_secret", repo.ID, map[string]any{
			"name": name,
		})
	}

	return c.NoContent(http.StatusNoContent)
}

func toActionSecretResponse(secret *entity.ActionSecret) actionSecretResponse {
	return actionSecretResponse{
		Name:      secret.Name,
		CreatedAt: secret.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: secret.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
