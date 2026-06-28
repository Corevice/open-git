package pr_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
	prusecase "github.com/open-git/backend/internal/usecase/pr"
	"github.com/google/uuid"
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

func (m *mockPullRequestRepo) ListByRepo(_ context.Context, _ repository.ListPullRequestsFilter) ([]*entity.PullRequest, int, error) {
	return m.prs, len(m.prs), nil
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

func (m *mockPullRequestRepo) NextNumber(_ context.Context, _ uuid.UUID) (int, error) {
	return 1, nil
}

type mockBranchProtectionRepo struct {
	protection *entity.BranchProtection
}

func (m *mockBranchProtectionRepo) GetForRef(_ context.Context, _ uuid.UUID, _ string) (*entity.BranchProtection, error) {
	if m.protection == nil {
		return nil, apperror.ErrNotFound
	}
	return m.protection, nil
}

type mockReviewRepo struct {
	satisfiedReviews     int
	hasOpenConversations bool
}

func (m *mockReviewRepo) CountSatisfiedReviews(_ context.Context, _ uuid.UUID) (int, error) {
	return m.satisfiedReviews, nil
}

func (m *mockReviewRepo) HasOpenConversations(_ context.Context, _ uuid.UUID) (bool, error) {
	return m.hasOpenConversations, nil
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

func (m *mockAuditLogRepo) InsertAuditLog(
	_ context.Context,
	_, _ uuid.UUID,
	action, targetType string,
	_ uuid.UUID,
	_ json.RawMessage,
) error {
	m.calls = append(m.calls, auditLogCall{
		action:     action,
		targetType: targetType,
	})
	return nil
}

type mockGitService struct {
	mergeErr error
}

func (m *mockGitService) BranchExists(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return true, nil
}

func (m *mockGitService) ResolveRef(_ context.Context, _ uuid.UUID, _ string) (string, error) {
	return "abc123", nil
}

func (m *mockGitService) Merge(_ context.Context, _ uuid.UUID, _, _, _ string) error {
	return m.mergeErr
}

type mockTxManager struct{}

func (mockTxManager) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type mockMembershipRepo struct {
	roles map[uuid.UUID]string
}

func (m *mockMembershipRepo) Add(_ context.Context, _ *entity.Membership) error {
	return nil
}

func (m *mockMembershipRepo) GetRole(_ context.Context, orgID, userID uuid.UUID) (string, error) {
	if m.roles == nil {
		return entity.RoleMember, nil
	}
	role, ok := m.roles[userID]
	if !ok {
		return "", domain.ErrNotFound
	}
	return role, nil
}

func (m *mockMembershipRepo) ListByOrg(_ context.Context, _ uuid.UUID, _, _ int) ([]*entity.Membership, error) {
	return nil, nil
}

func (m *mockMembershipRepo) UpdateRole(_ context.Context, _, _ uuid.UUID, _ string) error {
	return nil
}

func (m *mockMembershipRepo) Remove(_ context.Context, _, _ uuid.UUID) error {
	return nil
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
		&mockMembershipRepo{},
		&mockReviewRepo{},
		&mockWorkflowRunRepo{},
		&mockAuditLogRepo{},
		&mockGitService{},
		mockTxManager{},
	)

	_, err := uc.Execute(context.Background(), prusecase.MergePRInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
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
				RequiredReviews: 2,
				RequiredChecks:  []string{"ci/build"},
			},
		},
		&mockMembershipRepo{},
		&mockReviewRepo{satisfiedReviews: 1},
		&mockWorkflowRunRepo{
			runs: []*entity.WorkflowRun{
				{Workflow: "ci/build", Conclusion: "success"},
			},
		},
		&mockAuditLogRepo{},
		&mockGitService{},
		mockTxManager{},
	)

	_, err := uc.Execute(context.Background(), prusecase.MergePRInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
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
		&mockMembershipRepo{},
		&mockReviewRepo{},
		&mockWorkflowRunRepo{},
		&mockAuditLogRepo{},
		&mockGitService{mergeErr: apperror.ErrConflict},
		mockTxManager{},
	)

	_, err := uc.Execute(context.Background(), prusecase.MergePRInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
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
		&mockMembershipRepo{},
		&mockReviewRepo{},
		&mockWorkflowRunRepo{},
		auditRepo,
		&mockGitService{},
		mockTxManager{},
	)

	merged, err := uc.Execute(context.Background(), prusecase.MergePRInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
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

func TestEnforceAdminsFalseAdminBypassesProtection(t *testing.T) {
	pr := newOpenPR()
	prRepo := &mockPullRequestRepo{prs: []*entity.PullRequest{pr}}
	auditRepo := &mockAuditLogRepo{}
	actorID := uuid.New()

	uc := prusecase.NewMergePRUsecase(
		prRepo,
		&mockBranchProtectionRepo{
			protection: &entity.BranchProtection{
				EnforceAdmins:   false,
				RequiredReviews: 2,
				RequiredChecks:  []string{"ci/build"},
			},
		},
		&mockMembershipRepo{roles: map[uuid.UUID]string{actorID: entity.RoleAdmin}},
		&mockReviewRepo{satisfiedReviews: 0},
		&mockWorkflowRunRepo{},
		auditRepo,
		&mockGitService{},
		mockTxManager{},
	)

	merged, err := uc.Execute(context.Background(), prusecase.MergePRInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
		ActorID:        actorID,
		Number:         pr.Number,
	})
	if err != nil {
		t.Fatalf("expected admin bypass merge, got %v", err)
	}
	if merged.State != "merged" {
		t.Fatalf("expected state merged, got %q", merged.State)
	}
	if len(auditRepo.calls) != 2 {
		t.Fatalf("expected 2 audit log calls (admin bypass + merge), got %d", len(auditRepo.calls))
	}
	if auditRepo.calls[0].action != "pr.merge.admin_bypass" || auditRepo.calls[0].targetType != "branch_protection" {
		t.Fatalf("unexpected admin bypass audit payload: %+v", auditRepo.calls[0])
	}
}

