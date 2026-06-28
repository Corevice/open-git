package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/queue"
	"github.com/open-git/backend/internal/middleware"
	repointerface "github.com/open-git/backend/internal/repository"
)

const (
	defaultLogLimit = 1000
	maxLogLimit     = 5000
	streamKeepAlive = 15 * time.Second
)

type ActionsLogHandler struct {
	logRepo domainrepo.IJobLogRepository
	jobRepo domainrepo.IWorkflowJobRepository
	logSub  *queue.JobLogSubscriber
	repos   repointerface.IRepositoryRepository
}

func NewActionsLogHandler(
	logRepo domainrepo.IJobLogRepository,
	jobRepo domainrepo.IWorkflowJobRepository,
	logSub *queue.JobLogSubscriber,
	repos repointerface.IRepositoryRepository,
) *ActionsLogHandler {
	return &ActionsLogHandler{
		logRepo: logRepo,
		jobRepo: jobRepo,
		logSub:  logSub,
		repos:   repos,
	}
}

func (h *ActionsLogHandler) RegisterRoutes(g *echo.Group, authMiddleware echo.MiddlewareFunc) {
	readScope := middleware.RequireScope("read")

	g.GET("/repos/:owner/:repo/actions/runs/:runId/jobs/:jobId/logs/stream", h.StreamLogs, authMiddleware, readScope)
	g.GET("/repos/:owner/:repo/actions/runs/:runId/jobs/:jobId/logs", h.GetLogs, authMiddleware, readScope)
}

type logLineResponse struct {
	Step   int    `json:"step"`
	Line   int64  `json:"line"`
	TS     string `json:"ts"`
	Stream string `json:"stream"`
	Text   string `json:"text"`
}

type getLogsResponse struct {
	JobID        string            `json:"job_id"`
	Status       string            `json:"status"`
	TotalLines   int64             `json:"total_lines"`
	Lines        []logLineResponse `json:"lines"`
	NextFromLine *int64            `json:"next_from_line,omitempty"`
}

func (h *ActionsLogHandler) GetLogs(c echo.Context) error {
	repo, job, err := h.resolveJob(c)
	if err != nil {
		return err
	}
	_ = repo

	fromLine, err := parseInt64Query(c.QueryParam("from_line"), 0)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid from_line")
	}

	limit, err := parseInt64Query(c.QueryParam("limit"), defaultLogLimit)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid limit")
	}
	if limit <= 0 {
		limit = defaultLogLimit
	}
	if limit > maxLogLimit {
		limit = maxLogLimit
	}

	orgID := job.OrganizationID.String()
	jobID := job.ID.String()

	meta, err := h.logRepo.GetMeta(c.Request().Context(), orgID, jobID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) && !errors.Is(err, apperror.ErrNotFound) {
		return err
	}

	totalLines := int64(0)
	status := entity.WorkflowJobStatusRunning
	if meta != nil {
		totalLines = meta.TotalLines
		if meta.Status != "" {
			status = meta.Status
		}
	} else if job.Status != "" {
		status = job.Status
	}

	lines, err := h.logRepo.ListLines(c.Request().Context(), orgID, jobID, fromLine, int(limit))
	if err != nil {
		return err
	}

	respLines := make([]logLineResponse, 0, len(lines))
	for _, line := range lines {
		respLines = append(respLines, toLogLineResponse(line))
	}

	resp := getLogsResponse{
		JobID:      jobID,
		Status:     status,
		TotalLines: totalLines,
		Lines:      respLines,
	}

	nextFrom := fromLine + int64(len(lines))
	if nextFrom < totalLines {
		resp.NextFromLine = &nextFrom
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *ActionsLogHandler) StreamLogs(c echo.Context) error {
	repo, job, err := h.resolveJob(c)
	if err != nil {
		return err
	}
	_ = repo

	fromLine, err := parseLastEventID(c.Request().Header.Get("Last-Event-ID"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid Last-Event-ID")
	}

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return echo.NewHTTPError(http.StatusInternalServerError, "streaming not supported")
	}

	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("X-Accel-Buffering", "no")
	c.Response().WriteHeader(http.StatusOK)

	orgID := job.OrganizationID.String()
	jobID := job.ID.String()
	ctx := c.Request().Context()

	lines, err := h.logRepo.ListLines(ctx, orgID, jobID, fromLine, maxLogLimit)
	if err != nil {
		return err
	}

	for _, line := range lines {
		if line.LineNumber <= fromLine {
			continue
		}
		if err := writeLogSSE(c.Response(), flusher, line); err != nil {
			return err
		}
	}

	if isTerminalJobStatus(job.Status) {
		status := job.Status
		if job.Conclusion != "" {
			status = job.Conclusion
		}
		if err := writeDoneSSE(c.Response(), flusher, status); err != nil {
			return err
		}
		return nil
	}

	if h.logSub == nil {
		return nil
	}

	lineCh, doneCh, err := h.logSub.Subscribe(ctx, jobID, fromLine)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(streamKeepAlive)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case line, ok := <-lineCh:
			if !ok {
				return nil
			}
			if line == nil {
				continue
			}
			if err := writeLogSSE(c.Response(), flusher, line); err != nil {
				return err
			}
		case <-doneCh:
			status := job.Status
			if job.Conclusion != "" {
				status = job.Conclusion
			}
			_ = writeDoneSSE(c.Response(), flusher, status)
			return nil
		case <-ticker.C:
			if _, err := fmt.Fprint(c.Response(), ": ping\n\n"); err != nil {
				return err
			}
			flusher.Flush()
		}
	}
}

