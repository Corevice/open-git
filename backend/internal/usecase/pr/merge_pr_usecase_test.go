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
	"github.com/open-git/backend/internal/domain/service"
	prusecase "github.com/open-git/backend/internal/usecase/pr"
)

type mockPullRequestRepo struct {
	prs []*entity.PullRequest
}

func (m *mockPullRequestRepo) Create(_ context.Context, _ *entity.PullRequest) error {
	return nil
}

func (m *mockPullRequestRepo) GetByNumber(_ context.Context, _ uuid.UUID, number int) (*entity.PullRequest, error) {
	for _, pr := range m.prs {
		if pr.Number == number {
			return pr, nil
		}
	}
	return nil, errors.New("pull request not found")
}

func (m *mockPullRequestRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.PullRequest, error) {
	for _, pr := range m.prs {
		if pr.ID == id {
			return pr, nil
		}
	}
	return nil, errors.New("pull request not found")
}

func (m *mockPullRequestRepo) ListByRepo(_ context.Context, _ uuid.UUID, _ repository.ListPullRequestsFilter) ([]*entity.PullRequest, int, error) {
	return m.prs, len(m.prs), nil
}

func (m *mockPullRequestRepo) NextNumber(_ context.Context, _ uuid.UUID) (int, error) {
	return 1, nil
}

func (m *mockPullRequestRepo) Update(_ context.Context, pr *entity.PullRequest) error {
	for i, existing := range m.prs {
		if existing.ID == pr.ID {
			m.prs[i] = pr
			return nil
		}
	}
	return errors.New("pull request not found")
}

func (m *mockPullRequestRepo) SetMerged(_ context.Context, id uuid.UUID, mergedAt time.Time, mergedBy uuid.UUID, sha string) error {
	for i, pr := range m.prs {
		if pr.ID == id {
			m.prs[i].State = "merged"
			m.prs[i].MergedAt = &mergedAt
			m.prs[i].MergedBy = &mergedBy
			m.prs[i].MergeCommitSHA = sha
			return nil
		}
	}
	return errors.New("pull request not found")
}

type mockBranchProtectionRepo struct {
	protection *entity.BranchProtection
}

func (m *mockBranchProtectionRepo) GetByBranch(_ context.Context, _ uuid.UUID, _ string) (*entity.BranchProtection, error) {
	if m.protection == nil {
		return nil, apperror.ErrNotFound
	}
	return m.protection, nil
}

func (m *mockBranchProtectionRepo) Upsert(_ context.Context, _ *entity.BranchProtection) error {
	return nil
}

type mockReviewRepo struct {
	satisfiedReviews int
}

func (m *mockReviewRepo) Create(_ context.Context, _ *entity.Review) error {
	return nil
}

func (m *mockReviewRepo) GetByID(_ context.Context, _ uuid.UUID) (*entity.Review, error) {
	return nil, errors.New("review not found")
}

func (m *mockReviewRepo) ListByPR(_ context.Context, _ uuid.UUID) ([]*entity.Review, error) {
	return nil, nil
}

func (m *mockReviewRepo) CountSatisfiedReviews(_ context.Context, _ uuid.UUID) (int, error) {
	return m.satisfiedReviews, nil
}

func (m *mockReviewRepo) HasBlockingReviews(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}

type mockWorkflowRunRepo struct {
	runs []*entity.WorkflowRun
}

func (m *mockWorkflowRunRepo) ListByHeadSHA(_ context.Context, _ uuid.UUID, _ string) ([]*entity.WorkflowRun, error) {
	return m.runs, nil
}

type mockAuditLogRepo struct {
	calls []auditLogCall
}

type auditLogCall struct {
	action     string
	targetType string
}

func (m *mockAuditLogRepo) Create(_ context.Context, log *entity.AuditLog) error {
	m.calls = append(m.calls, auditLogCall{
		action:     log.Action,
		targetType: log.TargetType,
	})
	return nil
}

func (m *mockAuditLogRepo) List(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]*entity.AuditLog, int, error) {
	return nil, 0, nil
}

type mockGitService struct {
	mergeErr error
}

func (m *mockGitService) BranchExists(_ context.Context, _ string, _ string) (bool, error) {
	return true, nil
}

func (m *mockGitService) ResolveRef(_ context.Context, _ string, _ string) (string, error) {
	return "abc123", nil
}

func (m *mockGitService) Merge(_ context.Context, _ string, _, _, _ string) (string, error) {
	if m.mergeErr != nil {
		return "", m.mergeErr
	}
	return "abc123def456", nil
}

func (m *mockGitService) GetDiff(_ context.Context, _ string, _, _ string, _ int) ([]service.FileDiff, bool, error) {
	return nil, false, nil
}

func (m *mockGitService) GetMergeBase(_ context.Context, _ string, _, _ string) (string, error) {
	return "base123", nil
}

type mockMembershipRepo struct {
	role string
}

func (m *mockMembershipRepo) Add(_ context.Context, _ *entity.Membership) error {
	return nil
}

