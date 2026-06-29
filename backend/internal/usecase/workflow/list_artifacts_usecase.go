package workflow

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
)

type ListArtifactsInput struct {
	OrganizationID uuid.UUID
	RunID          uuid.UUID
}

type ListArtifactsUsecase struct {
	runRepo      WorkflowRunRepository
	artifactRepo ArtifactRepository
}

func NewListArtifactsUsecase(runRepo WorkflowRunRepository, artifactRepo ArtifactRepository) *ListArtifactsUsecase {
	return &ListArtifactsUsecase{
		runRepo:      runRepo,
		artifactRepo: artifactRepo,
	}
}

func (uc *ListArtifactsUsecase) Execute(ctx context.Context, input ListArtifactsInput) ([]*WorkflowArtifact, error) {
	run, err := uc.runRepo.GetByID(ctx, input.OrganizationID, input.RunID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, domain.ErrNotFound
	}
	if run.OrganizationID != input.OrganizationID {
		return nil, domain.ErrNotFound
	}

	return uc.artifactRepo.ListByRunID(ctx, input.RunID, input.OrganizationID)
}
