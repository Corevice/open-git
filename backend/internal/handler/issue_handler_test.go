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

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/handler"
	"github.com/open-git/backend/internal/middleware"
	issueusecase "github.com/open-git/backend/internal/usecase/issue"
)

var (
	issueTestOrgID   = uuid.MustParse("00000000-0000-0000-0000-000000000042")
	issueTestUserID  = int64(7)
	issueTestRepoID  = uuid.MustParse("00000000-0000-0000-0000-000000000099")
	issueTestIssueID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	issueTestOwner   = "alice"
	issueTestRepo    = "demo"
)

type issueHandlerMockRepo struct {
	issues      map[int]*entity.Issue
	getByNumber func(ctx context.Context, repoID uuid.UUID, number int) (*entity.Issue, error)
	updateFn    func(ctx context.Context, issue *entity.Issue) error
}

func (m *issueHandlerMockRepo) Create(_ context.Context, issue *entity.Issue) error {
	if m.issues == nil {
		m.issues = map[int]*entity.Issue{}
	}
	m.issues[issue.Number] = issue
	return nil
}

func (m *issueHandlerMockRepo) GetByNumber(ctx context.Context, repoID uuid.UUID, number int) (*entity.Issue, error) {
	if m.getByNumber != nil {
		return m.getByNumber(ctx, repoID, number)
	}
	if m.issues == nil {
		return nil, apperror.ErrNotFound
	}
	issue, ok := m.issues[number]
	if !ok {
		return nil, apperror.ErrNotFound
	}
	return issue, nil
}

func (m *issueHandlerMockRepo) GetByID(_ context.Context, _ uuid.UUID) (*entity.Issue, error) {
	return nil, nil
}

func (m *issueHandlerMockRepo) ListByRepo(_ context.Context, _ repository.ListIssuesFilter) ([]*entity.Issue, int, error) {
	issues := make([]*entity.Issue, 0, len(m.issues))
	for _, issue := range m.issues {
		issues = append(issues, issue)
	}
	return issues, len(issues), nil
}

func (m *issueHandlerMockRepo) Update(ctx context.Context, issue *entity.Issue) error {
	if m.updateFn != nil {
		if err := m.updateFn(ctx, issue); err != nil {
			return err
		}
	}
	if m.issues == nil {
		m.issues = map[int]*entity.Issue{}
	}
	m.issues[issue.Number] = issue
	return nil
}

func (m *issueHandlerMockRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }

func (m *issueHandlerMockRepo) Count(_ context.Context, _ repository.ListIssuesFilter) (int, error) {
	return len(m.issues), nil
}

func (m *issueHandlerMockRepo) NextNumber(_ context.Context, _ uuid.UUID) (int, error) {
	return len(m.issues) + 1, nil
}

type issueHandlerMockLabelRepo struct{}

func (issueHandlerMockLabelRepo) Create(context.Context, *entity.Label) error { return nil }
func (issueHandlerMockLabelRepo) GetByName(context.Context, uuid.UUID, string) (*entity.Label, error) {
	return nil, nil
}
func (issueHandlerMockLabelRepo) ListByRepo(context.Context, uuid.UUID, int, int) ([]*entity.Label, int, error) {
	return nil, 0, nil
}
func (issueHandlerMockLabelRepo) Update(context.Context, *entity.Label) error { return nil }
func (issueHandlerMockLabelRepo) Delete(context.Context, uuid.UUID) error     { return nil }
func (issueHandlerMockLabelRepo) AddToIssue(context.Context, uuid.UUID, int, uuid.UUID) error {
	return nil
}
func (issueHandlerMockLabelRepo) RemoveFromIssue(context.Context, uuid.UUID, int, uuid.UUID) error {
	return nil
}

type issueHandlerMockMilestoneRepo struct{}

func (issueHandlerMockMilestoneRepo) Create(context.Context, *entity.Milestone) error { return nil }
func (issueHandlerMockMilestoneRepo) GetByNumber(context.Context, uuid.UUID, int) (*entity.Milestone, error) {
	return nil, nil
}
func (issueHandlerMockMilestoneRepo) ListByRepo(context.Context, uuid.UUID, string, int, int) ([]*entity.Milestone, int, error) {
	return nil, 0, nil
}
func (issueHandlerMockMilestoneRepo) Update(context.Context, *entity.Milestone) error { return nil }
func (issueHandlerMockMilestoneRepo) Delete(context.Context, uuid.UUID) error         { return nil }
func (issueHandlerMockMilestoneRepo) NextNumber(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}
func (issueHandlerMockMilestoneRepo) IncrOpenCount(context.Context, uuid.UUID) error { return nil }
func (issueHandlerMockMilestoneRepo) DecrOpenCount(context.Context, uuid.UUID) error { return nil }

