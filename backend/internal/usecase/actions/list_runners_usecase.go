package actions

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type ListRunnersUsecase struct {
	runnerRepo domainrepo.IRunnerRepository
}

func NewListRunnersUsecase(runnerRepo domainrepo.IRunnerRepository) *ListRunnersUsecase {
	return &ListRunnersUsecase{runnerRepo: runnerRepo}
}

func (uc *ListRunnersUsecase) Execute(ctx context.Context, orgID uuid.UUID) ([]*entity.Runner, error) {
	return uc.runnerRepo.ListByOrg(ctx, orgID)
}
