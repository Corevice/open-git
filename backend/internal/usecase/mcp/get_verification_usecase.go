package mcp

import (
	"context"

	"github.com/google/uuid"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type GetLatestVerificationUsecase struct {
	repo domainrepo.IMCPVerificationRepository
}

func NewGetLatestVerificationUsecase(repo domainrepo.IMCPVerificationRepository) *GetLatestVerificationUsecase {
	return &GetLatestVerificationUsecase{repo: repo}
}

func (uc *GetLatestVerificationUsecase) Execute(
	ctx context.Context,
	orgID uuid.UUID,
) (*entity.MCPVerificationRun, []*entity.MCPVerificationCheck, error) {
	run, err := uc.repo.GetLatestRun(ctx, orgID)
	if err != nil {
		return nil, nil, err
	}
	if run == nil {
		return nil, nil, nil
	}

	checks, err := uc.repo.ListChecksByRun(ctx, run.ID, orgID)
	if err != nil {
		return nil, nil, err
	}

	return run, checks, nil
}

type ListVerificationHistoryUsecase struct {
	repo domainrepo.IMCPVerificationRepository
}

func NewListVerificationHistoryUsecase(repo domainrepo.IMCPVerificationRepository) *ListVerificationHistoryUsecase {
	return &ListVerificationHistoryUsecase{repo: repo}
}

func (uc *ListVerificationHistoryUsecase) Execute(
	ctx context.Context,
	orgID uuid.UUID,
	page, perPage int,
) ([]*entity.MCPVerificationRun, int64, error) {
	return uc.repo.ListRuns(ctx, orgID, page, perPage)
}

type GetJobStatusUsecase struct {
	repo domainrepo.IMCPVerificationRepository
}

func NewGetJobStatusUsecase(repo domainrepo.IMCPVerificationRepository) *GetJobStatusUsecase {
	return &GetJobStatusUsecase{repo: repo}
}

func (uc *GetJobStatusUsecase) Execute(
	ctx context.Context,
	runID, orgID uuid.UUID,
) (*entity.MCPVerificationRun, float64, error) {
	run, err := uc.repo.GetRunByID(ctx, runID, orgID)
	if err != nil {
		return nil, 0, err
	}
	if run == nil {
		return nil, 0, nil
	}

	return run, jobProgress(run.Status), nil
}

func jobProgress(status entity.RunStatus) float64 {
	switch status {
	case entity.RunStatusQueued:
		return 0.0
	case entity.RunStatusRunning:
		return 0.5
	case entity.RunStatusCompleted, entity.RunStatusErrored:
		return 1.0
	default:
		return 0.0
	}
}
