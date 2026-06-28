package repository_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func newCompatTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db := openTestDB(t)

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS compat_test_run (
			id TEXT PRIMARY KEY,
			suite TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'queued',
			triggered_by TEXT REFERENCES users(id),
			organization_id TEXT NOT NULL,
			total_endpoints INTEGER NOT NULL DEFAULT 0,
			passing INTEGER NOT NULL DEFAULT 0,
			failing INTEGER NOT NULL DEFAULT 0,
			unimplemented INTEGER NOT NULL DEFAULT 0,
			coverage_rate REAL NOT NULL DEFAULT 0,
			started_at TIMESTAMP,
			finished_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("create compat_test_run table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS compat_endpoint_result (
			id TEXT PRIMARY KEY,
			run_id TEXT NOT NULL REFERENCES compat_test_run(id) ON DELETE CASCADE,
			method TEXT NOT NULL,
			path TEXT NOT NULL,
			status TEXT NOT NULL,
			checks TEXT,
			diff TEXT
		)
	`)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("create compat_endpoint_result table: %v", err)
	}

	return db
}

func TestCompatRepository(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T, repo domainrepo.ICompatRepository, db *sqlx.DB)
	}{
		{name: "CreateRunGetRunRoundTrip", fn: testCompatCreateRunGetRunRoundTrip},
		{name: "GetRunUnknownIDReturnsNil", fn: testCompatGetRunUnknownIDReturnsNil},
		{name: "UpdateRunChangesStatus", fn: testCompatUpdateRunChangesStatus},
		{name: "ListRunsOrganizationIsolation", fn: testCompatListRunsOrganizationIsolation},
		{name: "CreateEndpointResultListEndpointResultsJSONRoundTrip", fn: testCompatEndpointResultJSONRoundTrip},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newCompatTestDB(t)
			repo := repository.NewCompatRepository(db)
			tt.fn(t, repo, db)
		})
	}
}

func testCompatCreateRunGetRunRoundTrip(t *testing.T, repo domainrepo.ICompatRepository, db *sqlx.DB) {
	t.Helper()

	orgID := createTestOrganization(t, db, "compat-org-a")
	userID := createTestUser(t, db, "compat-user-a")
	startedAt := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(time.Hour)

	run := &entity.CompatTestRun{
		Suite:          "rest-v3",
		Status:         entity.CompatStatusCompleted,
		TriggeredBy:    &userID,
		OrganizationID: orgID,
		TotalEndpoints: 10,
		Passing:        8,
		Failing:        1,
		Unimplemented:  1,
		CoverageRate:   0.8,
		StartedAt:      &startedAt,
		FinishedAt:     &finishedAt,
	}

	if err := repo.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	got, err := repo.GetRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got == nil {
		t.Fatal("expected run, got nil")
	}
	if got.ID != run.ID || got.Suite != run.Suite || got.Status != run.Status {
		t.Fatalf("unexpected run identity: %+v", got)
	}
	if got.OrganizationID != orgID {
		t.Fatalf("organization_id: got %v, want %v", got.OrganizationID, orgID)
	}
	if got.TriggeredBy == nil || *got.TriggeredBy != userID {
		t.Fatalf("triggered_by: got %+v, want %v", got.TriggeredBy, userID)
	}
	if got.TotalEndpoints != 10 || got.Passing != 8 || got.Failing != 1 || got.Unimplemented != 1 {
		t.Fatalf("unexpected counters: %+v", got)
	}
	if got.CoverageRate != 0.8 {
		t.Fatalf("coverage_rate: got %v, want 0.8", got.CoverageRate)
	}
	if got.StartedAt == nil || !got.StartedAt.Equal(startedAt) {
		t.Fatalf("started_at: got %+v, want %v", got.StartedAt, startedAt)
	}
	if got.FinishedAt == nil || !got.FinishedAt.Equal(finishedAt) {
		t.Fatalf("finished_at: got %+v, want %v", got.FinishedAt, finishedAt)
	}
}

func testCompatGetRunUnknownIDReturnsNil(t *testing.T, repo domainrepo.ICompatRepository, _ *sqlx.DB) {
	t.Helper()

	got, err := repo.GetRun(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetRun: expected nil error, got %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil run, got %+v", got)
	}
}

func testCompatUpdateRunChangesStatus(t *testing.T, repo domainrepo.ICompatRepository, db *sqlx.DB) {
	t.Helper()

	orgID := createTestOrganization(t, db, "compat-org-update")
	run := &entity.CompatTestRun{
		Suite:          "rest-v3",
		Status:         entity.CompatStatusQueued,
		OrganizationID: orgID,
	}
	if err := repo.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	startedAt := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(5 * time.Minute)
	run.Status = entity.CompatStatusCompleted
	run.TotalEndpoints = 5
	run.Passing = 4
	run.Failing = 1
	run.Unimplemented = 0
	run.CoverageRate = 0.8
	run.StartedAt = &startedAt
	run.FinishedAt = &finishedAt

	if err := repo.UpdateRun(context.Background(), run); err != nil {
		t.Fatalf("UpdateRun: %v", err)
	}

	got, err := repo.GetRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.Status != entity.CompatStatusCompleted {
		t.Fatalf("status: got %q, want %q", got.Status, entity.CompatStatusCompleted)
	}
	if got.TotalEndpoints != 5 || got.Passing != 4 || got.Failing != 1 {
		t.Fatalf("unexpected updated counters: %+v", got)
	}
	if got.OrganizationID != orgID || got.Suite != "rest-v3" {
		t.Fatalf("immutable fields changed: %+v", got)
	}
}

func testCompatListRunsOrganizationIsolation(t *testing.T, repo domainrepo.ICompatRepository, db *sqlx.DB) {
	t.Helper()

	orgA := createTestOrganization(t, db, "compat-list-org-a")
	orgB := createTestOrganization(t, db, "compat-list-org-b")

	older := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		orgID     uuid.UUID
		startedAt time.Time
	}{
		{orgID: orgA, startedAt: older},
		{orgID: orgA, startedAt: newer},
		{orgID: orgB, startedAt: older},
	} {
		startedAt := tc.startedAt
		run := &entity.CompatTestRun{
			Suite:          "rest-v3",
			Status:         entity.CompatStatusCompleted,
			OrganizationID: tc.orgID,
			StartedAt:      &startedAt,
		}
		if err := repo.CreateRun(context.Background(), run); err != nil {
			t.Fatalf("CreateRun: %v", err)
		}
	}

	runsA, err := repo.ListRuns(context.Background(), orgA, 10)
	if err != nil {
		t.Fatalf("ListRuns orgA: %v", err)
	}
	if len(runsA) != 2 {
		t.Fatalf("orgA: expected 2 runs, got %d", len(runsA))
	}
	if runsA[0].StartedAt == nil || runsA[1].StartedAt == nil {
		t.Fatal("expected started_at on listed runs")
	}
	if runsA[0].StartedAt.Before(*runsA[1].StartedAt) {
		t.Fatalf("expected started_at DESC order, got %v then %v", runsA[0].StartedAt, runsA[1].StartedAt)
	}
	for _, run := range runsA {
		if run.OrganizationID != orgA {
			t.Fatalf("orgA leak: got organization_id %v", run.OrganizationID)
		}
	}

	runsB, err := repo.ListRuns(context.Background(), orgB, 10)
	if err != nil {
		t.Fatalf("ListRuns orgB: %v", err)
	}
	if len(runsB) != 1 {
		t.Fatalf("orgB: expected 1 run, got %d", len(runsB))
	}
	if runsB[0].OrganizationID != orgB {
		t.Fatalf("orgB: got organization_id %v", runsB[0].OrganizationID)
	}

	limited, err := repo.ListRuns(context.Background(), orgA, 1)
	if err != nil {
		t.Fatalf("ListRuns limit: %v", err)
	}
	if len(limited) != 1 {
		t.Fatalf("limit: expected 1 run, got %d", len(limited))
	}
}

func testCompatEndpointResultJSONRoundTrip(t *testing.T, repo domainrepo.ICompatRepository, db *sqlx.DB) {
	t.Helper()

	orgID := createTestOrganization(t, db, "compat-endpoint-org")
	run := &entity.CompatTestRun{
		Suite:          "rest-v3",
		Status:         entity.CompatStatusCompleted,
		OrganizationID: orgID,
	}
	if err := repo.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	diffPayload := []map[string]any{
		{"field": "merged_by", "expected": "object|null", "actual": "missing"},
	}
	diffJSON, err := json.Marshal(diffPayload)
	if err != nil {
		t.Fatalf("marshal diff: %v", err)
	}

	endpoint := &entity.CompatEndpointResult{
		RunID:  run.ID,
		Method: "GET",
		Path:   "/repos/{owner}/{repo}/issues",
		Status: entity.CompatResultFail,
		Checks: &entity.CompatEndpointChecks{
			Schema:     true,
			StatusCode: true,
			Headers:    false,
			Pagination: true,
		},
		Diff: diffJSON,
	}
	if err := repo.CreateEndpointResult(context.Background(), endpoint); err != nil {
		t.Fatalf("CreateEndpointResult: %v", err)
	}

	results, err := repo.ListEndpointResults(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("ListEndpointResults: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 endpoint result, got %d", len(results))
	}

	got := results[0]
	if got.Method != "GET" || got.Path != "/repos/{owner}/{repo}/issues" {
		t.Fatalf("unexpected endpoint identity: %+v", got)
	}
	if got.Status != entity.CompatResultFail {
		t.Fatalf("status: got %q, want %q", got.Status, entity.CompatResultFail)
	}
	if got.Checks == nil {
		t.Fatal("expected checks, got nil")
	}
	if !got.Checks.Schema || !got.Checks.StatusCode || got.Checks.Headers || !got.Checks.Pagination {
		t.Fatalf("unexpected checks: %+v", got.Checks)
	}

	var decodedDiff []map[string]any
	if err := json.Unmarshal(got.Diff, &decodedDiff); err != nil {
		t.Fatalf("unmarshal diff: %v", err)
	}
	if len(decodedDiff) != 1 || decodedDiff[0]["field"] != "merged_by" {
		t.Fatalf("unexpected diff payload: %+v", decodedDiff)
	}
}
