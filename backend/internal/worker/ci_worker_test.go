package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

const ciTestSchema = `
CREATE TABLE organizations (
    id TEXT PRIMARY KEY,
    login TEXT NOT NULL,
    name TEXT NOT NULL,
    plan_tier TEXT NOT NULL DEFAULT 'free',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE repositories (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    name TEXT NOT NULL
);
CREATE TABLE action_secrets (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    repository_id TEXT,
    name TEXT NOT NULL,
    encrypted_value BYTEA NOT NULL,
    key_id TEXT NOT NULL DEFAULT '',
    visibility TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE workflow_runs (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    repository_id TEXT NOT NULL,
    workflow TEXT NOT NULL,
    status TEXT NOT NULL,
    conclusion TEXT,
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

func newCITestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec(ciTestSchema); err != nil {
		_ = db.Close()
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestSecretsAreMasked(t *testing.T) {
	db := newCITestDB(t)
	ctx := context.Background()

	orgID := "org-1"
	repoID := "repo-1"
	runID := "run-1"
	const secretValue = "supersecret-token-xyz"

	mustExec(t, db, `INSERT INTO organizations (id, login, name, plan_tier) VALUES (?, ?, ?, ?)`,
		orgID, "acme", "Acme", planTierPro)
	mustExec(t, db, `INSERT INTO repositories (id, organization_id, name) VALUES (?, ?, ?)`,
		repoID, orgID, "widgets")
	mustExec(t, db, `INSERT INTO action_secrets (id, organization_id, repository_id, name, encrypted_value) VALUES (?, ?, ?, ?, ?)`,
		"sec-1", orgID, repoID, "API_TOKEN", secretValue)
	mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`,
		runID, orgID, repoID, "ci.yml", ciStatusQueued)

	yamlSrc := []byte(`name: CI
on: push
jobs:
  build:
    steps:
      - name: leak
        run: echo $API_TOKEN
`)

	worker := NewCIWorker(db).WithCommandRunner(func(_ context.Context, _ string, env []string, _ string) ([]byte, error) {
		for _, kv := range env {
			if strings.HasPrefix(kv, "API_TOKEN=") {
				return []byte(strings.TrimPrefix(kv, "API_TOKEN=") + "\n"), nil
			}
		}
		return []byte("no token\n"), nil
	})

	payload, err := json.Marshal(CIRunPayload{
		WorkflowRunID:  runID,
		RepositoryID:   repoID,
		OrganizationID: orgID,
		WorkflowYAML:   yamlSrc,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	task := asynq.NewTask(TypeCIRun, payload)
	if err := worker.HandleCIRun(ctx, task); err != nil {
		t.Fatalf("HandleCIRun returned error: %v", err)
	}

	var status, conclusion sql.NullString
	err = db.QueryRowContext(ctx, `SELECT status, conclusion FROM workflow_runs WHERE id = ?`, runID).
		Scan(&status, &conclusion)
	if err != nil {
		t.Fatalf("query workflow_run: %v", err)
	}
	if status.String != ciStatusCompleted {
		t.Errorf("status: got %q, want %q", status.String, ciStatusCompleted)
	}
	if strings.Contains(conclusion.String, secretValue) {
		t.Errorf("conclusion contains plaintext secret %q: %q", secretValue, conclusion.String)
	}
	if !strings.Contains(conclusion.String, logMask) {
		t.Errorf("conclusion missing mask token %q: %q", logMask, conclusion.String)
	}
}

func TestFreeTierConcurrentLimit(t *testing.T) {
	db := newCITestDB(t)
	ctx := context.Background()

	orgID := "org-free"
	repoID := "repo-free"
	firstRun := "run-first"
	secondRun := "run-second"

	mustExec(t, db, `INSERT INTO organizations (id, login, name, plan_tier) VALUES (?, ?, ?, ?)`,
		orgID, "acme", "Acme", planTierFree)
	mustExec(t, db, `INSERT INTO repositories (id, organization_id, name) VALUES (?, ?, ?)`,
		repoID, orgID, "widgets")
	mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`,
		firstRun, orgID, repoID, "ci.yml", ciStatusInProgress)
	mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`,
		secondRun, orgID, repoID, "ci.yml", ciStatusQueued)

	yamlSrc := []byte(`name: CI
on: push
jobs:
  build:
    steps:
      - name: build
        run: echo hello
`)

	worker := NewCIWorker(db).WithCommandRunner(func(_ context.Context, _ string, _ []string, _ string) ([]byte, error) {
		return []byte("hello\n"), nil
	})

	payload, err := json.Marshal(CIRunPayload{
		WorkflowRunID:  secondRun,
		RepositoryID:   repoID,
		OrganizationID: orgID,
		WorkflowYAML:   yamlSrc,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	task := asynq.NewTask(TypeCIRun, payload)
	err = worker.HandleCIRun(ctx, task)
	if err == nil {
		t.Fatal("expected error for free-tier 2nd concurrent run, got nil")
	}
	if !errors.Is(err, ErrConcurrentLimitExceeded) {
		t.Errorf("expected ErrConcurrentLimitExceeded, got: %v", err)
	}

	var status, conclusion sql.NullString
	err = db.QueryRowContext(ctx, `SELECT status, conclusion FROM workflow_runs WHERE id = ?`, secondRun).
		Scan(&status, &conclusion)
	if err != nil {
		t.Fatalf("query second workflow_run: %v", err)
	}
	if status.String != ciStatusFailed {
		t.Errorf("second run status: got %q, want %q", status.String, ciStatusFailed)
	}
	if !strings.Contains(conclusion.String, ciConclusionRateLimited) {
		t.Errorf("expected conclusion to include %q, got %q", ciConclusionRateLimited, conclusion.String)
	}
}

func TestProTierAllowsManyConcurrent(t *testing.T) {
	db := newCITestDB(t)
	ctx := context.Background()

	orgID := "org-pro"
	repoID := "repo-pro"
	mustExec(t, db, `INSERT INTO organizations (id, login, name, plan_tier) VALUES (?, ?, ?, ?)`,
		orgID, "acme", "Acme", planTierPro)
	mustExec(t, db, `INSERT INTO repositories (id, organization_id, name) VALUES (?, ?, ?)`,
		repoID, orgID, "widgets")

	for i := 0; i < 5; i++ {
		mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`,
			"running-"+strconv.Itoa(i), orgID, repoID, "ci.yml", ciStatusInProgress)
	}
	runID := "candidate"
	mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`,
		runID, orgID, repoID, "ci.yml", ciStatusQueued)

	yamlSrc := []byte(`name: CI
on: push
jobs:
  build:
    steps:
      - name: build
        run: echo ok
`)
	worker := NewCIWorker(db).WithCommandRunner(func(_ context.Context, _ string, _ []string, _ string) ([]byte, error) {
		return []byte("ok\n"), nil
	})

	payload, _ := json.Marshal(CIRunPayload{
		WorkflowRunID:  runID,
		RepositoryID:   repoID,
		OrganizationID: orgID,
		WorkflowYAML:   yamlSrc,
	})
	task := asynq.NewTask(TypeCIRun, payload)
	if err := worker.HandleCIRun(ctx, task); err != nil {
		t.Fatalf("HandleCIRun unexpected error: %v", err)
	}
}

// runWorkflowRecording runs a workflow with a command runner that records the
// scripts it executes (in order) and fails any script containing "FAIL". It
// returns the executed scripts and the run's final conclusion.
func runWorkflowRecording(t *testing.T, yamlSrc string) ([]string, string) {
	t.Helper()
	db := newCITestDB(t)
	ctx := context.Background()
	orgID, repoID, runID := "org-x", "repo-x", "run-x"
	mustExec(t, db, `INSERT INTO organizations (id, login, name, plan_tier) VALUES (?, ?, ?, ?)`, orgID, "acme", "Acme", planTierPro)
	mustExec(t, db, `INSERT INTO repositories (id, organization_id, name) VALUES (?, ?, ?)`, repoID, orgID, "widgets")
	mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`, runID, orgID, repoID, "ci.yml", ciStatusQueued)

	var mu sync.Mutex
	var scripts []string
	worker := NewCIWorker(db).WithCommandRunner(func(_ context.Context, _ string, _ []string, script string) ([]byte, error) {
		mu.Lock()
		scripts = append(scripts, script)
		mu.Unlock()
		if strings.Contains(script, "FAIL") {
			return []byte("boom\n"), errors.New("exit status 1")
		}
		return []byte("ok\n"), nil
	})

	payload, _ := json.Marshal(CIRunPayload{WorkflowRunID: runID, RepositoryID: repoID, OrganizationID: orgID, WorkflowYAML: []byte(yamlSrc)})
	if err := worker.HandleCIRun(ctx, asynq.NewTask(TypeCIRun, payload)); err != nil {
		t.Fatalf("HandleCIRun: %v", err)
	}
	var conclusion sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT conclusion FROM workflow_runs WHERE id = ?`, runID).Scan(&conclusion); err != nil {
		t.Fatalf("query conclusion: %v", err)
	}
	// With no log repository wired, markRun stashes step logs after the enum
	// value in the conclusion column; the enum is the first line.
	return scripts, strings.SplitN(conclusion.String, "\n", 2)[0]
}

func TestCIEnvPrecedence(t *testing.T) {
	db := newCITestDB(t)
	ctx := context.Background()
	orgID, repoID, runID := "org-e", "repo-e", "run-e"
	mustExec(t, db, `INSERT INTO organizations (id, login, name, plan_tier) VALUES (?, ?, ?, ?)`, orgID, "acme", "Acme", planTierPro)
	mustExec(t, db, `INSERT INTO repositories (id, organization_id, name) VALUES (?, ?, ?)`, repoID, orgID, "widgets")
	mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`, runID, orgID, repoID, "ci.yml", ciStatusQueued)

	yamlSrc := `name: CI
on: push
env:
  L: workflow
  W_ONLY: w
jobs:
  build:
    env:
      L: job
      J_ONLY: j
    steps:
      - name: check
        run: echo hi
        env:
          L: step
          S_ONLY: s
`
	var gotEnv []string
	worker := NewCIWorker(db).WithCommandRunner(func(_ context.Context, _ string, env []string, _ string) ([]byte, error) {
		gotEnv = env
		return []byte("ok\n"), nil
	})
	payload, _ := json.Marshal(CIRunPayload{WorkflowRunID: runID, RepositoryID: repoID, OrganizationID: orgID, WorkflowYAML: []byte(yamlSrc)})
	if err := worker.HandleCIRun(ctx, asynq.NewTask(TypeCIRun, payload)); err != nil {
		t.Fatalf("HandleCIRun: %v", err)
	}

	want := map[string]string{"L": "step", "W_ONLY": "w", "J_ONLY": "j", "S_ONLY": "s"}
	got := map[string]string{}
	for _, kv := range gotEnv {
		if eq := strings.IndexByte(kv, '='); eq >= 0 {
			got[kv[:eq]] = kv[eq+1:]
		}
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("env %s = %q, want %q (full env: %v)", k, got[k], v, gotEnv)
		}
	}
}

func TestCINeedsSkipsDependentOnFailure(t *testing.T) {
	scripts, conclusion := runWorkflowRecording(t, `name: CI
on: push
jobs:
  a:
    steps:
      - run: echo FAIL
  b:
    needs: [a]
    steps:
      - run: echo B_RAN
`)
	joined := strings.Join(scripts, "\n")
	if strings.Contains(joined, "B_RAN") {
		t.Errorf("dependent job b ran despite needs=[a] failing; scripts=%v", scripts)
	}
	if conclusion != ciConclusionFailure {
		t.Errorf("run conclusion = %q, want %q", conclusion, ciConclusionFailure)
	}
}

func TestCIIndependentJobRunsDespiteOtherFailure(t *testing.T) {
	scripts, conclusion := runWorkflowRecording(t, `name: CI
on: push
jobs:
  a:
    steps:
      - run: echo FAIL
  b:
    steps:
      - run: echo B_RAN
`)
	if !strings.Contains(strings.Join(scripts, "\n"), "B_RAN") {
		t.Errorf("independent job b was not run after job a failed; scripts=%v", scripts)
	}
	if conclusion != ciConclusionFailure {
		t.Errorf("run conclusion = %q, want %q", conclusion, ciConclusionFailure)
	}
}

func TestCINeedsRunsInDependencyOrder(t *testing.T) {
	scripts, conclusion := runWorkflowRecording(t, `name: CI
on: push
jobs:
  zeta:
    needs: [alpha]
    steps:
      - run: echo ZETA
  alpha:
    steps:
      - run: echo ALPHA
`)
	if conclusion != ciConclusionSuccess {
		t.Fatalf("conclusion = %q, want success; scripts=%v", conclusion, scripts)
	}
	var alphaIdx, zetaIdx = -1, -1
	for i, s := range scripts {
		if strings.Contains(s, "ALPHA") {
			alphaIdx = i
		}
		if strings.Contains(s, "ZETA") {
			zetaIdx = i
		}
	}
	if alphaIdx < 0 || zetaIdx < 0 || alphaIdx > zetaIdx {
		t.Errorf("expected alpha (%d) to run before zeta (%d); scripts=%v", alphaIdx, zetaIdx, scripts)
	}
}

// TestCIIndependentJobsRunConcurrently proves independent jobs execute in
// parallel: each job's step blocks until it observes that the other job has
// also started. If execution were sequential the first job would wait forever
// (until its 2s barrier times out) and fail the run.
func TestCIIndependentJobsRunConcurrently(t *testing.T) {
	db := newCITestDB(t)
	ctx := context.Background()
	orgID, repoID, runID := "org-par", "repo-par", "run-par"
	mustExec(t, db, `INSERT INTO organizations (id, login, name, plan_tier) VALUES (?, ?, ?, ?)`, orgID, "acme", "Acme", planTierPro)
	mustExec(t, db, `INSERT INTO repositories (id, organization_id, name) VALUES (?, ?, ?)`, repoID, orgID, "widgets")
	mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`, runID, orgID, repoID, "ci.yml", ciStatusQueued)

	var started int32
	release := make(chan struct{})
	var releaseOnce sync.Once
	worker := NewCIWorker(db).WithCommandRunner(func(_ context.Context, _ string, _ []string, _ string) ([]byte, error) {
		if atomic.AddInt32(&started, 1) == 2 {
			releaseOnce.Do(func() { close(release) })
		}
		select {
		case <-release:
			return []byte("ok\n"), nil
		case <-time.After(2 * time.Second):
			return nil, errors.New("timeout: job did not run concurrently with its sibling")
		}
	})

	payload, _ := json.Marshal(CIRunPayload{
		WorkflowRunID: runID, RepositoryID: repoID, OrganizationID: orgID,
		WorkflowYAML: []byte(`name: CI
on: push
jobs:
  a:
    steps:
      - run: echo A
  b:
    steps:
      - run: echo B
`),
	})
	if err := worker.HandleCIRun(ctx, asynq.NewTask(TypeCIRun, payload)); err != nil {
		t.Fatalf("HandleCIRun: %v", err)
	}
	var conclusion sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT conclusion FROM workflow_runs WHERE id = ?`, runID).Scan(&conclusion); err != nil {
		t.Fatalf("query conclusion: %v", err)
	}
	if got := strings.SplitN(conclusion.String, "\n", 2)[0]; got != ciConclusionSuccess {
		t.Errorf("run conclusion = %q, want success — jobs did not run concurrently", got)
	}
}

// runWorkflowCapture runs a workflow and returns the scripts the runner
// actually received (post-interpolation), letting a caller adjust the payload.
func runWorkflowCapture(t *testing.T, yamlSrc string, mutate func(*CIRunPayload)) []string {
	t.Helper()
	db := newCITestDB(t)
	ctx := context.Background()
	orgID, repoID, runID := "org-c", "repo-c", "run-c"
	mustExec(t, db, `INSERT INTO organizations (id, login, name, plan_tier) VALUES (?, ?, ?, ?)`, orgID, "acme", "Acme", planTierPro)
	mustExec(t, db, `INSERT INTO repositories (id, organization_id, name) VALUES (?, ?, ?)`, repoID, orgID, "widgets")
	mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`, runID, orgID, repoID, "ci.yml", ciStatusQueued)

	var mu sync.Mutex
	var scripts []string
	worker := NewCIWorker(db).WithCommandRunner(func(_ context.Context, _ string, _ []string, script string) ([]byte, error) {
		mu.Lock()
		scripts = append(scripts, script)
		mu.Unlock()
		if strings.Contains(script, "FAIL") {
			return nil, errors.New("exit status 1")
		}
		return []byte("ok\n"), nil
	})

	p := CIRunPayload{WorkflowRunID: runID, RepositoryID: repoID, OrganizationID: orgID, WorkflowYAML: []byte(yamlSrc)}
	if mutate != nil {
		mutate(&p)
	}
	payload, _ := json.Marshal(p)
	if err := worker.HandleCIRun(ctx, asynq.NewTask(TypeCIRun, payload)); err != nil {
		t.Fatalf("HandleCIRun: %v", err)
	}
	return scripts
}

