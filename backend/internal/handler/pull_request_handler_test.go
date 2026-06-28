package handler_test

import (
	"bytes"
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
	pr    *entity.PullRequest
	pulls []*entity.PullRequest
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
	if m.pulls != nil {
		return m.pulls, len(m.pulls), nil
	}
	return []*entity.PullRequest{}, 0, nil
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

type prHandlerReviewCommentRepo struct{}

func (prHandlerReviewCommentRepo) Create(_ context.Context, _ *entity.ReviewComment) error { return nil }

func (prHandlerReviewCommentRepo) ListByPR(_ context.Context, _ uuid.UUID) ([]*entity.ReviewComment, error) {
	return nil, nil
}

func (prHandlerReviewCommentRepo) ListByReview(_ context.Context, _ uuid.UUID) ([]*entity.ReviewComment, error) {
	return nil, nil
}

type prHandlerWorkflowRunRepo struct{}

func (prHandlerWorkflowRunRepo) ListByHeadSHA(_ context.Context, _ uuid.UUID, _ string) ([]*entity.WorkflowRun, error) {
	return nil, nil
}

type prHandlerAuditLogRepo struct{}

func (prHandlerAuditLogRepo) Create(_ context.Context, _ *entity.AuditLog) error { return nil }

func (prHandlerAuditLogRepo) List(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]*entity.AuditLog, int, error) {
	return nil, 0, nil
}

func (prHandlerAuditLogRepo) InsertAuditLog(context.Context, uuid.UUID, uuid.UUID, string, string, uuid.UUID, json.RawMessage) error {
	return nil
}

type prHandlerGitService struct {
	diffs []service.FileDiff
}

func (m prHandlerGitService) BranchExists(_ context.Context, _ string, _ string) (bool, error) {
	return true, nil
}

func (m prHandlerGitService) ResolveRef(_ context.Context, _ string, _ string) (string, error) {
	return "abc123", nil
}

func (m prHandlerGitService) Merge(_ context.Context, _ string, _, _, _ string) (string, error) {
	return "abc123def456", nil
}

func (m prHandlerGitService) GetDiff(_ context.Context, _ string, _, _ string, _ int) ([]service.FileDiff, bool, error) {
	if len(m.diffs) > 0 {
		return m.diffs, false, nil
	}
	return []service.FileDiff{
		{Filename: "main.go", Status: "modified", Additions: 3, Deletions: 1, Patch: "@@ diff"},
	}, false, nil
}

func (m prHandlerGitService) GetMergeBase(_ context.Context, _ string, _, _ string) (string, error) {
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

type prHandlerOptions struct {
	prRepo  *prHandlerPRRepo
	gitSvc  prHandlerGitService
}

func newPullRequestHandlerEcho(t *testing.T, opts prHandlerOptions) *echo.Echo {
	t.Helper()

	pr := opts.prRepo
	if pr == nil {
		pr = &prHandlerPRRepo{
			pr: &entity.PullRequest{
				ID:             prTestPRID,
				OrganizationID: prTestOrgUUID,
				RepositoryID:   prTestRepoID,
				Number:         1,
				HeadRef:        "feature",
				BaseRef:        "main",
				BaseSHA:        "base-sha",
				HeadSHA:        "head-sha",
				State:          "open",
			},
		}
	}

	gitSvc := opts.gitSvc
	createPRUC := prusecase.NewCreatePRUsecase(
		pr,
		prHandlerAuditLogRepo{},
		gitSvc,
		prHandlerTxManager{},
		prHandlerMembershipRepo{},
	)
	mergePRUC := prusecase.NewMergePRUsecase(
		pr,
		prHandlerBranchProtectionRepo{},
		prHandlerReviewRepo{},
		prHandlerWorkflowRunRepo{},
		prHandlerAuditLogRepo{},
		gitSvc,
		prHandlerTxManager{},
		prHandlerMembershipRepo{},
	)
	createReviewUC := prusecase.NewCreateReviewUsecase(
		pr,
		prHandlerReviewRepo{},
		prHandlerAuditLogRepo{},
		prHandlerMembershipRepo{},
	)
	listReviewsUC := prusecase.NewListReviewsUsecase(
		pr,
		prHandlerReviewRepo{},
	)

	h := handler.NewPullRequestHandler(
		createPRUC,
		mergePRUC,
		createReviewUC,
		listReviewsUC,
		pr,
		prHandlerReviewRepo{},
		prHandlerReviewCommentRepo{},
		gitSvc,
		func(_ echo.Context, _, _ string) (*entity.Repository, error) {
			return prTestRepo(), nil
		},
	)

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g, prTestAuth)
	return e
}

