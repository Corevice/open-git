package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/handler"
	"github.com/open-git/backend/internal/middleware"
	workflowusecase "github.com/open-git/backend/internal/usecase/workflow"
)

var (
	wfTestUserID  = int64(7)
	wfTestOrgUUID = uuid.MustParse("00000000-0000-0000-0000-000000000042")
	wfTestRepoID  = uuid.MustParse("00000000-0000-0000-0000-000000000099")
	wfTestRunID   = uuid.MustParse("00000000-0000-0000-0000-000000000201")
	wfTestJobID   = uuid.MustParse("00000000-0000-0000-0000-000000000301")
)

type wfHandlerFullRunRepo struct {
	runs      []*entity.WorkflowRun
	run       *entity.WorkflowRun
	cancelled bool
	rerun     *entity.WorkflowRun
}

func (m *wfHandlerFullRunRepo) List(_ context.Context, _ workflowusecase.ListRunsFilter) ([]*entity.WorkflowRun, int, error) {
	return m.runs, len(m.runs), nil
}

func (m *wfHandlerFullRunRepo) GetByID(_ context.Context, _, _, _ uuid.UUID) (*entity.WorkflowRun, error) {
	return m.run, nil
}

func (m *wfHandlerFullRunRepo) Cancel(_ context.Context, _, _, _, _ uuid.UUID) error {
	m.cancelled = true
	return nil
}

func (m *wfHandlerFullRunRepo) Rerun(_ context.Context, _, _, _, _ uuid.UUID) (*entity.WorkflowRun, error) {
	if m.rerun != nil {
		return m.rerun, nil
	}
	return m.run, nil
}

func wfTestRepo() *entity.Repository {
	return &entity.Repository{
		ID:             wfTestRepoID,
		OrganizationID: wfTestOrgUUID,
		OwnerLogin:     "alice",
		Name:           "demo",
	}
}

func wfTestAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		middleware.SetAuthContext(c, wfTestUserID, []string{"read", "write", "repo"})
		return next(c)
	}
}

func newWorkflowRunHandlerEcho(t *testing.T, runRepo *wfHandlerFullRunRepo, resolveRepo func(echo.Context, string, string) (*entity.Repository, error)) *echo.Echo {
	t.Helper()

	if runRepo == nil {
		runRepo = &wfHandlerFullRunRepo{}
	}
	if resolveRepo == nil {
		resolveRepo = func(_ echo.Context, _, _ string) (*entity.Repository, error) {
			return wfTestRepo(), nil
		}
	}

	listRunsUC := workflowusecase.NewListRunsUsecase(runRepo)
	getRunUC := workflowusecase.NewGetRunUsecase(runRepo)
	cancelRunUC := workflowusecase.NewCancelRunUsecase(runRepo)
	rerunUC := workflowusecase.NewRerunRunUsecase(runRepo)
	listJobsUC := workflowusecase.NewListJobsUsecase(&wfHandlerJobRepo{})

	h := handler.NewWorkflowRunHandler(
		listRunsUC,
		getRunUC,
		cancelRunUC,
		rerunUC,
		listJobsUC,
		resolveRepo,
		nil,
		nil,
	)

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g, wfTestAuth)
	return e
}

type wfStreamJobLogRepo struct {
	chunks []*handler.JobLogChunk
}

