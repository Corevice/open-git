package importjob

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

var importPhaseOrder = []entity.ImportJobPhase{
	entity.ImportJobPhaseClone,
	entity.ImportJobPhaseMetadata,
	entity.ImportJobPhaseIssues,
	entity.ImportJobPhasePullRequests,
	entity.ImportJobPhaseWiki,
}

type RetryImportJobInput struct {
	OrganizationID uuid.UUID
	JobID          uuid.UUID
	CallerID       uuid.UUID
}

type RetryImportJobUsecase struct {
	importJobs   domainrepo.IImportJobRepository
	checkpoints  domainrepo.IImportPhaseCheckpointRepository
	memberships  domainrepo.IMembershipRepository
	enqueuer     GitHubImportEnqueuer
}

func NewRetryImportJobUsecase(
	importJobs domainrepo.IImportJobRepository,
	checkpoints domainrepo.IImportPhaseCheckpointRepository,
	memberships domainrepo.IMembershipRepository,
	client *asynq.Client,
) *RetryImportJobUsecase {
	return &RetryImportJobUsecase{
		importJobs:  importJobs,
		checkpoints: checkpoints,
		memberships: memberships,
		enqueuer:    newAsynqGitHubImportEnqueuer(client),
	}
}

func NewRetryImportJobUsecaseWithEnqueuer(
	importJobs domainrepo.IImportJobRepository,
	checkpoints domainrepo.IImportPhaseCheckpointRepository,
	memberships domainrepo.IMembershipRepository,
	enqueuer GitHubImportEnqueuer,
) *RetryImportJobUsecase {
	return &RetryImportJobUsecase{
		importJobs:  importJobs,
		checkpoints: checkpoints,
		memberships: memberships,
		enqueuer:    enqueuer,
	}
}

func (u *RetryImportJobUsecase) Execute(ctx context.Context, input RetryImportJobInput) (*entity.ImportJob, error) {
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
	if job.Status != entity.ImportJobStatusFailed {
		return nil, ErrInvalidTransition
	}

	resumePhase, err := u.resolveResumePhase(ctx, job.ID)
	if err != nil {
		return nil, err
	}

	if err := u.importJobs.UpdateStatus(ctx, job.ID, entity.ImportJobStatusQueued); err != nil {
		return nil, err
	}
	if err := u.importJobs.UpdatePhase(ctx, job.ID, resumePhase); err != nil {
		return nil, err
	}
	if err := u.enqueuer.EnqueueGitHubImport(ctx, job.ID, input.OrganizationID); err != nil {
		return nil, err
	}

	job.Status = entity.ImportJobStatusQueued
	job.Phase = resumePhase
	return job, nil
}

func (u *RetryImportJobUsecase) resolveResumePhase(ctx context.Context, jobID uuid.UUID) (entity.ImportJobPhase, error) {
	for _, phase := range importPhaseOrder {
		checkpoint, err := u.checkpoints.GetCheckpoint(ctx, jobID, phase)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return phase, nil
			}
			return "", err
		}
		if checkpoint == nil || !checkpoint.Completed {
			return phase, nil
		}
	}
	return entity.ImportJobPhaseClone, nil
}

func (u *RetryImportJobUsecase) checkCallerAdmin(ctx context.Context, organizationID, callerID uuid.UUID) error {
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
