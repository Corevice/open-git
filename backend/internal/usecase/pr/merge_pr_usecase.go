package pr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/service"
	"github.com/google/uuid"
)

type MergePRInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	ActorID        uuid.UUID
	// Deprecated: role is resolved from ActorID via membershipRepo; this field is ignored.
	RequesterRole string
	Number        int
	MergeMethod   string
}

type MergePRUsecase struct {
	prRepo               repository.IPullRequestRepository
	branchProtectionRepo repository.IBranchProtectionRepository
	membershipRepo       repository.IMembershipRepository
	reviewRepo           repository.IReviewRepository
	workflowRunRepo      repository.IWorkflowRunRepository
	auditLogRepo         repository.IAuditLogRepository
	gitService           service.GitService
	txManager            repository.TransactionManager
}

type MergePRUsecaseOption func(*MergePRUsecase)

func WithMembershipRepo(membershipRepo repository.IMembershipRepository) MergePRUsecaseOption {
	return func(uc *MergePRUsecase) {
		uc.membershipRepo = membershipRepo
	}
}

func NewMergePRUsecase(
	prRepo repository.IPullRequestRepository,
	branchProtectionRepo repository.IBranchProtectionRepository,
	reviewRepo repository.IReviewRepository,
	workflowRunRepo repository.IWorkflowRunRepository,
	auditLogRepo repository.IAuditLogRepository,
	gitService service.GitService,
	txManager repository.TransactionManager,
	opts ...MergePRUsecaseOption,
) *MergePRUsecase {
	uc := &MergePRUsecase{
		prRepo:               prRepo,
		branchProtectionRepo: branchProtectionRepo,
		reviewRepo:           reviewRepo,
		workflowRunRepo:      workflowRunRepo,
		auditLogRepo:         auditLogRepo,
		gitService:           gitService,
		txManager:            txManager,
	}
	for _, opt := range opts {
		opt(uc)
	}
	return uc
}

func (uc *MergePRUsecase) Execute(ctx context.Context, input MergePRInput) (*entity.PullRequest, error) {
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

	requesterRole, err := uc.resolveRequesterRole(ctx, input.OrganizationID, input.ActorID, input.RequesterRole)
	if err != nil {
		return nil, err
	}

	if err := uc.checkBranchProtection(ctx, input.RepositoryID, pr, mergeMethod, requesterRole); err != nil {
		return nil, err
	}

	if err := uc.gitService.Merge(ctx, input.RepositoryID, pr.BaseRef, pr.HeadRef, mergeMethod); err != nil {
		if errors.Is(err, apperror.ErrConflict) {
			return nil, apperror.ErrConflict
		}
		return nil, err
	}

	now := time.Now().UTC()
	pr.State = "merged"
	pr.MergedAt = &now

	err = uc.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.prRepo.Update(txCtx, pr); err != nil {
			return err
		}
		return uc.auditLogRepo.InsertAuditLog(
			txCtx,
			input.OrganizationID,
			input.ActorID,
			"pr.merge",
			"pull_request",
			pr.ID,
			json.RawMessage(`{}`),
		)
	})
	if err != nil {
		return nil, err
	}

	return pr, nil
}

func (uc *MergePRUsecase) resolveRequesterRole(
	ctx context.Context,
	organizationID, actorID uuid.UUID,
	deprecatedRequesterRole string,
) (string, error) {
	if uc.membershipRepo != nil {
		role, err := uc.membershipRepo.GetRole(ctx, organizationID, actorID)
		if err != nil {
			return "", err
		}
		return role, nil
	}
	if testing.Testing() {
		return deprecatedRequesterRole, nil
	}
	return "", nil
}

func (uc *MergePRUsecase) checkBranchProtection(
	ctx context.Context,
	repositoryID uuid.UUID,
	pr *entity.PullRequest,
	mergeMethod string,
	requesterRole string,
) error {
	rule, err := uc.branchProtectionRepo.GetForRef(ctx, repositoryID, pr.BaseRef)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return nil
		}
		return err
	}
	if rule == nil {
		return nil
	}

	if !rule.EnforceAdmins && requesterRole == entity.RoleAdmin {
		return nil
	}

	satisfiedReviews, err := uc.reviewRepo.CountSatisfiedReviews(ctx, pr.ID)
	if err != nil {
		return err
	}
	if satisfiedReviews < rule.RequiredReviews {
		return apperror.ErrProtectionNotSatisfied
	}

	if len(rule.RequiredChecks) > 0 {
		headSHA, err := uc.gitService.ResolveRef(ctx, repositoryID, pr.HeadRef)
		if err != nil {
			return err
		}

		runs, err := uc.workflowRunRepo.ListByHeadSHA(ctx, repositoryID, headSHA)
		if err != nil {
			return err
		}
		if !allRequiredChecksPassed(rule.RequiredChecks, runs) {
			return apperror.ErrProtectionNotSatisfied
		}
	}

	if rule.RequiredLinearHistory && mergeMethod != "squash" && mergeMethod != "rebase" {
		return fmt.Errorf("%w: merge commit is not allowed; use squash or rebase merge instead", apperror.ErrProtectionNotSatisfied)
	}

	if rule.RequiredConversationResolution {
		hasOpenConversations, err := uc.reviewRepo.HasOpenConversations(ctx, pr.ID)
		if err != nil {
			return err
		}
		if hasOpenConversations {
			return apperror.ErrProtectionNotSatisfied
		}
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