func (m *wfStreamJobLogRepo) ListByJobIDFromOffset(_ context.Context, _ uuid.UUID, offset int64, limit int) ([]*handler.JobLogChunk, error) {
	result := make([]*handler.JobLogChunk, 0, limit)
	for _, chunk := range m.chunks {
		if chunk.Offset < offset {
			continue
		}
		result = append(result, chunk)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

type wfStreamWorkflowJobRepo struct {
	job *entity.WorkflowJob
}

func (m *wfStreamWorkflowJobRepo) Create(_ context.Context, _ *entity.WorkflowJob) error {
	return nil
}

func (m *wfStreamWorkflowJobRepo) GetByID(_ context.Context, _ uuid.UUID) (*entity.WorkflowJob, error) {
	if m.job == nil {
		return nil, domain.ErrNotFound
	}
	return m.job, nil
}

func (m *wfStreamWorkflowJobRepo) AcquireForRunner(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ int) (bool, error) {
	return false, nil
}

func (m *wfStreamWorkflowJobRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _, _ string) error {
	return nil
}

func (m *wfStreamWorkflowJobRepo) Complete(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}

func (m *wfStreamWorkflowJobRepo) Cancel(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *wfStreamWorkflowJobRepo) ListQueued(_ context.Context, _ uuid.UUID) ([]*entity.WorkflowJob, error) {
	return nil, nil
}

func newWorkflowRunStreamHandler(t *testing.T, logRepo handler.IJobLogRepository, jobRepo *wfStreamWorkflowJobRepo) *handler.WorkflowRunHandler {
	t.Helper()

	return handler.NewWorkflowRunHandler(
		nil,
		nil,
		nil,
		nil,
		nil,
		func(_ echo.Context, _, _ string) (*entity.Repository, error) {
			return wfTestRepo(), nil
		},
		logRepo,
		jobRepo,
	)
}

type wfHandlerJobRepo struct{}

func (wfHandlerJobRepo) ListByRunID(_ context.Context, _, _, _ uuid.UUID) ([]*workflowusecase.WorkflowJob, error) {
	return []*workflowusecase.WorkflowJob{}, nil
}

func TestListRuns_StatusFilter(t *testing.T) {
	runRepo := &wfHandlerFullRunRepo{
		runs: []*entity.WorkflowRun{
			{
				ID:           wfTestRunID,
				RepositoryID: wfTestRepoID,
				Status:       entity.WorkflowStatusCompleted,
				Conclusion:   entity.WorkflowConclusionSuccess,
				Workflow:     "ci",
			},
			{
				ID:           uuid.New(),
				RepositoryID: wfTestRepoID,
				Status:       entity.WorkflowStatusCompleted,
				Conclusion:   "failure",
				Workflow:     "ci",
			},
		},
	}
	e := newWorkflowRunHandlerEcho(t, runRepo, nil)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/actions/runs?status=success", nil)
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
	runs, ok := resp["workflow_runs"].([]any)
	if !ok {
		t.Fatalf("workflow_runs missing: %v", resp)
	}
	if len(runs) != 1 {
		t.Fatalf("workflow_runs len = %d, want 1", len(runs))
	}
}

func TestListRuns_CrossOrgIsolation(t *testing.T) {
	runRepo := &wfHandlerFullRunRepo{
		runs: []*entity.WorkflowRun{},
	}
	e := newWorkflowRunHandlerEcho(t, runRepo, func(_ echo.Context, _, _ string) (*entity.Repository, error) {
		return nil, echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	})

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/actions/runs", nil)
	req.Host = "git.example.com"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestCancelRun_Success(t *testing.T) {
	runRepo := &wfHandlerFullRunRepo{
		run: &entity.WorkflowRun{
			ID:           wfTestRunID,
			RepositoryID: wfTestRepoID,
			Status:       "in_progress",
		},
	}
	e := newWorkflowRunHandlerEcho(t, runRepo, nil)

	req := httptest.NewRequest(http.MethodPost, "/repos/alice/demo/actions/runs/"+wfTestRunID.String()+"/cancel", nil)
	req.Host = "git.example.com"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	if !runRepo.cancelled {
		t.Fatal("expected run to be cancelled")
	}
}

func TestCancelRun_AlreadyComplete(t *testing.T) {
	runRepo := &wfHandlerFullRunRepo{
		run: &entity.WorkflowRun{
			ID:           wfTestRunID,
			RepositoryID: wfTestRepoID,
			Status:       entity.WorkflowStatusCompleted,
			Conclusion:   entity.WorkflowConclusionSuccess,
		},
	}
	e := newWorkflowRunHandlerEcho(t, runRepo, nil)

	req := httptest.NewRequest(http.MethodPost, "/repos/alice/demo/actions/runs/"+wfTestRunID.String()+"/cancel", nil)
	req.Host = "git.example.com"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
	if runRepo.cancelled {
		t.Fatal("expected cancel to be rejected before repository Cancel is called")
	}
}

func TestRerunRun_Returns202(t *testing.T) {
	runRepo := &wfHandlerFullRunRepo{
		run: &entity.WorkflowRun{
			ID:           wfTestRunID,
			RepositoryID: wfTestRepoID,
			Status:       entity.WorkflowStatusCompleted,
			Conclusion:   entity.WorkflowConclusionSuccess,
		},
	}
	e := newWorkflowRunHandlerEcho(t, runRepo, nil)

	req := httptest.NewRequest(http.MethodPost, "/repos/alice/demo/actions/runs/"+wfTestRunID.String()+"/rerun", nil)
	req.Host = "git.example.com"
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
}

func TestStreamJobLogs_ReturnsChunksFromOffset(t *testing.T) {
	runID := wfTestRunID
	jobRepo := &wfStreamWorkflowJobRepo{
		job: &entity.WorkflowJob{
			ID:             wfTestJobID,
			WorkflowRunID:  &runID,
			OrganizationID: wfTestOrgUUID,
			RepositoryID:   wfTestRepoID,
			Status:         entity.WorkflowJobStatusCompleted,
			Conclusion:     entity.WorkflowJobConclusionSuccess,
		},
	}
	logRepo := &wfStreamJobLogRepo{
		chunks: []*handler.JobLogChunk{
			{Offset: 0, Chunk: "chunk-at-offset-0"},
			{Offset: 1, Chunk: "chunk-at-offset-1"},
		},
	}
	h := newWorkflowRunStreamHandler(t, logRepo, jobRepo)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/actions/jobs/"+wfTestJobID.String()+"/logs/stream?offset=0", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/repos/:owner/:repo/actions/jobs/:job_id/logs/stream")
	c.SetParamNames("owner", "repo", "job_id")
	c.SetParamValues("alice", "demo", wfTestJobID.String())
	middleware.SetAuthContext(c, wfTestUserID, []string{"read", "repo"})

	if err := h.StreamJobLogs(c); err != nil {
		t.Fatalf("StreamJobLogs() error = %v", err)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content-type = %q, want text/event-stream", ct)
	}
	if buffering := rec.Header().Get("X-Accel-Buffering"); buffering != "no" {
		t.Fatalf("X-Accel-Buffering = %q, want no", buffering)
	}

	body := rec.Body.String()
	firstIdx := strings.Index(body, "data: chunk-at-offset-0")
	secondIdx := strings.Index(body, "data: chunk-at-offset-1")
	if firstIdx < 0 || secondIdx < 0 {
		t.Fatalf("expected both chunks in SSE body, body = %q", body)
	}
	if firstIdx > secondIdx {
		t.Fatalf("chunks out of offset order, body = %q", body)
	}
	if !strings.HasSuffix(body[:firstIdx+len("data: chunk-at-offset-0")], "data: chunk-at-offset-0") {
		t.Fatalf("first chunk missing data: prefix, body = %q", body)
	}
	if !strings.Contains(body, "data: chunk-at-offset-0\n\n") {
		t.Fatalf("first chunk missing SSE double newline terminator, body = %q", body)
	}
	if !strings.Contains(body, "data: chunk-at-offset-1\n\n") {
		t.Fatalf("second chunk missing SSE double newline terminator, body = %q", body)
	}
	if !strings.Contains(body, "data: {\"event\":\"done\"}\n\n") {
		t.Fatalf("expected done event, body = %q", body)
	}
}
