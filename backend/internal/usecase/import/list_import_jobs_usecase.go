package importjob

import (
	"context"

	"github.com/google/uuid"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type ListImportJobsInput struct {
	OrganizationID uuid.UUID
	Page           int
	PerPage        int
}

type ListImportJobsOutput struct {
	Jobs  []*entity.ImportJob
	Total int
}

type ListImportJobsUsecase struct {
	importJobs domainrepo.IImportJobRepository
}

func NewListImportJobsUsecase(importJobs domainrepo.IImportJobRepository) *ListImportJobsUsecase {
	return &ListImportJobsUsecase{importJobs: importJobs}
}

func (u *ListImportJobsUsecase) Execute(ctx context.Context, input ListImportJobsInput) (*ListImportJobsOutput, error) {
	page := input.Page
	if page < 1 {
		page = 1
	}
	perPage := input.PerPage
	if perPage < 1 {
		perPage = 30
	}

	jobs, total, err := u.importJobs.ListByOrg(ctx, input.OrganizationID, page, perPage)
	if err != nil {
		return nil, err
	}

	return &ListImportJobsOutput{
		Jobs:  jobs,
		Total: total,
	}, nil
}
