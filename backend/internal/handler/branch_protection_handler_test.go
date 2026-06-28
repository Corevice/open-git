package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/handler"
	repoUC "github.com/open-git/backend/internal/usecase/repository"
)

var (
	bpTestOrgID   = uuid.MustParse("00000000-0000-0000-0000-000000000010")
	bpTestRepoID  = uuid.MustParse("00000000-0000-0000-0000-000000000011")
	bpTestActorID = uuid.MustParse("00000000-0000-0000-0000-000000000012")
	bpTestUserID  = int64(12)
)

type branchProtectionStore struct {
	byPattern map[string]*handler.BranchProtectionDetail
	list      []*handler.BranchProtectionDetail
}

func bpKey(orgID, repoID uuid.UUID, pattern string) string {
	return fmt.Sprintf("%s:%s:%s", orgID, repoID, pattern)
}

type stubBranchProtectionReadRepo struct {
	store *branchProtectionStore
}

func (m *stubBranchProtectionReadRepo) GetByPattern(_ context.Context, orgID, repoID uuid.UUID, pattern string) (*handler.BranchProtectionDetail, error) {
	if m.store.byPattern == nil {
		return nil, apperror.ErrNotFound
	}
	rule, ok := m.store.byPattern[bpKey(orgID, repoID, pattern)]
	if !ok {
		return nil, apperror.ErrNotFound
	}
	return rule, nil
}

func (m *stubBranchProtectionReadRepo) ListByRepository(_ context.Context, orgID, repoID uuid.UUID) ([]*handler.BranchProtectionDetail, error) {
	if m.store.list != nil {
		return m.store.list, nil
	}
	if m.store.byPattern == nil {
		return []*handler.BranchProtectionDetail{}, nil
	}
	result := make([]*handler.BranchProtectionDetail, 0, len(m.store.byPattern))
	for key, rule := range m.store.byPattern {
		if key == bpKey(orgID, repoID, rule.Pattern) {
			result = append(result, rule)
		}
	}
	return result, nil
}

type stubBranchProtectionWriteRepo struct {
	store *branchProtectionStore
}

func (m *stubBranchProtectionWriteRepo) GetByPattern(_ context.Context, orgID, repoID uuid.UUID, pattern string) (*repoUC.BranchProtectionRule, error) {
	if m.store.byPattern == nil {
		return nil, apperror.ErrNotFound
	}
	rule, ok := m.store.byPattern[bpKey(orgID, repoID, pattern)]
	if !ok {
		return nil, apperror.ErrNotFound
	}
	return &repoUC.BranchProtectionRule{
		Pattern:                      rule.Pattern,
		RequiredApprovingReviewCount: rule.RequiredApprovingReviewCount,
	}, nil
}

func (m *stubBranchProtectionWriteRepo) Upsert(_ context.Context, orgID, repoID uuid.UUID, rule *repoUC.BranchProtectionRule) (*repoUC.BranchProtectionRule, error) {
	if m.store.byPattern == nil {
		m.store.byPattern = map[string]*handler.BranchProtectionDetail{}
	}
	m.store.byPattern[bpKey(orgID, repoID, rule.Pattern)] = &handler.BranchProtectionDetail{
		Pattern:                      rule.Pattern,
		RequiredApprovingReviewCount: rule.RequiredApprovingReviewCount,
	}
	return rule, nil
}

func (m *stubBranchProtectionWriteRepo) DeleteByPattern(_ context.Context, orgID, repoID uuid.UUID, pattern string) error {
	if m.store.byPattern == nil {
		return apperror.ErrNotFound
	}
	key := bpKey(orgID, repoID, pattern)
	if _, ok := m.store.byPattern[key]; !ok {
		return apperror.ErrNotFound
	}
	delete(m.store.byPattern, key)
	return nil
}

func testRepository() *entity.Repository {
	return &entity.Repository{
		ID:             bpTestRepoID,
		OrganizationID: bpTestOrgID,
		OwnerID:        bpTestActorID,
		Name:           "demo",
		OwnerLogin:     "alice",
		Visibility:     entity.VisibilityPublic,
		DefaultBranch:  "main",
	}
}

func newBranchProtectionEcho(
	t *testing.T,
	store *branchProtectionStore,
	isAdmin bool,
) (*echo.Echo, *handler.BranchProtectionHandler) {
	t.Helper()

	readRepo := &stubBranchProtectionReadRepo{store: store}
	writeRepo := &stubBranchProtectionWriteRepo{store: store}

	auditLog := &listMockAuditLogRepo{}
	upsertUC := repoUC.NewUpsertBranchProtectionUsecase(writeRepo, auditLog)
	deleteUC := repoUC.NewDeleteBranchProtectionUsecase(writeRepo, auditLog)

	testRepo := testRepository()
	resolveRepo := func(_ echo.Context, _, _ string) (*entity.Repository, error) {
		return testRepo, nil
	}
	checkRepoAdmin := func(_ echo.Context, _ *entity.Repository) error {
		if !isAdmin {
			return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "Forbidden"})
		}
		return nil
	}

	h := handler.NewBranchProtectionHandler(readRepo, upsertUC, deleteUC, resolveRepo, checkRepoAdmin)

	e := echo.New()
	v3 := e.Group("/api/v3")
	h.RegisterRoutes(v3, authMiddleware(bpTestUserID))

	internal := e.Group("/api/internal")
	h.RegisterInternalRoutes(internal, authMiddleware(bpTestUserID))

	return e, h
}

