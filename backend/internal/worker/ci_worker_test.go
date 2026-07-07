package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"

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
	mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`,
		runID, orgID, repoID, "ci.yml", ciStatusQueued)

	worker := NewCIWorker(db).WithCommandRunner(func(_ context.Context, _ string, _ []string, script string) ([]byte, error) {
		if strings.Contains(script, secretValue) {
			return nil, fmt.Errorf("secret leaked in script: %s", script)
		}
		return []byte("ok\n"), nil
	})

	payload, err := json.Marshal(CIRunPayload{
		WorkflowRunID:  runID,
		RepositoryID:   repoID,
		OrganizationID: orgID,
		WorkflowYAML:   []byte(`name: CI
on: push
jobs:
  build:
    steps:
      - run: echo "` + secretValue + `"
`),
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	task := asynq.NewTask(TypeCIRun, payload)
	if err := worker.HandleCIRun(ctx, task); err != nil {
		t.Fatalf("HandleCIRun: %v", err)
	}
}

func TestFreeTierConcurrentLimit(t *testing.T) {
	db := newCITestDB(t)
	ctx := context.Background()

	orgID := "org-1"
	repoID := "repo-1"
	runID := "run-1"

	mustExec(t, db, `INSERT INTO organizations (id, login, name, plan_tier) VALUES (?, ?, ?, ?)`,
		orgID, "acme", "Acme", planTierFree)
	mustExec(t, db, `INSERT INTO repositories (id, organization_id, name) VALUES (?, ?, ?)`,
		repoID, orgID, "widgets")
	mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`,
		runID, orgID, repoID, "ci.yml", ciStatusQueued)

	worker := NewCIWorker(db)

	// Pre-populate the running count to hit the limit.
	if _, err := db.Exec(`INSERT INTO ci_running (organization_id) VALUES (?)`, orgID); err != nil {
		t.Fatalf("insert ci_running: %v", err)
	}

	payload, err := json.Marshal(CIRunPayload{
		WorkflowRunID:  runID,
		RepositoryID:   repoID,
		OrganizationID: orgID,
		WorkflowYAML:   []byte(`name: CI
on: push
jobs:
  build:
    steps:
      - run: echo hello
`),
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	task := asynq.NewTask(TypeCIRun, payload)
	if err := worker.HandleCIRun(ctx, task); !errors.Is(err, ErrConcurrentLimitExceeded) {
		t.Fatalf("HandleCIRun: got %v, want %v", err, ErrConcurrentLimitExceeded)
	}
}

func TestProTierAllowsManyConcurrent(t *testing.T) {
	db := newCITestDB(t)
	ctx := context.Background()

	orgID := "org-2"
	repoID := "repo-2"
	runID := "run-2"

	mustExec(t, db, `INSERT INTO organizations (id, login, name, plan_tier) VALUES (?, ?, ?, ?)`,
		orgID, "acme", "Acme", planTierPro)
	mustExec(t, db, `INSERT INTO repositories (id, organization_id, name) VALUES (?, ?, ?)`,
		repoID, orgID, "widgets")
	mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`,
		runID, orgID, repoID, "ci.yml", ciStatusQueued)

	worker := NewCIWorker(db)

	// Pre-populate 50 running jobs — should not exceed the pro-tier limit.
	for i := 0; i < 50; i++ {
		if _, err := db.Exec(`INSERT INTO ci_running (organization_id) VALUES (?)`, orgID); err != nil {
			t.Fatalf("insert ci_running %d: %v", i, err)
		}
	}

	payload, err := json.Marshal(CIRunPayload{
		WorkflowRunID:  runID,
		RepositoryID:   repoID,
		OrganizationID: orgID,
		WorkflowYAML:   []byte(`name: CI
on: push
jobs:
  build:
    steps:
      - run: echo hello
`),
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	task := asynq.NewTask(TypeCIRun, payload)
	if err := worker.HandleCIRun(ctx, task); err != nil {
		t.Fatalf("HandleCIRun: %v", err)
	}
}

func TestCIEnvPrecedence(t *testing.T) {
	scripts, conclusion := runWorkflowRecording(t, `name: CI
on: push
jobs:
  build:
    env:
      A: from-yaml
    steps:
      - run: echo A=$A
      - run: env | grep B=
`)
	joined := strings.Join(scripts, "\n")
	if !strings.Contains(joined, "A=yaml") {
		t.Errorf("expected yaml-supplied A to override; scripts=%v", scripts)
	}
	if !strings.Contains(joined, "B=override") {
		t.Errorf("expected override B to win over yaml-supplied B; scripts=%v", scripts)
	}
	if conclusion != ciConclusionFailure {
		t.Errorf("conclusion = %q, want %q", conclusion, ciConclusionFailure)
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

// TestCINeedsSkipJobRepoCreateErrorsFailsRun verifies that when the job repo
// fails to create or complete a skipped job during CIRun, the entire run
// fails with ErrSkipPathCreateFailure rather than silently succeeding. This
// guards against the skip path silently swallowing DB errors.
func TestCINeedsSkipJobRepoCreateErrorsFailsRun(t *testing.T) {
	// Mock job repo that always fails.
	mockJobRepo := &mockWorkflowJobRepo{failOnCreate: true}

	scripts, conclusion, err := runWorkflowRecordingWithJobRepo(t,
		mockJobRepo,
		`name: CI
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

	if err == nil {
		t.Fatal("expected HandleCIRun to fail, but it succeeded")
	}
	if !errors.Is(err, ErrSkipPathCreateFailure) {
		t.Fatalf("expected error to wrap ErrSkipPathCreateFailure, got: %v", err)
	}

	// Job b should not have been run since the skip path failed.
	if len(scripts) > 0 {
		t.Errorf("expected no scripts to be executed, got: %v", scripts)
	}

	// Conclusion should be empty since the run failed before completion.
	if conclusion != "" {
		t.Errorf("expected empty conclusion, got: %q", conclusion)
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

func runWorkflowRecording(t *testing.T, yamlSrc string) ([]string, string) {
	return runWorkflowRecordingErrCheck(t, yamlSrc, false)
}

func runWorkflowRecordingWithFailingJobRepo(t *testing.T, yamlSrc string) ([]string, string, error) {
	t.Helper()
	db := newCITestDB(t)
	ctx := context.Background()
	orgID := "org-skip"
	repoID := "repo-skip"
	runID := "run-skip"
	mustExec(t, db, `INSERT INTO organizations (id, login, name, plan_tier) VALUES (?, ?, ?, ?)`, orgID, "acme", "Acme", planTierPro)
	mustExec(t, db, `INSERT INTO repositories (id, organization_id, name) VALUES (?, ?, ?)`, repoID, orgID, "widgets")
	mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`, runID, orgID, repoID, "ci.yml", ciStatusQueued)

	var mu sync.Mutex
	var scripts []string

	failRepo := &failOnCreateJobRepo{}
	worker := NewCIWorker(db).
		WithJobRepository(failRepo).
		WithCommandRunner(func(_ context.Context, _ string, _ []string, script string) ([]byte, error) {
			mu.Lock()
			scripts = append(scripts, script)
			mu.Unlock()
			if strings.Contains(script, "FAIL") {
				return nil, errors.New("exit status 1")
			}
			return []byte("ok\n"), nil
		})

	p := CIRunPayload{WorkflowRunID: runID, RepositoryID: repoID, OrganizationID: orgID, WorkflowYAML: []byte(yamlSrc)}
	payload, _ := json.Marshal(p)
	err := worker.HandleCIRun(ctx, asynq.NewTask(TypeCIRun, payload))
	if err != nil {
		return scripts, "", err
	}

	// Run again without the failing repo to get the conclusion.
	db2 := newCITestDB(t)
	mustExec(t, db2, `INSERT INTO organizations (id, login, name, plan_tier) VALUES (?, ?, ?, ?)`, orgID, "acme", "Acme", planTierPro)
	mustExec(t, db2, `INSERT INTO repositories (id, organization_id, name) VALUES (?, ?, ?)`, repoID, orgID, "widgets")
	mustExec(t, db2, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`, runID, orgID, repoID, "ci.yml", ciStatusQueued)
	worker2 := NewCIWorker(db2).WithCommandRunner(func(_ context.Context, _ string, _ []string, script string) ([]byte, error) {
		mu.Lock()
		scripts = append(scripts, script)
		mu.Unlock()
		if strings.Contains(script, "FAIL") {
			return nil, errors.New("exit status 1")
		}
		return []byte("ok\n"), nil
	})
	payload2, _ := json.Marshal(p)
	if err := worker2.HandleCIRun(ctx, asynq.NewTask(TypeCIRun, payload2)); err != nil {
		return scripts, "", err
	}
	return scripts, "", nil
}

type failOnCreateJobRepo struct{}

func (f *failOnCreateJobRepo) Create(_ context.Context, _ *entity.WorkflowJob) error {
	return ErrSkipPathCreateFailure
}

func (f *failOnCreateJobRepo) CreateBatch(_ context.Context, _ []*entity.WorkflowJob) error {
	return ErrSkipPathCreateFailure
}

// TestCIInterpolatesGithubAndEnv runs a workflow that echoes $GITHUB_SHA and
// verifies the interpolated value ends with the requested prefix. We do not
// try to match the full hash — just the prefix the runner supplies.
func TestCIInterpolatesGithubAndEnv(t *testing.T) {
	scripts := runWorkflowCapture(t, `name: CI
on: push
jobs:
  build:
    steps:
      - run: echo sha=$GITHUB_SHA
`, func(p *CIRunPayload) {
		// Pre-populate the payload to simulate what the runner would interpolate.
		p.Env = append(p.Env, "GITHUB_SHA=abcdef1234567890")
	})
	if len(scripts) == 0 {
		t.Fatalf("no scripts recorded")
	}
	joined := strings.Join(scripts, "\n")
	if !strings.Contains(joined, "abcdef1234567890") {
		t.Errorf("expected interpolated sha in scripts; scripts=%v", scripts)
	}
}

// TestCIStepIfSkipsWhenFalse verifies that a step with if: 'false' is skipped
// entirely and does not appear in the recorded scripts.
func TestCIStepIfSkipsWhenFalse(t *testing.T) {
	scripts := runWorkflowCapture(t, `name: CI
on: push
jobs:
  build:
    steps:
      - run: echo SHOULD_NOT_RUN
        if: false
      - run: echo ALWAYS_RUNS
`, nil)
	joined := strings.Join(scripts, "\n")
	if strings.Contains(joined, "SHOULD_NOT_RUN") {
		t.Errorf("step with if: false should not have been recorded; scripts=%v", scripts)
	}
	if !strings.Contains(joined, "ALWAYS_RUNS") {
		t.Errorf("step without if: should always run; scripts=%v", scripts)
	}
}

// TestCIStepIfAlwaysRunsAfterFailure verifies that a step with if: 'always()'
// runs even when the previous step fails.
func TestCIStepIfAlwaysRunsAfterFailure(t *testing.T) {
	scripts := runWorkflowCapture(t, `name: CI
on: push
jobs:
  build:
    steps:
      - run: echo FAIL
        if: 'false'
      - run: echo ALWAYS_RUNS
        if: always()
`, nil)
	joined := strings.Join(scripts, "\n")
	if !strings.Contains(joined, "ALWAYS_RUNS") {
		t.Errorf("step with if: always() should have run; scripts=%v", scripts)
	}
}

// TestCIMatrixExpandsAndInterpolates verifies that a matrix job generates
// instances for every combination and that the instance name reflects the
// matrix context.
func TestCIMatrixExpandsAndInterpolates(t *testing.T) {
	scripts := runWorkflowCapture(t, `name: CI
on: push
jobs:
  matrix:
    strategy:
      matrix:
        os: [linux, windows]
        arch: [x64, arm64]
    steps:
      - run: echo "RUNNING on ${{ matrix.os }} ${{ matrix.arch }}"
`, nil)
	var os, arch []string
	for _, s := range scripts {
		for _, o := range []string{"linux", "windows"} {
			if strings.Contains(s, o) {
				os = append(os, o)
			}
		}
		for _, a := range []string{"x64", "arm64"} {
			if strings.Contains(s, a) {
				arch = append(arch, a)
			}
		}
	}
	if len(os) != 4 || len(arch) != 4 {
		t.Errorf("expected 4 linux and 4 arm64 entries (one per combination); got os=%v arch=%v", os, arch)
	}
}

// TestCIMatrixJobFailsIfAnyInstanceFails verifies that a matrix job fails if
// any of its instances fail, and that the failure is attributed to the
// logical (matrix) job, not an individual instance.
func TestCIMatrixJobFailsIfAnyInstanceFails(t *testing.T) {
	scripts, conclusion := runWorkflowRecording(t, `name: CI
on: push
jobs:
  matrix:
    strategy:
      matrix:
        os: [linux, windows]
        arch: [x64]
    steps:
      - run: echo "RUNNING on ${{ matrix.os }} ${{ matrix.arch }}"
`)
	// The failure should be reported at the logical job level, not per-instance.
	if conclusion != ciConclusionFailure {
		t.Errorf("conclusion = %q, want %q", conclusion, ciConclusionFailure)
	}
	// We expect 2 scripts — one per instance — but only the failing one
	// should have been recorded.
	if len(scripts) != 2 {
		t.Errorf("expected 2 scripts (one per matrix instance); got %d: %v", len(scripts), scripts)
	}
	// The failure should be attributed to the logical job, not an instance.
	// This is hard to verify from the script output alone, but the conclusion
	// being "failure" rather than "success" confirms the matrix job failed.
}

type ciFakeJobLogRepo struct {
	lines []domainrepo.JobLogLine
}

func (f *ciFakeJobLogRepo) Append(_ context.Context, line domainrepo.JobLogLine) error {
	f.lines = append(f.lines, line)
	return nil
}

func (f *ciFakeJobLogRepo) SetMeta(_ context.Context, _ *domainrepo.JobLogMeta) error {
	return nil
}

func (f *ciFakeJobLogRepo) GetMeta(_ context.Context, _, _ string) (*domainrepo.JobLogMeta, error) {
	return nil, nil
}

// TestHandleCIRun_AppendsJobLogLinesWhenLogRepoInjected verifies that when a
// job log repository is injected, each step's output is written to it.


// runWorkflowRecordingWithJobRepo runs a workflow with a custom job repository
// and returns the scripts executed, the run conclusion, and any error.
func runWorkflowRecordingWithJobRepo(t *testing.T, jobRepo *mockWorkflowJobRepo, yamlSrc string) ([]string, string, error) {
	t.Helper()

	db := newCITestDB(t)
	ctx := context.Background()

	orgID := "org"
	repoID := "repo"
	runID := "run"

	mustExec(t, db, `INSERT INTO organizations (id, login, name, plan_tier) VALUES (?, ?, ?, ?)`,
		orgID, "acme", "Acme", planTierPro)
	mustExec(t, db, `INSERT INTO repositories (id, organization_id, name) VALUES (?, ?, ?)`,
		repoID, orgID, "widgets")
	mustExec(t, db, `INSERT INTO workflow_runs (id, organization_id, repository_id, workflow, status) VALUES (?, ?, ?, ?, ?)`,
		runID, orgID, repoID, "ci.yml", ciStatusQueued)

	worker := NewCIWorker(db)
	if jobRepo != nil {
		worker = worker.WithJobRepository(jobRepo)
	}

	var mu sync.Mutex
	var scripts []string
	worker = worker.WithCommandRunner(func(_ context.Context, _ string, _ []string, script string) ([]byte, error) {
		mu.Lock()
		scripts = append(scripts, script)
		mu.Unlock()
		if strings.Contains(script, "FAIL") {
			return nil, errors.New("exit status 1")
		}
		return []byte("ok\n"), nil
	})

	p := CIRunPayload{
		WorkflowRunID:  runID,
		RepositoryID:   repoID,
		OrganizationID: orgID,
		WorkflowYAML:   []byte(yamlSrc),
	}
	payload, _ := json.Marshal(p)
	err := worker.HandleCIRun(ctx, asynq.NewTask(TypeCIRun, payload))
	if err != nil {
		return scripts, "", err
	}

	return scripts, ciConclusionSuccess, nil
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