func TestCIInterpolatesGithubAndEnv(t *testing.T) {
	scripts := runWorkflowCapture(t, `name: CI
on: push
env:
  GREETING: hi
jobs:
  build:
    steps:
      - run: echo "ref=${{ github.ref_name }} sha=${{ github.sha }} n=${{ github.run_number }} g=${{ env.GREETING }}"
`, func(p *CIRunPayload) {
		p.HeadBranch = "main"
		p.HeadSHA = "deadbeef"
		p.RunNumber = 42
	})
	if len(scripts) != 1 {
		t.Fatalf("expected 1 script, got %d: %v", len(scripts), scripts)
	}
	want := `echo "ref=main sha=deadbeef n=42 g=hi"`
	if strings.TrimSpace(scripts[0]) != want {
		t.Errorf("interpolated script = %q, want %q", scripts[0], want)
	}
}

func TestCIStepIfSkipsWhenFalse(t *testing.T) {
	scripts := runWorkflowCapture(t, `name: CI
on: push
jobs:
  build:
    steps:
      - run: echo ALWAYS_ONE
      - if: github.ref_name == 'nonexistent'
        run: echo SHOULD_SKIP
      - run: echo ALWAYS_TWO
`, func(p *CIRunPayload) { p.HeadBranch = "main" })
	joined := strings.Join(scripts, "\n")
	if strings.Contains(joined, "SHOULD_SKIP") {
		t.Errorf("step with false if: ran; scripts=%v", scripts)
	}
	if !strings.Contains(joined, "ALWAYS_ONE") || !strings.Contains(joined, "ALWAYS_TWO") {
		t.Errorf("unconditional steps did not both run; scripts=%v", scripts)
	}
}

