package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/handler"
	workflowusecase "github.com/open-git/backend/internal/usecase/workflow"
)

var (
	wfJobTestJobID = uuid.MustParse("00000000-0000-0000-0000-000000000301")
)

type wfHandlerJobLogRepo struct {
	chunks []*workflowusecase.JobLog
}

func (m *wfHandlerJobLogRepo) ListChunks(_ context.Context, _, _ uuid.UUID, _ int64) ([]*workflowusecase.JobLog, error) {
	return m.chunks, nil
}

type wfHandlerJobDetailRepo struct {
	job *workflowusecase.WorkflowJob
}

func (m *wfHandlerJobDetailRepo) GetByID(_ context.Context, _, _ uuid.UUID) (*workflowusecase.WorkflowJob, error) {
	if m.job == nil {
		return nil, apperror.ErrNotFound
	}
	return m.job, nil
}

func newWorkflowJobHandlerEcho(
	t *testing.T,
	jobRepo *wfHandlerJobDetailRepo,
	logRepo *wfHandlerJobLogRepo,
	resolveRepo func(echo.Context, string, string) (*entity.Repository, error),
) *echo.Echo {
	t.Helper()

	if jobRepo == nil {
		jobRepo = &wfHandlerJobDetailRepo{}
	}
	if logRepo == nil {
		logRepo = &wfHandlerJobLogRepo{}
	}
	if resolveRepo == nil {
		resolveRepo = func(_ echo.Context, _, _ string) (*entity.Repository, error) {
			return wfTestRepo(), nil
		}
	}

	getJobUC := workflowusecase.NewGetJobUsecase(jobRepo)
	listStepsUC := workflowusecase.NewListStepsUsecase(&wfHandlerStepRepo{})
	h := handler.NewWorkflowJobHandler(getJobUC, listStepsUC, logRepo, jobRepo, resolveRepo)

	e := echo.New()
	g := e.Group("")
	h.RegisterRoutes(g, wfTestAuth)
	return e
}

type wfHandlerStepRepo struct{}

func (wfHandlerStepRepo) ListByJobID(_ context.Context, _, _ uuid.UUID) ([]*workflowusecase.WorkflowStep, error) {
	return nil, nil
}

func TestGetJob_Success(t *testing.T) {
	started := time.Now().UTC()
	jobRepo := &wfHandlerJobDetailRepo{
		job: &workflowusecase.WorkflowJob{
			ID:        wfJobTestJobID,
			RunID:     wfTestRunID,
			Name:      "build",
			Status:    "completed",
			StartedAt: &started,
		},
	}
	e := newWorkflowJobHandlerEcho(t, jobRepo, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/actions/jobs/"+wfJobTestJobID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestGetLogs_ReturnsPlainText(t *testing.T) {
	jobRepo := &wfHandlerJobDetailRepo{
		job: &workflowusecase.WorkflowJob{
			ID:     wfJobTestJobID,
			RunID:  wfTestRunID,
			Name:   "build",
			Status: "completed",
		},
	}
	logRepo := &wfHandlerJobLogRepo{
		chunks: []*workflowusecase.JobLog{
			{JobID: wfJobTestJobID, Chunk: "line one\nline two\n"},
		},
	}
	e := newWorkflowJobHandlerEcho(t, jobRepo, logRepo, nil)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/actions/jobs/"+wfJobTestJobID.String()+"/logs", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Fatalf("content-type = %q, want text/plain", ct)
	}
	if body := rec.Body.String(); body != "line one\nline two\n" {
		t.Fatalf("body = %q, want log chunk concatenation", body)
	}
}

func TestStreamJobLogs_SetsEventStreamContentType(t *testing.T) {
	jobRepo := &wfHandlerJobDetailRepo{
		job: &workflowusecase.WorkflowJob{
			ID:     wfJobTestJobID,
			RunID:  wfTestRunID,
			Name:   "build",
			Status: "in_progress",
		},
	}
	logRepo := &wfHandlerJobLogRepo{
		chunks: []*workflowusecase.JobLog{
			{JobID: wfJobTestJobID, Chunk: "streaming line\n"},
		},
	}
	e := newWorkflowJobHandlerEcho(t, jobRepo, logRepo, nil)

	req := httptest.NewRequest(http.MethodGet, "/repos/alice/demo/actions/jobs/"+wfJobTestJobID.String()+"/logs/stream", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content-type = %q, want text/event-stream", ct)
	}
	if !strings.Contains(rec.Body.String(), "data: streaming line") {
		t.Fatalf("expected SSE data frame, body = %q", rec.Body.String())
	}
}
