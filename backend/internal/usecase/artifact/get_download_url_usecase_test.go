package artifact_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	artifactusecase "github.com/open-git/backend/internal/usecase/artifact"
)

type getDownloadRepo struct {
	artifact *entity.Artifact
}

func (m *getDownloadRepo) Create(context.Context, *entity.Artifact) error { return nil }

func (m *getDownloadRepo) GetByID(_ context.Context, id, orgID uuid.UUID) (*entity.Artifact, error) {
	if m.artifact == nil || m.artifact.ID != id || m.artifact.OrganizationID != orgID {
		return nil, domain.ErrNotFound
	}
	copyArtifact := *m.artifact
	return &copyArtifact, nil
}

func (m *getDownloadRepo) ListByRun(context.Context, uuid.UUID, uuid.UUID) ([]*entity.Artifact, error) {
	return nil, nil
}

func (m *getDownloadRepo) UpdateStatus(context.Context, uuid.UUID, entity.ArtifactStatus, int64) error {
	return nil
}

func (m *getDownloadRepo) SoftDelete(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func (m *getDownloadRepo) ListExpired(context.Context, int) ([]*entity.Artifact, error) {
	return nil, nil
}

func (m *getDownloadRepo) DeleteByRunID(context.Context, uuid.UUID) error { return nil }

var _ domainrepo.IArtifactRepository = (*getDownloadRepo)(nil)

type getDownloadStorage struct {
	url string
}

func (m *getDownloadStorage) PresignedPutURL(context.Context, string, string, time.Duration) (string, error) {
	return "", nil
}

func (m *getDownloadStorage) PresignedGetURL(context.Context, string, string, time.Duration) (string, error) {
	return m.url, nil
}

func (m *getDownloadStorage) DeleteObject(context.Context, string, string) error { return nil }

func TestGetArtifactDownloadURLExpired(t *testing.T) {
	orgID := uuid.New()
	artifactID := uuid.New()
	repo := &getDownloadRepo{
		artifact: &entity.Artifact{
			ID:             artifactID,
			OrganizationID: orgID,
			StorageKey:     "key",
			ExpiresAt:      time.Now().UTC().Add(-time.Hour),
		},
	}
	uc := artifactusecase.NewGetArtifactDownloadURLUsecase(repo, &getDownloadStorage{url: "https://minio.example/download"}, "artifacts")

	_, err := uc.Execute(context.Background(), artifactID, orgID)
	if !errors.Is(err, artifactusecase.ErrArtifactExpired) {
		t.Fatalf("expected ErrArtifactExpired, got %v", err)
	}
}

func TestGetArtifactDownloadURLCrossOrg(t *testing.T) {
	orgID := uuid.New()
	otherOrgID := uuid.New()
	artifactID := uuid.New()
	repo := &getDownloadRepo{
		artifact: &entity.Artifact{
			ID:             artifactID,
			OrganizationID: orgID,
			StorageKey:     "key",
			ExpiresAt:      time.Now().UTC().Add(time.Hour),
		},
	}
	uc := artifactusecase.NewGetArtifactDownloadURLUsecase(repo, &getDownloadStorage{url: "https://minio.example/download"}, "artifacts")

	_, err := uc.Execute(context.Background(), artifactID, otherOrgID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetArtifactDownloadURLValid(t *testing.T) {
	orgID := uuid.New()
	artifactID := uuid.New()
	repo := &getDownloadRepo{
		artifact: &entity.Artifact{
			ID:             artifactID,
			OrganizationID: orgID,
			StorageKey:     "key",
			ExpiresAt:      time.Now().UTC().Add(time.Hour),
		},
	}
	uc := artifactusecase.NewGetArtifactDownloadURLUsecase(repo, &getDownloadStorage{url: "https://minio.example/download"}, "artifacts")

	url, err := uc.Execute(context.Background(), artifactID, orgID)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if url != "https://minio.example/download" {
		t.Fatalf("url = %q", url)
	}
}
