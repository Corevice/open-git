package pr_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
	prusecase "github.com/open-git/backend/internal/usecase/pr"
)

type listReviewsMockRepoRepo struct {
	repo *entity.Repository
}

func (m *listReviewsMockRepoRepo) Create(_ context.Context, _ *entity.Repository) error {
	return nil
}

func (m *listReviewsMockRepoRepo) GetByOwnerAndName(_ context.Context, _ uuid.UUID, _ string) (*entity.Repository, error) {
	return nil, nil
}

func (m *listReviewsMockRepoRepo) GetByOwnerLoginAndName(_ context.Context, _, _ string) (*entity.Repository, error) {
	return m.repo, nil
}

func (m *listReviewsMockRepoRepo) ListByOrg(_ context.Context, _ uuid.UUID, _, _ int) ([]*entity.Repository, error) {
	return nil, nil
}

func (m *listReviewsMockRepoRepo) CountByOrg(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (m *listReviewsMockRepoRepo) ListByOwner(_ context.Context, _ uuid.UUID, _, _ int) ([]*entity.Repository, error) {
	return nil, nil
}

func (m *listReviewsMockRepoRepo) CountByOwner(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (m *listReviewsMockRepoRepo) UpdateVisibility(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func (m *listReviewsMockRepoRepo) UpdateName(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func (m *listReviewsMockRepoRepo) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

type listReviewsMockPRRepo struct {
	pr *entity.PullRequest
}

func (m *listReviewsMockPRRepo) Create(_ context.Context, _ *entity.PullRequest) error {
	return nil
}

func (m *listReviewsMockPRRepo) GetByNumber(_ context.Context, _ uuid.UUID, _ int) (*entity.PullRequest, error) {
	return m.pr, nil
}

func (m *listReviewsMockPRRepo) GetByID(_ context.Context, _ uuid.UUID) (*entity.PullRequest, error) {
	return nil, nil
}

func (m *listReviewsMockPRRepo) ListByRepo(_ context.Context, _ uuid.UUID, _ repository.ListPullRequestsFilter) ([]*entity.PullRequest, int, error) {
	return nil, 0, nil
}

func (m *listReviewsMockPRRepo) NextNumber(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (m *listReviewsMockPRRepo) Update(_ context.Context, _ *entity.PullRequest) error {
	return nil
}

func (m *listReviewsMockPRRepo) SetMerged(_ context.Context, _ uuid.UUID, _ time.Time, _ uuid.UUID, _ string) error {
	return nil
}

type listReviewsMockReviewRepo struct {
	reviews []*entity.Review
}

func (m *listReviewsMockReviewRepo) Create(_ context.Context, _ *entity.Review) error {
	return nil
}

func (m *listReviewsMockReviewRepo) GetByID(_ context.Context, _ uuid.UUID) (*entity.Review, error) {
	return nil, nil
}

func (m *listReviewsMockReviewRepo) ListByPR(_ context.Context, _ uuid.UUID) ([]*entity.Review, error) {
	return m.reviews, nil
}

func (m *listReviewsMockReviewRepo) CountSatisfiedReviews(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (m *listReviewsMockReviewRepo) HasBlockingReviews(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}

func TestListReviewsEmptyList(t *testing.T) {
	repoID := uuid.New()
	prID := uuid.New()

	uc := prusecase.NewListReviewsUsecase(
		&listReviewsMockRepoRepo{
			repo: &entity.Repository{ID: repoID},
		},
		&listReviewsMockPRRepo{
			pr: &entity.PullRequest{
				ID:           prID,
				RepositoryID: repoID,
				Number:       1,
			},
		},
		&listReviewsMockReviewRepo{reviews: nil},
	)

	reviews, err := uc.Execute(context.Background(), "alice", "demo", 1)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if reviews == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(reviews) != 0 {
		t.Fatalf("expected 0 reviews, got %d", len(reviews))
	}
}

func TestListReviewsPopulatedList(t *testing.T) {
	repoID := uuid.New()
	prID := uuid.New()
	expected := []*entity.Review{
		{ID: uuid.New(), PullRequestID: prID, State: entity.ReviewStateApproved},
		{ID: uuid.New(), PullRequestID: prID, State: entity.ReviewStateCommented},
	}

	uc := prusecase.NewListReviewsUsecase(
		&listReviewsMockRepoRepo{
			repo: &entity.Repository{ID: repoID},
		},
		&listReviewsMockPRRepo{
			pr: &entity.PullRequest{
				ID:           prID,
				RepositoryID: repoID,
				Number:       1,
			},
		},
		&listReviewsMockReviewRepo{reviews: expected},
	)

	reviews, err := uc.Execute(context.Background(), "alice", "demo", 1)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(reviews) != 2 {
		t.Fatalf("expected 2 reviews, got %d", len(reviews))
	}
	if reviews[0].ID != expected[0].ID || reviews[1].ID != expected[1].ID {
		t.Fatalf("expected reviews to pass through unchanged, got %+v", reviews)
	}
}
