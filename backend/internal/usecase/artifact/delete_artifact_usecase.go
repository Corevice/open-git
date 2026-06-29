package artifact

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type DeleteArtifactUsecase struct {
	repo    domainrepo.IArtifactRepository
	storage ArtifactStorage
	bucket  string
}

func NewDeleteArtifactUsecase(
	repo domainrepo.IArtifactRepository,
	storage ArtifactStorage,
	bucket string,
) *DeleteArtifactUsecase {
	return &DeleteArtifactUsecase{
		repo:    repo,
		storage: storage,
		bucket:  bucket,
	}
}

func (u *DeleteArtifactUsecase) Execute(ctx context.Context, artifactID, orgID uuid.UUID) error {
	artifact, err := u.repo.GetByID(ctx, artifactID, orgID)
	if err != nil {
		return err
	}
	if artifact.OrganizationID != orgID {
		return domain.ErrNotFound
	}

	if err := u.storage.DeleteObject(ctx, u.bucket, artifact.StorageKey); err != nil {
		return err
	}
	return u.repo.SoftDelete(ctx, artifactID, orgID)
}
