package importjob

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type CancelImportJobInput struct {
	OrganizationID uuid.UUID
	JobID          uuid.UUID
	CallerID       uuid.UUID
}

type CancelImportJobUsecase struct {
	importJobs  domainrepo.IImportJobRepository
	memberships domainrepo.IMembershipRepository
}

func NewCancelImportJobUsecase(
	importJobs domainrepo.IImportJobRepository,
	memberships domainrepo.IMembershipRepository,
) *CancelImportJobUsecase {
	return &CancelImportJobUsecase{
		importJobs:  importJobs,
		memberships: memberships,
	}
}

func (u *CancelImportJobUsecase) Execute(ctx context.Context, input CancelImportJobInput) (*entity.ImportJob, error) {
	if err := u.checkCallerAdmin(ctx, input.OrganizationID, input.CallerID); err != nil {
		return nil, err
	}

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

	switch job.Status {
	case entity.ImportJobStatusQueued, entity.ImportJobStatusRunning, entity.ImportJobStatusPaused:
	default:
		return nil, ErrInvalidTransition
	}

	if err := u.importJobs.UpdateStatus(ctx, job.ID, entity.ImportJobStatusCancelled); err != nil {
		return nil, err
	}

	job.Status = entity.ImportJobStatusCancelled
	return job, nil
}

func (u *CancelImportJobUsecase) checkCallerAdmin(ctx context.Context, organizationID, callerID uuid.UUID) error {
	role, err := u.memberships.GetRole(ctx, organizationID, callerID)
	if errors.Is(err, domain.ErrNotFound) {
		return ErrForbidden
	}
	if err != nil {
		return err
	}
	if role != entity.RoleOwner && role != entity.RoleAdmin {
		return ErrForbidden
	}
	return nil
}
