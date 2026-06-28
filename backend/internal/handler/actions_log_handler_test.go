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
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/handler"
	repointerface "github.com/open-git/backend/internal/repository"
)

var (
	actionsLogTestOrgID  = uuid.MustParse("00000000-0000-0000-0000-000000000042")
	actionsLogTestRepoID = uuid.MustParse("00000000-0000-0000-0000-000000000099")
	actionsLogTestRunID  = uuid.MustParse("00000000-0000-0000-0000-000000000201")
	actionsLogTestJobID  = uuid.MustParse("00000000-0000-0000-0000-000000000301")
)

type fakeJobLogRepo struct {
	lines []*entity.JobLogLine
	meta  *domainrepo.JobLogMeta
}

func (f *fakeJobLogRepo) AppendLines(_ context.Context, lines []*entity.JobLogLine) error {
	f.lines = append(f.lines, lines...)
	return nil
}

func (f *fakeJobLogRepo) ListLines(_ context.Context, _, jobID string, fromLine int64, limit int) ([]*entity.JobLogLine, error) {
	result := make([]*entity.JobLogLine, 0)
	for _, line := range f.lines {
		if line.JobID != jobID || line.LineNumber <= fromLine {
			continue
		}
		result = append(result, line)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (f *fakeJobLogRepo) CountLines(_ context.Context, _, jobID string) (int64, error) {
	var n int64
	for _, line := range f.lines {
		if line.JobID == jobID {
			n++
		}
	}
	return n, nil
}

func (f *fakeJobLogRepo) SetMeta(_ context.Context, meta *domainrepo.JobLogMeta) error {
	f.meta = meta
	return nil
}

func (f *fakeJobLogRepo) GetMeta(_ context.Context, _, jobID string) (*domainrepo.JobLogMeta, error) {
	if f.meta != nil && f.meta.JobID == jobID {
		return f.meta, nil
	}
	return &domainrepo.JobLogMeta{JobID: jobID, Status: entity.WorkflowJobStatusCompleted, TotalLines: int64(len(f.lines))}, nil
}

type fakeWorkflowJobRepo struct {
	jobs map[uuid.UUID]*entity.WorkflowJob
}

func (f *fakeWorkflowJobRepo) Create(_ context.Context, job *entity.WorkflowJob) error {
	if f.jobs == nil {
		f.jobs = make(map[uuid.UUID]*entity.WorkflowJob)
	}
	f.jobs[job.ID] = job
	return nil
}

func (f *fakeWorkflowJobRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.WorkflowJob, error) {
	if f.jobs == nil {
		return nil, domain.ErrNotFound
	}
	job, ok := f.jobs[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return job, nil
}

func (f *fakeWorkflowJobRepo) AcquireForRunner(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ int) (bool, error) {
	return false, nil
}

func (f *fakeWorkflowJobRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _, _ string) error {
	return nil
}

func (f *fakeWorkflowJobRepo) Complete(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}

func (f *fakeWorkflowJobRepo) Cancel(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (f *fakeWorkflowJobRepo) ListQueued(_ context.Context, _ uuid.UUID) ([]*entity.WorkflowJob, error) {
	return nil, nil
}

type fakeRepoLookup struct {
	repo *entity.Repository
}

func (f *fakeRepoLookup) Create(_ context.Context, _ *entity.Repository) error { return nil }

func (f *fakeRepoLookup) GetByOwnerAndName(_ context.Context, _ uuid.UUID, _ string) (*entity.Repository, error) {
	return nil, domain.ErrNotFound
}

func (f *fakeRepoLookup) GetByOwnerLoginAndName(_ context.Context, _, _ string) (*entity.Repository, error) {
	if f.repo == nil {
		return nil, domain.ErrNotFound
	}
	return f.repo, nil
}

func (f *fakeRepoLookup) ListByOrg(_ context.Context, _ uuid.UUID, _, _ int) ([]*entity.Repository, error) {
	return nil, nil
}

func (f *fakeRepoLookup) CountByOrg(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (f *fakeRepoLookup) ListByOwner(_ context.Context, _ uuid.UUID, _, _ int) ([]*entity.Repository, error) {
	return nil, nil
}

func (f *fakeRepoLookup) CountByOwner(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (f *fakeRepoLookup) UpdateVisibility(_ context.Context, _ uuid.UUID, _ string) error { return nil }

func (f *fakeRepoLookup) UpdateName(_ context.Context, _ uuid.UUID, _ string) error { return nil }

func (f *fakeRepoLookup) Delete(_ context.Context, _ uuid.UUID) error { return nil }

var _ repointerface.IRepositoryRepository = (*fakeRepoLookup)(nil)

func actionsLogTestRepo() *entity.Repository {
	return &entity.Repository{
		ID:             actionsLogTestRepoID,
		OrganizationID: actionsLogTestOrgID,
		OwnerLogin:     "alice",
		Name:           "demo",
	}
}

func actionsLogTestJob() *entity.WorkflowJob {
	runID := actionsLogTestRunID
	return &entity.WorkflowJob{
		ID:             actionsLogTestJobID,
		WorkflowRunID:  &runID,
		OrganizationID: actionsLogTestOrgID,
		RepositoryID:   actionsLogTestRepoID,
		Name:           "build",
		Status:         entity.WorkflowJobStatusCompleted,
		Conclusion:     entity.WorkflowJobConclusionSuccess,
	}
}

func newActionsLogHandlerEcho(
	t *testing.T,
	logRepo domainrepo.IJobLogRepository,
	jobRepo domainrepo.IWorkflowJobRepository,
	repos repointerface.IRepositoryRepository,
) *echo.Echo {
	t.Helper()

	h := handler.NewActionsLogHandler(logRepo, jobRepo, nil, repos)
	e := echo.New()
	g := e.Group("/api")
	h.RegisterRoutes(g, wfTestAuth)
	return e
}

func TestActionsLogGetLogs_NotFound(t *testing.T) {
	jobRepo := &fakeWorkflowJobRepo{jobs: map[uuid.UUID]*entity.WorkflowJob{}}
	e := newActionsLogHandlerEcho(t, &fakeJobLogRepo{}, jobRepo, &fakeRepoLookup{repo: actionsLogTestRepo()})

	req := httptest.NewRequest(http.MethodGet, "/api/repos/alice/demo/actions/runs/"+actionsLogTestRunID.String()+"/jobs/"+actionsLogTestJobID.String()+"/logs", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestActionsLogGetLogs_Pagination(t *testing.T) {
	job := actionsLogTestJob()
	jobRepo := &fakeWorkflowJobRepo{jobs: map[uuid.UUID]*entity.WorkflowJob{job.ID: job}}
	logRepo := &fakeJobLogRepo{
		meta: &domainrepo.JobLogMeta{
			JobID:      job.ID.String(),
			Status:     entity.WorkflowJobStatusCompleted,
			TotalLines: 5,
		},
		lines: []*entity.JobLogLine{
			{JobID: job.ID.String(), LineNumber: 1, StepIndex: 0, Stream: entity.LogStreamStdout, Text: "one", CreatedAt: time.Now().UTC()},
			{JobID: job.ID.String(), LineNumber: 2, StepIndex: 0, Stream: entity.LogStreamStdout, Text: "two", CreatedAt: time.Now().UTC()},
			{JobID: job.ID.String(), LineNumber: 3, StepIndex: 0, Stream: entity.LogStreamStdout, Text: "three", CreatedAt: time.Now().UTC()},
		},
	}
	e := newActionsLogHandlerEcho(t, logRepo, jobRepo, &fakeRepoLookup{repo: actionsLogTestRepo()})

	req := httptest.NewRequest(http.MethodGet, "/api/repos/alice/demo/actions/runs/"+actionsLogTestRunID.String()+"/jobs/"+actionsLogTestJobID.String()+"/logs?from_line=0&limit=2", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		NextFromLine *int64 `json:"next_from_line"`
		Lines        []any  `json:"lines"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp.Lines) != 2 {
		t.Fatalf("lines len = %d, want 2", len(resp.Lines))
	}
	if resp.NextFromLine == nil || *resp.NextFromLine != 2 {
		t.Fatalf("next_from_line = %v, want 2", resp.NextFromLine)
	}
}

func TestActionsLogStreamLogs_NotFound(t *testing.T) {
	jobRepo := &fakeWorkflowJobRepo{jobs: map[uuid.UUID]*entity.WorkflowJob{}}
	e := newActionsLogHandlerEcho(t, &fakeJobLogRepo{}, jobRepo, &fakeRepoLookup{repo: actionsLogTestRepo()})

	req := httptest.NewRequest(http.MethodGet, "/api/repos/alice/demo/actions/runs/"+actionsLogTestRunID.String()+"/jobs/"+actionsLogTestJobID.String()+"/logs/stream", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestActionsLogStreamLogs_LastEventID(t *testing.T) {
	job := actionsLogTestJob()
	job.Status = entity.WorkflowJobStatusInProgress
	jobRepo := &fakeWorkflowJobRepo{jobs: map[uuid.UUID]*entity.WorkflowJob{job.ID: job}}
	logRepo := &fakeJobLogRepo{
		lines: []*entity.JobLogLine{
			{JobID: job.ID.String(), LineNumber: 3, StepIndex: 0, Stream: entity.LogStreamStdout, Text: "line3", CreatedAt: time.Now().UTC()},
			{JobID: job.ID.String(), LineNumber: 4, StepIndex: 0, Stream: entity.LogStreamStdout, Text: "line4", CreatedAt: time.Now().UTC()},
			{JobID: job.ID.String(), LineNumber: 6, StepIndex: 0, Stream: entity.LogStreamStdout, Text: "line6", CreatedAt: time.Now().UTC()},
		},
	}
	e := newActionsLogHandlerEcho(t, logRepo, jobRepo, &fakeRepoLookup{repo: actionsLogTestRepo()})

	req := httptest.NewRequest(http.MethodGet, "/api/repos/alice/demo/actions/runs/"+actionsLogTestRunID.String()+"/jobs/"+actionsLogTestJobID.String()+"/logs/stream", nil)
	req.Header.Set("Last-Event-ID", "5")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "line3") || strings.Contains(body, "line4") {
		t.Fatalf("expected lines <= 5 to be skipped, body = %q", body)
	}
	if !strings.Contains(body, "line6") {
		t.Fatalf("expected line6 in body, body = %q", body)
	}
	if !strings.HasPrefix(body, "id: 6") && !strings.Contains(body, "\nid: 6\n") {
		t.Fatalf("expected SSE id frame for line 6, body = %q", body)
	}
}

func TestActionsLogTenantIsolation(t *testing.T) {
	otherOrg := uuid.MustParse("00000000-0000-0000-0000-000000000999")
	runID := actionsLogTestRunID
	job := &entity.WorkflowJob{
		ID:             actionsLogTestJobID,
		WorkflowRunID:  &runID,
		OrganizationID: otherOrg,
		RepositoryID:   actionsLogTestRepoID,
		Name:           "build",
		Status:         entity.WorkflowJobStatusCompleted,
	}
	jobRepo := &fakeWorkflowJobRepo{jobs: map[uuid.UUID]*entity.WorkflowJob{job.ID: job}}
	e := newActionsLogHandlerEcho(t, &fakeJobLogRepo{}, jobRepo, &fakeRepoLookup{repo: actionsLogTestRepo()})

	req := httptest.NewRequest(http.MethodGet, "/api/repos/alice/demo/actions/runs/"+actionsLogTestRunID.String()+"/jobs/"+actionsLogTestJobID.String()+"/logs", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d for tenant mismatch, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestActionsLogStreamLogs_TerminalJobSendsDone(t *testing.T) {
	job := actionsLogTestJob()
	jobRepo := &fakeWorkflowJobRepo{jobs: map[uuid.UUID]*entity.WorkflowJob{job.ID: job}}
	logRepo := &fakeJobLogRepo{
		lines: []*entity.JobLogLine{
			{JobID: job.ID.String(), LineNumber: 1, StepIndex: 0, Stream: entity.LogStreamStdout, Text: "done-line", CreatedAt: time.Now().UTC()},
		},
	}
	e := newActionsLogHandlerEcho(t, logRepo, jobRepo, &fakeRepoLookup{repo: actionsLogTestRepo()})

	req := httptest.NewRequest(http.MethodGet, "/api/repos/alice/demo/actions/runs/"+actionsLogTestRunID.String()+"/jobs/"+actionsLogTestJobID.String()+"/logs/stream", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content-type = %q, want text/event-stream", ct)
	}
	if !strings.Contains(rec.Body.String(), "event: done") {
		t.Fatalf("expected done event, body = %q", rec.Body.String())
	}
}