func TestCIStepIfAlwaysRunsAfterFailure(t *testing.T) {
	scripts := runWorkflowCapture(t, `name: CI
on: push
jobs:
  build:
    steps:
      - run: echo FAIL
      - run: echo SKIPPED_DEFAULT
      - if: always()
        run: echo CLEANUP
`, nil)
	joined := strings.Join(scripts, "\n")
	if strings.Contains(joined, "SKIPPED_DEFAULT") {
		t.Errorf("default step ran after a failure; scripts=%v", scripts)
	}
	if !strings.Contains(joined, "CLEANUP") {
		t.Errorf("if: always() step did not run after failure; scripts=%v", scripts)
	}
}

func TestCIJobLevelIfSkipsWholeJob(t *testing.T) {
	scripts := runWorkflowCapture(t, `name: CI
on: push
jobs:
  always:
    steps:
      - run: echo ALWAYS_JOB
  gated:
    if: github.ref_name == 'nope'
    steps:
      - run: echo GATED_SHOULD_NOT_RUN
`, func(p *CIRunPayload) { p.HeadBranch = "main" })
	joined := strings.Join(scripts, "\n")
	if strings.Contains(joined, "GATED_SHOULD_NOT_RUN") {
		t.Errorf("job with false job-level if: ran; scripts=%v", scripts)
	}
	if !strings.Contains(joined, "ALWAYS_JOB") {
		t.Errorf("ungated job did not run; scripts=%v", scripts)
	}
}

