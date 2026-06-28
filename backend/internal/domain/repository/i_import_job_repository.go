package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IImportJobRepository interface {
	Create(ctx context.Context, job *entity.ImportJob) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.ImportJob, error)
	GetByIDAndOrg(ctx context.Context, id, orgID uuid.UUID) (*entity.ImportJob, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID, page, perPage int) ([]*entity.ImportJob, int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entity.ImportJobStatus) error
	UpdatePhase(ctx context.Context, id uuid.UUID, phase entity.ImportJobPhase) error
	UpdateProgress(ctx context.Context, id uuid.UUID, progress entity.ImportProgress) error
	SetError(ctx context.Context, id uuid.UUID, errMsg string) error
	SetTargetRepository(ctx context.Context, id, repoID uuid.UUID) error
}

type IImportUserMappingRepository interface {
	UpsertMapping(ctx context.Context, m *entity.ImportUserMapping) error
	GetMappingByLogin(ctx context.Context, jobID uuid.UUID, githubLogin string) (*entity.ImportUserMapping, error)
	ListMappings(ctx context.Context, jobID uuid.UUID) ([]*entity.ImportUserMapping, error)
}

type IImportPhaseCheckpointRepository interface {
	SaveCheckpoint(ctx context.Context, cp *entity.ImportPhaseCheckpoint) error
	GetCheckpoint(ctx context.Context, jobID uuid.UUID, phase entity.ImportJobPhase) (*entity.ImportPhaseCheckpoint, error)
	MarkPhaseComplete(ctx context.Context, jobID uuid.UUID, phase entity.ImportJobPhase) error
}
