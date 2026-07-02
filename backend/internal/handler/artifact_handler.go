package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/middleware"
	artifactusecase "github.com/open-git/backend/internal/usecase/artifact"
)

type ArtifactHandler struct {
	createArtifactUC   *artifactusecase.CreateArtifactUsecase
	completeArtifactUC *artifactusecase.CompleteArtifactUsecase
	listArtifactsUC    *artifactusecase.ListArtifactsUsecase
	getDownloadURLUC   *artifactusecase.GetArtifactDownloadURLUsecase
	deleteArtifactUC   *artifactusecase.DeleteArtifactUsecase
	resolveRepo        func(c echo.Context, owner, repo string) (*entity.Repository, error)
	access             *RepoAccess
}

func NewArtifactHandler(
	createArtifactUC *artifactusecase.CreateArtifactUsecase,
	completeArtifactUC *artifactusecase.CompleteArtifactUsecase,
	listArtifactsUC *artifactusecase.ListArtifactsUsecase,
	getDownloadURLUC *artifactusecase.GetArtifactDownloadURLUsecase,
	deleteArtifactUC *artifactusecase.DeleteArtifactUsecase,
	resolveRepo func(c echo.Context, owner, repo string) (*entity.Repository, error),
) *ArtifactHandler {
	return &ArtifactHandler{
		createArtifactUC:   createArtifactUC,
		completeArtifactUC: completeArtifactUC,
		listArtifactsUC:    listArtifactsUC,
		getDownloadURLUC:   getDownloadURLUC,
		deleteArtifactUC:   deleteArtifactUC,
		resolveRepo:        resolveRepo,
	}
}

func (h *ArtifactHandler) SetAccess(a *RepoAccess) { h.access = a }

func (h *ArtifactHandler) RegisterRoutes(g *echo.Group, auth echo.MiddlewareFunc) {
	readScope := middleware.RequireScope("read")
	writeScope := middleware.RequireScope("write")

	g.POST("/repos/:owner/:repo/actions/runs/:run_id/artifacts", h.CreateArtifact, auth, writeScope)
	g.PATCH("/repos/:owner/:repo/actions/runs/:run_id/artifacts/:artifact_id", h.CompleteArtifact, auth, writeScope)
	g.GET("/repos/:owner/:repo/actions/runs/:run_id/artifacts", h.ListArtifacts, auth, readScope)
	g.GET("/repos/:owner/:repo/actions/artifacts/:artifact_id/zip", h.DownloadArtifact, auth, readScope)
	g.DELETE("/repos/:owner/:repo/actions/artifacts/:artifact_id", h.DeleteArtifact, auth, writeScope)
}

type createArtifactRequest struct {
	Name          string `json:"name"`
	RetentionDays int    `json:"retention_days"`
}

type createArtifactResponse struct {
	ID        string `json:"id"`
	UploadURL string `json:"upload_url"`
	Name      string `json:"name"`
}

type completeArtifactRequest struct {
	Status string `json:"status"`
	Size   int64  `json:"size"`
}

type listArtifactsResponse struct {
	TotalCount int                `json:"total_count"`
	Artifacts  []artifactResponse `json:"artifacts"`
}

type artifactResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	SizeInBytes int64  `json:"size_in_bytes"`
	ExpiresAt   string `json:"expires_at"`
	CreatedAt   string `json:"created_at"`
	Expired     bool   `json:"expired"`
}

func (h *ArtifactHandler) CreateArtifact(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}
	if err := h.access.EnsureWrite(c, repo); err != nil {
		return err
	}

	runID, err := uuid.Parse(c.Param("run_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid run_id"})
	}

	var req createArtifactRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid request body"})
	}

	retentionDays := req.RetentionDays
	if retentionDays == 0 {
		retentionDays = 90
	}

	artifact, uploadURL, err := h.createArtifactUC.Execute(
		c.Request().Context(),
		repo.OrganizationID,
		repo.ID,
		runID,
		req.Name,
		retentionDays,
	)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, createArtifactResponse{
		ID:        artifact.ID.String(),
		UploadURL: uploadURL,
		Name:      artifact.Name,
	})
}

func (h *ArtifactHandler) CompleteArtifact(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}
	if err := h.access.EnsureWrite(c, repo); err != nil {
		return err
	}

	artifactID, err := parseArtifactID(c.Param("artifact_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid artifact_id"})
	}

	var req completeArtifactRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid request body"})
	}

	if err := h.completeArtifactUC.Execute(c.Request().Context(), artifactID, req.Size); err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

func (h *ArtifactHandler) ListArtifacts(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}
	if err := h.access.EnsureRead(c, repo); err != nil {
		return err
	}

	runID, err := uuid.Parse(c.Param("run_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid run_id"})
	}

	artifacts, err := h.listArtifactsUC.Execute(c.Request().Context(), runID, repo.OrganizationID)
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
	if err := h.access.EnsureRead(c, repo); err != nil {
		return err
	}
	orgID := repo.OrganizationID

	artifactID, err := parseArtifactID(c.Param("artifact_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid artifact_id"})
	}

	signedURL, err := h.getDownloadURLUC.Execute(c.Request().Context(), artifactID, orgID)
	if err != nil {
		if errors.Is(err, artifactusecase.ErrArtifactExpired) {
			return echo.NewHTTPError(http.StatusGone, map[string]string{"message": "artifact expired"})
		}
		return err
	}

	return c.Redirect(http.StatusFound, signedURL)
}

func (h *ArtifactHandler) DeleteArtifact(c echo.Context) error {
	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return err
	}
	if err := h.access.EnsureWrite(c, repo); err != nil {
		return err
	}
	orgID := repo.OrganizationID

	artifactID, err := parseArtifactID(c.Param("artifact_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid artifact_id"})
	}

	if err := h.deleteArtifactUC.Execute(c.Request().Context(), artifactID, orgID); err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *ArtifactHandler) resolveOrgID(c echo.Context) (uuid.UUID, error) {
	if raw, ok := c.Get("org_id").(uuid.UUID); ok && raw != uuid.Nil {
		return raw, nil
	}
	if raw, ok := c.Get("org_id").(string); ok && raw != "" {
		orgID, err := uuid.Parse(raw)
		if err != nil {
			return uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid org_id"})
		}
		return orgID, nil
	}

	repo, err := h.resolveRepo(c, c.Param("owner"), c.Param("repo"))
	if err != nil {
		return uuid.Nil, err
	}
	return repo.OrganizationID, nil
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
	now := time.Now().UTC()
	return artifactResponse{
		ID:          middleware.UUIDToInt64(artifact.ID),
		Name:        artifact.Name,
		SizeInBytes: artifact.SizeBytes,
		ExpiresAt:   artifact.ExpiresAt.UTC().Format(time.RFC3339),
		CreatedAt:   artifact.CreatedAt.UTC().Format(time.RFC3339),
		Expired:     now.After(artifact.ExpiresAt),
	}
}
