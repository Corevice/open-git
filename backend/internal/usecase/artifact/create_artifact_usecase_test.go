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

type mockArtifactRepo struct {
	created     *entity.Artifact
	createErr   error
	existing    map[string]*entity.Artifact
}

func (m *mockArtifactRepo) Create(_ context.Context, artifact *entity.Artifact) error {
	if m.createErr != nil {
		return m.createErr
	}
	copyArtifact := *artifact
	m.created = &copyArtifact
	if m.existing == nil {
		m.existing = map[string]*entity.Artifact{}
	}
	key := artifact.RunID.String() + ":" + artifact.Name
	if _, ok := m.existing[key]; ok {
		return domain.ErrConflict
	}
	m.existing[key] = &copyArtifact
	return nil
}

func (m *mockArtifactRepo) GetByID(context.Context, uuid.UUID, uuid.UUID) (*entity.Artifact, error) {
	return nil, domain.ErrNotFound
}

func (m *mockArtifactRepo) ListByRun(context.Context, uuid.UUID, uuid.UUID) ([]*entity.Artifact, error) {
	return nil, nil
}

func (m *mockArtifactRepo) UpdateStatus(context.Context, uuid.UUID, entity.ArtifactStatus, int64) error {
	return nil
}

func (m *mockArtifactRepo) SoftDelete(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (m *mockArtifactRepo) ListExpired(context.Context, int) ([]*entity.Artifact, error) {
	return nil, nil
}

func (m *mockArtifactRepo) DeleteByRunID(context.Context, uuid.UUID) error {
	return nil
}

var _ domainrepo.IArtifactRepository = (*mockArtifactRepo)(nil)

type mockArtifactStorage struct {
	putURL string
}

func (m *mockArtifactStorage) PresignedPutURL(context.Context, string, string, time.Duration) (string, error) {
	return m.putURL, nil
}

func (m *mockArtifactStorage) PresignedGetURL(context.Context, string, string, time.Duration) (string, error) {
	return "", nil
}

func (m *mockArtifactStorage) DeleteObject(context.Context, string, string) error {
	return nil
}

var _ artifactusecase.ArtifactStorage = (*mockArtifactStorage)(nil)

func TestCreateArtifactUsecaseValidInput(t *testing.T) {
	repo := &mockArtifactRepo{}
	storage := &mockArtifactStorage{putURL: "https://minio.example/upload"}
	uc := artifactusecase.NewCreateArtifactUsecase(repo, storage, "artifacts")

	orgID := uuid.New()
	repoID := uuid.New()
	runID := uuid.New()

	artifact, uploadURL, err := uc.Execute(context.Background(), orgID, repoID, runID, "build-output", 30)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if artifact == nil {
		t.Fatal("expected artifact")
	}
	if uploadURL != "https://minio.example/upload" {
		t.Fatalf("upload URL = %q", uploadURL)
	}
	if repo.created == nil || repo.created.Name != "build-output" {
		t.Fatalf("artifact not persisted: %+v", repo.created)
	}
}

func TestCreateArtifactUsecaseRejectsPathTraversal(t *testing.T) {
	uc := artifactusecase.NewCreateArtifactUsecase(&mockArtifactRepo{}, &mockArtifactStorage{}, "artifacts")

	_, _, err := uc.Execute(context.Background(), uuid.New(), uuid.New(), uuid.New(), "../secret", 30)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestCreateArtifactUsecaseRejectsRetentionDaysOutOfRange(t *testing.T) {
	uc := artifactusecase.NewCreateArtifactUsecase(&mockArtifactRepo{}, &mockArtifactStorage{}, "artifacts")

	_, _, err := uc.Execute(context.Background(), uuid.New(), uuid.New(), uuid.New(), "build-output", 91)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestCreateArtifactUsecaseDuplicateName(t *testing.T) {
	runID := uuid.New()
	repo := &mockArtifactRepo{
		existing: map[string]*entity.Artifact{
			runID.String() + ":build-output": {Name: "build-output", RunID: runID},
		},
	}
	uc := artifactusecase.NewCreateArtifactUsecase(repo, &mockArtifactStorage{putURL: "https://minio.example/upload"}, "artifacts")

	_, _, err := uc.Execute(context.Background(), uuid.New(), uuid.New(), runID, "build-output", 30)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}
