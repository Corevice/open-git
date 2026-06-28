package pr_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
	prusecase "github.com/open-git/backend/internal/usecase/pr"
)

type createReviewMockRepoRepo struct {
	repo *entity.Repository
}

func (m *createReviewMockRepoRepo) Create(_ context.Context, _ *entity.Repository) error {
	return nil
}

func (m *createReviewMockRepoRepo) GetByOwnerAndName(_ context.Context, _ uuid.UUID, _ string) (*entity.Repository, error) {
	return nil, nil
}

func (m *createReviewMockRepoRepo) GetByOwnerLoginAndName(_ context.Context, _, _ string) (*entity.Repository, error) {
	return m.repo, nil
}

func (m *createReviewMockRepoRepo) ListByOrg(_ context.Context, _ uuid.UUID, _, _ int) ([]*entity.Repository, error) {
	return nil, nil
}

func (m *createReviewMockRepoRepo) CountByOrg(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (m *createReviewMockRepoRepo) ListByOwner(_ context.Context, _ uuid.UUID, _, _ int) ([]*entity.Repository, error) {
	return nil, nil
}

func (m *createReviewMockRepoRepo) CountByOwner(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (m *createReviewMockRepoRepo) UpdateVisibility(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func (m *createReviewMockRepoRepo) UpdateName(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func (m *createReviewMockRepoRepo) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

type createReviewMockPRRepo struct {
	pr    *entity.PullRequest
	err   error
}

func (m *createReviewMockPRRepo) Create(_ context.Context, _ *entity.PullRequest) error {
	return nil
}

func (m *createReviewMockPRRepo) GetByNumber(_ context.Context, _ uuid.UUID, _ int) (*entity.PullRequest, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.pr, nil
}

func (m *createReviewMockPRRepo) GetByID(_ context.Context, _ uuid.UUID) (*entity.PullRequest, error) {
	return nil, nil
}

func (m *createReviewMockPRRepo) ListByRepo(_ context.Context, _ uuid.UUID, _ repository.ListPullRequestsFilter) ([]*entity.PullRequest, int, error) {
	return nil, 0, nil
}

func (m *createReviewMockPRRepo) NextNumber(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (m *createReviewMockPRRepo) Update(_ context.Context, _ *entity.PullRequest) error {
	return nil
}

func (m *createReviewMockPRRepo) SetMerged(_ context.Context, _ uuid.UUID, _ time.Time, _ uuid.UUID, _ string) error {
	return nil
}

type createReviewMockReviewRepo struct {
	createErr error
	created   *entity.Review
}

func (m *createReviewMockReviewRepo) Create(_ context.Context, review *entity.Review) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.created = review
	return nil
}

func (m *createReviewMockReviewRepo) GetByID(_ context.Context, _ uuid.UUID) (*entity.Review, error) {
	return nil, nil
}

func (m *createReviewMockReviewRepo) ListByPR(_ context.Context, _ uuid.UUID) ([]*entity.Review, error) {
	return nil, nil
}

func (m *createReviewMockReviewRepo) CountSatisfiedReviews(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (m *createReviewMockReviewRepo) HasBlockingReviews(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}

func TestCreateReviewApprovedEvent(t *testing.T) {
	repoID := uuid.New()
	prID := uuid.New()
	actorID := uuid.New()
	reviewRepo := &createReviewMockReviewRepo{}

	uc := prusecase.NewCreateReviewUsecase(
		&createReviewMockRepoRepo{
			repo: &entity.Repository{ID: repoID},
		},
		&createReviewMockPRRepo{
			pr: &entity.PullRequest{
				ID:           prID,
				RepositoryID: repoID,
				Number:       1,
				HeadSHA:      "abc123",
			},
		},
		reviewRepo,
	)

	review, err := uc.Execute(context.Background(), "alice", "demo", prusecase.CreateReviewInput{
		PRNumber: 1,
		ActorID:  actorID,
		Body:     "looks good",
		Event:    "approved",
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if review == nil {
		t.Fatal("expected review, got nil")
	}
	if review.State != entity.ReviewStateApproved {
		t.Fatalf("expected state %q, got %q", entity.ReviewStateApproved, review.State)
	}
	if review.ReviewerID != actorID {
		t.Fatalf("expected reviewer %v, got %v", actorID, review.ReviewerID)
	}
	if reviewRepo.created == nil {
		t.Fatal("expected reviewRepo.Create to be called")
	}
}

func TestCreateReviewInvalidEvent(t *testing.T) {
	uc := prusecase.NewCreateReviewUsecase(
		&createReviewMockRepoRepo{
			repo: &entity.Repository{ID: uuid.New()},
		},
		&createReviewMockPRRepo{
			pr: &entity.PullRequest{ID: uuid.New(), Number: 1},
		},
		&createReviewMockReviewRepo{},
	)

	_, err := uc.Execute(context.Background(), "alice", "demo", prusecase.CreateReviewInput{
		PRNumber: 1,
		ActorID:  uuid.New(),
		Body:     "looks good",
		Event:    "invalid",
	})
	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestCreateReviewPRNotFound(t *testing.T) {
	uc := prusecase.NewCreateReviewUsecase(
		&createReviewMockRepoRepo{
			repo: &entity.Repository{ID: uuid.New()},
		},
		&createReviewMockPRRepo{
			pr: &entity.PullRequest{ID: uuid.New(), Number: 1},
		},
		&createReviewMockReviewRepo{
			createErr: apperror.ErrNotFound,
		},
	)

	_, err := uc.Execute(context.Background(), "alice", "demo", prusecase.CreateReviewInput{
		PRNumber: 1,
		ActorID:  uuid.New(),
		Body:     "looks good",
		Event:    "approved",
	})
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
