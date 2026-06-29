package pr

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/service"
)

type MergePRInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	GitPath        string
	ActorID        uuid.UUID
	Number         int
	MergeMethod    string
}

type MergePRUsecase struct {
	prRepo               repository.IPullRequestRepository
	branchProtectionRepo repository.IBranchProtectionRepository
	reviewRepo           repository.IReviewRepository
	workflowRunRepo      repository.IWorkflowRunRepository
	auditLogRepo         repository.IAuditLogRepository
	gitService           service.GitService
	txManager            repository.ITransactionManager
	membershipRepo       repository.IMembershipRepository
}

func NewMergePRUsecase(
	prRepo repository.IPullRequestRepository,
	branchProtectionRepo repository.IBranchProtectionRepository,
	reviewRepo repository.IReviewRepository,
	workflowRunRepo repository.IWorkflowRunRepository,
	auditLogRepo repository.IAuditLogRepository,
	gitService service.GitService,
	txManager repository.ITransactionManager,
	membershipRepo repository.IMembershipRepository,
) *MergePRUsecase {
	return &MergePRUsecase{
		prRepo:               prRepo,
		branchProtectionRepo: branchProtectionRepo,
		reviewRepo:           reviewRepo,
		workflowRunRepo:      workflowRunRepo,
		auditLogRepo:         auditLogRepo,
		gitService:           gitService,
		txManager:            txManager,
		membershipRepo:       membershipRepo,
	}
}

func (uc *MergePRUsecase) Execute(ctx context.Context, input MergePRInput) (*entity.PullRequest, error) {
	if err := validateGitPath(input.GitPath); err != nil {
		return nil, err
	}
	if err := uc.checkActorWriteAccess(ctx, input.OrganizationID, input.ActorID); err != nil {
		return nil, err
	}

	pr, err := uc.prRepo.GetByNumber(ctx, input.RepositoryID, input.Number)
	if err != nil {
		return nil, err
	}
	if pr.State == "merged" {
		return nil, apperror.ErrAlreadyMerged
	}

	mergeMethod := input.MergeMethod
	if mergeMethod == "" {
		mergeMethod = "merge"
	}

	if err := uc.checkBranchProtection(ctx, input.RepositoryID, input.GitPath, pr); err != nil {
		return nil, err
	}

	mergeSHA, err := uc.gitService.Merge(ctx, input.GitPath, pr.BaseRef, pr.HeadRef, mergeMethod)
	if err != nil {
		_ = uc.auditLogRepo.Create(ctx, &entity.AuditLog{
			ID:             uuid.New(),
			OrganizationID: input.OrganizationID,
			ActorID:        input.ActorID,
			Action:         "pr.merge.failed",
			TargetType:     "pull_request",
			TargetID:       pr.ID.String(),
		})
		if errors.Is(err, service.ErrMergeConflict) {
			return nil, apperror.ErrConflict
		}
		if errors.Is(err, apperror.ErrConflict) {
			return nil, apperror.ErrConflict
		}
		return nil, err
	}

	now := time.Now().UTC()
	pr.State = "merged"
	pr.MergedAt = &now
	pr.MergedBy = &input.ActorID
	pr.MergeCommitSHA = mergeSHA

	err = uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		if err := uc.prRepo.SetMerged(txCtx, pr.ID, now, input.ActorID, mergeSHA); err != nil {
			return err
		}
		return uc.auditLogRepo.Create(txCtx, &entity.AuditLog{
			ID:             uuid.New(),
			OrganizationID: input.OrganizationID,
			ActorID:        input.ActorID,
			Action:         "pr.merge",
			TargetType:     "pull_request",
			TargetID:       pr.ID.String(),
		})
	})
	if err != nil {
		return nil, err
	}

	return pr, nil
}

func (uc *MergePRUsecase) checkBranchProtection(ctx context.Context, repositoryID uuid.UUID, gitPath string, pr *entity.PullRequest) error {
	protection, err := uc.branchProtectionRepo.GetByBranch(ctx, repositoryID, pr.BaseRef)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return nil
		}
		return err
	}
	if protection == nil {
		return nil
	}

	satisfiedReviews, err := uc.reviewRepo.CountSatisfiedReviews(ctx, pr.ID)
	if err != nil {
		return err
	}
	if satisfiedReviews < protection.RequiredApprovingReviews {
		return apperror.ErrProtectionNotSatisfied
	}

	if len(protection.RequiredStatusChecks) == 0 {
		return nil
	}

	headSHA, err := uc.gitService.ResolveRef(ctx, gitPath, pr.HeadRef)
	if err != nil {
		return err
	}

	runs, err := uc.workflowRunRepo.ListByHeadSHA(ctx, repositoryID, headSHA)
	if err != nil {
		return err
	}
	if !allRequiredChecksPassed(protection.RequiredStatusChecks, runs) {
		return apperror.ErrProtectionNotSatisfied
	}

	return nil
}

func allRequiredChecksPassed(required []string, runs []*entity.WorkflowRun) bool {
	passed := make(map[string]bool, len(runs))
	for _, run := range runs {
		if run.Conclusion == "success" {
			passed[run.Workflow] = true
		}
	}
	for _, check := range required {
		if !passed[check] {
			return false
		}
	}
	return true
}

func (uc *MergePRUsecase) checkActorWriteAccess(ctx context.Context, organizationID, actorID uuid.UUID) error {
	role, err := uc.membershipRepo.GetRole(ctx, organizationID, actorID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.ErrForbidden
	}
	if err != nil {
		return err
	}
	if role != entity.RoleOwner && role != entity.RoleAdmin {
		return domain.ErrForbidden
	}
	return nil
}
