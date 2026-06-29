package artifact

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

const maxArtifactSizeBytes int64 = 2 * 1024 * 1024 * 1024

type CompleteArtifactUsecase struct {
	repo domainrepo.IArtifactRepository
}

func NewCompleteArtifactUsecase(repo domainrepo.IArtifactRepository) *CompleteArtifactUsecase {
	return &CompleteArtifactUsecase{repo: repo}
}

func (u *CompleteArtifactUsecase) Execute(ctx context.Context, artifactID uuid.UUID, sizeInBytes int64) error {
	if sizeInBytes > maxArtifactSizeBytes {
		return domain.ErrValidation
	}
	return u.repo.UpdateStatus(ctx, artifactID, entity.ArtifactStatusCompleted, sizeInBytes)
}