func TestEnforceAdminsTrueAdminBlockedByReviews(t *testing.T) {
	pr := newOpenPR()
	actorID := uuid.New()

	uc := prusecase.NewMergePRUsecase(
		&mockPullRequestRepo{prs: []*entity.PullRequest{pr}},
		&mockBranchProtectionRepo{
			protection: &entity.BranchProtection{
				EnforceAdmins:   true,
				RequiredReviews: 2,
			},
		},
		&mockMembershipRepo{roles: map[uuid.UUID]string{actorID: entity.RoleAdmin}},
		&mockReviewRepo{satisfiedReviews: 1},
		&mockWorkflowRunRepo{},
		&mockAuditLogRepo{},
		&mockGitService{},
		mockTxManager{},
	)

	_, err := uc.Execute(context.Background(), prusecase.MergePRInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
		ActorID:        actorID,
		Number:         pr.Number,
	})
	if !errors.Is(err, apperror.ErrProtectionNotSatisfied) {
		t.Fatalf("expected ErrProtectionNotSatisfied, got %v", err)
	}
}

func TestEnforceAdminsFalseAdminBypassesRequiredLinearHistory(t *testing.T) {
	pr := newOpenPR()
	prRepo := &mockPullRequestRepo{prs: []*entity.PullRequest{pr}}
	auditRepo := &mockAuditLogRepo{}
	actorID := uuid.New()

	uc := prusecase.NewMergePRUsecase(
		prRepo,
		&mockBranchProtectionRepo{
			protection: &entity.BranchProtection{
				EnforceAdmins:         false,
				RequiredLinearHistory: true,
			},
		},
		&mockMembershipRepo{roles: map[uuid.UUID]string{actorID: entity.RoleAdmin}},
		&mockReviewRepo{},
		&mockWorkflowRunRepo{},
		auditRepo,
		&mockGitService{},
		mockTxManager{},
	)

	merged, err := uc.Execute(context.Background(), prusecase.MergePRInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
		ActorID:        actorID,
		Number:         pr.Number,
		MergeMethod:    "merge",
	})
	if err != nil {
		t.Fatalf("expected admin bypass merge, got %v", err)
	}
	if merged.State != "merged" {
		t.Fatalf("expected state merged, got %q", merged.State)
	}
	if len(auditRepo.calls) != 2 {
		t.Fatalf("expected 2 audit log calls (admin bypass + merge), got %d", len(auditRepo.calls))
	}
	if auditRepo.calls[0].action != "pr.merge.admin_bypass" {
		t.Fatalf("unexpected admin bypass audit payload: %+v", auditRepo.calls[0])
	}
}