func TestCIMatrixExpandsAndInterpolates(t *testing.T) {
	scripts := runWorkflowCapture(t, `name: CI
on: push
jobs:
  build:
    strategy:
      matrix:
        os: [linux, windows]
        go: ['1.21', '1.22']
    steps:
      - run: echo "os=${{ matrix.os }} go=${{ matrix.go }}"
`, nil)
	if len(scripts) != 4 {
		t.Fatalf("expected 4 matrix instances, got %d: %v", len(scripts), scripts)
	}
	want := map[string]bool{
		`echo "os=linux go=1.21"`:   false,
		`echo "os=linux go=1.22"`:   false,
		`echo "os=windows go=1.21"`: false,
		`echo "os=windows go=1.22"`: false,
	}
	for _, s := range scripts {
		s = strings.TrimSpace(s)
		if _, ok := want[s]; !ok {
			t.Errorf("unexpected matrix script %q", s)
			continue
		}
		want[s] = true
	}
	for combo, seen := range want {
		if !seen {
			t.Errorf("matrix combination not executed: %q", combo)
		}
	}
}

func TestCIMatrixJobFailsIfAnyInstanceFails(t *testing.T) {
	scripts, conclusion := runWorkflowRecording(t, `name: CI
on: push
jobs:
  build:
    strategy:
      matrix:
        n: [ok, FAIL, ok2]
    steps:
      - run: echo ${{ matrix.n }}
  after:
    needs: [build]
    steps:
      - run: echo AFTER_RAN
`)
	if conclusion != ciConclusionFailure {
		t.Errorf("run conclusion = %q, want failure (one matrix instance failed)", conclusion)
	}
	if strings.Contains(strings.Join(scripts, "\n"), "AFTER_RAN") {
		t.Errorf("dependent job ran though a matrix instance failed; scripts=%v", scripts)
	}
}

