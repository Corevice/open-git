package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func seedPullRequestFixtures(t *testing.T, db *sqlx.DB) (orgID, repoID, repo2ID, authorID uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	orgID = uuid.New()
	repoID = uuid.New()
	repo2ID = uuid.New()
	authorID = uuid.New()

	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("exec %q: %v", query, err)
		}
	}

	exec(`INSERT INTO organizations (id, login, name) VALUES (?, ?, ?)`, orgID, "acme", "Acme")
	exec(`INSERT INTO users (id, login, email, password_hash) VALUES (?, ?, ?, ?)`, authorID, "alice", "alice@example.com", "hash")
	exec(`INSERT INTO repositories (id, organization_id, owner_id, name) VALUES (?, ?, ?, ?)`, repoID, orgID, authorID, "demo")
	exec(`INSERT INTO repositories (id, organization_id, owner_id, name) VALUES (?, ?, ?, ?)`, repo2ID, orgID, authorID, "other")
	return orgID, repoID, repo2ID, authorID
}

func TestPullRequestRepository_CreateGetByNumber(t *testing.T) {
	db := openTestDB(t)
	orgID, repoID, _, authorID := seedPullRequestFixtures(t, db)
	repo := repository.NewPullRequestRepository(db)

	pr := &entity.PullRequest{
		ID:             uuid.New(),
		OrganizationID: orgID,
		RepositoryID:   repoID,
		Number:         1,
		Title:          "Add feature",
		Body:           "body text",
		HeadRef:        "feature",
		BaseRef:        "main",
		HeadSHA:        "abc123",
		BaseSHA:        "def456",
		State:          entity.PullRequestStateOpen,
		AuthorID:       authorID,
	}

	if err := repo.Create(context.Background(), pr); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByNumber(context.Background(), repoID, 1)
	if err != nil {
		t.Fatalf("GetByNumber: %v", err)
	}
	if got == nil {
		t.Fatal("expected pull request, got nil")
	}
	if got.Title != pr.Title || got.Body != pr.Body || got.HeadRef != pr.HeadRef || got.BaseRef != pr.BaseRef {
		t.Fatalf("unexpected pull request: %+v", got)
	}
}

