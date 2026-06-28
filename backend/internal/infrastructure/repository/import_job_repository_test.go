package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func newImportTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	schema := `
		CREATE TABLE organizations (
			id TEXT PRIMARY KEY,
			login TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			plan_tier TEXT NOT NULL DEFAULT 'free',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			login TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE import_jobs (
			id TEXT PRIMARY KEY,
			organization_id TEXT NOT NULL REFERENCES organizations(id),
			created_by TEXT NOT NULL REFERENCES users(id),
			source_url TEXT NOT NULL,
			target_repository_id TEXT,
			target_name TEXT NOT NULL,
			include TEXT NOT NULL DEFAULT '[]',
			status TEXT NOT NULL DEFAULT 'queued',
			phase TEXT NOT NULL DEFAULT 'clone',
			progress TEXT NOT NULL DEFAULT '{}',
			token_secret_ref TEXT,
			error TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX idx_import_jobs_org_status ON import_jobs(organization_id, status);
		CREATE INDEX idx_import_jobs_org_created ON import_jobs(organization_id, created_at);

		CREATE TABLE import_user_mappings (
			id TEXT PRIMARY KEY,
			import_job_id TEXT NOT NULL REFERENCES import_jobs(id),
			github_login TEXT NOT NULL,
			github_display_name TEXT NOT NULL DEFAULT '',
			local_user_id TEXT REFERENCES users(id),
			UNIQUE(import_job_id, github_login)
		);

		CREATE TABLE import_phase_checkpoints (
			import_job_id TEXT NOT NULL REFERENCES import_jobs(id),
			phase TEXT NOT NULL,
			last_cursor TEXT NOT NULL DEFAULT '',
			completed INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (import_job_id, phase)
		);
	`
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		t.Fatalf("create schema: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })
	return sqlx.NewDb(db, "sqlite3")
}

func insertImportTestOrg(t *testing.T, db *sqlx.DB, login string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(
		`INSERT INTO organizations (id, login, name) VALUES (?, ?, ?)`,
		id.String(), login, login,
	)
	if err != nil {
		t.Fatalf("insert org %s: %v", login, err)
	}
	return id
}

func insertImportTestUser(t *testing.T, db *sqlx.DB, login string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(
		`INSERT INTO users (id, login, email, password_hash) VALUES (?, ?, ?, ?)`,
		id.String(), login, login+"@example.com", "hashed",
	)
	if err != nil {
		t.Fatalf("insert user %s: %v", login, err)
	}
	return id
}

func TestImportJobRepository_Create(t *testing.T) {
	db := newImportTestDB(t)
	repo := repository.NewImportJobRepository(db)

	orgID := insertImportTestOrg(t, db, "import-org")
	userID := insertImportTestUser(t, db, "import-user")

	job := &entity.ImportJob{
		OrganizationID: orgID,
		CreatedBy:      userID,
		SourceURL:      "https://github.com/acme/demo",
		TargetName:     "demo",
		Include:        []string{"code", "issues"},
		Status:         entity.ImportJobStatusQueued,
		Phase:          entity.ImportJobPhaseClone,
	}
	if err := repo.Create(context.Background(), job); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if job.ID == uuid.Nil {
		t.Fatal("expected job ID to be assigned")
	}

	got, err := repo.GetByID(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected job, got nil")
	}
	if got.SourceURL != job.SourceURL || got.TargetName != job.TargetName {
		t.Fatalf("unexpected job: %+v", got)
	}
	if len(got.Include) != 2 {
		t.Fatalf("expected 2 include items, got %d", len(got.Include))
	}
}

func TestImportJobRepository_GetByIDAndOrgCrossOrg(t *testing.T) {
	db := newImportTestDB(t)
	repo := repository.NewImportJobRepository(db)

	orgA := insertImportTestOrg(t, db, "org-a")
	orgB := insertImportTestOrg(t, db, "org-b")
	userID := insertImportTestUser(t, db, "cross-org-user")

	job := &entity.ImportJob{
		OrganizationID: orgA,
		CreatedBy:      userID,
		SourceURL:      "https://github.com/acme/private",
		TargetName:     "private",
		Include:        []string{"code"},
	}
	if err := repo.Create(context.Background(), job); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByIDAndOrg(context.Background(), job.ID, orgA)
	if err != nil {
		t.Fatalf("GetByIDAndOrg same org: %v", err)
	}
	if got == nil || got.ID != job.ID {
		t.Fatalf("unexpected job: %+v", got)
	}

	_, err = repo.GetByIDAndOrg(context.Background(), job.ID, orgB)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows for cross-org read, got %v", err)
	}
}

