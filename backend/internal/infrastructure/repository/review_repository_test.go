package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func seedReviewFixtures(t *testing.T, db *sqlx.DB) (prID, reviewerID uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	orgID := uuid.New()
	repoID := uuid.New()
	authorID := uuid.New()
	reviewerID = uuid.New()

	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("exec %q: %v", query, err)
		}
	}

	exec(`INSERT INTO organizations (id, login, name) VALUES (?, ?, ?)`, orgID, "acme", "Acme")
	exec(`INSERT INTO users (id, login, email, password_hash) VALUES (?, ?, ?, ?)`, authorID, "alice", "alice@example.com", "hash")
	exec(`INSERT INTO users (id, login, email, password_hash) VALUES (?, ?, ?, ?)`, reviewerID, "bob", "bob@example.com", "hash")
	exec(`INSERT INTO repositories (id, organization_id, owner_id, name) VALUES (?, ?, ?, ?)`, repoID, orgID, authorID, "demo")

	prRepo := repository.NewPullRequestRepository(db)
	prID = uuid.New()
	if err := prRepo.Create(ctx, &entity.PullRequest{
		ID:             prID,
		OrganizationID: orgID,
		RepositoryID:   repoID,
		Number:         1,
		Title:          "Review me",
		HeadRef:        "feature",
		BaseRef:        "main",
		State:          entity.PullRequestStateOpen,
		AuthorID:       authorID,
	}); err != nil {
		t.Fatalf("create pull request: %v", err)
	}

	return prID, reviewerID
}

func TestReviewRepository_CreateListByPR(t *testing.T) {
	db := openTestDB(t)
	prID, reviewerID := seedReviewFixtures(t, db)
	repo := repository.NewReviewRepository(db)
	ctx := context.Background()

	submittedAt := time.Now().UTC()
	review := &entity.Review{
		ID:            uuid.New(),
		PullRequestID: prID,
		ReviewerID:    reviewerID,
		State:         entity.ReviewStateCommented,
		Body:          "Looks good",
		CommitSHA:     "abc123",
		SubmittedAt:   &submittedAt,
	}

	if err := repo.Create(ctx, review); err != nil {
		t.Fatalf("Create: %v", err)
	}

	reviews, err := repo.ListByPR(ctx, prID)
	if err != nil {
		t.Fatalf("ListByPR: %v", err)
	}
	if len(reviews) != 1 {
		t.Fatalf("expected 1 review, got %d", len(reviews))
	}
	if reviews[0].State != entity.ReviewStateCommented {
		t.Fatalf("unexpected review state %q", reviews[0].State)
	}
}

func TestReviewRepository_CountSatisfiedReviews(t *testing.T) {
	db := openTestDB(t)
	prID, reviewerID := seedReviewFixtures(t, db)
	repo := repository.NewReviewRepository(db)
	ctx := context.Background()

	count, err := repo.CountSatisfiedReviews(ctx, prID)
	if err != nil {
		t.Fatalf("CountSatisfiedReviews empty: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 satisfied reviews, got %d", count)
	}

	submittedAt := time.Now().UTC()
	if err := repo.Create(ctx, &entity.Review{
		ID:            uuid.New(),
		PullRequestID: prID,
		ReviewerID:    reviewerID,
		State:         entity.ReviewStateApproved,
		SubmittedAt:   &submittedAt,
	}); err != nil {
		t.Fatalf("Create approved review: %v", err)
	}

	count, err = repo.CountSatisfiedReviews(ctx, prID)
	if err != nil {
		t.Fatalf("CountSatisfiedReviews approved: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 satisfied review, got %d", count)
	}
}

func TestReviewRepository_HasBlockingReviews(t *testing.T) {
	db := openTestDB(t)
	prID, reviewerID := seedReviewFixtures(t, db)
	repo := repository.NewReviewRepository(db)
	ctx := context.Background()

	blocking, err := repo.HasBlockingReviews(ctx, prID)
	if err != nil {
		t.Fatalf("HasBlockingReviews empty: %v", err)
	}
	if blocking {
		t.Fatal("expected no blocking reviews")
	}

	submittedAt := time.Now().UTC()
	if err := repo.Create(ctx, &entity.Review{
		ID:            uuid.New(),
		PullRequestID: prID,
		ReviewerID:    reviewerID,
		State:         entity.ReviewStateChangesRequested,
		SubmittedAt:   &submittedAt,
	}); err != nil {
		t.Fatalf("Create changes requested review: %v", err)
	}

	blocking, err = repo.HasBlockingReviews(ctx, prID)
	if err != nil {
		t.Fatalf("HasBlockingReviews blocking: %v", err)
	}
	if !blocking {
		t.Fatal("expected blocking review")
	}
}
