package pr_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	prusecase "github.com/open-git/backend/internal/usecase/pr"
)

type trackingReviewRepo struct {
	reviews []*entity.Review
}

func (m *trackingReviewRepo) Create(_ context.Context, review *entity.Review) error {
	m.reviews = append(m.reviews, review)
	return nil
}

func (m *trackingReviewRepo) GetByID(_ context.Context, _ uuid.UUID) (*entity.Review, error) {
	return nil, errors.New("review not found")
}

func (m *trackingReviewRepo) ListByPR(_ context.Context, _ uuid.UUID) ([]*entity.Review, error) {
	return m.reviews, nil
}

func (m *trackingReviewRepo) CountSatisfiedReviews(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (m *trackingReviewRepo) HasBlockingReviews(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}

type noMembershipRepo struct{}

func (noMembershipRepo) Add(_ context.Context, _ *entity.Membership) error {
	return nil
}

func (noMembershipRepo) GetRole(_ context.Context, _ uuid.UUID, _ uuid.UUID) (string, error) {
	return "", domain.ErrNotFound
}

func (noMembershipRepo) ListByOrg(_ context.Context, _ uuid.UUID, _, _ int) ([]*entity.Membership, error) {
	return nil, nil
}

func (noMembershipRepo) UpdateRole(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) error {
	return nil
}

func (noMembershipRepo) Remove(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

func newOpenPRForReview() (*entity.PullRequest, uuid.UUID) {
	authorID := uuid.New()
	pr := &entity.PullRequest{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		RepositoryID:   uuid.New(),
		Number:         1,
		HeadRef:        "feature",
		BaseRef:        "main",
		HeadSHA:        "abc123",
		State:          entity.PullRequestStateOpen,
		AuthorID:       authorID,
	}
	return pr, authorID
}

func TestCreateReviewApproveSuccess(t *testing.T) {
	pr, authorID := newOpenPRForReview()
	reviewerID := uuid.New()
	auditRepo := &mockAuditLogRepo{}

	uc := prusecase.NewCreateReviewUsecase(
		&mockPullRequestRepo{prs: []*entity.PullRequest{pr}},
		&trackingReviewRepo{},
		auditRepo,
		&mockMembershipRepo{},
	)

	review, err := uc.Execute(context.Background(), prusecase.CreateReviewInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
		Number:         pr.Number,
		ActorID:        reviewerID,
		Event:          "APPROVE",
		Body:           "looks good",
	})
	if err != nil {
		t.Fatalf("create review: %v", err)
	}
	if review.State != entity.ReviewStateApproved {
		t.Fatalf("expected state %q, got %q", entity.ReviewStateApproved, review.State)
	}
	if review.ReviewerID != reviewerID {
		t.Fatalf("expected reviewer %v, got %v", reviewerID, review.ReviewerID)
	}
	if review.ReviewerID == authorID {
		t.Fatal("reviewer should not be the PR author")
	}
	if len(auditRepo.calls) != 1 {
		t.Fatalf("expected 1 audit log call, got %d", len(auditRepo.calls))
	}
	if auditRepo.calls[0].action != "pr.review.approve" || auditRepo.calls[0].targetType != "pull_request" {
		t.Fatalf("unexpected audit payload: %+v", auditRepo.calls[0])
	}
}

func TestCreateReviewCommentSuccess(t *testing.T) {
	pr, authorID := newOpenPRForReview()
	auditRepo := &mockAuditLogRepo{}

	uc := prusecase.NewCreateReviewUsecase(
		&mockPullRequestRepo{prs: []*entity.PullRequest{pr}},
		&trackingReviewRepo{},
		auditRepo,
		&mockMembershipRepo{},
	)

	review, err := uc.Execute(context.Background(), prusecase.CreateReviewInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
		Number:         pr.Number,
		ActorID:        authorID,
		Event:          "COMMENT",
		Body:           "note to self",
	})
	if err != nil {
		t.Fatalf("create review: %v", err)
	}
	if review.State != entity.ReviewStateCommented {
		t.Fatalf("expected state %q, got %q", entity.ReviewStateCommented, review.State)
	}
	if len(auditRepo.calls) != 0 {
		t.Fatalf("expected no audit log calls, got %d", len(auditRepo.calls))
	}
}

func TestCreateReviewRequestChangesSuccess(t *testing.T) {
	pr, _ := newOpenPRForReview()
	reviewerID := uuid.New()
	auditRepo := &mockAuditLogRepo{}

	uc := prusecase.NewCreateReviewUsecase(
		&mockPullRequestRepo{prs: []*entity.PullRequest{pr}},
		&trackingReviewRepo{},
		auditRepo,
		&mockMembershipRepo{},
	)

	review, err := uc.Execute(context.Background(), prusecase.CreateReviewInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
		Number:         pr.Number,
		ActorID:        reviewerID,
		Event:          "REQUEST_CHANGES",
		Body:           "please fix",
	})
	if err != nil {
		t.Fatalf("create review: %v", err)
	}
	if review.State != entity.ReviewStateChangesRequested {
		t.Fatalf("expected state %q, got %q", entity.ReviewStateChangesRequested, review.State)
	}
	if len(auditRepo.calls) != 1 {
		t.Fatalf("expected 1 audit log call, got %d", len(auditRepo.calls))
	}
	if auditRepo.calls[0].action != "pr.review.request_changes" {
		t.Fatalf("unexpected audit action %q", auditRepo.calls[0].action)
	}
}

func TestCreateReviewInvalidEvent(t *testing.T) {
	pr, _ := newOpenPRForReview()

	uc := prusecase.NewCreateReviewUsecase(
		&mockPullRequestRepo{prs: []*entity.PullRequest{pr}},
		&trackingReviewRepo{},
		&mockAuditLogRepo{},
		&mockMembershipRepo{},
	)

	_, err := uc.Execute(context.Background(), prusecase.CreateReviewInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
		Number:         pr.Number,
		ActorID:        uuid.New(),
		Event:          "DISMISS",
	})
	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestCreateReviewClosedPR(t *testing.T) {
	pr, _ := newOpenPRForReview()
	pr.State = entity.PullRequestStateClosed

	uc := prusecase.NewCreateReviewUsecase(
		&mockPullRequestRepo{prs: []*entity.PullRequest{pr}},
		&trackingReviewRepo{},
		&mockAuditLogRepo{},
		&mockMembershipRepo{},
	)

	_, err := uc.Execute(context.Background(), prusecase.CreateReviewInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
		Number:         pr.Number,
		ActorID:        uuid.New(),
		Event:          "APPROVE",
	})
	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestCreateReviewSelfReview(t *testing.T) {
	pr, authorID := newOpenPRForReview()

	uc := prusecase.NewCreateReviewUsecase(
		&mockPullRequestRepo{prs: []*entity.PullRequest{pr}},
		&trackingReviewRepo{},
		&mockAuditLogRepo{},
		&mockMembershipRepo{},
	)

	_, err := uc.Execute(context.Background(), prusecase.CreateReviewInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
		Number:         pr.Number,
		ActorID:        authorID,
		Event:          "APPROVE",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCreateReviewNoOrgMembership(t *testing.T) {
	pr, _ := newOpenPRForReview()

	uc := prusecase.NewCreateReviewUsecase(
		&mockPullRequestRepo{prs: []*entity.PullRequest{pr}},
		&trackingReviewRepo{},
		&mockAuditLogRepo{},
		noMembershipRepo{},
	)

	_, err := uc.Execute(context.Background(), prusecase.CreateReviewInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
		Number:         pr.Number,
		ActorID:        uuid.New(),
		Event:          "COMMENT",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}
