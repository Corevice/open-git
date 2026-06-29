package pr_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	prusecase "github.com/open-git/backend/internal/usecase/pr"
)

type listReviewsRepo struct {
	reviews []*entity.Review
}

func (m *listReviewsRepo) Create(_ context.Context, _ *entity.Review) error {
	return nil
}

func (m *listReviewsRepo) GetByID(_ context.Context, _ uuid.UUID) (*entity.Review, error) {
	return nil, nil
}

func (m *listReviewsRepo) ListByPR(_ context.Context, _ uuid.UUID) ([]*entity.Review, error) {
	return m.reviews, nil
}

func (m *listReviewsRepo) CountSatisfiedReviews(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (m *listReviewsRepo) HasBlockingReviews(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}

func TestListReviewsEmpty(t *testing.T) {
	pr := newOpenPR()

	uc := prusecase.NewListReviewsUsecase(
		&mockPullRequestRepo{prs: []*entity.PullRequest{pr}},
		&listReviewsRepo{reviews: nil},
	)

	reviews, err := uc.Execute(context.Background(), prusecase.ListReviewsInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
		Number:         pr.Number,
	})
	if err != nil {
		t.Fatalf("list reviews: %v", err)
	}
	if reviews == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(reviews) != 0 {
		t.Fatalf("expected 0 reviews, got %d", len(reviews))
	}
}

func TestListReviewsMultiple(t *testing.T) {
	pr := newOpenPR()
	reviews := []*entity.Review{
		{ID: uuid.New(), PullRequestID: pr.ID, State: entity.ReviewStateApproved},
		{ID: uuid.New(), PullRequestID: pr.ID, State: entity.ReviewStateCommented},
		{ID: uuid.New(), PullRequestID: pr.ID, State: entity.ReviewStateChangesRequested},
	}

	uc := prusecase.NewListReviewsUsecase(
		&mockPullRequestRepo{prs: []*entity.PullRequest{pr}},
		&listReviewsRepo{reviews: reviews},
	)

	got, err := uc.Execute(context.Background(), prusecase.ListReviewsInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
		Number:         pr.Number,
	})
	if err != nil {
		t.Fatalf("list reviews: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 reviews, got %d", len(got))
	}
}
