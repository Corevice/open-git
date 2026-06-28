package importjob

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type GetImportJobInput struct {
	OrganizationID uuid.UUID
	JobID          uuid.UUID
}

type GetImportJobUsecase struct {
	importJobs domainrepo.IImportJobRepository
}

func NewGetImportJobUsecase(importJobs domainrepo.IImportJobRepository) *GetImportJobUsecase {
	return &GetImportJobUsecase{importJobs: importJobs}
}

func (u *GetImportJobUsecase) Execute(ctx context.Context, input GetImportJobInput) (*entity.ImportJob, error) {
	job, err := u.importJobs.GetByIDAndOrg(ctx, input.JobID, input.OrganizationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if job == nil {
		return nil, ErrNotFound
	}
	return job, nil
}
