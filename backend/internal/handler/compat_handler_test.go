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

	"github.com/open-git/backend/internal/compat"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/handler"
	"github.com/open-git/backend/internal/middleware"
	compatusecase "github.com/open-git/backend/internal/usecase/compat"
)

var (
	compatTestUserID  = int64(7)
	compatTestOrgID   = uuid.MustParse("00000000-0000-0000-0000-000000000042")
	compatTestRunID   = uuid.MustParse("00000000-0000-0000-0000-000000000301")
	compatUnknownID   = uuid.MustParse("00000000-0000-0000-0000-000000000999")
)

type handlerCompatRepo struct {
	runs     map[uuid.UUID]*entity.CompatTestRun
	endpoints map[uuid.UUID][]*entity.CompatEndpointResult
}

func (m *handlerCompatRepo) CreateRun(_ context.Context, run *entity.CompatTestRun) error {
	if m.runs == nil {
		m.runs = map[uuid.UUID]*entity.CompatTestRun{}
	}
	if run.ID == uuid.Nil {
		run.ID = uuid.New()
	}
	copyRun := *run
	m.runs[run.ID] = &copyRun
	return nil
}

func (m *handlerCompatRepo) UpdateRun(_ context.Context, run *entity.CompatTestRun) error {
	if m.runs == nil {
		return nil
	}
	copyRun := *run
	m.runs[run.ID] = &copyRun
	return nil
}

func (m *handlerCompatRepo) GetRun(_ context.Context, id uuid.UUID) (*entity.CompatTestRun, error) {
	if m.runs == nil {
		return nil, nil
	}
	run, ok := m.runs[id]
	if !ok {
		return nil, nil
	}
	copyRun := *run
	return &copyRun, nil
}

func (m *handlerCompatRepo) ListRuns(_ context.Context, orgID uuid.UUID, limit int) ([]*entity.CompatTestRun, error) {
	if m.runs == nil {
		return []*entity.CompatTestRun{}, nil
	}
	runs := make([]*entity.CompatTestRun, 0)
	for _, run := range m.runs {
		if run.OrganizationID == orgID {
			copyRun := *run
			runs = append(runs, &copyRun)
		}
	}
	if limit > 0 && len(runs) > limit {
		runs = runs[:limit]
	}
	return runs, nil
}

func (m *handlerCompatRepo) CreateEndpointResult(_ context.Context, result *entity.CompatEndpointResult) error {
	if m.endpoints == nil {
		m.endpoints = map[uuid.UUID][]*entity.CompatEndpointResult{}
	}
	if result.ID == uuid.Nil {
		result.ID = uuid.New()
	}
	copyResult := *result
	m.endpoints[result.RunID] = append(m.endpoints[result.RunID], &copyResult)
	return nil
}

func (m *handlerCompatRepo) ListEndpointResults(_ context.Context, runID uuid.UUID) ([]*entity.CompatEndpointResult, error) {
	results := m.endpoints[runID]
	if len(results) == 0 {
		return []*entity.CompatEndpointResult{}, nil
	}
	out := make([]*entity.CompatEndpointResult, 0, len(results))
	for _, result := range results {
		copyResult := *result
		out = append(out, &copyResult)
	}
	return out, nil
}

var _ repository.ICompatRepository = (*handlerCompatRepo)(nil)

func compatAdminAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		middleware.SetAuthContext(c, compatTestUserID, []string{"admin"})
		return next(c)
	}
}

func compatRequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if middleware.UserIDFromContext(c) == 0 {
			return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
		}
		return next(c)
	}
}

func newCompatHandlerEcho(t *testing.T, repo *handlerCompatRepo, auth echo.MiddlewareFunc) *echo.Echo {
	t.Helper()

	getReportUC := compatusecase.NewGetReportUsecase(repo)
	triggerRunUC := compatusecase.NewTriggerRunUsecase(repo, &compat.Runner{})
	h := handler.NewCompatHandler(getReportUC, triggerRunUC, repo)

	e := echo.New()
	g := e.Group("/api/v1")
	h.RegisterRoutes(g, auth)
	return e
}

