package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IMCPVerificationRepository interface {
	CreateRun(ctx context.Context, run *entity.MCPVerificationRun) error
	GetRunByID(ctx context.Context, id, orgID uuid.UUID) (*entity.MCPVerificationRun, error)
	UpdateRun(ctx context.Context, run *entity.MCPVerificationRun) error
	DeleteRun(ctx context.Context, id, orgID uuid.UUID) error
	GetLatestRun(ctx context.Context, orgID uuid.UUID) (*entity.MCPVerificationRun, error)
	ListRuns(ctx context.Context, orgID uuid.UUID, page, perPage int) ([]*entity.MCPVerificationRun, int64, error)
	GetActiveRun(ctx context.Context, orgID uuid.UUID) (*entity.MCPVerificationRun, error)
	CountRunsThisMonth(ctx context.Context, orgID uuid.UUID) (int64, error)
	BatchCreateChecks(ctx context.Context, checks []*entity.MCPVerificationCheck) error
	ListChecksByRun(ctx context.Context, runID, orgID uuid.UUID) ([]*entity.MCPVerificationCheck, error)
}
