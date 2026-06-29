package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/handler"
	"github.com/open-git/backend/internal/middleware"
	artifactusecase "github.com/open-git/backend/internal/usecase/artifact"
)

var (
	artifactTestOrgID  = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	artifactTestRepoID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	artifactTestRunID  = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	artifactTestID     = uuid.MustParse("44444444-4444-4444-4444-444444444444")
)

type handlerArtifactRepo struct {
	byID map[uuid.UUID]*entity.Artifact
	byRun []*entity.Artifact
}

func (m *handlerArtifactRepo) Create(context.Context, *entity.Artifact) error { return nil }

func (m *handlerArtifactRepo) GetByID(_ context.Context, id, orgID uuid.UUID) (*entity.Artifact, error) {
	artifact, ok := m.byID[id]
	if !ok || artifact.OrganizationID != orgID {
		return nil, domain.ErrNotFound
	}
	copyArtifact := *artifact
	return &copyArtifact, nil
}

func (m *handlerArtifactRepo) ListByRun(_ context.Context, runID, orgID uuid.UUID) ([]*entity.Artifact, error) {
	var artifacts []*entity.Artifact
	for _, artifact := range m.byRun {
		if artifact.RunID == runID && artifact.OrganizationID == orgID {
			copyArtifact := *artifact
			artifacts = append(artifacts, &copyArtifact)
		}
	}
	return artifacts, nil
}

func (m *handlerArtifactRepo) UpdateStatus(context.Context, uuid.UUID, entity.ArtifactStatus, int64) error {
	return nil
}

func (m *handlerArtifactRepo) SoftDelete(_ context.Context, id, orgID uuid.UUID) error {
	if artifact, ok := m.byID[id]; ok && artifact.OrganizationID == orgID {
		delete(m.byID, id)
		return nil
	}
	return domain.ErrNotFound
}

func (m *handlerArtifactRepo) ListExpired(context.Context, int) ([]*entity.Artifact, error) {
	return nil, nil
}

func (m *handlerArtifactRepo) DeleteByRunID(context.Context, uuid.UUID) error { return nil }

var _ domainrepo.IArtifactRepository = (*handlerArtifactRepo)(nil)

type handlerArtifactStorage struct {
	getURL string
}

func (m *handlerArtifactStorage) PresignedPutURL(context.Context, string, string, time.Duration) (string, error) {
	return "https://minio.example/upload", nil
}

func (m *handlerArtifactStorage) PresignedGetURL(context.Context, string, string, time.Duration) (string, error) {
	return m.getURL, nil
}

func (m *handlerArtifactStorage) DeleteObject(context.Context, string, string) error { return nil }

func newArtifactHandler(repo *handlerArtifactRepo, storage *handlerArtifactStorage) *handler.ArtifactHandler {
	createUC := artifactusecase.NewCreateArtifactUsecase(repo, storage, "artifacts")
	completeUC := artifactusecase.NewCompleteArtifactUsecase(repo)
	listUC := artifactusecase.NewListArtifactsUsecase(repo)
	downloadUC := artifactusecase.NewGetArtifactDownloadURLUsecase(repo, storage, "artifacts")
	deleteUC := artifactusecase.NewDeleteArtifactUsecase(repo, storage, "artifacts")

	resolveRepo := func(c echo.Context, owner, repoName string) (*entity.Repository, error) {
		return &entity.Repository{
			ID:             artifactTestRepoID,
			OrganizationID: artifactTestOrgID,
			Name:           repoName,
		}, nil
	}

	return handler.NewArtifactHandler(createUC, completeUC, listUC, downloadUC, deleteUC, resolveRepo)
}

func TestArtifactHandlerListJSONShape(t *testing.T) {
	repo := &handlerArtifactRepo{
		byRun: []*entity.Artifact{
			{
				ID:             artifactTestID,
				OrganizationID: artifactTestOrgID,
				RunID:          artifactTestRunID,
				Name:           "build-output",
				SizeBytes:      1024,
				CreatedAt:      time.Now().UTC(),
				ExpiresAt:      time.Now().UTC().Add(24 * time.Hour),
			},
		},
	}
	h := newArtifactHandler(repo, &handlerArtifactStorage{})

	e := echo.New()
	g := e.Group("/api/v3")
	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middleware.SetAuthContext(c, 1, []string{"read"})
			return next(c)
		}
	})
	h.RegisterRoutes(g, func(next echo.HandlerFunc) echo.HandlerFunc { return next })

	req := httptest.NewRequest(http.MethodGet, "/api/v3/repos/acme/demo/actions/runs/"+artifactTestRunID.String()+"/artifacts", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if _, ok := body["total_count"]; !ok {
		t.Fatalf("missing total_count: %s", rec.Body.String())
	}
	if _, ok := body["artifacts"]; !ok {
		t.Fatalf("missing artifacts: %s", rec.Body.String())
	}
}

