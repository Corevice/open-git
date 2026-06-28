package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/queue"
)

const mcpHTTPTimeout = 30 * time.Second

var mcpRetryDelays = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	4 * time.Second,
}

type MCPVerificationWorker struct {
	repo       domainrepo.IMCPVerificationRepository
	httpClient *http.Client
	baseURL    string
}

func NewMCPVerificationWorker(
	repo domainrepo.IMCPVerificationRepository,
	baseURL string,
) *MCPVerificationWorker {
	return &MCPVerificationWorker{
		repo: repo,
		httpClient: &http.Client{
			Timeout: mcpHTTPTimeout,
		},
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (w *MCPVerificationWorker) WithHTTPClient(client *http.Client) *MCPVerificationWorker {
	w.httpClient = client
	return w
}

func (w *MCPVerificationWorker) HandleMCPVerification(ctx context.Context, task *asynq.Task) (retErr error) {
	var payload queue.MCPVerificationPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal mcp verification payload: %w", err)
	}

	runID, err := uuid.Parse(payload.RunID)
	if err != nil {
		return fmt.Errorf("parse run_id: %w", err)
	}
	orgID, err := uuid.Parse(payload.OrganizationID)
	if err != nil {
		return fmt.Errorf("parse organization_id: %w", err)
	}

	run, err := w.repo.GetRunByID(ctx, runID, orgID)
	if err != nil {
		return fmt.Errorf("get run: %w", err)
	}
	if run == nil {
		return fmt.Errorf("run not found: %s", runID)
	}

	now := time.Now().UTC()
	run.Status = entity.RunStatusRunning
	run.StartedAt = &now
	if err := w.repo.UpdateRun(ctx, run); err != nil {
		return fmt.Errorf("update run to running: %w", err)
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			finishedAt := time.Now().UTC()
			run.Status = entity.RunStatusErrored
			run.FinishedAt = &finishedAt
			_ = w.repo.UpdateRun(ctx, run)
			retErr = fmt.Errorf("mcp verification panic: %v", recovered)
		}
	}()

	checks := make([]*entity.MCPVerificationCheck, 0)
	for _, target := range payload.Targets {
		switch target {
		case "graphql":
			checks = append(checks, w.graphqlSkipChecks(runID, orgID)...)
		case "rest":
			checks = append(checks, w.checkREST(ctx, runID, orgID, payload.RepositoryFullName, "")...)
		case "auth":
			checks = append(checks, w.checkAuth(ctx, runID, orgID, ""))
		}
	}

	if err := w.repo.BatchCreateChecks(ctx, checks); err != nil {
		return fmt.Errorf("batch create checks: %w", err)
	}

	overall := entity.ComputeOverallStatus(checks)
	run.OverallStatus = &overall
	finishedAt := time.Now().UTC()
	run.Status = entity.RunStatusCompleted
	run.FinishedAt = &finishedAt
	if err := w.repo.UpdateRun(ctx, run); err != nil {
		return fmt.Errorf("update run to completed: %w", err)
	}

	return nil
}

func (w *MCPVerificationWorker) graphqlSkipChecks(runID, orgID uuid.UUID) []*entity.MCPVerificationCheck {
	reason := "graphql endpoint not available"
	return []*entity.MCPVerificationCheck{
		newSkipCheck(runID, orgID, "graphql.viewer", entity.CheckCategoryGraphQL, reason),
		newSkipCheck(runID, orgID, "graphql.repository", entity.CheckCategoryGraphQL, reason),
	}
}

