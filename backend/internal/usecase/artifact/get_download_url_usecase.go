package artifact

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

var ErrArtifactExpired = errors.New("artifact expired")

type GetArtifactDownloadURLUsecase struct {
	repo    domainrepo.IArtifactRepository
	storage ArtifactStorage
	bucket  string
}

func NewGetArtifactDownloadURLUsecase(
	repo domainrepo.IArtifactRepository,
	storage ArtifactStorage,
	bucket string,
) *GetArtifactDownloadURLUsecase {
	return &GetArtifactDownloadURLUsecase{
		repo:    repo,
		storage: storage,
		bucket:  bucket,
	}
}

func (u *GetArtifactDownloadURLUsecase) Execute(ctx context.Context, artifactID, orgID uuid.UUID) (string, error) {
	artifact, err := u.repo.GetByID(ctx, artifactID, orgID)
	if err != nil {
		return "", err
	}
	if artifact.OrganizationID != orgID {
		return "", domain.ErrNotFound
	}
	if time.Now().UTC().After(artifact.ExpiresAt) {
		return "", ErrArtifactExpired
	}

	return u.storage.PresignedGetURL(ctx, u.bucket, artifact.StorageKey, presignedURLExpiry)
}