func TestImportJobRepository_ListByOrg(t *testing.T) {
	db := newImportTestDB(t)
	repo := repository.NewImportJobRepository(db)

	orgID := insertImportTestOrg(t, db, "list-org")
	userID := insertImportTestUser(t, db, "list-user")

	for i := 0; i < 3; i++ {
		job := &entity.ImportJob{
			OrganizationID: orgID,
			CreatedBy:      userID,
			SourceURL:      "https://github.com/acme/repo",
			TargetName:     "repo",
			Include:        []string{"issues"},
		}
		if err := repo.Create(context.Background(), job); err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
	}

	jobs, total, err := repo.ListByOrg(context.Background(), orgID, 1, 2)
	if err != nil {
		t.Fatalf("ListByOrg: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected total 3, got %d", total)
	}
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs on page 1, got %d", len(jobs))
	}
}

func TestImportJobRepository_UpdateStatus(t *testing.T) {
	db := newImportTestDB(t)
	repo := repository.NewImportJobRepository(db)

	orgID := insertImportTestOrg(t, db, "status-org")
	userID := insertImportTestUser(t, db, "status-user")

	job := &entity.ImportJob{
		OrganizationID: orgID,
		CreatedBy:      userID,
		SourceURL:      "https://github.com/acme/status",
		TargetName:     "status",
		Include:        []string{"code"},
	}
	if err := repo.Create(context.Background(), job); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.UpdateStatus(context.Background(), job.ID, entity.ImportJobStatusRunning); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	got, err := repo.GetByID(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != entity.ImportJobStatusRunning {
		t.Fatalf("expected status running, got %s", got.Status)
	}
}

func TestImportJobRepository_UpdateProgress(t *testing.T) {
	db := newImportTestDB(t)
	repo := repository.NewImportJobRepository(db)

	orgID := insertImportTestOrg(t, db, "progress-org")
	userID := insertImportTestUser(t, db, "progress-user")

	job := &entity.ImportJob{
		OrganizationID: orgID,
		CreatedBy:      userID,
		SourceURL:      "https://github.com/acme/progress",
		TargetName:     "progress",
		Include:        []string{"issues"},
	}
	if err := repo.Create(context.Background(), job); err != nil {
		t.Fatalf("Create: %v", err)
	}

	progress := entity.ImportProgress{
		"issues": {Done: 5, Total: 10},
	}
	if err := repo.UpdateProgress(context.Background(), job.ID, progress); err != nil {
		t.Fatalf("UpdateProgress: %v", err)
	}

	got, err := repo.GetByID(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	issuesProgress, ok := got.Progress["issues"]
	if !ok {
		t.Fatal("expected issues progress key")
	}
	if issuesProgress.Done != 5 || issuesProgress.Total != 10 {
		t.Fatalf("unexpected progress: %+v", issuesProgress)
	}
}

func TestImportUserMappingRepository_UpsertMapping(t *testing.T) {
	db := newImportTestDB(t)
	repo := repository.NewImportJobRepository(db)

	orgID := insertImportTestOrg(t, db, "mapping-org")
	userID := insertImportTestUser(t, db, "mapping-user")

	job := &entity.ImportJob{
		OrganizationID: orgID,
		CreatedBy:      userID,
		SourceURL:      "https://github.com/acme/mapping",
		TargetName:     "mapping",
		Include:        []string{"code"},
	}
	if err := repo.Create(context.Background(), job); err != nil {
		t.Fatalf("Create job: %v", err)
	}

	mapping := &entity.ImportUserMapping{
		ImportJobID:       job.ID,
		GitHubLogin:       "octocat",
		GitHubDisplayName: "The Octocat",
		LocalUserID:       &userID,
	}
	if err := repo.UpsertMapping(context.Background(), mapping); err != nil {
		t.Fatalf("UpsertMapping insert: %v", err)
	}

	mapping.GitHubDisplayName = "Updated Octocat"
	mapping.LocalUserID = nil
	if err := repo.UpsertMapping(context.Background(), mapping); err != nil {
		t.Fatalf("UpsertMapping update: %v", err)
	}

	got, err := repo.GetMappingByLogin(context.Background(), job.ID, "octocat")
	if err != nil {
		t.Fatalf("GetMappingByLogin: %v", err)
	}
	if got == nil {
		t.Fatal("expected mapping, got nil")
	}
	if got.GitHubDisplayName != "Updated Octocat" {
		t.Fatalf("expected updated display name, got %s", got.GitHubDisplayName)
	}
	if got.LocalUserID != nil {
		t.Fatal("expected local user ID to be cleared on upsert")
	}
}

func TestImportUserMappingRepository_GetMappingByLogin(t *testing.T) {
	db := newImportTestDB(t)
	repo := repository.NewImportJobRepository(db)

	orgID := insertImportTestOrg(t, db, "lookup-org")
	userID := insertImportTestUser(t, db, "lookup-user")

	job := &entity.ImportJob{
		OrganizationID: orgID,
		CreatedBy:      userID,
		SourceURL:      "https://github.com/acme/lookup",
		TargetName:     "lookup",
		Include:        []string{"code"},
	}
	if err := repo.Create(context.Background(), job); err != nil {
		t.Fatalf("Create job: %v", err)
	}

	if err := repo.UpsertMapping(context.Background(), &entity.ImportUserMapping{
		ImportJobID:       job.ID,
		GitHubLogin:       "ghost",
		GitHubDisplayName: "Ghost User",
	}); err != nil {
		t.Fatalf("UpsertMapping: %v", err)
	}

	got, err := repo.GetMappingByLogin(context.Background(), job.ID, "ghost")
	if err != nil {
		t.Fatalf("GetMappingByLogin: %v", err)
	}
	if got == nil || got.GitHubLogin != "ghost" {
		t.Fatalf("unexpected mapping: %+v", got)
	}
}

func TestImportPhaseCheckpointRepository_SaveCheckpoint(t *testing.T) {
	db := newImportTestDB(t)
	repo := repository.NewImportJobRepository(db)

	orgID := insertImportTestOrg(t, db, "checkpoint-org")
	userID := insertImportTestUser(t, db, "checkpoint-user")

	job := &entity.ImportJob{
		OrganizationID: orgID,
		CreatedBy:      userID,
		SourceURL:      "https://github.com/acme/checkpoint",
		TargetName:     "checkpoint",
		Include:        []string{"issues"},
	}
	if err := repo.Create(context.Background(), job); err != nil {
		t.Fatalf("Create job: %v", err)
	}

	cp := &entity.ImportPhaseCheckpoint{
		ImportJobID: job.ID,
		Phase:       entity.ImportJobPhaseIssues,
		LastCursor:  "cursor-42",
		Completed:   false,
	}
	if err := repo.SaveCheckpoint(context.Background(), cp); err != nil {
		t.Fatalf("SaveCheckpoint insert: %v", err)
	}

	cp.LastCursor = "cursor-99"
	if err := repo.SaveCheckpoint(context.Background(), cp); err != nil {
		t.Fatalf("SaveCheckpoint update: %v", err)
	}

	got, err := repo.GetCheckpoint(context.Background(), job.ID, entity.ImportJobPhaseIssues)
	if err != nil {
		t.Fatalf("GetCheckpoint: %v", err)
	}
	if got == nil {
		t.Fatal("expected checkpoint, got nil")
	}
	if got.LastCursor != "cursor-99" {
		t.Fatalf("expected cursor-99, got %s", got.LastCursor)
	}
	if got.Completed {
		t.Fatal("expected checkpoint to remain incomplete")
	}
}

func TestImportPhaseCheckpointRepository_MarkPhaseComplete(t *testing.T) {
	db := newImportTestDB(t)
	repo := repository.NewImportJobRepository(db)

	orgID := insertImportTestOrg(t, db, "complete-org")
	userID := insertImportTestUser(t, db, "complete-user")

	job := &entity.ImportJob{
		OrganizationID: orgID,
		CreatedBy:      userID,
		SourceURL:      "https://github.com/acme/complete",
		TargetName:     "complete",
		Include:        []string{"wiki"},
	}
	if err := repo.Create(context.Background(), job); err != nil {
		t.Fatalf("Create job: %v", err)
	}

	if err := repo.SaveCheckpoint(context.Background(), &entity.ImportPhaseCheckpoint{
		ImportJobID: job.ID,
		Phase:       entity.ImportJobPhaseWiki,
		LastCursor:  "cursor-wiki",
		Completed:   false,
	}); err != nil {
		t.Fatalf("SaveCheckpoint: %v", err)
	}

	if err := repo.MarkPhaseComplete(context.Background(), job.ID, entity.ImportJobPhaseWiki); err != nil {
		t.Fatalf("MarkPhaseComplete: %v", err)
	}

	got, err := repo.GetCheckpoint(context.Background(), job.ID, entity.ImportJobPhaseWiki)
	if err != nil {
		t.Fatalf("GetCheckpoint: %v", err)
	}
	if got == nil {
		t.Fatal("expected checkpoint, got nil")
	}
	if !got.Completed {
		t.Fatal("expected checkpoint to be marked complete")
	}
}