func (w *MCPVerificationWorker) checkREST(
	ctx context.Context,
	runID, orgID uuid.UUID,
	repoFullName, token string,
) []*entity.MCPVerificationCheck {
	owner, repo := splitRepositoryFullName(repoFullName)
	endpoints := []struct {
		checkID        string
		path           string
		expectedStatus int
	}{
		{"rest.user", "/api/v3/user", http.StatusOK},
		{"rest.repository", fmt.Sprintf("/api/v3/repos/%s/%s", owner, repo), http.StatusOK},
		{"rest.issues", fmt.Sprintf("/api/v3/repos/%s/%s/issues", owner, repo), http.StatusOK},
		{"rest.pulls", fmt.Sprintf("/api/v3/repos/%s/%s/pulls", owner, repo), http.StatusOK},
	}

	checks := make([]*entity.MCPVerificationCheck, 0, len(endpoints))
	for _, endpoint := range endpoints {
		checks = append(checks, w.performHTTPCheck(
			ctx,
			runID,
			orgID,
			endpoint.checkID,
			entity.CheckCategoryREST,
			endpoint.path,
			token,
			endpoint.expectedStatus,
		))
	}
	return checks
}

func (w *MCPVerificationWorker) checkAuth(ctx context.Context, runID, orgID uuid.UUID, token string) *entity.MCPVerificationCheck {
	return w.performHTTPCheck(
		ctx,
		runID,
		orgID,
		"auth.bearer",
		entity.CheckCategoryAuth,
		"/api/v3/user",
		token,
		http.StatusOK,
	)
}

func (w *MCPVerificationWorker) performHTTPCheck(
	ctx context.Context,
	runID, orgID uuid.UUID,
	checkID string,
	category entity.CheckCategory,
	path, token string,
	expectedStatus int,
) *entity.MCPVerificationCheck {
	check := &entity.MCPVerificationCheck{
		ID:             uuid.New(),
		RunID:          runID,
		OrganizationID: orgID,
		CheckID:        checkID,
		Category:       category,
		Expected:       mustJSON(map[string]any{"status_code": expectedStatus}),
		CreatedAt:      time.Now().UTC(),
	}

	start := time.Now()
	statusCode, body, err := w.doRequestWithRetry(ctx, path, token)
	check.DurationMS = int(time.Since(start).Milliseconds())

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			timeout := "timeout"
			check.Status = entity.CheckStatusFail
			check.Error = &timeout
			return check
		}
		errMsg := err.Error()
		check.Status = entity.CheckStatusFail
		check.Error = &errMsg
		return check
	}

	check.Actual = mustJSON(map[string]any{
		"status_code": statusCode,
		"body":        body,
	})
	if statusCode == expectedStatus {
		check.Status = entity.CheckStatusPass
	} else {
		check.Status = entity.CheckStatusFail
	}
	return check
}

func (w *MCPVerificationWorker) doRequestWithRetry(ctx context.Context, path, token string) (int, string, error) {
	url := w.baseURL + path
	var lastErr error

	for attempt := 0; attempt <= len(mcpRetryDelays); attempt++ {
		statusCode, body, err := w.doRequest(ctx, url, token)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return 0, "", context.DeadlineExceeded
			}
			lastErr = err
		} else if statusCode != http.StatusTooManyRequests {
			return statusCode, body, nil
		}

		if attempt == len(mcpRetryDelays) {
			if err == nil {
				return statusCode, body, nil
			}
			return 0, "", lastErr
		}

		delay := mcpRetryDelays[attempt]
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return 0, "", context.DeadlineExceeded
			}
			return 0, "", ctx.Err()
		case <-timer.C:
		}
	}

	return 0, "", lastErr
}

func (w *MCPVerificationWorker) doRequest(ctx context.Context, url, token string) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, "", err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return 0, "", context.DeadlineExceeded
		}
		return 0, "", err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return 0, "", err
	}

	return resp.StatusCode, string(bodyBytes), nil
}

func newSkipCheck(runID, orgID uuid.UUID, checkID string, category entity.CheckCategory, reason string) *entity.MCPVerificationCheck {
	return &entity.MCPVerificationCheck{
		ID:             uuid.New(),
		RunID:          runID,
		OrganizationID: orgID,
		CheckID:        checkID,
		Category:       category,
		Status:         entity.CheckStatusSkip,
		Error:          &reason,
		CreatedAt:      time.Now().UTC(),
	}
}

func splitRepositoryFullName(fullName string) (owner, repo string) {
	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) != 2 {
		return fullName, ""
	}
	return parts[0], parts[1]
}

func mustJSON(value any) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage("{}")
	}
	return data
}