func TestGetReportReturns200WithCoverageStructure(t *testing.T) {
	finishedAt := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &handlerCompatRepo{
		runs: map[uuid.UUID]*entity.CompatTestRun{
			compatTestRunID: {
				ID:             compatTestRunID,
				Suite:          "rest-v3",
				Status:         entity.CompatStatusCompleted,
				OrganizationID: compatTestOrgID,
				TotalEndpoints: 2,
				Passing:        1,
				Failing:        1,
				FinishedAt:     &finishedAt,
			},
		},
		endpoints: map[uuid.UUID][]*entity.CompatEndpointResult{
			compatTestRunID: {
				{
					RunID:  compatTestRunID,
					Method: "GET",
					Path:   "/user",
					Status: entity.CompatResultPass,
					Checks: &entity.CompatEndpointChecks{
						Schema:     true,
						StatusCode: true,
						Headers:    true,
						Pagination: true,
					},
				},
				{
					RunID:  compatTestRunID,
					Method: "POST",
					Path:   "/repos/{owner}/{repo}/pulls",
					Status: entity.CompatResultFail,
					Checks: &entity.CompatEndpointChecks{
						Schema:     false,
						StatusCode: true,
						Headers:    true,
						Pagination: true,
					},
				},
			},
		},
	}

	e := newCompatHandlerEcho(t, repo, compatAdminAuth)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/internal/compat/report?organization_id="+compatTestOrgID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := resp["generated_at"]; !ok {
		t.Fatalf("missing generated_at: %v", resp)
	}

	coverage, ok := resp["coverage"].(map[string]any)
	if !ok {
		t.Fatalf("coverage = %v, want object", resp["coverage"])
	}
	for _, key := range []string{"total_endpoints", "passing", "failing", "unimplemented", "rate"} {
		if _, ok := coverage[key]; !ok {
			t.Fatalf("coverage missing %s: %v", key, coverage)
		}
	}

	endpoints, ok := resp["endpoints"].([]any)
	if !ok {
		t.Fatalf("endpoints = %v, want array", resp["endpoints"])
	}
	if len(endpoints) != 2 {
		t.Fatalf("len(endpoints) = %d, want 2", len(endpoints))
	}
}

func TestTriggerRunReturns202WithJobID(t *testing.T) {
	repo := &handlerCompatRepo{}
	e := newCompatHandlerEcho(t, repo, compatAdminAuth)

	body := bytes.NewBufferString(`{"suite":"rest-v3","filter":["issues"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/compat/run?organization_id="+compatTestOrgID.String(), body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusAccepted, rec.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["job_id"] == "" {
		t.Fatalf("job_id = %q, want non-empty", resp["job_id"])
	}
	if resp["status"] != "queued" {
		t.Fatalf("status = %q, want queued", resp["status"])
	}
}

func TestGetRunStatusReturns200WithStatusField(t *testing.T) {
	repo := &handlerCompatRepo{
		runs: map[uuid.UUID]*entity.CompatTestRun{
			compatTestRunID: {
				ID:     compatTestRunID,
				Status: entity.CompatStatusRunning,
				Suite:  "rest-v3",
			},
		},
	}
	e := newCompatHandlerEcho(t, repo, compatAdminAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/internal/compat/run/"+compatTestRunID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["job_id"] != compatTestRunID.String() {
		t.Fatalf("job_id = %q, want %q", resp["job_id"], compatTestRunID.String())
	}
	if resp["status"] != entity.CompatStatusRunning {
		t.Fatalf("status = %q, want %q", resp["status"], entity.CompatStatusRunning)
	}
}

func TestGetRunStatusReturns404ForUnknownJobID(t *testing.T) {
	repo := &handlerCompatRepo{}
	e := newCompatHandlerEcho(t, repo, compatAdminAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/internal/compat/run/"+compatUnknownID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestCompatUnauthenticatedRequestReturns401(t *testing.T) {
	repo := &handlerCompatRepo{}
	e := newCompatHandlerEcho(t, repo, compatRequireAuth)

	tests := []struct {
		name   string
		method string
		path   string
		body   *bytes.Buffer
	}{
		{
			name:   "GetReport",
			method: http.MethodGet,
			path:   "/api/v1/internal/compat/report",
		},
		{
			name:   "TriggerRun",
			method: http.MethodPost,
			path:   "/api/v1/internal/compat/run",
			body:   bytes.NewBufferString(`{"suite":"rest-v3"}`),
		},
		{
			name:   "GetRunStatus",
			method: http.MethodGet,
			path:   "/api/v1/internal/compat/run/" + compatTestRunID.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != nil {
				req = httptest.NewRequest(tt.method, tt.path, tt.body)
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
			}
		})
	}
}
