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
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/handler"
	"github.com/open-git/backend/internal/infrastructure/queue"
	"github.com/open-git/backend/internal/middleware"
	mcpusecase "github.com/open-git/backend/internal/usecase/mcp"
)

var (
	mcpTestUserID = int64(7)
	mcpTestOrgID  = uuid.MustParse("00000000-0000-0000-0000-000000000042")
	mcpTestRunID  = uuid.MustParse("00000000-0000-0000-0000-000000000501")
)

type handlerMCPRepo struct {
	runs                []*entity.MCPVerificationRun
	checks              []*entity.MCPVerificationCheck
	countRunsThisMonth  int64
	countRunsThisMonthSet bool
}

var _ domainrepo.IMCPVerificationRepository = (*handlerMCPRepo)(nil)

func (m *handlerMCPRepo) CreateRun(_ context.Context, run *entity.MCPVerificationRun) error {
	if run.ID == uuid.Nil {
		run.ID = uuid.New()
	}
	if run.CreatedAt.IsZero() {
		run.CreatedAt = time.Now().UTC()
	}
	stored := *run
	m.runs = append(m.runs, &stored)
	return nil
}

func (m *handlerMCPRepo) GetRunByID(_ context.Context, id, orgID uuid.UUID) (*entity.MCPVerificationRun, error) {
	for _, run := range m.runs {
		if run.ID == id && run.OrganizationID == orgID {
			stored := *run
			return &stored, nil
		}
	}
	return nil, nil
}

func (m *handlerMCPRepo) UpdateRun(_ context.Context, run *entity.MCPVerificationRun) error {
	for i, existing := range m.runs {
		if existing.ID == run.ID && existing.OrganizationID == run.OrganizationID {
			stored := *run
			m.runs[i] = &stored
			return nil
		}
	}
	return nil
}

func (m *handlerMCPRepo) DeleteRun(_ context.Context, id, orgID uuid.UUID) error {
	filtered := make([]*entity.MCPVerificationRun, 0, len(m.runs))
	found := false
	for _, run := range m.runs {
		if run.ID == id && run.OrganizationID == orgID {
			found = true
			continue
		}
		filtered = append(filtered, run)
	}
	if !found {
		return nil
	}
	m.runs = filtered
	return nil
}

func (m *handlerMCPRepo) GetLatestRun(_ context.Context, orgID uuid.UUID) (*entity.MCPVerificationRun, error) {
	var latest *entity.MCPVerificationRun
	for _, run := range m.runs {
		if run.OrganizationID != orgID {
			continue
		}
		if latest == nil || run.CreatedAt.After(latest.CreatedAt) {
			stored := *run
			latest = &stored
		}
	}
	return latest, nil
}

func (m *handlerMCPRepo) ListRuns(_ context.Context, orgID uuid.UUID, _, _ int) ([]*entity.MCPVerificationRun, int64, error) {
	return nil, 0, nil
}

func (m *handlerMCPRepo) GetActiveRun(_ context.Context, orgID uuid.UUID) (*entity.MCPVerificationRun, error) {
	for _, run := range m.runs {
		if run.OrganizationID != orgID {
			continue
		}
		if run.Status == entity.RunStatusQueued || run.Status == entity.RunStatusRunning {
			stored := *run
			return &stored, nil
		}
	}
	return nil, nil
}

func (m *handlerMCPRepo) CountRunsThisMonth(_ context.Context, orgID uuid.UUID) (int64, error) {
	if m.countRunsThisMonthSet {
		return m.countRunsThisMonth, nil
	}

	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	var count int64
	for _, run := range m.runs {
		if run.OrganizationID == orgID && !run.CreatedAt.Before(monthStart) {
			count++
		}
	}
	return count, nil
}

func (m *handlerMCPRepo) BatchCreateChecks(_ context.Context, checks []*entity.MCPVerificationCheck) error {
	return nil
}

func (m *handlerMCPRepo) ListChecksByRun(_ context.Context, runID, orgID uuid.UUID) ([]*entity.MCPVerificationCheck, error) {
	var out []*entity.MCPVerificationCheck
	for _, check := range m.checks {
		if check.RunID == runID && check.OrganizationID == orgID {
			stored := *check
			out = append(out, &stored)
		}
	}
	return out, nil
}

type handlerMCPAuditRepo struct{}

func (handlerMCPAuditRepo) Create(context.Context, *entity.AuditLog) error {
	return nil
}

func (handlerMCPAuditRepo) List(context.Context, uuid.UUID, string, int, int) ([]*entity.AuditLog, int, error) {
	return nil, 0, nil
}

func (handlerMCPAuditRepo) InsertAuditLog(context.Context, uuid.UUID, uuid.UUID, string, string, uuid.UUID, json.RawMessage) error {
	return nil
}

type handlerMCPEnqueuer struct{}

func (handlerMCPEnqueuer) EnqueueMCPVerification(_ context.Context, _ queue.MCPVerificationPayload) error {
	return nil
}

func mcpAdminAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		middleware.SetAuthContext(c, mcpTestUserID, []string{"admin"})
		return next(c)
	}
}

func newMCPVerificationHandlerEcho(t *testing.T, repo *handlerMCPRepo) *echo.Echo {
	t.Helper()

	runUC := mcpusecase.NewRunVerificationUsecaseWithDeps(repo, handlerMCPAuditRepo{}, handlerMCPEnqueuer{})
	getLatestUC := mcpusecase.NewGetLatestVerificationUsecase(repo)
	listHistoryUC := mcpusecase.NewListVerificationHistoryUsecase(repo)
	getJobUC := mcpusecase.NewGetJobStatusUsecase(repo)
	deleteUC := mcpusecase.NewDeleteVerificationUsecase(repo, handlerMCPAuditRepo{})

	h := handler.NewMCPVerificationHandler(runUC, getLatestUC, listHistoryUC, getJobUC, deleteUC)

	e := echo.New()
	g := e.Group("/api/v1")
	h.RegisterRoutes(g, mcpAdminAuth)
	return e
}

func TestRunVerification_202(t *testing.T) {
	repo := &handlerMCPRepo{}
	e := newMCPVerificationHandlerEcho(t, repo)

	body := []byte(`{"repository":"acme/widgets","targets":["rest"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/verification/run?organization_id="+mcpTestOrgID.String(), bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusAccepted, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := resp["job_id"]; !ok {
		t.Fatalf("expected job_id in response, got %v", resp)
	}
	if resp["status"] != "queued" {
		t.Fatalf("status = %v, want queued", resp["status"])
	}
}

func TestRunVerification_409(t *testing.T) {
	repo := &handlerMCPRepo{
		runs: []*entity.MCPVerificationRun{{
			ID:                 uuid.New(),
			OrganizationID:     mcpTestOrgID,
			RepositoryFullName: "acme/widgets",
			Status:             entity.RunStatusRunning,
			CreatedAt:          time.Now().UTC(),
		}},
	}
	e := newMCPVerificationHandlerEcho(t, repo)

	body := []byte(`{"repository":"acme/widgets","targets":["rest"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/verification/run?organization_id="+mcpTestOrgID.String(), bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestRunVerification_403(t *testing.T) {
	repo := &handlerMCPRepo{
		countRunsThisMonthSet: true,
		countRunsThisMonth:    10,
	}
	e := newMCPVerificationHandlerEcho(t, repo)

	body := []byte(`{"repository":"acme/widgets","targets":["rest"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/verification/run?organization_id="+mcpTestOrgID.String(), bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestGetLatest_200(t *testing.T) {
	finishedAt := time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)
	overall := entity.OverallStatusCompatible
	repo := &handlerMCPRepo{
		runs: []*entity.MCPVerificationRun{{
			ID:                 mcpTestRunID,
			OrganizationID:     mcpTestOrgID,
			RepositoryFullName: "octo-org/hello-repo",
			Status:             entity.RunStatusCompleted,
			OverallStatus:      &overall,
			FinishedAt:         &finishedAt,
			CreatedAt:          finishedAt,
		}},
		checks: []*entity.MCPVerificationCheck{{
			ID:             uuid.New(),
			RunID:          mcpTestRunID,
			OrganizationID: mcpTestOrgID,
			CheckID:        "graphql.viewer",
			Category:       entity.CheckCategoryGraphQL,
			Status:         entity.CheckStatusPass,
			Expected:       json.RawMessage(`{"status_code":200}`),
			Actual:         json.RawMessage(`{"status_code":200}`),
			DurationMS:     120,
			CreatedAt:      finishedAt,
		}},
	}
	e := newMCPVerificationHandlerEcho(t, repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mcp/verification/latest?organization_id="+mcpTestOrgID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["run_id"] != mcpTestRunID.String() {
		t.Fatalf("run_id = %v, want %s", resp["run_id"], mcpTestRunID)
	}
	if resp["repository"] != "octo-org/hello-repo" {
		t.Fatalf("repository = %v, want octo-org/hello-repo", resp["repository"])
	}
	checks, ok := resp["checks"].([]any)
	if !ok || len(checks) != 1 {
		t.Fatalf("checks = %v, want one check", resp["checks"])
	}
}

func TestDeleteRun_204(t *testing.T) {
	repo := &handlerMCPRepo{
		runs: []*entity.MCPVerificationRun{{
			ID:                 mcpTestRunID,
			OrganizationID:     mcpTestOrgID,
			RepositoryFullName: "octo-org/hello-repo",
			Status:             entity.RunStatusCompleted,
			CreatedAt:          time.Now().UTC(),
		}},
	}
	e := newMCPVerificationHandlerEcho(t, repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/mcp/verification/runs/"+mcpTestRunID.String()+"?organization_id="+mcpTestOrgID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
}
