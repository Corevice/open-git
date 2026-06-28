package pr

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	repointerface "github.com/open-git/backend/internal/repository"
)

type ReviewCommentInput struct {
	Path string
	Line int
	Body string
}

type CreateReviewInput struct {
	PRNumber int
	ActorID  uuid.UUID
	Body     string
	Event    string
	Comments []ReviewCommentInput
}

var allowedReviewEvents = map[string]string{
	"approved":          entity.ReviewStateApproved,
	"changes_requested": entity.ReviewStateChangesRequested,
	"commented":         entity.ReviewStateCommented,
}

type CreateReviewUsecase struct {
	repos      repointerface.IRepositoryRepository
	prRepo     domainrepo.IPullRequestRepository
	reviewRepo domainrepo.IReviewRepository
}

func NewCreateReviewUsecase(
	repos repointerface.IRepositoryRepository,
	prRepo domainrepo.IPullRequestRepository,
	reviewRepo domainrepo.IReviewRepository,
) *CreateReviewUsecase {
	return &CreateReviewUsecase{
		repos:      repos,
		prRepo:     prRepo,
		reviewRepo: reviewRepo,
	}
}

func (uc *CreateReviewUsecase) Execute(ctx context.Context, owner, repo string, input CreateReviewInput) (*entity.Review, error) {
	state, ok := allowedReviewEvents[input.Event]
	if !ok {
		return nil, apperror.ErrValidation
	}

	repository, err := uc.repos.GetByOwnerLoginAndName(ctx, owner, repo)
	if err != nil || repository == nil {
		return nil, apperror.ErrNotFound
	}

	pr, err := uc.prRepo.GetByNumber(ctx, repository.ID, input.PRNumber)
	if err != nil {
		return nil, err
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

	return review, nil
}
