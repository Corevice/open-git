package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

const workflowTestSchema = `
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
CREATE TABLE workflows (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL REFERENCES organizations(id),
    repository_id TEXT NOT NULL REFERENCES repositories(id),
    name TEXT NOT NULL,
    path TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'active' CHECK (state IN ('active', 'disabled')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(repository_id, path)
);
CREATE TABLE workflow_revisions (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    commit_sha TEXT NOT NULL,
    raw_content_hash TEXT NOT NULL,
    parse_status TEXT NOT NULL DEFAULT 'pending' CHECK (parse_status IN ('valid', 'invalid', 'pending')),
    ir TEXT NOT NULL DEFAULT '{}',
    parsed_at TIMESTAMP,
    UNIQUE(workflow_id, commit_sha)
);
CREATE TABLE workflow_diagnostics (
    id TEXT PRIMARY KEY,
    workflow_revision_id TEXT NOT NULL REFERENCES workflow_revisions(id) ON DELETE CASCADE,
    line INTEGER NOT NULL DEFAULT 0,
    col INTEGER NOT NULL DEFAULT 0,
    severity TEXT NOT NULL CHECK (severity IN ('error', 'warning', 'info')),
    message TEXT NOT NULL
);
`

type workflowTestSeed struct {
	OrgID  uuid.UUID
	RepoID uuid.UUID
}

