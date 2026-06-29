package artifact

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

const presignedURLExpiry = 5 * time.Minute

type CreateArtifactUsecase struct {
	repo    domainrepo.IArtifactRepository
	storage ArtifactStorage
	bucket  string
}

func NewCreateArtifactUsecase(
	repo domainrepo.IArtifactRepository,
	storage ArtifactStorage,
	bucket string,
) *CreateArtifactUsecase {
	return &CreateArtifactUsecase{
		repo:    repo,
		storage: storage,
		bucket:  bucket,
	}
}

func (u *CreateArtifactUsecase) Execute(
	ctx context.Context,
	orgID, repoID, runID uuid.UUID,
	name string,
	retentionDays int,
) (*entity.Artifact, string, error) {
	if err := validateArtifactName(name); err != nil {
		return nil, "", err
	}
	if retentionDays < 1 || retentionDays > 90 {
		return nil, "", domain.ErrValidation
	}

	artifactID := uuid.New()
	now := time.Now().UTC()
	storageKey := artifactStorageKey(orgID, repoID, runID, artifactID, name)

	artifact := &entity.Artifact{
		ID:             artifactID,
		OrganizationID: orgID,
		RunID:          runID,
		Name:           name,
		StorageKey:     storageKey,
		CreatedAt:      now,
		ExpiresAt:      now.Add(time.Duration(retentionDays) * 24 * time.Hour),
	}

	if err := u.repo.Create(ctx, artifact); err != nil {
		return nil, "", err
	}

	uploadURL, err := u.storage.PresignedPutURL(ctx, u.bucket, storageKey, presignedURLExpiry)
	if err != nil {
		return nil, "", err
	}

	return artifact, uploadURL, nil
}

func artifactStorageKey(orgID, repoID, runID, artifactID uuid.UUID, name string) string {
	return fmt.Sprintf("org/%s/repo/%s/runs/%s/%s/%s.zip", orgID, repoID, runID, artifactID, name)
}

func validateArtifactName(name string) error {
	if len(name) < 1 || len(name) > 255 {
		return domain.ErrValidation
	}
	if strings.Contains(name, "../") || strings.HasPrefix(name, "/") {
		return domain.ErrValidation
	}
	return nil
}