func (h *ActionsLogHandler) resolveJob(c echo.Context) (*entity.Repository, *entity.WorkflowJob, error) {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	runIDParam := c.Param("runId")
	jobIDParam := c.Param("jobId")

	repository, err := h.repos.GetByOwnerLoginAndName(c.Request().Context(), owner, repoName)
	if err != nil {
		return nil, nil, err
	}
	if repository == nil {
		return nil, nil, echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}

	runID, err := uuid.Parse(runIDParam)
	if err != nil {
		return nil, nil, echo.NewHTTPError(http.StatusBadRequest, "invalid runId")
	}
	jobID, err := uuid.Parse(jobIDParam)
	if err != nil {
		return nil, nil, echo.NewHTTPError(http.StatusBadRequest, "invalid jobId")
	}

	job, err := h.jobRepo.GetByID(c.Request().Context(), jobID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, apperror.ErrNotFound) {
			return nil, nil, echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		return nil, nil, err
	}
	if job == nil {
		return nil, nil, echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}

	if job.OrganizationID != repository.OrganizationID {
		return nil, nil, echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}
	if job.WorkflowRunID == nil || *job.WorkflowRunID != runID {
		return nil, nil, echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}
	if job.RepositoryID != repository.ID {
		return nil, nil, echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}

	return repository, job, nil
}

func parseLastEventID(raw string) (int64, error) {
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	if v < 0 {
		return 0, fmt.Errorf("negative line number")
	}
	return v, nil
}

func parseInt64Query(raw string, defaultVal int64) (int64, error) {
	if raw == "" {
		return defaultVal, nil
	}
	return strconv.ParseInt(raw, 10, 64)
}

func isTerminalJobStatus(status string) bool {
	switch status {
	case entity.WorkflowJobStatusCompleted,
		entity.WorkflowJobStatusFailed,
		entity.WorkflowJobStatusCancelled:
		return true
	default:
		return false
	}
}

func toLogLineResponse(line *entity.JobLogLine) logLineResponse {
	return logLineResponse{
		Step:   line.StepIndex,
		Line:   line.LineNumber,
		TS:     line.CreatedAt.UTC().Format(time.RFC3339),
		Stream: line.Stream,
		Text:   line.Text,
	}
}

func writeLogSSE(w http.ResponseWriter, flusher http.Flusher, line *entity.JobLogLine) error {
	payload, err := json.Marshal(toLogLineResponse(line))
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "id: %d\nevent: log\ndata: %s\n\n", line.LineNumber, payload); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

func writeDoneSSE(w http.ResponseWriter, flusher http.Flusher, status string) error {
	payload, err := json.Marshal(map[string]string{"status": status})
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: done\ndata: %s\n\n", payload); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
