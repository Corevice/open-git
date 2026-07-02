package handler

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/middleware"
	secretusecase "github.com/open-git/backend/internal/usecase/secret"
)

type listRepoSecretsUseCase interface {
	Execute(ctx context.Context, orgID, repoID uuid.UUID) ([]*entity.ActionSecret, error)
}

type listOrgSecretsUseCase interface {
	Execute(ctx context.Context, orgID uuid.UUID) ([]*entity.ActionSecret, error)
}

type getActionSecretUseCase interface {
	Execute(ctx context.Context, orgID uuid.UUID, repoID *uuid.UUID, name string) (*entity.ActionSecret, error)
}

type upsertActionSecretUseCase interface {
	Execute(ctx context.Context, orgID uuid.UUID, repoID *uuid.UUID, input secretusecase.UpsertActionSecretInput) (created bool, err error)
}

type deleteActionSecretUseCase interface {
	Execute(ctx context.Context, orgID uuid.UUID, repoID *uuid.UUID, actorID uuid.UUID, name string) error
}

type getPublicKeyUseCase interface {
	Execute() (keyID, base64PublicKey string)
}

type actionSecretCryptor interface {
	DecryptSealedBox(ciphertext []byte) ([]byte, error)
}

type SecretHandler struct {
	listRepoSecretsUC listRepoSecretsUseCase
	listOrgSecretsUC  listOrgSecretsUseCase
	getSecretUC       getActionSecretUseCase
	upsertSecretUC    upsertActionSecretUseCase
	deleteSecretUC    deleteActionSecretUseCase
	getPublicKeyUC    getPublicKeyUseCase
	secretRepo        domainrepo.IActionSecretRepository
	repoRepo          domainrepo.IRepositoryRepository
	enc               actionSecretCryptor
	resolveRepo       func(c echo.Context, owner, repo string) (*entity.Repository, error)
	resolveOrg        func(c echo.Context, orgLogin string) (uuid.UUID, error)
	access            *RepoAccess
}

// SetAccess wires repository/organization authorization for secret management.
func (h *SecretHandler) SetAccess(a *RepoAccess) { h.access = a }

func NewSecretHandler(
	listRepoSecretsUC listRepoSecretsUseCase,
	listOrgSecretsUC listOrgSecretsUseCase,
	getSecretUC getActionSecretUseCase,
	upsertSecretUC upsertActionSecretUseCase,
	deleteSecretUC deleteActionSecretUseCase,
	getPublicKeyUC getPublicKeyUseCase,
	secretRepo domainrepo.IActionSecretRepository,
	repoRepo domainrepo.IRepositoryRepository,
	enc actionSecretCryptor,
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error),
	resolveOrg func(c echo.Context, orgLogin string) (uuid.UUID, error),
) *SecretHandler {
	return &SecretHandler{
		listRepoSecretsUC: listRepoSecretsUC,
		listOrgSecretsUC:  listOrgSecretsUC,
		getSecretUC:       getSecretUC,
		upsertSecretUC:    upsertSecretUC,
		deleteSecretUC:    deleteSecretUC,
		getPublicKeyUC:    getPublicKeyUC,
		secretRepo:        secretRepo,
		repoRepo:          repoRepo,
		enc:               enc,
		resolveRepo:       resolveRepo,
		resolveOrg:        resolveOrg,
	}
}

func (h *SecretHandler) RegisterRoutes(g *echo.Group, auth echo.MiddlewareFunc) {
	writeScope := middleware.RequireScope("write")
	adminScope := middleware.RequireScope("admin")

	g.GET("/repos/:owner/:repo/actions/secrets", h.ListRepoSecrets, auth, writeScope)
	g.GET("/repos/:owner/:repo/actions/secrets/public-key", h.GetRepoPublicKey, auth, writeScope)
	g.GET("/repos/:owner/:repo/actions/secrets/:secret_name", h.GetRepoSecret, auth, writeScope)
	g.PUT("/repos/:owner/:repo/actions/secrets/:secret_name", h.UpsertRepoSecret, auth, adminScope)
	g.DELETE("/repos/:owner/:repo/actions/secrets/:secret_name", h.DeleteRepoSecret, auth, adminScope)

	g.GET("/orgs/:org/actions/secrets", h.ListOrgSecrets, auth, writeScope)
	g.GET("/orgs/:org/actions/secrets/public-key", h.GetOrgPublicKey, auth, writeScope)
	g.GET("/orgs/:org/actions/secrets/:secret_name", h.GetOrgSecret, auth, writeScope)
	g.PUT("/orgs/:org/actions/secrets/:secret_name", h.UpsertOrgSecret, auth, adminScope)
	g.DELETE("/orgs/:org/actions/secrets/:secret_name", h.DeleteOrgSecret, auth, adminScope)
	g.GET("/orgs/:org/actions/secrets/:secret_name/repositories", h.GetOrgSecretRepos, auth, writeScope)
	g.PUT("/orgs/:org/actions/secrets/:secret_name/repositories", h.SetOrgSecretRepos, auth, adminScope)
}