func TestGetBranchProtection_Found(t *testing.T) {
	store := &branchProtectionStore{
		byPattern: map[string]*handler.BranchProtectionDetail{
			bpKey(bpTestOrgID, bpTestRepoID, "main"): {
				Pattern:                          "main",
				RequiredApprovingReviewCount:     2,
				DismissStaleReviews:              true,
				RequireCodeOwnerReviews:          false,
				RequiredStatusChecksStrict:       true,
				RequiredStatusChecksContexts:     []string{"ci/build"},
				EnforceAdmins:                    true,
				AllowForcePushes:                 false,
				AllowDeletions:                   false,
				RequiredLinearHistory:            false,
				RequiredConversationResolution:   false,
			},
		},
	}
	e, _ := newBranchProtectionEcho(t, store, true)

	req := httptest.NewRequest(http.MethodGet, "/api/v3/repos/alice/demo/branches/main/protection", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	rsc, ok := resp["required_status_checks"].(map[string]any)
	if !ok {
		t.Fatalf("missing required_status_checks: %v", resp)
	}
	if rsc["strict"] != true {
		t.Fatalf("expected strict=true, got %v", rsc["strict"])
	}

	rpr, ok := resp["required_pull_request_reviews"].(map[string]any)
	if !ok {
		t.Fatalf("missing required_pull_request_reviews: %v", resp)
	}
	if rpr["required_approving_review_count"].(float64) != 2 {
		t.Fatalf("expected review count 2, got %v", rpr["required_approving_review_count"])
	}

	if resp["restrictions"] != nil {
		t.Fatalf("expected restrictions null, got %v", resp["restrictions"])
	}
	if resp["enforce_admins"] != true {
		t.Fatalf("expected enforce_admins=true, got %v", resp["enforce_admins"])
	}
}

func TestGetBranchProtection_NotFound(t *testing.T) {
	store := &branchProtectionStore{byPattern: map[string]*handler.BranchProtectionDetail{}}
	e, _ := newBranchProtectionEcho(t, store, true)

	req := httptest.NewRequest(http.MethodGet, "/api/v3/repos/alice/demo/branches/main/protection", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestPutBranchProtection_AdminSuccess(t *testing.T) {
	store := &branchProtectionStore{byPattern: map[string]*handler.BranchProtectionDetail{}}
	e, _ := newBranchProtectionEcho(t, store, true)

	body := `{
		"required_status_checks": {"strict": true, "contexts": ["ci/build"]},
		"enforce_admins": true,
		"required_pull_request_reviews": {
			"dismiss_stale_reviews": true,
			"require_code_owner_reviews": false,
			"required_approving_review_count": 2
		},
		"restrictions": null,
		"allow_force_pushes": false,
		"allow_deletions": false,
		"required_linear_history": false,
		"required_conversation_resolution": false
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/v3/repos/alice/demo/branches/main/protection", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["restrictions"] != nil {
		t.Fatalf("expected restrictions null, got %v", resp["restrictions"])
	}
	rpr := resp["required_pull_request_reviews"].(map[string]any)
	if rpr["required_approving_review_count"].(float64) != 2 {
		t.Fatalf("expected review count 2, got %v", rpr["required_approving_review_count"])
	}
}

func TestPutBranchProtection_NonAdminForbidden(t *testing.T) {
	store := &branchProtectionStore{byPattern: map[string]*handler.BranchProtectionDetail{}}
	e, _ := newBranchProtectionEcho(t, store, false)

	body := `{
		"required_pull_request_reviews": {"required_approving_review_count": 1}
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/v3/repos/alice/demo/branches/main/protection", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestPutBranchProtection_InvalidReviewCount422(t *testing.T) {
	store := &branchProtectionStore{byPattern: map[string]*handler.BranchProtectionDetail{}}
	e, _ := newBranchProtectionEcho(t, store, true)

	body := `{
		"required_pull_request_reviews": {"required_approving_review_count": 99}
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/v3/repos/alice/demo/branches/main/protection", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

func TestDeleteBranchProtection_Success204(t *testing.T) {
	store := &branchProtectionStore{
		byPattern: map[string]*handler.BranchProtectionDetail{
			bpKey(bpTestOrgID, bpTestRepoID, "main"): {
				Pattern:                      "main",
				RequiredApprovingReviewCount: 1,
			},
		},
	}
	e, _ := newBranchProtectionEcho(t, store, true)

	req := httptest.NewRequest(http.MethodDelete, "/api/v3/repos/alice/demo/branches/main/protection", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
}

func TestDeleteBranchProtection_NotFound404(t *testing.T) {
	store := &branchProtectionStore{byPattern: map[string]*handler.BranchProtectionDetail{}}
	e, _ := newBranchProtectionEcho(t, store, true)

	req := httptest.NewRequest(http.MethodDelete, "/api/v3/repos/alice/demo/branches/main/protection", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestListBranchProtections_ReturnsArray(t *testing.T) {
	store := &branchProtectionStore{
		list: []*handler.BranchProtectionDetail{
			{
				Pattern:                      "main",
				RequiredApprovingReviewCount: 1,
				RequiredStatusChecksContexts: []string{},
			},
			{
				Pattern:                      "release/*",
				RequiredApprovingReviewCount: 2,
				RequiredStatusChecksContexts: []string{"ci/test"},
			},
		},
	}
	e, _ := newBranchProtectionEcho(t, store, true)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/repos/alice/demo/branch-protections", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(resp))
	}
	for _, item := range resp {
		if item["restrictions"] != nil {
			t.Fatalf("expected restrictions null, got %v", item["restrictions"])
		}
	}
}