func TestEnforceAdminsFalseAdminBypassesRequiredConversationResolution(t *testing.T) {
	pr := newOpenPR()
	prRepo := &mockPullRequestRepo{prs: []*entity.PullRequest{pr}}
	auditRepo := &mockAuditLogRepo{}
	actorID := uuid.New()

	uc := prusecase.NewMergePRUsecase(
		prRepo,
		&mockBranchProtectionRepo{
			protection: &entity.BranchProtection{
				EnforceAdmins:                  false,
				RequiredConversationResolution: true,
			},
		},
		&mockMembershipRepo{roles: map[uuid.UUID]string{actorID: entity.RoleAdmin}},
		&mockReviewRepo{hasOpenConversations: true},
		&mockWorkflowRunRepo{},
		auditRepo,
		&mockGitService{},
		mockTxManager{},
	)

	merged, err := uc.Execute(context.Background(), prusecase.MergePRInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
		ActorID:        actorID,
		Number:         pr.Number,
	})
	if err != nil {
		t.Fatalf("expected admin bypass merge, got %v", err)
	}
	if merged.State != "merged" {
		t.Fatalf("expected state merged, got %q", merged.State)
	}
	if len(auditRepo.calls) != 2 {
		t.Fatalf("expected 2 audit log calls (admin bypass + merge), got %d", len(auditRepo.calls))
	}
	if auditRepo.calls[0].action != "pr.merge.admin_bypass" {
		t.Fatalf("unexpected admin bypass audit payload: %+v", auditRepo.calls[0])
	}
}

func TestRequiredLinearHistoryBlocksMergeCommit(t *testing.T) {
	pr := newOpenPR()

	uc := prusecase.NewMergePRUsecase(
		&mockPullRequestRepo{prs: []*entity.PullRequest{pr}},
		&mockBranchProtectionRepo{
			protection: &entity.BranchProtection{
				RequiredLinearHistory: true,
			},
		},
		&mockMembershipRepo{},
		&mockReviewRepo{},
		&mockWorkflowRunRepo{},
		&mockAuditLogRepo{},
		&mockGitService{},
		mockTxManager{},
	)

	_, err := uc.Execute(context.Background(), prusecase.MergePRInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
		ActorID:        uuid.New(),
		Number:         pr.Number,
		MergeMethod:    "merge",
	})
	if !errors.Is(err, apperror.ErrProtectionNotSatisfied) {
		t.Fatalf("expected ErrProtectionNotSatisfied, got %v", err)
	}
}

func TestRequiredLinearHistoryAllowsSquash(t *testing.T) {
	pr := newOpenPR()

	uc := prusecase.NewMergePRUsecase(
		&mockPullRequestRepo{prs: []*entity.PullRequest{pr}},
		&mockBranchProtectionRepo{
			protection: &entity.BranchProtection{
				RequiredLinearHistory: true,
			},
		},
		&mockMembershipRepo{},
		&mockReviewRepo{},
		&mockWorkflowRunRepo{},
		&mockAuditLogRepo{},
		&mockGitService{},
		mockTxManager{},
	)

	merged, err := uc.Execute(context.Background(), prusecase.MergePRInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
		ActorID:        uuid.New(),
		Number:         pr.Number,
		MergeMethod:    "squash",
	})
	if err != nil {
		t.Fatalf("expected squash merge to succeed, got %v", err)
	}
	if merged.State != "merged" {
		t.Fatalf("expected state merged, got %q", merged.State)
	}
}

func TestRequiredConversationResolutionBlocksMerge(t *testing.T) {
	pr := newOpenPR()

	uc := prusecase.NewMergePRUsecase(
		&mockPullRequestRepo{prs: []*entity.PullRequest{pr}},
		&mockBranchProtectionRepo{
			protection: &entity.BranchProtection{
				RequiredConversationResolution: true,
			},
		},
		&mockMembershipRepo{},
		&mockReviewRepo{hasOpenConversations: true},
		&mockWorkflowRunRepo{},
		&mockAuditLogRepo{},
		&mockGitService{},
		mockTxManager{},
	)

	_, err := uc.Execute(context.Background(), prusecase.MergePRInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
		ActorID:        uuid.New(),
		Number:         pr.Number,
	})
	if !errors.Is(err, apperror.ErrProtectionNotSatisfied) {
		t.Fatalf("expected ErrProtectionNotSatisfied, got %v", err)
	}
}
