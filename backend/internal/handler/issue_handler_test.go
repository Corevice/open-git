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
	"github.com/open-git/backend/internal/handler"
	"github.com/open-git/backend/internal/middleware"
	issueusecase "github.com/open-git/backend/internal/usecase/issue"
)

var (
	issueTestUserID  = int64(7)
	issueTestOrgUUID = uuid.MustParse("00000000-0000-0000-0000-000000000042")
	issueTestRepoID  = uuid.MustParse("00000000-0000-0000-0000-000000000099")
	issueTestIssueID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
)

type handlerIssueRepo struct {
	issues map[int]*entity.Issue
}

func (m *handlerIssueRepo) Create(_ context.Context, _ *entity.Issue) error { return nil }

func (m *handlerIssueRepo) GetByNumber(_ context.Context, _ uuid.UUID, number int) (*entity.Issue, error) {
	if m.issues == nil {
		return nil, nil
	}
	return m.issues[number], nil
}

func (m *handlerIssueRepo) GetByID(_ context.Context, _ uuid.UUID) (*entity.Issue, error) {
	return nil, nil
}

func (m *handlerIssueRepo) ListByRepo(_ context.Context, filter repository.ListIssuesFilter) ([]*entity.Issue, int, error) {
	if m.issues == nil {
		return []*entity.Issue{}, 0, nil
	}
	issues := make([]*entity.Issue, 0, len(m.issues))
	for _, issue := range m.issues {
		issues = append(issues, issue)
	}
	return issues, len(issues), nil
}

func (m *handlerIssueRepo) Update(_ context.Context, issue *entity.Issue) error {
	if m.issues == nil {
		m.issues = map[int]*entity.Issue{}
	}
	m.issues[issue.Number] = issue
	return nil
}

func (m *handlerIssueRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }

func (m *handlerIssueRepo) Count(_ context.Context, _ repository.ListIssuesFilter) (int, error) {
	return len(m.issues), nil
}

func (m *handlerIssueRepo) NextNumber(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil }

type handlerAuditLogRepo struct{}

func (m *handlerAuditLogRepo) InsertAuditLog(context.Context, uuid.UUID, uuid.UUID, string, string, uuid.UUID, json.RawMessage) error {
	return nil
}

func (handlerAuditLogRepo) Create(context.Context, *entity.AuditLog) error {
	return nil
}

func (handlerAuditLogRepo) List(context.Context, uuid.UUID, string, int, int) ([]*entity.AuditLog, int, error) {
	return nil, 0, nil
}

func issueTestRepo() *entity.Repository {
	return &entity.Repository{
		ID:             issueTestRepoID,
		OrganizationID: issueTestOrgUUID,
		OwnerLogin:     "alice",
		Name:           "demo",
	}
}

func issueTestAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		middleware.SetAuthContext(c, issueTestUserID, []string{"repo"})
		return next(c)
	}
}

func newIssueHandlerEcho(t *testing.T, issueRepo *handlerIssueRepo) *echo.Echo {
	t.Helper()

	listIssuesUC := issueusecase.NewListIssuesUsecase(issueRepo)
	updateIssueUC := issueusecase.NewUpdateIssueUsecase(issueRepo, nil, nil, &handlerAuditLogRepo{})

	h := handler.NewIssueHandler(
		nil,
		listIssuesUC,
		nil,
		updateIssueUC,
		func(_ echo.Context, _, _ string) (*entity.Repository, error) {
			return issueTestRepo(), nil
		},
	)

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g, issueTestAuth)
	return e
}

func TestPatchIssueClose(t *testing.T) {
	issueRepo := &handlerIssueRepo{
		issues: map[int]*entity.Issue{
			1: {
				ID:           issueTestIssueID,
				RepositoryID: issueTestRepoID,
				Number:       1,
				Title:        "Bug",
				State:        "open",
				AuthorLogin:  "alice",
			},
		},
	}
	e := newIssueHandlerEcho(t, issueRepo)

	body := bytes.NewBufferString(`{"state":"closed"}`)
	req := httptest.NewRequest(http.MethodPatch, "/repos/alice/demo/issues/1", body)
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
	issueRepo := &handlerIssueRepo{
		issues: map[int]*entity.Issue{
			1: {
				ID:           issueTestIssueID,
				RepositoryID: issueTestRepoID,
				Number:       1,
				Title:        "Bug",
				State:        "closed",
				AuthorLogin:  "alice",
			},
		},
	}
	e := newIssueHandlerEcho(t, issueRepo)

	body := bytes.NewBufferString(`{"state":"open"}`)
	req := httptest.NewRequest(http.MethodPatch, "/repos/alice/demo/issues/1", body)
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
	e := newIssueHandlerEcho(t, &handlerIssueRepo{issues: map[int]*entity.Issue{}})

	body := bytes.NewBufferString(`{"state":"closed"}`)
	req := httptest.NewRequest(http.MethodPatch, "/repos/alice/demo/issues/999", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestPatchIssueInvalidState(t *testing.T) {
	issueRepo := &handlerIssueRepo{
		issues: map[int]*entity.Issue{
			1: {
				ID:           issueTestIssueID,
				RepositoryID: issueTestRepoID,
				Number:       1,
				Title:        "Bug",
				State:        "open",
				AuthorLogin:  "alice",
			},
		},
	}
	e := newIssueHandlerEcho(t, issueRepo)

	body := bytes.NewBufferString(`{"state":"invalid"}`)
	req := httptest.NewRequest(http.MethodPatch, "/repos/alice/demo/issues/1", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

func TestListIssuesContainsNodeIDAndHTMLURL(t *testing.T) {
	issueRepo := &handlerIssueRepo{
		issues: map[int]*entity.Issue{
			1: {
				ID:           issueTestIssueID,
				RepositoryID: issueTestRepoID,
				Number:       1,
				Title:        "Bug",
				State:        "open",
				AuthorLogin:  "alice",
				CreatedAt:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}
	e := newIssueHandlerEcho(t, issueRepo)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/issues", nil)
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
	if htmlURL != "https://git.example.com/alice/demo/issues/1" {
		t.Fatalf("html_url = %q, want https://git.example.com/alice/demo/issues/1", htmlURL)
	}
	if _, ok := item["id"].(float64); !ok {
		t.Fatalf("id = %v (%T), want numeric", item["id"], item["id"])
	}
	if _, ok := item["created_at"]; !ok {
		t.Fatalf("created_at missing from response: %v", item)
	}
}
