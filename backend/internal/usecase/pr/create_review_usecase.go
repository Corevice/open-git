package pr

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
)

var reviewEventToState = map[string]string{
	"APPROVE":         entity.ReviewStateApproved,
	"REQUEST_CHANGES": entity.ReviewStateChangesRequested,
	"COMMENT":         entity.ReviewStateCommented,
}

type CreateReviewInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	Number         int
	ActorID        uuid.UUID
	Event          string
	Body           string
}

type CreateReviewUsecase struct {
	prRepo         repository.IPullRequestRepository
	reviewRepo     repository.IReviewRepository
	auditLogRepo   repository.IAuditLogRepository
	membershipRepo repository.IMembershipRepository
}

func NewCreateReviewUsecase(
	prRepo repository.IPullRequestRepository,
	reviewRepo repository.IReviewRepository,
	auditLogRepo repository.IAuditLogRepository,
	membershipRepo repository.IMembershipRepository,
) *CreateReviewUsecase {
	return &CreateReviewUsecase{
		prRepo:         prRepo,
		reviewRepo:     reviewRepo,
		auditLogRepo:   auditLogRepo,
		membershipRepo: membershipRepo,
	}
}

func (uc *CreateReviewUsecase) Execute(ctx context.Context, input CreateReviewInput) (*entity.Review, error) {
	if err := uc.checkActorAccess(ctx, input.OrganizationID, input.ActorID); err != nil {
		return nil, err
	}

	pr, err := uc.prRepo.GetByNumber(ctx, input.RepositoryID, input.Number)
	if err != nil {
		return nil, err
	}
	if pr.OrganizationID != input.OrganizationID {
		return nil, apperror.ErrNotFound
	}
	if pr.State == entity.PullRequestStateClosed || pr.State == entity.PullRequestStateMerged {
		return nil, apperror.ErrValidation
	}

	state, ok := reviewEventToState[strings.ToUpper(input.Event)]
	if !ok {
		return nil, apperror.ErrValidation
	}

	if input.ActorID == pr.AuthorID &&
		(state == entity.ReviewStateApproved || state == entity.ReviewStateChangesRequested) {
		return nil, domain.ErrForbidden
	}

	now := time.Now().UTC()
	review := &entity.Review{
		ID:            uuid.New(),
		PullRequestID: pr.ID,
		ReviewerID:    input.ActorID,
		State:         state,
		Body:          input.Body,
		CommitSHA:     pr.HeadSHA,
		SubmittedAt:   &now,
		CreatedAt:     now,
	}

	if err := uc.reviewRepo.Create(ctx, review); err != nil {
		return nil, err
	}

	if state == entity.ReviewStateApproved || state == entity.ReviewStateChangesRequested {
		action := "pr.review.approve"
		if state == entity.ReviewStateChangesRequested {
			action = "pr.review.request_changes"
		}
		if err := uc.auditLogRepo.Create(ctx, &entity.AuditLog{
			ID:             uuid.New(),
			OrganizationID: input.OrganizationID,
			ActorID:        input.ActorID,
			Action:         action,
			TargetType:     "pull_request",
			TargetID:       pr.ID.String(),
		}); err != nil {
			return nil, err
		}
	}

	return review, nil
}

func (uc *CreateReviewUsecase) checkActorAccess(ctx context.Context, organizationID, actorID uuid.UUID) error {
	// Personal repositories use the owner's user id as the organization id and
	// have no membership row, so the owner is authorized directly.
	if organizationID != uuid.Nil && organizationID == actorID {
		return nil
	}
	_, err := uc.membershipRepo.GetRole(ctx, organizationID, actorID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.ErrForbidden
	}
	return err
}
