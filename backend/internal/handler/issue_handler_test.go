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
	issueTestUserID  = uuid.MustParse("00000000-0000-0000-0000-000000000007")
	issueTestRepoID  = uuid.MustParse("00000000-0000-0000-0000-000000000099")
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

func (m *issueHandlerMockRepo) ListByRepo(_ context.Context, _ repository.ListIssuesFilter) ([]*entity.Issue, int, error) {
	issues := make([]*entity.Issue, 0, len(m.issues))
	for _, issue := range m.issues {
		issues = append(issues, issue)
	}
	return issues, len(issues), nil
}

func (m *issueHandlerMockRepo) NextNumber(_ context.Context, _ uuid.UUID) (int, error) {
	return len(m.issues) + 1, nil
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

type issueHandlerMockAuditLog struct{}

func (issueHandlerMockAuditLog) InsertAuditLog(context.Context, uuid.UUID, uuid.UUID, string, string, uuid.UUID, json.RawMessage) error {
	return nil
}

type issueHandlerMockTxManager struct{}

func (issueHandlerMockTxManager) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func issueHandlerAuthMiddleware(orgID, userID uuid.UUID) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			middleware.SetActor(c, middleware.Actor{
				UserID:         userID,
				OrganizationID: orgID,
			})
			c.Set("scopes", []string{"repo"})
			return next(c)
		}
	}
}

func newIssueHandlerEcho(
	t *testing.T,
	issueRepo *issueHandlerMockRepo,
) *echo.Echo {
	t.Helper()

	createUC := issueusecase.NewCreateIssueUsecase(issueRepo, issueHandlerMockAuditLog{}, issueHandlerMockTxManager{})
	listUC := issueusecase.NewListIssuesUsecase(issueRepo)
	getUC := issueusecase.NewGetIssueUsecase(issueRepo)
	updateUC := issueusecase.NewUpdateIssueUsecase(issueRepo, issueHandlerMockAuditLog{}, issueHandlerMockTxManager{})

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
		ID:             uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		OrganizationID: issueTestOrgID,
		RepositoryID:   issueTestRepoID,
		Number:         number,
		Title:          "Bug report",
		Body:           "Something broke",
		State:          state,
		AuthorLogin:    issueTestOwner,
		Labels: []entity.IssueLabel{
			{Name: "bug", Color: "ff0000", Description: "Bug label"},
		},
		CommentsCount: 2,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
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

func strPtr(s string) *string {
	return &s
}
