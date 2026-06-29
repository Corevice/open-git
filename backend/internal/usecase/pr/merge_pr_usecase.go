package pr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/service"
	"github.com/google/uuid"
)

const mergeMethodMerge = "merge"

type MergePRInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	ActorID        uuid.UUID // membership user ID; passed as userID to IMembershipRepository.GetRole
	Number         int
	MergeMethod    string
	RequesterRole  string // optional; when empty, resolved via membershipRepo
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

func NewMergePRUsecase(
	prRepo repository.IPullRequestRepository,
	branchProtectionRepo repository.IBranchProtectionRepository,
	membershipRepo repository.IMembershipRepository,
	reviewRepo repository.IReviewRepository,
	workflowRunRepo repository.IWorkflowRunRepository,
	auditLogRepo repository.IAuditLogRepository,
	gitService service.GitService,
	txManager repository.TransactionManager,
) *MergePRUsecase {
	if membershipRepo == nil {
		panic("membershipRepo is required")
	}
	return &MergePRUsecase{
		prRepo:               prRepo,
		branchProtectionRepo: branchProtectionRepo,
		membershipRepo:       membershipRepo,
		reviewRepo:           reviewRepo,
		workflowRunRepo:      workflowRunRepo,
		auditLogRepo:         auditLogRepo,
		gitService:           gitService,
		txManager:            txManager,
	}
}

func (uc *MergePRUsecase) Execute(ctx context.Context, input MergePRInput) (*entity.PullRequest, error) {
	pr, err := uc.prRepo.GetByNumber(ctx, input.RepositoryID, input.Number)
	if err != nil {
		return nil, err
	}
	if pr.State == "merged" {
		return nil, apperror.ErrAlreadyMerged
	}

	mergeMethod := normalizeMergeMethod(input.MergeMethod)

	requesterRole := input.RequesterRole
	if requesterRole == "" {
		var err error
		requesterRole, err = uc.resolveRequesterRole(ctx, input.OrganizationID, input.ActorID)
		if err != nil {
			return nil, err
		}
	}

	if err := uc.checkBranchProtection(
		ctx,
		input.OrganizationID,
		input.ActorID,
		input.RepositoryID,
		pr,
		mergeMethod,
		requesterRole,
	); err != nil {
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

func normalizeMergeMethod(method string) string {
	if method == "" {
		return mergeMethodMerge
	}
	return method
}

func mergeMethodCreatesMergeCommit(method string) bool {
	return method == mergeMethodMerge
}

func (uc *MergePRUsecase) resolveRequesterRole(
	ctx context.Context,
	organizationID, actorID uuid.UUID,
) (string, error) {
	// GetRole argument order is (organizationID, userID) per IMembershipRepository.
	role, err := uc.membershipRepo.GetRole(ctx, organizationID, actorID)
	if errors.Is(err, domain.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return role, nil
}

func (uc *MergePRUsecase) checkBranchProtection(
	ctx context.Context,
	organizationID, actorID, repositoryID uuid.UUID,
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

	// When enforce_admins is false, org admins may bypass every branch-protection
	// check (required reviews, required checks, linear history, conversation
	// resolution, etc.). This mirrors GitHub-style admin bypass and is audited.
	if !rule.EnforceAdmins && requesterRole == entity.RoleAdmin {
		if err := uc.logAdminProtectionBypass(ctx, organizationID, actorID, pr.ID); err != nil {
			return err
		}
		return nil
	}

	if rule.RequiredLinearHistory && mergeMethodCreatesMergeCommit(mergeMethod) {
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

	return nil
}

func (uc *MergePRUsecase) logAdminProtectionBypass(
	ctx context.Context,
	organizationID, actorID, pullRequestID uuid.UUID,
) error {
	if uc.auditLogRepo == nil {
		return nil
	}
	return uc.auditLogRepo.InsertAuditLog(
		ctx,
		organizationID,
		actorID,
		"pr.merge.admin_bypass",
		"branch_protection",
		pullRequestID,
		json.RawMessage(`{}`),
	)
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