func TestPullRequestRepository_ListByRepoFiltersState(t *testing.T) {
	db := openTestDB(t)
	orgID, repoID, _, authorID := seedPullRequestFixtures(t, db)
	repo := repository.NewPullRequestRepository(db)
	ctx := context.Background()

	openPR := &entity.PullRequest{
		ID: uuid.New(), OrganizationID: orgID, RepositoryID: repoID, Number: 1,
		Title: "Open PR", HeadRef: "feature", BaseRef: "main", State: entity.PullRequestStateOpen, AuthorID: authorID,
	}
	closedPR := &entity.PullRequest{
		ID: uuid.New(), OrganizationID: orgID, RepositoryID: repoID, Number: 2,
		Title: "Closed PR", HeadRef: "fix", BaseRef: "main", State: entity.PullRequestStateClosed, AuthorID: authorID,
	}
	for _, pr := range []*entity.PullRequest{openPR, closedPR} {
		if err := repo.Create(ctx, pr); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	openList, openTotal, err := repo.ListByRepo(ctx, repoID, domainrepo.ListPullRequestsFilter{State: entity.PullRequestStateOpen})
	if err != nil {
		t.Fatalf("ListByRepo open: %v", err)
	}
	if openTotal != 1 || len(openList) != 1 || openList[0].State != entity.PullRequestStateOpen {
		t.Fatalf("expected 1 open PR, got total=%d len=%d", openTotal, len(openList))
	}

	allList, allTotal, err := repo.ListByRepo(ctx, repoID, domainrepo.ListPullRequestsFilter{State: "all"})
	if err != nil {
		t.Fatalf("ListByRepo all: %v", err)
	}
	if allTotal != 2 || len(allList) != 2 {
		t.Fatalf("expected 2 PRs for state=all, got total=%d len=%d", allTotal, len(allList))
	}
}

func TestPullRequestRepository_NextNumberIncrements(t *testing.T) {
	db := openTestDB(t)
	orgID, repoID, repo2ID, authorID := seedPullRequestFixtures(t, db)
	repo := repository.NewPullRequestRepository(db)
	ctx := context.Background()

	first, err := repo.NextNumber(ctx, repoID)
	if err != nil {
		t.Fatalf("NextNumber first: %v", err)
	}
	if first != 1 {
		t.Fatalf("expected first number 1, got %d", first)
	}

	if err := repo.Create(ctx, &entity.PullRequest{
		ID: uuid.New(), OrganizationID: orgID, RepositoryID: repoID, Number: first,
		Title: "First", HeadRef: "a", BaseRef: "main", State: entity.PullRequestStateOpen, AuthorID: authorID,
	}); err != nil {
		t.Fatalf("Create first: %v", err)
	}

	second, err := repo.NextNumber(ctx, repoID)
	if err != nil {
		t.Fatalf("NextNumber second: %v", err)
	}
	if second != 2 {
		t.Fatalf("expected second number 2, got %d", second)
	}

	otherFirst, err := repo.NextNumber(ctx, repo2ID)
	if err != nil {
		t.Fatalf("NextNumber other repo: %v", err)
	}
	if otherFirst != 1 {
		t.Fatalf("expected independent sequence 1 for other repo, got %d", otherFirst)
	}
}

func TestPullRequestRepository_SetMerged(t *testing.T) {
	db := openTestDB(t)
	orgID, repoID, _, authorID := seedPullRequestFixtures(t, db)
	repo := repository.NewPullRequestRepository(db)
	ctx := context.Background()

	pr := &entity.PullRequest{
		ID: uuid.New(), OrganizationID: orgID, RepositoryID: repoID, Number: 1,
		Title: "Merge me", HeadRef: "feature", BaseRef: "main", State: entity.PullRequestStateOpen, AuthorID: authorID,
	}
	if err := repo.Create(ctx, pr); err != nil {
		t.Fatalf("Create: %v", err)
	}

	mergedAt := time.Now().UTC()
	mergeSHA := "merge-sha-123"
	if err := repo.SetMerged(ctx, pr.ID, mergedAt, authorID, mergeSHA); err != nil {
		t.Fatalf("SetMerged: %v", err)
	}

	got, err := repo.GetByID(ctx, pr.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.State != entity.PullRequestStateMerged {
		t.Fatalf("expected merged state, got %q", got.State)
	}
	if got.MergeCommitSHA != mergeSHA {
		t.Fatalf("expected merge commit sha %q, got %q", mergeSHA, got.MergeCommitSHA)
	}
	if got.MergedBy == nil || *got.MergedBy != authorID {
		t.Fatalf("expected merged_by %s, got %+v", authorID, got.MergedBy)
	}
}

func TestPullRequestRepository_UpdateTitle(t *testing.T) {
	db := openTestDB(t)
	orgID, repoID, _, authorID := seedPullRequestFixtures(t, db)
	repo := repository.NewPullRequestRepository(db)
	ctx := context.Background()

	pr := &entity.PullRequest{
		ID: uuid.New(), OrganizationID: orgID, RepositoryID: repoID, Number: 1,
		Title: "Original", HeadRef: "feature", BaseRef: "main", State: entity.PullRequestStateOpen, AuthorID: authorID,
	}
	if err := repo.Create(ctx, pr); err != nil {
		t.Fatalf("Create: %v", err)
	}

	pr.Title = "Updated title"
	if err := repo.Update(ctx, pr); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.GetByNumber(ctx, repoID, 1)
	if err != nil {
		t.Fatalf("GetByNumber: %v", err)
	}
	if got.Title != "Updated title" {
		t.Fatalf("expected updated title, got %q", got.Title)
	}
}