type issueHandlerMockAuditLog struct{}

func (issueHandlerMockAuditLog) InsertAuditLog(context.Context, uuid.UUID, uuid.UUID, string, string, uuid.UUID, json.RawMessage) error {
	return nil
}
func (issueHandlerMockAuditLog) Create(context.Context, *entity.AuditLog) error { return nil }
func (issueHandlerMockAuditLog) List(context.Context, uuid.UUID, string, int, int) ([]*entity.AuditLog, int, error) {
	return nil, 0, nil
}

type issueHandlerMockTxManager struct{}

func (issueHandlerMockTxManager) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func issueHandlerAuthMiddleware(orgID uuid.UUID, userID int64) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middleware.SetActorContext(c, userID, orgID, []string{"repo"})
			return next(c)
		}
	}
}

func newIssueHandlerEcho(t *testing.T, issueRepo *issueHandlerMockRepo) *echo.Echo {
	t.Helper()

	createUC := issueusecase.NewCreateIssueUsecase(issueRepo, issueHandlerMockAuditLog{}, issueHandlerMockTxManager{})
	listUC := issueusecase.NewListIssuesUsecase(issueRepo)
	getUC := issueusecase.NewGetIssueUsecase(issueRepo)
	updateUC := issueusecase.NewUpdateIssueUsecase(
		issueRepo,
		issueHandlerMockLabelRepo{},
		issueHandlerMockMilestoneRepo{},
		issueHandlerMockAuditLog{},
	)

	resolveRepo := func(_ echo.Context, owner, name string) (*entity.Repository, error) {
		if owner == issueTestOwner && name == issueTestRepo {
			return &entity.Repository{
				ID:             issueTestRepoID,
				OrganizationID: issueTestOrgID,
				OwnerLogin:     owner,
				Name:           name,
			}, nil
		}
		return nil, echo.NewHTTPError(http.StatusNotFound, "not found")
	}

	h := handler.NewIssueHandler(createUC, listUC, getUC, updateUC, nil, resolveRepo)

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g, issueHandlerAuthMiddleware(issueTestOrgID, issueTestUserID))
	return e
}

