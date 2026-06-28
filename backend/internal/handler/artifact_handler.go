package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/storage"
	"github.com/open-git/backend/internal/middleware"
)

const artifactDownloadExpiry = 5 * time.Minute

type IArtifactRepository interface {
	ListByRunID(ctx context.Context, orgID, repoID, runID uuid.UUID) ([]*entity.Artifact, error)
	GetByID(ctx context.Context, orgID, artifactID uuid.UUID) (*entity.Artifact, error)
}

type ArtifactHandler struct {
	artifactRepo IArtifactRepository
	storage      *storage.MinIOStorage
	resolveRepo  func(c echo.Context, owner, repo string) (*entity.Repository, error)
}

func NewArtifactHandler(
	artifactRepo IArtifactRepository,
	storage *storage.MinIOStorage,
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error),
) *ArtifactHandler {
	return &ArtifactHandler{
		artifactRepo: artifactRepo,
		storage:      storage,
		resolveRepo:  resolveRepo,
	}
}

func (h *ArtifactHandler) RegisterRoutes(g *echo.Group, auth echo.MiddlewareFunc) {
	readScope := middleware.RequireScope("read")
	g.GET("/repos/:owner/:repo/actions/runs/:run_id/artifacts", h.ListArtifacts, auth, readScope)
	g.GET("/repos/:owner/:repo/actions/artifacts/:artifact_id/zip", h.DownloadArtifact, auth, readScope)
}

type listArtifactsResponse struct {
	TotalCount int                 `json:"total_count"`
	Artifacts  []artifactResponse  `json:"artifacts"`
}

type artifactResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	SizeInBytes int64  `json:"size_in_bytes"`
	ExpiresAt   string `json:"expires_at"`
	CreatedAt   string `json:"created_at"`
}

func (h *ArtifactHandler) ListArtifacts(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	runID, err := uuid.Parse(c.Param("run_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid run_id")
	}

	artifacts, err := h.artifactRepo.ListByRunID(c.Request().Context(), repo.OrganizationID, repo.ID, runID)
	if err != nil {
		return err
	}

	responses := make([]artifactResponse, 0, len(artifacts))
	for _, artifact := range artifacts {
		responses = append(responses, toArtifactResponse(artifact))
	}

	return c.JSON(http.StatusOK, listArtifactsResponse{
		TotalCount: len(responses),
		Artifacts:  responses,
	})
}

func (h *ArtifactHandler) DownloadArtifact(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}

	artifactID, err := parseArtifactID(c.Param("artifact_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid artifact_id")
	}

	artifact, err := h.artifactRepo.GetByID(c.Request().Context(), repo.OrganizationID, artifactID)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return err
	}
	if artifact == nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}

	if h.storage == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "storage unavailable"})
	}

	signedURL, err := h.storage.PresignedGetURL(c.Request().Context(), artifact.StorageKey, artifactDownloadExpiry)
	if err != nil {
		return err
	}

	return c.Redirect(http.StatusFound, signedURL.String())
}

func parseArtifactID(raw string) (uuid.UUID, error) {
	if id, err := uuid.Parse(raw); err == nil {
		return id, nil
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return uuid.Nil, err
	}
	return middleware.Int64ToUUID(parsed), nil
}

func toArtifactResponse(artifact *entity.Artifact) artifactResponse {
	return artifactResponse{
		ID:          middleware.UUIDToInt64(artifact.ID),
		Name:        artifact.Name,
		SizeInBytes: artifact.SizeBytes,
		ExpiresAt:   artifact.ExpiresAt.UTC().Format(time.RFC3339),
		CreatedAt:   artifact.CreatedAt.UTC().Format(time.RFC3339),
	}
}