func TestArtifactHandlerDownloadRedirect(t *testing.T) {
	repo := &handlerArtifactRepo{
		byID: map[uuid.UUID]*entity.Artifact{
			artifactTestID: {
				ID:             artifactTestID,
				OrganizationID: artifactTestOrgID,
				StorageKey:     "key",
				ExpiresAt:      time.Now().UTC().Add(time.Hour),
			},
		},
	}
	h := newArtifactHandler(repo, &handlerArtifactStorage{getURL: "https://minio.example/download"})

	e := echo.New()
	g := e.Group("/api/v3")
	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middleware.SetAuthContext(c, 1, []string{"read"})
			return next(c)
		}
	})
	h.RegisterRoutes(g, func(next echo.HandlerFunc) echo.HandlerFunc { return next })

	req := httptest.NewRequest(http.MethodGet, "/api/v3/repos/acme/demo/actions/artifacts/"+artifactTestID.String()+"/zip", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "https://minio.example/download" {
		t.Fatalf("Location = %q", got)
	}
}

func TestArtifactHandlerDownloadExpired(t *testing.T) {
	repo := &handlerArtifactRepo{
		byID: map[uuid.UUID]*entity.Artifact{
			artifactTestID: {
				ID:             artifactTestID,
				OrganizationID: artifactTestOrgID,
				StorageKey:     "key",
				ExpiresAt:      time.Now().UTC().Add(-time.Hour),
			},
		},
	}
	h := newArtifactHandler(repo, &handlerArtifactStorage{getURL: "https://minio.example/download"})

	e := echo.New()
	g := e.Group("/api/v3")
	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middleware.SetAuthContext(c, 1, []string{"read"})
			return next(c)
		}
	})
	h.RegisterRoutes(g, func(next echo.HandlerFunc) echo.HandlerFunc { return next })

	req := httptest.NewRequest(http.MethodGet, "/api/v3/repos/acme/demo/actions/artifacts/"+artifactTestID.String()+"/zip", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusGone {
		t.Fatalf("expected 410, got %d", rec.Code)
	}
}

func TestArtifactHandlerDeleteForbiddenWithoutWriteScope(t *testing.T) {
	repo := &handlerArtifactRepo{
		byID: map[uuid.UUID]*entity.Artifact{
			artifactTestID: {
				ID:             artifactTestID,
				OrganizationID: artifactTestOrgID,
				StorageKey:     "key",
			},
		},
	}
	h := newArtifactHandler(repo, &handlerArtifactStorage{})

	e := echo.New()
	g := e.Group("/api/v3")
	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middleware.SetAuthContext(c, 1, []string{"read"})
			return next(c)
		}
	})
	h.RegisterRoutes(g, func(next echo.HandlerFunc) echo.HandlerFunc { return next })

	req := httptest.NewRequest(http.MethodDelete, "/api/v3/repos/acme/demo/actions/artifacts/"+artifactTestID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestArtifactHandlerDeleteNoContent(t *testing.T) {
	repo := &handlerArtifactRepo{
		byID: map[uuid.UUID]*entity.Artifact{
			artifactTestID: {
				ID:             artifactTestID,
				OrganizationID: artifactTestOrgID,
				StorageKey:     "key",
			},
		},
	}
	h := newArtifactHandler(repo, &handlerArtifactStorage{})

	e := echo.New()
	g := e.Group("/api/v3")
	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middleware.SetAuthContext(c, 1, []string{"write"})
			return next(c)
		}
	})
	h.RegisterRoutes(g, func(next echo.HandlerFunc) echo.HandlerFunc { return next })

	req := httptest.NewRequest(http.MethodDelete, "/api/v3/repos/acme/demo/actions/artifacts/"+artifactTestID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}