type secretListResponse struct {
	TotalCount int              `json:"total_count"`
	Secrets    []secretResponse `json:"secrets"`
}

type secretResponse struct {
	Name       string `json:"name"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Visibility string `json:"visibility,omitempty"`
}

type publicKeyResponse struct {
	KeyID string `json:"key_id"`
	Key   string `json:"key"`
}

type upsertRepoSecretRequest struct {
	EncryptedValue string `json:"encrypted_value"`
	KeyID          string `json:"key_id"`
}

type upsertOrgSecretRequest struct {
	EncryptedValue        string   `json:"encrypted_value"`
	KeyID                 string   `json:"key_id"`
	Visibility            string   `json:"visibility"`
	SelectedRepositoryIDs []int64  `json:"selected_repository_ids"`
}

type setOrgSecretReposRequest struct {
	SelectedRepositoryIDs []int64 `json:"selected_repository_ids"`
}

type orgSecretReposResponse struct {
	TotalCount   int                    `json:"total_count"`
	Repositories []orgSecretRepoSummary `json:"repositories"`
}

type orgSecretRepoSummary struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func (h *SecretHandler) ListRepoSecrets(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return mapSecretErr(c, err)
	}
	if err := h.access.EnsureAdmin(c, repo); err != nil {
		return mapSecretErr(c, err)
	}

	secrets, err := h.listRepoSecretsUC.Execute(c.Request().Context(), repo.OrganizationID, repo.ID)
	if err != nil {
		return mapSecretErr(c, err)
	}

	return c.JSON(http.StatusOK, toSecretListResponse(secrets, false))
}

func (h *SecretHandler) ListOrgSecrets(c echo.Context) error {
	orgID, err := h.resolveOrg(c, c.Param("org"))
	if err != nil {
		return mapSecretErr(c, err)
	}
	if err := h.access.EnsureOrgAdmin(c, orgID); err != nil {
		return mapSecretErr(c, err)
	}

	secrets, err := h.listOrgSecretsUC.Execute(c.Request().Context(), orgID)
	if err != nil {
		return mapSecretErr(c, err)
	}

	return c.JSON(http.StatusOK, toSecretListResponse(secrets, true))
}

func (h *SecretHandler) GetRepoSecret(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return mapSecretErr(c, err)
	}
	if err := h.access.EnsureAdmin(c, repo); err != nil {
		return mapSecretErr(c, err)
	}

	secret, err := h.getSecretUC.Execute(
		c.Request().Context(),
		repo.OrganizationID,
		&repo.ID,
		c.Param("secret_name"),
	)
	if err != nil {
		return mapSecretErr(c, err)
	}

	return c.JSON(http.StatusOK, toSecretResponse(secret, false))
}

func (h *SecretHandler) GetOrgSecret(c echo.Context) error {
	orgID, err := h.resolveOrg(c, c.Param("org"))
	if err != nil {
		return mapSecretErr(c, err)
	}
	if err := h.access.EnsureOrgAdmin(c, orgID); err != nil {
		return mapSecretErr(c, err)
	}

	secret, err := h.getSecretUC.Execute(c.Request().Context(), orgID, nil, c.Param("secret_name"))
	if err != nil {
		return mapSecretErr(c, err)
	}

	return c.JSON(http.StatusOK, toSecretResponse(secret, true))
}

func (h *SecretHandler) GetRepoPublicKey(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return mapSecretErr(c, err)
	}
	if err := h.access.EnsureAdmin(c, repo); err != nil {
		return mapSecretErr(c, err)
	}

	keyID, key := h.getPublicKeyUC.Execute()
	return c.JSON(http.StatusOK, publicKeyResponse{KeyID: keyID, Key: key})
}

func (h *SecretHandler) GetOrgPublicKey(c echo.Context) error {
	orgID, err := h.resolveOrg(c, c.Param("org"))
	if err != nil {
		return mapSecretErr(c, err)
	}
	if err := h.access.EnsureOrgAdmin(c, orgID); err != nil {
		return mapSecretErr(c, err)
	}

	keyID, key := h.getPublicKeyUC.Execute()
	return c.JSON(http.StatusOK, publicKeyResponse{KeyID: keyID, Key: key})
}

func (h *SecretHandler) UpsertRepoSecret(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return mapSecretErr(c, err)
	}
	if err := h.access.EnsureAdmin(c, repo); err != nil {
		return mapSecretErr(c, err)
	}

	var req upsertRepoSecretRequest
	if err := c.Bind(&req); err != nil {
		return RespondGitHubError(c, http.StatusUnprocessableEntity, "Invalid request body", nil)
	}

	plaintext, err := h.decryptEncryptedValue(req.EncryptedValue)
	if err != nil {
		return mapSecretErr(c, err)
	}

	created, err := h.upsertSecretUC.Execute(
		c.Request().Context(),
		repo.OrganizationID,
		&repo.ID,
		secretusecase.UpsertActionSecretInput{
			ActorID:        middleware.UserUUIDFromContext(c),
			Name:           c.Param("secret_name"),
			PlaintextValue: plaintext,
		},
	)
	if err != nil {
		return mapSecretErr(c, err)
	}

	if created {
		return c.NoContent(http.StatusCreated)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *SecretHandler) UpsertOrgSecret(c echo.Context) error {
	orgID, err := h.resolveOrg(c, c.Param("org"))
	if err != nil {
		return mapSecretErr(c, err)
	}
	if err := h.access.EnsureOrgAdmin(c, orgID); err != nil {
		return mapSecretErr(c, err)
	}

	var req upsertOrgSecretRequest
	if err := c.Bind(&req); err != nil {
		return RespondGitHubError(c, http.StatusUnprocessableEntity, "Invalid request body", nil)
	}

	plaintext, err := h.decryptEncryptedValue(req.EncryptedValue)
	if err != nil {
		return mapSecretErr(c, err)
	}

	selectedRepoIDs, err := int64sToUUIDs(req.SelectedRepositoryIDs)
	if err != nil {
		return mapSecretErr(c, err)
	}

	visibility := secretusecase.SecretVisibility(req.Visibility)
	if visibility == "" {
		visibility = secretusecase.VisibilityPrivate
	}

	created, err := h.upsertSecretUC.Execute(
		c.Request().Context(),
		orgID,
		nil,
		secretusecase.UpsertActionSecretInput{
			ActorID:         middleware.UserUUIDFromContext(c),
			Name:            c.Param("secret_name"),
			PlaintextValue:  plaintext,
			Visibility:      visibility,
			SelectedRepoIDs: selectedRepoIDs,
		},
	)
	if err != nil {
		return mapSecretErr(c, err)
	}

	if created {
		return c.NoContent(http.StatusCreated)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *SecretHandler) DeleteRepoSecret(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return mapSecretErr(c, err)
	}
	if err := h.access.EnsureAdmin(c, repo); err != nil {
		return mapSecretErr(c, err)
	}

	err = h.deleteSecretUC.Execute(
		c.Request().Context(),
		repo.OrganizationID,
		&repo.ID,
		middleware.UserUUIDFromContext(c),
		c.Param("secret_name"),
	)
	if err != nil {
		return mapSecretErr(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *SecretHandler) DeleteOrgSecret(c echo.Context) error {
	orgID, err := h.resolveOrg(c, c.Param("org"))
	if err != nil {
		return mapSecretErr(c, err)
	}
	if err := h.access.EnsureOrgAdmin(c, orgID); err != nil {
		return mapSecretErr(c, err)
	}

	err = h.deleteSecretUC.Execute(
		c.Request().Context(),
		orgID,
		nil,
		middleware.UserUUIDFromContext(c),
		c.Param("secret_name"),
	)
	if err != nil {
		return mapSecretErr(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *SecretHandler) GetOrgSecretRepos(c echo.Context) error {
	orgID, err := h.resolveOrg(c, c.Param("org"))
	if err != nil {
		return mapSecretErr(c, err)
	}
	if err := h.access.EnsureOrgAdmin(c, orgID); err != nil {
		return mapSecretErr(c, err)
	}

	secret, err := h.getSecretUC.Execute(c.Request().Context(), orgID, nil, c.Param("secret_name"))
	if err != nil {
		return mapSecretErr(c, err)
	}

	repoIDs, err := h.secretRepo.GetSelectedRepositories(c.Request().Context(), orgID, secret.ID)
	if err != nil {
		return mapSecretErr(c, err)
	}

	repositories, err := h.toOrgSecretRepoSummaries(c.Request().Context(), orgID, repoIDs)
	if err != nil {
		return mapSecretErr(c, err)
	}

	return c.JSON(http.StatusOK, orgSecretReposResponse{
		TotalCount:   len(repositories),
		Repositories: repositories,
	})
}

func (h *SecretHandler) SetOrgSecretRepos(c echo.Context) error {
	orgID, err := h.resolveOrg(c, c.Param("org"))
	if err != nil {
		return mapSecretErr(c, err)
	}
	if err := h.access.EnsureOrgAdmin(c, orgID); err != nil {
		return mapSecretErr(c, err)
	}

	var req setOrgSecretReposRequest
	if err := c.Bind(&req); err != nil {
		return RespondGitHubError(c, http.StatusUnprocessableEntity, "Invalid request body", nil)
	}

	secret, err := h.getSecretUC.Execute(c.Request().Context(), orgID, nil, c.Param("secret_name"))
	if err != nil {
		return mapSecretErr(c, err)
	}

	selectedRepoIDs, err := int64sToUUIDs(req.SelectedRepositoryIDs)
	if err != nil {
		return mapSecretErr(c, err)
	}

	if err := h.secretRepo.SetSelectedRepositories(c.Request().Context(), orgID, secret.ID, selectedRepoIDs); err != nil {
		return mapSecretErr(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *SecretHandler) decryptEncryptedValue(encryptedValue string) (string, error) {
	if strings.TrimSpace(encryptedValue) == "" {
		return "", fmt.Errorf("%w: encrypted_value is required", apperror.ErrValidation)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encryptedValue)
	if err != nil {
		return "", fmt.Errorf("%w: invalid encrypted_value encoding", apperror.ErrValidation)
	}

	plaintext, err := h.enc.DecryptSealedBox(ciphertext)
	if err != nil {
		// A ciphertext that cannot be opened with the server's key is malformed
		// client input (e.g. not sealed against the repo public key), not a
		// server fault, so surface it as a 422 rather than a 500.
		return "", fmt.Errorf("%w: encrypted_value could not be decrypted", apperror.ErrValidation)
	}

	return string(plaintext), nil
}


func (h *SecretHandler) toOrgSecretRepoSummaries(ctx context.Context, orgID uuid.UUID, repoIDs []uuid.UUID) ([]orgSecretRepoSummary, error) {
	nameByID := make(map[uuid.UUID]string, len(repoIDs))
	if h.repoRepo != nil && len(repoIDs) > 0 {
		repos, err := h.repoRepo.ListByOrg(ctx, orgID, 1, 1000)
		if err != nil {
			return nil, err
		}
		for _, repo := range repos {
			nameByID[repo.ID] = repo.Name
		}
	}

	summaries := make([]orgSecretRepoSummary, 0, len(repoIDs))
	for _, repoID := range repoIDs {
		summaries = append(summaries, orgSecretRepoSummary{
			ID:   middleware.UUIDToInt64(repoID),
			Name: nameByID[repoID],
		})
	}
	return summaries, nil
}

func toSecretListResponse(secrets []*entity.ActionSecret, includeVisibility bool) secretListResponse {
	responses := make([]secretResponse, 0, len(secrets))
	for _, secret := range secrets {
		responses = append(responses, toSecretResponse(secret, includeVisibility))
	}
	return secretListResponse{
		TotalCount: len(responses),
		Secrets:    responses,
	}
}

func toSecretResponse(secret *entity.ActionSecret, includeVisibility bool) secretResponse {
	resp := secretResponse{
		Name:      secret.Name,
		CreatedAt: secret.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: secret.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if includeVisibility && secret.Visibility != "" {
		resp.Visibility = secret.Visibility
	}
	return resp
}

func int64sToUUIDs(ids []int64) ([]uuid.UUID, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		result = append(result, middleware.Int64ToUUID(id))
	}
	return result, nil
}

func mapSecretErr(c echo.Context, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, apperror.ErrNotFound) {
		return RespondGitHubError(c, http.StatusNotFound, "Not Found", nil)
	}
	if errors.Is(err, apperror.ErrValidation) {
		message := strings.TrimPrefix(err.Error(), apperror.ErrValidation.Error()+": ")
		if message == apperror.ErrValidation.Error() {
			message = "Validation Failed"
		}
		return RespondGitHubError(c, http.StatusUnprocessableEntity, message, validationFieldErrors(message))
	}
	if errors.Is(err, domain.ErrForbidden) {
		return RespondGitHubError(c, http.StatusForbidden, "Forbidden", nil)
	}
	if errors.Is(err, domain.ErrNotFound) {
		return RespondGitHubError(c, http.StatusNotFound, "Not Found", nil)
	}
	return err
}

func validationFieldErrors(message string) []GitHubFieldError {
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "name"):
		return []GitHubFieldError{{Resource: "secret", Field: "name", Code: "invalid"}}
	case strings.Contains(lower, "value"), strings.Contains(lower, "encrypted_value"):
		return []GitHubFieldError{{Resource: "secret", Field: "encrypted_value", Code: "invalid"}}
	default:
		return nil
	}
}
