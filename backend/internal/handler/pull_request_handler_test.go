package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/service"
	"github.com/open-git/backend/internal/handler"
	"github.com/open-git/backend/internal/middleware"
	prusecase "github.com/open-git/backend/internal/usecase/pr"
)

var (
	prTestUserID  = int64(7)
	prTestOrgUUID = uuid.MustParse("00000000-0000-0000-0000-000000000042")
	prTestRepoID  = uuid.MustParse("00000000-0000-0000-0000-000000000099")
	prTestPRID    = uuid.MustParse("00000000-0000-0000-0000-000000000001")
)

type prHandlerPRRepo struct {
	pr *entity.PullRequest
}

func (m *prHandlerPRRepo) Create(_ context.Context, _ *entity.PullRequest) error { return nil }

func (m *prHandlerPRRepo) GetByNumber(_ context.Context, _ uuid.UUID, number int) (*entity.PullRequest, error) {
	if m.pr != nil && m.pr.Number == number {
		return m.pr, nil
	}
	return nil, nil
}

func (m *prHandlerPRRepo) GetByID(_ context.Context, _ uuid.UUID) (*entity.PullRequest, error) {
	return nil, nil
}

func (m *prHandlerPRRepo) ListByRepo(_ context.Context, _ uuid.UUID, _ repository.ListPullRequestsFilter) ([]*entity.PullRequest, int, error) {
	return nil, 0, nil
}

func (m *prHandlerPRRepo) NextNumber(_ context.Context, _ uuid.UUID) (int, error) { return 1, nil }

func (m *prHandlerPRRepo) Update(_ context.Context, _ *entity.PullRequest) error { return nil }

func (m *prHandlerPRRepo) SetMerged(_ context.Context, id uuid.UUID, mergedAt time.Time, mergedBy uuid.UUID, sha string) error {
	if m.pr != nil && m.pr.ID == id {
		m.pr.State = "merged"
		m.pr.MergedAt = &mergedAt
		m.pr.MergedBy = &mergedBy
		m.pr.MergeCommitSHA = sha
	}
	return nil
}

type prHandlerBranchProtectionRepo struct{}

func (prHandlerBranchProtectionRepo) GetByBranch(_ context.Context, _ uuid.UUID, _ string) (*entity.BranchProtection, error) {
	return nil, nil
}

func (prHandlerBranchProtectionRepo) Upsert(_ context.Context, _ *entity.BranchProtection) error {
	return nil
}

type prHandlerReviewRepo struct{}

func (prHandlerReviewRepo) Create(_ context.Context, _ *entity.Review) error { return nil }

func (prHandlerReviewRepo) GetByID(_ context.Context, _ uuid.UUID) (*entity.Review, error) {
	return nil, nil
}

func (prHandlerReviewRepo) ListByPR(_ context.Context, _ uuid.UUID) ([]*entity.Review, error) {
	return nil, nil
}

func (prHandlerReviewRepo) CountSatisfiedReviews(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (prHandlerReviewRepo) HasBlockingReviews(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}

type prHandlerWorkflowRunRepo struct{}

func (prHandlerWorkflowRunRepo) ListByHeadSHA(_ context.Context, _ uuid.UUID, _ string) ([]*entity.WorkflowRun, error) {
	return nil, nil
}

type prHandlerAuditLogRepo struct{}

func (prHandlerAuditLogRepo) Create(_ context.Context, _ *entity.AuditLog) error { return nil }

type prHandlerGitService struct{}

func (prHandlerGitService) BranchExists(_ context.Context, _ string, _ string) (bool, error) {
	return true, nil
}

func (prHandlerGitService) ResolveRef(_ context.Context, _ string, _ string) (string, error) {
	return "abc123", nil
}

func (prHandlerGitService) Merge(_ context.Context, _ string, _, _, _ string) (string, error) {
	return "abc123def456", nil
}

func (prHandlerGitService) GetDiff(_ context.Context, _ string, _, _ string, _ int) ([]service.FileDiff, bool, error) {
	return nil, false, nil
}

func (prHandlerGitService) GetMergeBase(_ context.Context, _ string, _, _ string) (string, error) {
	return "base123", nil
}

type prHandlerMembershipRepo struct{}

func (prHandlerMembershipRepo) Add(_ context.Context, _ *entity.Membership) error { return nil }

func (prHandlerMembershipRepo) GetRole(_ context.Context, _ uuid.UUID, _ uuid.UUID) (string, error) {
	return entity.RoleOwner, nil
}

func (prHandlerMembershipRepo) ListByOrg(_ context.Context, _ uuid.UUID, _ int, _ int) ([]*entity.Membership, error) {
	return nil, nil
}

func (prHandlerMembershipRepo) UpdateRole(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) error {
	return nil
}

func (prHandlerMembershipRepo) Remove(_ context.Context, _ uuid.UUID, _ uuid.UUID) error { return nil }

type prHandlerTxManager struct{}

func (prHandlerTxManager) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func prTestRepo() *entity.Repository {
	return &entity.Repository{
		ID:             prTestRepoID,
		OrganizationID: prTestOrgUUID,
		OwnerLogin:     "alice",
		Name:           "demo",
		GitPath:        "/tmp/demo.git",
	}
}

func prTestAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		middleware.SetAuthContext(c, prTestUserID, []string{"repo"})
		return next(c)
	}
}

func newPullRequestHandlerEcho(t *testing.T) *echo.Echo {
	t.Helper()

	pr := &entity.PullRequest{
		ID:             prTestPRID,
		OrganizationID: prTestOrgUUID,
		RepositoryID:   prTestRepoID,
		Number:         1,
		HeadRef:        "feature",
		BaseRef:        "main",
		State:          "open",
	}

	mergePRUC := prusecase.NewMergePRUsecase(
		&prHandlerPRRepo{pr: pr},
		prHandlerBranchProtectionRepo{},
		prHandlerReviewRepo{},
		prHandlerWorkflowRunRepo{},
		prHandlerAuditLogRepo{},
		prHandlerGitService{},
		prHandlerTxManager{},
		prHandlerMembershipRepo{},
	)

	h := handler.NewPullRequestHandler(
		nil,
		mergePRUC,
		nil,
		func(_ echo.Context, _, _ string) (*entity.Repository, error) {
			return prTestRepo(), nil
		},
	)

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g, prTestAuth)
	return e
}

func TestPutMergePullRequest(t *testing.T) {
	e := newPullRequestHandlerEcho(t)

	req := httptest.NewRequest(http.MethodPut, "/repos/alice/demo/pulls/1/merge", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Host = "git.example.com"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if merged, ok := resp["merged"].(bool); !ok || !merged {
		t.Fatalf("merged = %v, want true", resp["merged"])
	}
}
