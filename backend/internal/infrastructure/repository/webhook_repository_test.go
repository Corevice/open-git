package repository_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/crypto"
	"github.com/open-git/backend/internal/infrastructure/database"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func newWebhookTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := database.RunMigrations(db, "sqlite", "../../../migrations"); err != nil {
		_ = db.Close()
		t.Fatalf("run migrations: %v", err)
	}

	_, err = db.Exec(`ALTER TABLE webhooks ADD COLUMN secret_encrypted BLOB`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column") {
		_ = db.Close()
		t.Fatalf("add secret_encrypted column: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })
	return sqlx.NewDb(db, "sqlite3")
}

func newTestEncryptor() *crypto.SecretEncryptor {
	return crypto.NewSecretEncryptor(bytes.Repeat([]byte{0x11}, 32))
}

func createTestRepositoryRecord(t *testing.T, db *sqlx.DB, orgID, ownerID uuid.UUID, name string) uuid.UUID {
	t.Helper()

	repoRepo := repository.NewRepositoryRepository(db)
	repo := &entity.Repository{
		OrganizationID: orgID,
		OwnerID:        ownerID,
		Name:           name,
		Visibility:     entity.VisibilityPrivate,
		DefaultBranch:  "main",
	}
	if err := repoRepo.Create(context.Background(), repo); err != nil {
		t.Fatalf("create repository: %v", err)
	}
	return repo.ID
}

func TestWebhookRepository_CreateGetByID(t *testing.T) {
	db := newWebhookTestDB(t)
	enc := newTestEncryptor()
	repo := repository.NewWebhookRepository(db, enc)

	orgID := createTestOrganization(t, db, "hook-org")
	userID := createTestUser(t, db, "hook-user")
	repoID := createTestRepositoryRecord(t, db, orgID, userID, "demo")

	plaintextSecret := []byte("super-secret")
	webhook := &entity.Webhook{
		OrganizationID:  orgID,
		RepositoryID:  &repoID,
		URL:             "https://example.com/hook",
		ContentType:     entity.ContentTypeJSON,
		SecretEncrypted: plaintextSecret,
		Events:          []string{"push"},
		Active:          true,
	}
	if err := repo.Create(context.Background(), webhook); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(context.Background(), webhook.ID, orgID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected webhook, got nil")
	}
	if got.URL != webhook.URL || got.ContentType != webhook.ContentType {
		t.Fatalf("unexpected webhook fields: %+v", got)
	}
	if bytes.Equal(got.SecretEncrypted, plaintextSecret) {
		t.Fatal("expected encrypted secret in storage, got plaintext")
	}
	decrypted, err := enc.Decrypt(got.SecretEncrypted)
	if err != nil {
		t.Fatalf("Decrypt stored secret: %v", err)
	}
	if !bytes.Equal(decrypted, plaintextSecret) {
		t.Fatalf("decrypted secret mismatch: got %q, want %q", decrypted, plaintextSecret)
	}
}

func TestWebhookRepository_GetByIDWrongOrg(t *testing.T) {
	db := newWebhookTestDB(t)
	repo := repository.NewWebhookRepository(db, newTestEncryptor())

	orgID := createTestOrganization(t, db, "hook-org-a")
	otherOrgID := createTestOrganization(t, db, "hook-org-b")
	userID := createTestUser(t, db, "hook-user-a")
	repoID := createTestRepositoryRecord(t, db, orgID, userID, "demo")

	webhook := &entity.Webhook{
		OrganizationID: orgID,
		RepositoryID:   &repoID,
		URL:            "https://example.com/hook",
		ContentType:    entity.ContentTypeJSON,
		Events:         []string{"push"},
		Active:         true,
	}
	if err := repo.Create(context.Background(), webhook); err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err := repo.GetByID(context.Background(), webhook.ID, otherOrgID)
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestWebhookRepository_ListByRepo(t *testing.T) {
	db := newWebhookTestDB(t)
	repo := repository.NewWebhookRepository(db, newTestEncryptor())

	orgID := createTestOrganization(t, db, "list-org")
	userID := createTestUser(t, db, "list-user")
	repoID := createTestRepositoryRecord(t, db, orgID, userID, "demo")

	for i := 0; i < 2; i++ {
		webhook := &entity.Webhook{
			OrganizationID: orgID,
			RepositoryID:   &repoID,
			URL:            "https://example.com/hook" + string(rune('a'+i)),
			ContentType:    entity.ContentTypeJSON,
			Events:         []string{"push"},
			Active:         true,
		}
		if err := repo.Create(context.Background(), webhook); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	webhooks, total, err := repo.ListByRepo(context.Background(), orgID, repoID, 1, 10)
	if err != nil {
		t.Fatalf("ListByRepo: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}
	if len(webhooks) != 2 {
		t.Fatalf("expected 2 webhooks, got %d", len(webhooks))
	}
}

func TestWebhookRepository_Update(t *testing.T) {
	db := newWebhookTestDB(t)
	repo := repository.NewWebhookRepository(db, newTestEncryptor())

	orgID := createTestOrganization(t, db, "update-org")
	userID := createTestUser(t, db, "update-user")
	repoID := createTestRepositoryRecord(t, db, orgID, userID, "demo")

	webhook := &entity.Webhook{
		OrganizationID: orgID,
		RepositoryID:   &repoID,
		URL:            "https://example.com/old",
		ContentType:    entity.ContentTypeJSON,
		Events:         []string{"push"},
		Active:         true,
	}
	if err := repo.Create(context.Background(), webhook); err != nil {
		t.Fatalf("Create: %v", err)
	}

	webhook.URL = "https://example.com/new"
	webhook.Active = false
	if err := repo.Update(context.Background(), webhook); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.GetByID(context.Background(), webhook.ID, orgID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.URL != "https://example.com/new" || got.Active {
		t.Fatalf("unexpected updated webhook: %+v", got)
	}
}

func TestWebhookRepository_Delete(t *testing.T) {
	db := newWebhookTestDB(t)
	repo := repository.NewWebhookRepository(db, newTestEncryptor())

	orgID := createTestOrganization(t, db, "delete-org")
	userID := createTestUser(t, db, "delete-user")
	repoID := createTestRepositoryRecord(t, db, orgID, userID, "demo")

	webhook := &entity.Webhook{
		OrganizationID: orgID,
		RepositoryID:   &repoID,
		URL:            "https://example.com/hook",
		ContentType:    entity.ContentTypeJSON,
		Events:         []string{"push"},
		Active:         true,
	}
	if err := repo.Create(context.Background(), webhook); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(context.Background(), webhook.ID, orgID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := repo.GetByID(context.Background(), webhook.ID, orgID)
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestWebhookRepository_ListActiveByRepoAndEvent(t *testing.T) {
	db := newWebhookTestDB(t)
	repo := repository.NewWebhookRepository(db, newTestEncryptor())

	orgID := createTestOrganization(t, db, "active-org")
	userID := createTestUser(t, db, "active-user")
	repoID := createTestRepositoryRecord(t, db, orgID, userID, "demo")

	activePush := &entity.Webhook{
		OrganizationID: orgID,
		RepositoryID:   &repoID,
		URL:            "https://example.com/push",
		ContentType:    entity.ContentTypeJSON,
		Events:         []string{"push"},
		Active:         true,
	}
	if err := repo.Create(context.Background(), activePush); err != nil {
		t.Fatalf("Create active push: %v", err)
	}

	activeWildcard := &entity.Webhook{
		OrganizationID: orgID,
		RepositoryID:   &repoID,
		URL:            "https://example.com/all",
		ContentType:    entity.ContentTypeJSON,
		Events:         []string{"*"},
		Active:         true,
	}
	if err := repo.Create(context.Background(), activeWildcard); err != nil {
		t.Fatalf("Create active wildcard: %v", err)
	}

	inactivePush := &entity.Webhook{
		OrganizationID: orgID,
		RepositoryID:   &repoID,
		URL:            "https://example.com/inactive",
		ContentType:    entity.ContentTypeJSON,
		Events:         []string{"push"},
		Active:         false,
	}
	if err := repo.Create(context.Background(), inactivePush); err != nil {
		t.Fatalf("Create inactive push: %v", err)
	}

	webhooks, err := repo.ListActiveByRepoAndEvent(context.Background(), orgID, repoID, "push")
	if err != nil {
		t.Fatalf("ListActiveByRepoAndEvent: %v", err)
	}
	if len(webhooks) != 2 {
		t.Fatalf("expected 2 active webhooks, got %d", len(webhooks))
	}
}