func sampleIssue(number int, state string) *entity.Issue {
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	return &entity.Issue{
		ID:             issueTestIssueID,
		OrganizationID: issueTestOrgID,
		RepositoryID:   issueTestRepoID,
		Number:         number,
		Title:          "Bug report",
		Body:           "Something broke",
		State:          state,
		AuthorLogin:    issueTestOwner,
		Labels: []entity.Label{
			{Name: "bug", Color: "ff0000", Description: "Bug label"},
		},
		CommentsCount: 2,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func strPtr(s string) *string {
	return &s
}

func TestIssueGetOK(t *testing.T) {
	issueRepo := &issueHandlerMockRepo{
		issues: map[int]*entity.Issue{
			1: sampleIssue(1, "open"),
		},
	}
	e := newIssueHandlerEcho(t, issueRepo)

	req := httptest.NewRequest(http.MethodGet, "/repos/"+issueTestOwner+"/"+issueTestRepo+"/issues/1", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["number"].(float64) != 1 {
		t.Fatalf("number = %v, want 1", resp["number"])
	}
	user, ok := resp["user"].(map[string]any)
	if !ok || user["login"] != issueTestOwner {
		t.Fatalf("user.login = %v, want %q", user["login"], issueTestOwner)
	}
	labels, ok := resp["labels"].([]any)
	if !ok || len(labels) != 1 {
		t.Fatalf("labels = %v, want 1 label", resp["labels"])
	}
	if resp["created_at"] != "2026-06-28T12:00:00Z" {
		t.Fatalf("created_at = %v, want RFC3339 timestamp", resp["created_at"])
	}
}

func TestIssueGetNotFound(t *testing.T) {
	issueRepo := &issueHandlerMockRepo{
		getByNumber: func(context.Context, uuid.UUID, int) (*entity.Issue, error) {
			return nil, apperror.ErrNotFound
		},
	}
	e := newIssueHandlerEcho(t, issueRepo)

	req := httptest.NewRequest(http.MethodGet, "/repos/"+issueTestOwner+"/"+issueTestRepo+"/issues/99", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestIssueCreateEmptyTitle422(t *testing.T) {
	e := newIssueHandlerEcho(t, &issueHandlerMockRepo{})

	body := bytes.NewBufferString(`{"title":"","body":"hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/repos/"+issueTestOwner+"/"+issueTestRepo+"/issues", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

func TestIssuePatchStateChangeOK(t *testing.T) {
	closedAt := time.Date(2026, 6, 28, 13, 0, 0, 0, time.UTC)
	issueRepo := &issueHandlerMockRepo{
		issues: map[int]*entity.Issue{
			1: sampleIssue(1, "open"),
		},
		updateFn: func(_ context.Context, issue *entity.Issue) error {
			if issue.State == "closed" {
				issue.StateReason = strPtr("completed")
				issue.ClosedAt = &closedAt
			}
			return nil
		},
	}

	e := newIssueHandlerEcho(t, issueRepo)

	body := bytes.NewBufferString(`{"state":"closed","state_reason":"completed"}`)
	req := httptest.NewRequest(http.MethodPatch, "/repos/"+issueTestOwner+"/"+issueTestRepo+"/issues/1", body)
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
	if resp["state"] != "closed" {
		t.Fatalf("state = %v, want closed", resp["state"])
	}
	if resp["closed_at"] == nil {
		t.Fatalf("closed_at = nil, want RFC3339 string")
	}
}

func TestPatchIssueClose(t *testing.T) {
	issueRepo := &issueHandlerMockRepo{
		issues: map[int]*entity.Issue{
			1: {
				ID:           issueTestIssueID,
				RepositoryID: issueTestRepoID,
				Number:       1,
				Title:        "Bug",
				State:        "open",
				AuthorLogin:  issueTestOwner,
			},
		},
	}
	e := newIssueHandlerEcho(t, issueRepo)

	body := bytes.NewBufferString(`{"state":"closed"}`)
	req := httptest.NewRequest(http.MethodPatch, "/repos/"+issueTestOwner+"/"+issueTestRepo+"/issues/1", body)
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
	if resp["state"] != "closed" {
		t.Fatalf("state = %v, want closed", resp["state"])
	}
}

func TestPatchIssueOpen(t *testing.T) {
	issueRepo := &issueHandlerMockRepo{
		issues: map[int]*entity.Issue{
			1: {
				ID:           issueTestIssueID,
				RepositoryID: issueTestRepoID,
				Number:       1,
				Title:        "Bug",
				State:        "closed",
				AuthorLogin:  issueTestOwner,
			},
		},
	}
	e := newIssueHandlerEcho(t, issueRepo)

	body := bytes.NewBufferString(`{"state":"open"}`)
	req := httptest.NewRequest(http.MethodPatch, "/repos/"+issueTestOwner+"/"+issueTestRepo+"/issues/1", body)
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
	if resp["state"] != "open" {
		t.Fatalf("state = %v, want open", resp["state"])
	}
}

func TestPatchIssueNotFound(t *testing.T) {
	e := newIssueHandlerEcho(t, &issueHandlerMockRepo{issues: map[int]*entity.Issue{}})

	body := bytes.NewBufferString(`{"state":"closed"}`)
	req := httptest.NewRequest(http.MethodPatch, "/repos/"+issueTestOwner+"/"+issueTestRepo+"/issues/999", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestPatchIssueInvalidState(t *testing.T) {
	issueRepo := &issueHandlerMockRepo{
		issues: map[int]*entity.Issue{
			1: {
				ID:           issueTestIssueID,
				RepositoryID: issueTestRepoID,
				Number:       1,
				Title:        "Bug",
				State:        "open",
				AuthorLogin:  issueTestOwner,
			},
		},
	}
	e := newIssueHandlerEcho(t, issueRepo)

	body := bytes.NewBufferString(`{"state":"invalid"}`)
	req := httptest.NewRequest(http.MethodPatch, "/repos/"+issueTestOwner+"/"+issueTestRepo+"/issues/1", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

func TestListIssuesContainsNodeIDAndHTMLURL(t *testing.T) {
	issueRepo := &issueHandlerMockRepo{
		issues: map[int]*entity.Issue{
			1: {
				ID:           issueTestIssueID,
				RepositoryID: issueTestRepoID,
				Number:       1,
				Title:        "Bug",
				State:        "open",
				AuthorLogin:  issueTestOwner,
			},
		},
	}
	e := newIssueHandlerEcho(t, issueRepo)

	req := httptest.NewRequest(http.MethodGet, "/repos/"+issueTestOwner+"/"+issueTestRepo+"/issues", nil)
	req.Host = "git.example.com"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("len = %d, want 1", len(resp))
	}

	item := resp[0]
	nodeID, ok := item["node_id"].(string)
	if !ok || nodeID == "" {
		t.Fatalf("node_id = %v, want non-empty string", item["node_id"])
	}
	htmlURL, ok := item["html_url"].(string)
	if !ok || htmlURL == "" {
		t.Fatalf("html_url = %v, want non-empty string", item["html_url"])
	}
	if htmlURL != "https://git.example.com/"+issueTestOwner+"/"+issueTestRepo+"/issues/1" {
		t.Fatalf("html_url = %q, want https://git.example.com/%s/%s/issues/1", htmlURL, issueTestOwner, issueTestRepo)
	}
}
