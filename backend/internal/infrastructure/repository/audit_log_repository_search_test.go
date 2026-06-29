package repository_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func setupAuditLogSearchColumns(t *testing.T, db *sqlx.DB) {
	t.Helper()

	if _, err := db.Exec(`ALTER TABLE audit_logs ADD COLUMN ip_address TEXT NOT NULL DEFAULT ''`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			t.Fatalf("add ip_address column: %v", err)
		}
	}
	if _, err := db.Exec(`ALTER TABLE audit_logs ADD COLUMN user_agent TEXT NOT NULL DEFAULT ''`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			t.Fatalf("add user_agent column: %v", err)
		}
	}
}

func newAuditLogSearchRepo(t *testing.T, db *sqlx.DB) domainrepo.IAuditLogSearchRepository {
	t.Helper()

	repo := repository.NewAuditLogRepository(db)
	searchRepo, ok := repo.(domainrepo.IAuditLogSearchRepository)
	if !ok {
		t.Fatal("audit log repository does not implement IAuditLogSearchRepository")
	}
	return searchRepo
}

func seedAuditLogSearchFixtures(t *testing.T, db *sqlx.DB) (orgID, actorID uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	orgID = uuid.New()
	actorID = uuid.New()

	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("exec %q: %v", query, err)
		}
	}

	exec(`INSERT INTO organizations (id, login, name) VALUES (?, ?, ?)`, orgID, "acme", "Acme")
	exec(`INSERT INTO users (id, login, email, password_hash) VALUES (?, ?, ?, ?)`, actorID, "alice", "alice@example.com", "hash")
	return orgID, actorID
}

func insertAuditLogRow(
	t *testing.T,
	db *sqlx.DB,
	orgID, actorID uuid.UUID,
	action, actorLogin string,
	createdAt time.Time,
) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO audit_logs (
			id, organization_id, actor_id, actor_login, action, target_type, target_id,
			metadata, ip_address, user_agent, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(),
		orgID.String(),
		actorID.String(),
		actorLogin,
		action,
		"repository",
		"repo-1",
		"{}",
		"",
		"",
		createdAt,
	)
	if err != nil {
		t.Fatalf("insert audit log: %v", err)
	}
}

func TestSearch_PhraseFilter(t *testing.T) {
	db := openTestDB(t)
	setupAuditLogSearchColumns(t, db)
	orgID, actorID := seedAuditLogSearchFixtures(t, db)
	repo := newAuditLogSearchRepo(t, db)
	ctx := context.Background()

	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	insertAuditLogRow(t, db, orgID, actorID, "repo.delete", "alice", base)
	insertAuditLogRow(t, db, orgID, actorID, "member.add", "bob", base.Add(time.Hour))

	logs, total, err := repo.Search(ctx, domainrepo.AuditLogSearchInput{
		OrganizationID: orgID,
		Phrase:         "delete",
		Page:           1,
		PerPage:        30,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if total != 1 || len(logs) != 1 {
		t.Fatalf("expected 1 log, got total=%d len=%d", total, len(logs))
	}
	if logs[0].Action != "repo.delete" {
		t.Fatalf("action: got %q, want repo.delete", logs[0].Action)
	}
}

func TestSearch_DateRangeFilter(t *testing.T) {
	db := openTestDB(t)
	setupAuditLogSearchColumns(t, db)
	orgID, actorID := seedAuditLogSearchFixtures(t, db)
	repo := newAuditLogSearchRepo(t, db)
	ctx := context.Background()

	early := time.Date(2025, 5, 1, 10, 0, 0, 0, time.UTC)
	middle := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	late := time.Date(2025, 7, 1, 10, 0, 0, 0, time.UTC)

	insertAuditLogRow(t, db, orgID, actorID, "repo.create", "alice", early)
	insertAuditLogRow(t, db, orgID, actorID, "repo.update", "alice", middle)
	insertAuditLogRow(t, db, orgID, actorID, "repo.delete", "alice", late)

	after := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	before := time.Date(2025, 6, 30, 23, 59, 59, 0, time.UTC)

	logs, total, err := repo.Search(ctx, domainrepo.AuditLogSearchInput{
		OrganizationID: orgID,
		After:          &after,
		Before:         &before,
		Page:           1,
		PerPage:        30,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if total != 1 || len(logs) != 1 {
		t.Fatalf("expected 1 log in date range, got total=%d len=%d", total, len(logs))
	}
	if logs[0].Action != "repo.update" {
		t.Fatalf("action: got %q, want repo.update", logs[0].Action)
	}
}

func TestSearch_CombinedFilters(t *testing.T) {
	db := openTestDB(t)
	setupAuditLogSearchColumns(t, db)
	orgID, actorID := seedAuditLogSearchFixtures(t, db)
	repo := newAuditLogSearchRepo(t, db)
	ctx := context.Background()

	inRange := time.Date(2025, 6, 10, 12, 0, 0, 0, time.UTC)
	outOfRange := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)

	insertAuditLogRow(t, db, orgID, actorID, "repo.delete", "alice", inRange)
	insertAuditLogRow(t, db, orgID, actorID, "member.delete", "alice", inRange)
	insertAuditLogRow(t, db, orgID, actorID, "repo.delete", "alice", outOfRange)

	after := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	before := time.Date(2025, 6, 30, 23, 59, 59, 0, time.UTC)

	logs, total, err := repo.Search(ctx, domainrepo.AuditLogSearchInput{
		OrganizationID: orgID,
		Phrase:         "repo",
		Action:         "repo.delete",
		After:          &after,
		Before:         &before,
		Page:           1,
		PerPage:        30,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if total != 1 || len(logs) != 1 {
		t.Fatalf("expected 1 combined match, got total=%d len=%d", total, len(logs))
	}
	if logs[0].Action != "repo.delete" {
		t.Fatalf("action: got %q, want repo.delete", logs[0].Action)
	}
	if !logs[0].CreatedAt.Equal(inRange) {
		t.Fatalf("createdAt: got %v, want %v", logs[0].CreatedAt, inRange)
	}
}