func newWorkflowTestDB(t *testing.T) (*sqlx.DB, workflowTestSeed) {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec(workflowTestSchema); err != nil {
		_ = db.Close()
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	sqlxDB := sqlx.NewDb(db, "sqlite3")

	orgID := uuid.New()
	repoID := uuid.New()
	if _, err := sqlxDB.Exec(
		`INSERT INTO organizations (id, login, name) VALUES (?, ?, ?)`,
		orgID.String(), "acme", "Acme",
	); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	if _, err := sqlxDB.Exec(
		`INSERT INTO repositories (id, organization_id, name) VALUES (?, ?, ?)`,
		repoID.String(), orgID.String(), "widgets",
	); err != nil {
		t.Fatalf("insert repo: %v", err)
	}

	return sqlxDB, workflowTestSeed{OrgID: orgID, RepoID: repoID}
}

func TestWorkflowUpsert_CreateAndRetrieve(t *testing.T) {
	db, seed := newWorkflowTestDB(t)
	repo := repository.NewWorkflowRepository(db)
	ctx := context.Background()

	wf := &entity.Workflow{
		OrganizationID: seed.OrgID,
		RepositoryID:   seed.RepoID,
		Name:           "CI",
		Path:           ".github/workflows/ci.yml",
		State:          entity.WorkflowStateActive,
	}
	if err := repo.Upsert(ctx, wf); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := repo.GetByID(ctx, seed.OrgID, wf.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID returned nil")
	}
	if got.Name != wf.Name {
		t.Fatalf("Name = %q, want %q", got.Name, wf.Name)
	}
	if got.Path != wf.Path {
		t.Fatalf("Path = %q, want %q", got.Path, wf.Path)
	}
	if got.State != entity.WorkflowStateActive {
		t.Fatalf("State = %q, want %q", got.State, entity.WorkflowStateActive)
	}
	if got.OrganizationID != seed.OrgID {
		t.Fatalf("OrganizationID = %v, want %v", got.OrganizationID, seed.OrgID)
	}
	if got.RepositoryID != seed.RepoID {
		t.Fatalf("RepositoryID = %v, want %v", got.RepositoryID, seed.RepoID)
	}
}

func TestWorkflowUpsert_Idempotent(t *testing.T) {
	db, seed := newWorkflowTestDB(t)
	repo := repository.NewWorkflowRepository(db)
	ctx := context.Background()

	wf := &entity.Workflow{
		OrganizationID: seed.OrgID,
		RepositoryID:   seed.RepoID,
		Name:           "CI",
		Path:           ".github/workflows/ci.yml",
		State:          entity.WorkflowStateActive,
	}
	if err := repo.Upsert(ctx, wf); err != nil {
		t.Fatalf("first Upsert: %v", err)
	}

	first, err := repo.GetByID(ctx, seed.OrgID, wf.ID)
	if err != nil {
		t.Fatalf("GetByID after first upsert: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	wf.Name = "CI Updated"
	if err := repo.Upsert(ctx, wf); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}

	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM workflows WHERE repository_id = ? AND path = ?`, seed.RepoID, wf.Path); err != nil {
		t.Fatalf("count workflows: %v", err)
	}
	if count != 1 {
		t.Fatalf("workflow row count = %d, want 1", count)
	}

	second, err := repo.GetByPath(ctx, seed.OrgID, seed.RepoID, wf.Path)
	if err != nil {
		t.Fatalf("GetByPath: %v", err)
	}
	if second.Name != "CI Updated" {
		t.Fatalf("Name = %q, want %q", second.Name, "CI Updated")
	}
	if !second.UpdatedAt.After(first.UpdatedAt) {
		t.Fatalf("UpdatedAt did not advance: first=%v second=%v", first.UpdatedAt, second.UpdatedAt)
	}
}

func TestWorkflowGetByPath_TenantIsolation(t *testing.T) {
	db, seed := newWorkflowTestDB(t)
	repo := repository.NewWorkflowRepository(db)
	ctx := context.Background()

	wf := &entity.Workflow{
		OrganizationID: seed.OrgID,
		RepositoryID:   seed.RepoID,
		Name:           "CI",
		Path:           ".github/workflows/ci.yml",
		State:          entity.WorkflowStateActive,
	}
	if err := repo.Upsert(ctx, wf); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	wrongOrgID := uuid.New()
	got, err := repo.GetByPath(ctx, wrongOrgID, seed.RepoID, wf.Path)
	if err != nil {
		t.Fatalf("GetByPath: %v", err)
	}
	if got != nil {
		t.Fatalf("GetByPath with wrong orgID = %+v, want nil", got)
	}
}

func TestWorkflowListByRepo(t *testing.T) {
	db, seed := newWorkflowTestDB(t)
	repo := repository.NewWorkflowRepository(db)
	ctx := context.Background()

	paths := []string{".github/workflows/b.yml", ".github/workflows/a.yml"}
	for _, path := range paths {
		wf := &entity.Workflow{
			OrganizationID: seed.OrgID,
			RepositoryID:   seed.RepoID,
			Name:           path,
			Path:           path,
			State:          entity.WorkflowStateActive,
		}
		if err := repo.Upsert(ctx, wf); err != nil {
			t.Fatalf("Upsert %s: %v", path, err)
		}
	}

	list, err := repo.ListByRepo(ctx, seed.OrgID, seed.RepoID)
	if err != nil {
		t.Fatalf("ListByRepo: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want 2", len(list))
	}
	if list[0].Path != ".github/workflows/a.yml" || list[1].Path != ".github/workflows/b.yml" {
		t.Fatalf("unexpected order: %q, %q", list[0].Path, list[1].Path)
	}
}

func TestWorkflowSaveRevision_GetLatest(t *testing.T) {
	db, seed := newWorkflowTestDB(t)
	repo := repository.NewWorkflowRepository(db)
	ctx := context.Background()

	wf := &entity.Workflow{
		OrganizationID: seed.OrgID,
		RepositoryID:   seed.RepoID,
		Name:           "CI",
		Path:           ".github/workflows/ci.yml",
		State:          entity.WorkflowStateActive,
	}
	if err := repo.Upsert(ctx, wf); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	earlier := time.Now().UTC().Add(-time.Hour)
	later := time.Now().UTC()

	revs := []*entity.WorkflowRevision{
		{
			WorkflowID:     wf.ID,
			CommitSHA:      "abc111",
			RawContentHash: "hash1",
			ParseStatus:    entity.ParseStatusValid,
			IR:             `{"jobs":{}}`,
			ParsedAt:       &earlier,
		},
		{
			WorkflowID:     wf.ID,
			CommitSHA:      "def222",
			RawContentHash: "hash2",
			ParseStatus:    entity.ParseStatusValid,
			IR:             `{"jobs":{}}`,
			ParsedAt:       &later,
		},
	}
	for _, rev := range revs {
		if err := repo.SaveRevision(ctx, rev); err != nil {
			t.Fatalf("SaveRevision: %v", err)
		}
	}

	latest, err := repo.GetLatestRevision(ctx, wf.ID)
	if err != nil {
		t.Fatalf("GetLatestRevision: %v", err)
	}
	if latest == nil {
		t.Fatal("GetLatestRevision returned nil")
	}
	if latest.CommitSHA != "def222" {
		t.Fatalf("CommitSHA = %q, want def222", latest.CommitSHA)
	}
}

func TestWorkflowSaveRevision_Idempotent(t *testing.T) {
	db, seed := newWorkflowTestDB(t)
	repo := repository.NewWorkflowRepository(db)
	ctx := context.Background()

	wf := &entity.Workflow{
		OrganizationID: seed.OrgID,
		RepositoryID:   seed.RepoID,
		Name:           "CI",
		Path:           ".github/workflows/ci.yml",
		State:          entity.WorkflowStateActive,
	}
	if err := repo.Upsert(ctx, wf); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	parsedAt := time.Now().UTC()
	rev := &entity.WorkflowRevision{
		WorkflowID:     wf.ID,
		CommitSHA:      "abc111",
		RawContentHash: "hash1",
		ParseStatus:    entity.ParseStatusValid,
		IR:             `{"jobs":{}}`,
		ParsedAt:       &parsedAt,
	}
	if err := repo.SaveRevision(ctx, rev); err != nil {
		t.Fatalf("first SaveRevision: %v", err)
	}
	if err := repo.SaveRevision(ctx, rev); err != nil {
		t.Fatalf("second SaveRevision: %v", err)
	}

	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM workflow_revisions WHERE workflow_id = ? AND commit_sha = ?`, wf.ID, rev.CommitSHA); err != nil {
		t.Fatalf("count revisions: %v", err)
	}
	if count != 1 {
		t.Fatalf("revision row count = %d, want 1", count)
	}
}

func TestWorkflowSaveDiagnostics_ListByRevision(t *testing.T) {
	db, seed := newWorkflowTestDB(t)
	repo := repository.NewWorkflowRepository(db)
	ctx := context.Background()

	wf := &entity.Workflow{
		OrganizationID: seed.OrgID,
		RepositoryID:   seed.RepoID,
		Name:           "CI",
		Path:           ".github/workflows/ci.yml",
		State:          entity.WorkflowStateActive,
	}
	if err := repo.Upsert(ctx, wf); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	parsedAt := time.Now().UTC()
	rev := &entity.WorkflowRevision{
		WorkflowID:     wf.ID,
		CommitSHA:      "abc111",
		RawContentHash: "hash1",
		ParseStatus:    entity.ParseStatusInvalid,
		IR:             `{}`,
		ParsedAt:       &parsedAt,
	}
	if err := repo.SaveRevision(ctx, rev); err != nil {
		t.Fatalf("SaveRevision: %v", err)
	}

	diags := []*entity.WorkflowDiagnostic{
		{Line: 10, Col: 1, Severity: entity.SeverityError, Message: "missing jobs"},
		{Line: 3, Col: 5, Severity: entity.SeverityWarning, Message: "deprecated key"},
		{Line: 3, Col: 1, Severity: entity.SeverityInfo, Message: "hint"},
	}
	if err := repo.SaveDiagnostics(ctx, rev.ID, diags); err != nil {
		t.Fatalf("SaveDiagnostics: %v", err)
	}

	got, err := repo.ListDiagnosticsByRevision(ctx, rev.ID)
	if err != nil {
		t.Fatalf("ListDiagnosticsByRevision: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len(got) = %d, want 3", len(got))
	}
	if got[0].Line != 3 || got[0].Col != 1 {
		t.Fatalf("first diag = line %d col %d, want line 3 col 1", got[0].Line, got[0].Col)
	}
	if got[1].Line != 3 || got[1].Col != 5 {
		t.Fatalf("second diag = line %d col %d, want line 3 col 5", got[1].Line, got[1].Col)
	}
	if got[2].Line != 10 {
		t.Fatalf("third diag line = %d, want 10", got[2].Line)
	}
	for _, d := range got {
		if d.ID == uuid.Nil {
			t.Fatal("diagnostic ID was not generated")
		}
	}
}
