package artifact

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type ListArtifactsUsecase struct {
	repo domainrepo.IArtifactRepository
}

func NewListArtifactsUsecase(repo domainrepo.IArtifactRepository) *ListArtifactsUsecase {
	return &ListArtifactsUsecase{repo: repo}
}

func (u *ListArtifactsUsecase) Execute(ctx context.Context, runID, orgID uuid.UUID) ([]*entity.Artifact, error) {
	return u.repo.ListByRun(ctx, runID, orgID)
}
