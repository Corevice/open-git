package artifact_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	artifactusecase "github.com/open-git/backend/internal/usecase/artifact"
)

type deleteArtifactRepo struct {
	artifact    *entity.Artifact
	softDeleted bool
}

func (m *deleteArtifactRepo) Create(context.Context, *entity.Artifact) error { return nil }

func (m *deleteArtifactRepo) GetByID(_ context.Context, id, orgID uuid.UUID) (*entity.Artifact, error) {
	if m.artifact == nil || m.artifact.ID != id || m.artifact.OrganizationID != orgID {
		return nil, nil
	}
	copyArtifact := *m.artifact
	return &copyArtifact, nil
}

func (m *deleteArtifactRepo) ListByRun(context.Context, uuid.UUID, uuid.UUID) ([]*entity.Artifact, error) {
	return nil, nil
}

func (m *deleteArtifactRepo) UpdateStatus(context.Context, uuid.UUID, entity.ArtifactStatus, int64) error {
	return nil
}

func (m *deleteArtifactRepo) SoftDelete(_ context.Context, id, orgID uuid.UUID) error {
	if m.artifact != nil && m.artifact.ID == id && m.artifact.OrganizationID == orgID {
		m.softDeleted = true
	}
	return nil
}

func (m *deleteArtifactRepo) ListExpired(context.Context, int) ([]*entity.Artifact, error) {
	return nil, nil
}

func (m *deleteArtifactRepo) DeleteByRunID(context.Context, uuid.UUID) error { return nil }

var _ domainrepo.IArtifactRepository = (*deleteArtifactRepo)(nil)

type deleteArtifactStorage struct {
	deletedKey string
}

func (m *deleteArtifactStorage) PresignedPutURL(context.Context, string, string, time.Duration) (string, error) {
	return "", nil
}

func (m *deleteArtifactStorage) PresignedGetURL(context.Context, string, string, time.Duration) (string, error) {
	return "", nil
}

func (m *deleteArtifactStorage) DeleteObject(_ context.Context, _, key string) error {
	m.deletedKey = key
	return nil
}

func TestDeleteArtifactUsecaseCallsStorageAndRepo(t *testing.T) {
	orgID := uuid.New()
	artifactID := uuid.New()
	repo := &deleteArtifactRepo{
		artifact: &entity.Artifact{
			ID:             artifactID,
			OrganizationID: orgID,
			StorageKey:     "org/key.zip",
		},
	}
	storage := &deleteArtifactStorage{}
	uc := artifactusecase.NewDeleteArtifactUsecase(repo, storage, "artifacts")

	if err := uc.Execute(context.Background(), artifactID, orgID); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if storage.deletedKey != "org/key.zip" {
		t.Fatalf("DeleteObject key = %q", storage.deletedKey)
	}
	if !repo.softDeleted {
		t.Fatal("expected SoftDelete to be called")
	}
}
