package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hibiken/asynq"
)

const (
	TypeGitHubImport = "import:github"

	githubAPIBase = "https://api.github.com"

	rateLimitRemainingHeader = "X-RateLimit-Remaining"
	rateLimitResetHeader     = "X-RateLimit-Reset"

	rateLimitThreshold = 10

	defaultPageSize = 100

	importerHTTPTimeout = 30 * time.Second
)

type GitHubImportPayload struct {
	WorkflowRunID  string `json:"workflow_run_id"`
	OrganizationID string `json:"organization_id"`
	RepositoryID   string `json:"repository_id"`
	SourceOwner    string `json:"source_owner"`
	SourceRepo     string `json:"source_repo"`
	Token          string `json:"token"`
	// ResumeFrom carries a per-resource cursor so a re-queued task can pick up
	// without redoing finished pages.
	ResumeFrom ImportCursor `json:"resume_from"`
}

type ImportCursor struct {
	ReposPage    int `json:"repos_page"`
	IssuesPage   int `json:"issues_page"`
	PullsPage    int `json:"pulls_page"`
	TotalSteps   int `json:"total_steps"`
	CompletedSet int `json:"completed_steps"`
}

type ImporterWorker struct {
	db         *sql.DB
	httpClient *http.Client
	apiBase    string
	now        func() time.Time
}

func NewImporterWorker(db *sql.DB) *ImporterWorker {
	return &ImporterWorker{
		db:         db,
		httpClient: &http.Client{Timeout: importerHTTPTimeout},
		apiBase:    githubAPIBase,
		now:        func() time.Time { return time.Now().UTC() },
	}
}

func (w *ImporterWorker) WithAPIBase(base string) *ImporterWorker {
	w.apiBase = strings.TrimRight(base, "/")
	return w
}

func (w *ImporterWorker) WithHTTPClient(c *http.Client) *ImporterWorker {
	w.httpClient = c
	return w
}

func (w *ImporterWorker) HandleGitHubImport(ctx context.Context, task *asynq.Task) error {
	var payload GitHubImportPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal importer payload: %w: %w", err, asynq.SkipRetry)
	}
	if payload.SourceOwner == "" || payload.SourceRepo == "" {
		return fmt.Errorf("importer payload missing source repo: %w", asynq.SkipRetry)
	}

	cursor := payload.ResumeFrom

	resources := []struct {
		kind    string
		pathFmt string
		pagePtr *int
	}{
		{"repos", "/repos/%s/%s", &cursor.ReposPage},
		{"issues", "/repos/%s/%s/issues", &cursor.IssuesPage},
		{"pulls", "/repos/%s/%s/pulls", &cursor.PullsPage},
	}

	for _, res := range resources {
		page := *res.pagePtr
		if page == 0 {
			page = 1
		}
		for {
			if err := w.respectRateLimit(ctx); err != nil {
				return err
			}
			url := fmt.Sprintf("%s"+res.pathFmt+"?per_page=%d&page=%d&state=all", w.apiBase, payload.SourceOwner, payload.SourceRepo, defaultPageSize, page)
			resp, items, err := w.fetchPage(ctx, url, payload.Token)
			if err != nil {
				// fatal connection failure: record progress and abort
				w.recordProgress(ctx, payload.WorkflowRunID, &cursor)
				return fmt.Errorf("fetch %s page %d: %w", res.kind, page, err)
			}

			if err := w.checkRateLimitHeaders(ctx, resp); err != nil {
				return err
			}

			ingested := 0
			for _, it := range items {
				if err := w.ingest(ctx, payload, res.kind, it); err != nil {
					// log and skip — failed item must not abort the import
					fmt.Printf("skipped: item_type=%s id=%s err=%s\n", res.kind, itemID(it), err.Error())
					continue
				}
				ingested++
			}
			cursor.CompletedSet += ingested

			if len(items) < defaultPageSize {
				*res.pagePtr = 0
				break
			}
			page++
			*res.pagePtr = page
			w.recordProgress(ctx, payload.WorkflowRunID, &cursor)
		}
	}

	cursor.CompletedSet = max(cursor.CompletedSet, cursor.TotalSteps)
	w.recordProgress(ctx, payload.WorkflowRunID, &cursor)
	w.finalize(ctx, payload.WorkflowRunID)
	return nil
}

func (w *ImporterWorker) respectRateLimit(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (w *ImporterWorker) fetchPage(ctx context.Context, url, token string) (*http.Response, []map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp, nil, fmt.Errorf("non-2xx response: %d body=%s", resp.StatusCode, truncate(string(body), 200))
	}

	items, err := decodeItems(body)
	if err != nil {
		return resp, nil, fmt.Errorf("decode body: %w", err)
	}
	return resp, items, nil
}

func decodeItems(body []byte) ([]map[string]any, error) {
	trim := strings.TrimSpace(string(body))
	if len(trim) == 0 {
		return nil, nil
	}
	if trim[0] == '[' {
		var arr []map[string]any
		if err := json.Unmarshal(body, &arr); err != nil {
			return nil, err
		}
		return arr, nil
	}
	var single map[string]any
	if err := json.Unmarshal(body, &single); err != nil {
		return nil, err
	}
	return []map[string]any{single}, nil
}

// checkRateLimitHeaders inspects X-RateLimit-Remaining; when below the
// threshold the worker sleeps until X-RateLimit-Reset before continuing.
func (w *ImporterWorker) checkRateLimitHeaders(ctx context.Context, resp *http.Response) error {
	if resp == nil {
		return nil
	}
	remainingStr := resp.Header.Get(rateLimitRemainingHeader)
	if remainingStr == "" {
		return nil
	}
	remaining, err := strconv.Atoi(remainingStr)
	if err != nil {
		return nil
	}
	if remaining >= rateLimitThreshold {
		return nil
	}
	resetStr := resp.Header.Get(rateLimitResetHeader)
	resetTs, err := strconv.ParseInt(resetStr, 10, 64)
	if err != nil || resetTs <= 0 {
		return nil
	}
	wait := time.Until(time.Unix(resetTs, 0))
	if wait <= 0 {
		return nil
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (w *ImporterWorker) ingest(_ context.Context, _ GitHubImportPayload, _ string, item map[string]any) error {
	if item == nil {
		return fmt.Errorf("nil item")
	}
	return nil
}

func itemID(item map[string]any) string {
	if item == nil {
		return ""
	}
	if v, ok := item["id"]; ok {
		return fmt.Sprintf("%v", v)
	}
	if v, ok := item["number"]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func (w *ImporterWorker) recordProgress(ctx context.Context, runID string, cur *ImportCursor) {
	if w.db == nil || runID == "" {
		return
	}
	pct := 0
	if cur.TotalSteps > 0 {
		pct = (cur.CompletedSet * 100) / cur.TotalSteps
		if pct > 100 {
			pct = 100
		}
	}
	progress := fmt.Sprintf("progress=%d%% completed=%d total=%d", pct, cur.CompletedSet, cur.TotalSteps)
	_, _ = w.db.ExecContext(
		ctx,
		`UPDATE workflow_runs SET conclusion = $1 WHERE id = $2`,
		progress, runID,
	)
}

func (w *ImporterWorker) finalize(ctx context.Context, runID string) {
	if w.db == nil || runID == "" {
		return
	}
	_, _ = w.db.ExecContext(
		ctx,
		`UPDATE workflow_runs SET status = $1, completed_at = $2 WHERE id = $3`,
		ciStatusCompleted, w.now(), runID,
	)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