func TestCICheckoutInvokedWithRepoAndRef(t *testing.T) {
	db := newCITestDB(t)
	ctx := context.Background()
	orgID, repoID, runID := "org-co", "repo-co", "run-co"
	mustExec(t, db, `INSERT INTO organizations (id, login, name, plan_tier) VALUES (?, ?, ?, ?)`, orgID, "acme", "Acme", planTierPro)
	mustExec(t, db, `INSERT INTO repositories (id, organization_id, name) VALUES (?, ?, ?)`, repoID, orgID, "widgets")
	mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`, runID, orgID, repoID, "ci.yml", ciStatusQueued)

	var gotPath, gotRef, gotDest string
	var checkoutCalls int
	worker := NewCIWorker(db).
		WithCheckout(func(_ context.Context, gitPath, ref, dest string) error {
			checkoutCalls++
			gotPath, gotRef, gotDest = gitPath, ref, dest
			return nil
		}).
		WithCommandRunner(func(_ context.Context, _ string, _ []string, _ string) ([]byte, error) {
			return []byte("ok\n"), nil
		})

	p := CIRunPayload{
		WorkflowRunID: runID, RepositoryID: repoID, OrganizationID: orgID,
		WorkflowYAML: []byte(`name: CI
on: push
jobs:
  build:
    steps:
      - uses: actions/checkout@v4
      - run: echo built
`),
		HeadSHA:     "cafe1234",
		RepoGitPath: "/data/git/alice/demo.git",
	}
	payload, _ := json.Marshal(p)
	if err := worker.HandleCIRun(ctx, asynq.NewTask(TypeCIRun, payload)); err != nil {
		t.Fatalf("HandleCIRun: %v", err)
	}
	if checkoutCalls != 1 {
		t.Fatalf("checkout called %d times, want 1", checkoutCalls)
	}
	if gotPath != "/data/git/alice/demo.git" {
		t.Errorf("checkout gitPath = %q, want repo path", gotPath)
	}
	if gotRef != "cafe1234" {
		t.Errorf("checkout ref = %q, want head sha", gotRef)
	}
	if gotDest == "" {
		t.Errorf("checkout dest was empty")
	}
}

func TestCIUnsupportedActionSkippedNotFailed(t *testing.T) {
	scripts, conclusion := runWorkflowRecording(t, `name: CI
on: push
jobs:
  build:
    steps:
      - uses: some/marketplace-action@v3
      - run: echo STILL_RAN
`)
	if conclusion != ciConclusionSuccess {
		t.Errorf("run conclusion = %q, want success (unsupported action must not fail)", conclusion)
	}
	if !strings.Contains(strings.Join(scripts, "\n"), "STILL_RAN") {
		t.Errorf("run step after unsupported action did not run; scripts=%v", scripts)
	}
}

func TestCIDockerContainerActionRuns(t *testing.T) {
	db := newCITestDB(t)
	ctx := context.Background()
	orgID, repoID, runID := "org-dk", "repo-dk", "run-dk"
	mustExec(t, db, `INSERT INTO organizations (id, login, name, plan_tier) VALUES (?, ?, ?, ?)`, orgID, "acme", "Acme", planTierPro)
	mustExec(t, db, `INSERT INTO repositories (id, organization_id, name) VALUES (?, ?, ?)`, repoID, orgID, "widgets")
	mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`, runID, orgID, repoID, "ci.yml", ciStatusQueued)

	var gotImage string
	var gotEnv []string
	worker := NewCIWorker(db).
		WithContainerAction(func(_ context.Context, image, _ string, env []string) ([]byte, error) {
			gotImage = image
			gotEnv = env
			return []byte("container ran\n"), nil
		})

	payload, _ := json.Marshal(CIRunPayload{
		WorkflowRunID: runID, RepositoryID: repoID, OrganizationID: orgID,
		WorkflowYAML: []byte(`name: CI
on: push
jobs:
  build:
    steps:
      - uses: docker://alpine:3
        with:
          greeting: hello
          my-name: world
`),
	})
	if err := worker.HandleCIRun(ctx, asynq.NewTask(TypeCIRun, payload)); err != nil {
		t.Fatalf("HandleCIRun: %v", err)
	}
	if gotImage != "alpine:3" {
		t.Errorf("container image = %q, want alpine:3", gotImage)
	}
	joined := strings.Join(gotEnv, "\n")
	if !strings.Contains(joined, "INPUT_GREETING=hello") {
		t.Errorf("expected INPUT_GREETING=hello in env; got %v", gotEnv)
	}
	if !strings.Contains(joined, "INPUT_MY_NAME=world") {
		t.Errorf("expected INPUT_MY_NAME=world (dash normalized) in env; got %v", gotEnv)
	}
}