func TestGetPullRequestsEmpty(t *testing.T) {
	e := newPullRequestHandlerEcho(t, prHandlerOptions{
		prRepo: &prHandlerPRRepo{},
	})

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/pulls", nil)
	req.Host = "git.example.com"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp []any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp) != 0 {
		t.Fatalf("expected empty array, got %d items", len(resp))
	}
}

func TestPostPullRequestSameBaseHead(t *testing.T) {
	e := newPullRequestHandlerEcho(t, prHandlerOptions{
		prRepo: &prHandlerPRRepo{},
	})

	body := `{"title":"Test PR","head":"main","base":"main"}`
	req := httptest.NewRequest(http.MethodPost, "/repos/alice/demo/pulls", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Host = "git.example.com"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

func TestGetPullRequestFiles(t *testing.T) {
	e := newPullRequestHandlerEcho(t, prHandlerOptions{})

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/pulls/1/files", nil)
	req.Host = "git.example.com"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Files     []map[string]any `json:"files"`
		Truncated bool             `json:"truncated"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Files) == 0 {
		t.Fatal("expected at least one file in response")
	}
	if resp.Files[0]["filename"] != "main.go" {
		t.Fatalf("filename = %v, want main.go", resp.Files[0]["filename"])
	}
}

func TestPostReviewValidBodyReturns201(t *testing.T) {
	e := newPullRequestHandlerEcho(t, prHandlerOptions{})

	body := `{"event":"APPROVE","body":"looks good"}`
	req := httptest.NewRequest(http.MethodPost, "/repos/alice/demo/pulls/1/reviews", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Host = "git.example.com"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

func TestGetReviewsReturns200Array(t *testing.T) {
	e := newPullRequestHandlerEcho(t, prHandlerOptions{})

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/pulls/1/reviews", nil)
	req.Host = "git.example.com"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp []any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp == nil {
		t.Fatal("expected JSON array, got null")
	}
}

func TestPostReviewInvalidEvent(t *testing.T) {
	e := newPullRequestHandlerEcho(t, prHandlerOptions{})

	body := `{"event":"INVALID","body":"looks good"}`
	req := httptest.NewRequest(http.MethodPost, "/repos/alice/demo/pulls/1/reviews", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Host = "git.example.com"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestPutMergePullRequest(t *testing.T) {
	e := newPullRequestHandlerEcho(t, prHandlerOptions{})

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

func TestPutMergeAlreadyMergedPullRequest(t *testing.T) {
	now := time.Now().UTC()
	e := newPullRequestHandlerEcho(t, prHandlerOptions{
		prRepo: &prHandlerPRRepo{
			pr: &entity.PullRequest{
				ID:             prTestPRID,
				OrganizationID: prTestOrgUUID,
				RepositoryID:   prTestRepoID,
				Number:         1,
				HeadRef:        "feature",
				BaseRef:        "main",
				State:          "merged",
				MergedAt:       &now,
			},
		},
	})

	req := httptest.NewRequest(http.MethodPut, "/repos/alice/demo/pulls/1/merge", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Host = "git.example.com"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusMethodNotAllowed, rec.Body.String())
	}
}