func (m *mockMembershipRepo) GetRole(_ context.Context, _ uuid.UUID, _ uuid.UUID) (string, error) {
	if m.role == "" {
		return entity.RoleOwner, nil
	}
	return m.role, nil
}

func (m *mockMembershipRepo) ListByOrg(_ context.Context, _ uuid.UUID, _ int, _ int) ([]*entity.Membership, error) {
	return nil, nil
}

func (m *mockMembershipRepo) UpdateRole(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) error {
	return nil
}

func (m *mockMembershipRepo) Remove(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

type mockTxManager struct{}

func (mockTxManager) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func newOpenPR() *entity.PullRequest {
	return &entity.PullRequest{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		RepositoryID:   uuid.New(),
		Number:         1,
		HeadRef:        "feature",
		BaseRef:        "main",
		State:          "open",
	}
}

func TestAlreadyMerged(t *testing.T) {
	pr := newOpenPR()
	pr.State = "merged"
	now := time.Now().UTC()
	pr.MergedAt = &now

	uc := prusecase.NewMergePRUsecase(
		&mockPullRequestRepo{prs: []*entity.PullRequest{pr}},
		&mockBranchProtectionRepo{},
		&mockReviewRepo{},
		&mockWorkflowRunRepo{},
		&mockAuditLogRepo{},
		&mockGitService{},
		mockTxManager{},
		&mockMembershipRepo{},
	)

	_, err := uc.Execute(context.Background(), prusecase.MergePRInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
		GitPath:        "/tmp/repo.git",
		ActorID:        uuid.New(),
		Number:         pr.Number,
	})
	if !errors.Is(err, apperror.ErrAlreadyMerged) {
		t.Fatalf("expected ErrAlreadyMerged, got %v", err)
	}
}

func TestProtectionNotSatisfied(t *testing.T) {
	pr := newOpenPR()

	uc := prusecase.NewMergePRUsecase(
		&mockPullRequestRepo{prs: []*entity.PullRequest{pr}},
		&mockBranchProtectionRepo{
			protection: &entity.BranchProtection{
				RequiredApprovingReviews: 2,
				RequiredStatusChecks:     []string{"ci/build"},
			},
		},
		&mockReviewRepo{satisfiedReviews: 1},
		&mockWorkflowRunRepo{
			runs: []*entity.WorkflowRun{
				{Workflow: "ci/build", Conclusion: "success"},
			},
		},
		&mockAuditLogRepo{},
		&mockGitService{},
		mockTxManager{},
		&mockMembershipRepo{},
	)

	_, err := uc.Execute(context.Background(), prusecase.MergePRInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
		GitPath:        "/tmp/repo.git",
		ActorID:        uuid.New(),
		Number:         pr.Number,
	})
	if !errors.Is(err, apperror.ErrProtectionNotSatisfied) {
		t.Fatalf("expected ErrProtectionNotSatisfied, got %v", err)
	}
}

func TestConflict(t *testing.T) {
	pr := newOpenPR()

	uc := prusecase.NewMergePRUsecase(
		&mockPullRequestRepo{prs: []*entity.PullRequest{pr}},
		&mockBranchProtectionRepo{},
		&mockReviewRepo{},
		&mockWorkflowRunRepo{},
		&mockAuditLogRepo{},
		&mockGitService{mergeErr: apperror.ErrConflict},
		mockTxManager{},
		&mockMembershipRepo{},
	)

	_, err := uc.Execute(context.Background(), prusecase.MergePRInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
		GitPath:        "/tmp/repo.git",
		ActorID:        uuid.New(),
		Number:         pr.Number,
	})
	if !errors.Is(err, apperror.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestSuccessfulMerge(t *testing.T) {
	pr := newOpenPR()
	prRepo := &mockPullRequestRepo{prs: []*entity.PullRequest{pr}}
	auditRepo := &mockAuditLogRepo{}

	uc := prusecase.NewMergePRUsecase(
		prRepo,
		&mockBranchProtectionRepo{},
		&mockReviewRepo{},
		&mockWorkflowRunRepo{},
		auditRepo,
		&mockGitService{},
		mockTxManager{},
		&mockMembershipRepo{},
	)

	merged, err := uc.Execute(context.Background(), prusecase.MergePRInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
		GitPath:        "/tmp/repo.git",
		ActorID:        uuid.New(),
		Number:         pr.Number,
	})
	if err != nil {
		t.Fatalf("merge pull request: %v", err)
	}
	if merged.State != "merged" {
		t.Fatalf("expected state merged, got %q", merged.State)
	}
	if merged.MergedAt == nil {
		t.Fatal("expected merged_at to be set")
	}
	if len(auditRepo.calls) != 1 {
		t.Fatalf("expected 1 audit log call, got %d", len(auditRepo.calls))
	}
	if auditRepo.calls[0].action != "pr.merge" || auditRepo.calls[0].targetType != "pull_request" {
		t.Fatalf("unexpected audit payload: %+v", auditRepo.calls[0])
	}
}