func TestCIDockerContainerActionFailureFailsJob(t *testing.T) {
	db := newCITestDB(t)
	ctx := context.Background()
	orgID, repoID, runID := "org-dkf", "repo-dkf", "run-dkf"
	mustExec(t, db, `INSERT INTO organizations (id, login, name, plan_tier) VALUES (?, ?, ?, ?)`, orgID, "acme", "Acme", planTierPro)
	mustExec(t, db, `INSERT INTO repositories (id, organization_id, name) VALUES (?, ?, ?)`, repoID, orgID, "widgets")
	mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`, runID, orgID, repoID, "ci.yml", ciStatusQueued)

	worker := NewCIWorker(db).
		WithContainerAction(func(_ context.Context, _, _ string, _ []string) ([]byte, error) {
			return []byte("boom\n"), errors.New("exit status 1")
		})
	payload, _ := json.Marshal(CIRunPayload{
		WorkflowRunID: runID, RepositoryID: repoID, OrganizationID: orgID,
		WorkflowYAML: []byte(`name: CI
on: push
jobs:
  build:
    steps:
      - uses: docker://alpine:3
`),
	})
	if err := worker.HandleCIRun(ctx, asynq.NewTask(TypeCIRun, payload)); err != nil {
		t.Fatalf("HandleCIRun: %v", err)
	}
	var conclusion sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT conclusion FROM workflow_runs WHERE id = ?`, runID).Scan(&conclusion); err != nil {
		t.Fatalf("query: %v", err)
	}
	if got := strings.SplitN(conclusion.String, "\n", 2)[0]; got != ciConclusionFailure {
		t.Errorf("conclusion = %q, want failure (container action errored)", got)
	}
}

func mustExec(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatalf("exec %q: %v", q, err)
	}
}

type ciFakeJobLogRepo struct {
	lines []*entity.JobLogLine
}

func (f *ciFakeJobLogRepo) AppendLines(_ context.Context, lines []*entity.JobLogLine) error {
	f.lines = append(f.lines, lines...)
	return nil
}

func (f *ciFakeJobLogRepo) ListLines(_ context.Context, _, _ string, _ int64, _ int) ([]*entity.JobLogLine, error) {
	return f.lines, nil
}

func (f *ciFakeJobLogRepo) CountLines(_ context.Context, _, _ string) (int64, error) {
	return int64(len(f.lines)), nil
}

func (f *ciFakeJobLogRepo) SetMeta(_ context.Context, _ *domainrepo.JobLogMeta) error {
	return nil
}

func (f *ciFakeJobLogRepo) GetMeta(_ context.Context, _, _ string) (*domainrepo.JobLogMeta, error) {
	return nil, nil
}

func TestHandleCIRun_AppendsJobLogLinesWhenLogRepoInjected(t *testing.T) {
	db := newCITestDB(t)
	ctx := context.Background()

	orgID := "org-log"
	repoID := "repo-log"
	runID := "run-log"

	mustExec(t, db, `INSERT INTO organizations (id, login, name, plan_tier) VALUES (?, ?, ?, ?)`,
		orgID, "acme", "Acme", planTierPro)
	mustExec(t, db, `INSERT INTO repositories (id, organization_id, name) VALUES (?, ?, ?)`,
		repoID, orgID, "widgets")
	mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`,
		runID, orgID, repoID, "ci.yml", ciStatusQueued)

	yamlSrc := []byte(`name: CI
on: push
jobs:
  build:
    steps:
      - name: greet
        run: echo hello
      - name: world
        run: echo world
`)

	logRepo := &ciFakeJobLogRepo{}
	worker := NewCIWorker(db).
		WithLogRepository(logRepo).
		WithStreamingCommandRunner(func(_ context.Context, _ string, _ []string, _ string, _ int, sink func(stream, line string)) error {
			sink(entity.LogStreamStdout, "hello")
			return nil
		})

	payload, err := json.Marshal(CIRunPayload{
		WorkflowRunID:  runID,
		RepositoryID:   repoID,
		OrganizationID: orgID,
		WorkflowYAML:   yamlSrc,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	task := asynq.NewTask(TypeCIRun, payload)
	if err := worker.HandleCIRun(ctx, task); err != nil {
		t.Fatalf("HandleCIRun returned error: %v", err)
	}

	if len(logRepo.lines) < 2 {
		t.Fatalf("expected at least one log line per step, got %d", len(logRepo.lines))
	}
	for _, line := range logRepo.lines {
		if line.RunID != runID || line.RepositoryID != repoID || line.OrganizationID != orgID {
			t.Fatalf("unexpected log line scope: %+v", line)
		}
	}
}
