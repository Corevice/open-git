package repository_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/crypto"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func setupActionSecretTables(t *testing.T, db *sqlx.DB) {
	t.Helper()

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS action_secrets (
			id TEXT PRIMARY KEY,
			organization_id TEXT NOT NULL,
			repository_id TEXT,
			name TEXT NOT NULL,
			encrypted_value BLOB NOT NULL,
			key_id TEXT NOT NULL DEFAULT '',
			visibility TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_action_secrets_org_repo_name
			ON action_secrets (organization_id, IFNULL(repository_id, ''), name);
		CREATE TABLE IF NOT EXISTS action_secret_repositories (
			secret_id TEXT NOT NULL,
			repository_id TEXT NOT NULL,
			PRIMARY KEY (secret_id, repository_id)
		);
	`)
	if err != nil {
		t.Fatalf("setup action secret tables: %v", err)
	}
}

func newActionSecretTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db := openTestDB(t)
	setupActionSecretTables(t, db)
	return db
}

func newTestActionSecretEncryptor() *crypto.ActionSecretEncryptor {
	return crypto.NewActionSecretEncryptor(bytes.Repeat([]byte{0x33}, 32))
}

func insertOrgActionSecret(
	t *testing.T,
	db *sqlx.DB,
	enc *crypto.ActionSecretEncryptor,
	orgID uuid.UUID,
	name, visibility string,
	innerValue []byte,
) uuid.UUID {
	t.Helper()

	encrypted, err := enc.Encrypt(innerValue)
	if err != nil {
		t.Fatalf("encrypt org secret: %v", err)
	}

	id := uuid.New()
	now := time.Now().UTC()
	_, err = db.Exec(`
		INSERT INTO action_secrets (
			id, organization_id, repository_id, name, encrypted_value, key_id, visibility, created_at, updated_at
		) VALUES (?, ?, NULL, ?, ?, '', ?, ?, ?)
	`, id, orgID, name, encrypted, visibility, now, now)
	if err != nil {
		t.Fatalf("insert org secret: %v", err)
	}
	return id
}

func TestActionSecretRepository_Upsert(t *testing.T) {
	db := newActionSecretTestDB(t)
	enc := newTestActionSecretEncryptor()
	repo := repository.NewActionSecretRepository(db, enc)

	orgID := createTestOrganization(t, db, "secret-org")
	userID := createTestUser(t, db, "secret-user")
	repoID := createTestRepositoryRecord(t, db, orgID, userID, "demo")

	secret := &entity.ActionSecret{
		OrganizationID: orgID,
		RepositoryID:   repoID,
		Name:           "MY_SECRET",
		EncryptedValue: string([]byte("aes-encrypted-value")),
	}

	created, err := repo.Upsert(context.Background(), secret)
	if err != nil {
		t.Fatalf("Upsert create: %v", err)
	}
	if !created {
		t.Fatal("expected created=true on first upsert")
	}

	created, err = repo.Upsert(context.Background(), secret)
	if err != nil {
		t.Fatalf("Upsert update: %v", err)
	}
	if created {
		t.Fatal("expected created=false on second upsert")
	}
}

func TestActionSecretRepository_GetByName(t *testing.T) {
	db := newActionSecretTestDB(t)
	enc := newTestActionSecretEncryptor()
	repo := repository.NewActionSecretRepository(db, enc)

	orgID := createTestOrganization(t, db, "get-secret-org")
	otherOrgID := createTestOrganization(t, db, "other-secret-org")
	userID := createTestUser(t, db, "get-secret-user")
	repoID := createTestRepositoryRecord(t, db, orgID, userID, "demo")

	secret := &entity.ActionSecret{
		OrganizationID: orgID,
		RepositoryID:   repoID,
		Name:           "FETCH_ME",
		EncryptedValue: string([]byte("aes-encrypted-value")),
	}
	if _, err := repo.Upsert(context.Background(), secret); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := repo.GetByName(context.Background(), orgID, &repoID, "FETCH_ME")
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	if got == nil || got.Name != "FETCH_ME" {
		t.Fatalf("unexpected secret: %+v", got)
	}

	_, err = repo.GetByName(context.Background(), otherOrgID, &repoID, "FETCH_ME")
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for other org, got %v", err)
	}
}

func TestActionSecretRepository_Delete(t *testing.T) {
	db := newActionSecretTestDB(t)
	enc := newTestActionSecretEncryptor()
	repo := repository.NewActionSecretRepository(db, enc)

	orgID := createTestOrganization(t, db, "delete-secret-org")
	userID := createTestUser(t, db, "delete-secret-user")
	repoID := createTestRepositoryRecord(t, db, orgID, userID, "demo")

	secret := &entity.ActionSecret{
		OrganizationID: orgID,
		RepositoryID:   repoID,
		Name:           "DELETE_ME",
		EncryptedValue: string([]byte("aes-encrypted-value")),
	}
	if _, err := repo.Upsert(context.Background(), secret); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	if err := repo.Delete(context.Background(), orgID, &repoID, "DELETE_ME"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := repo.GetByName(context.Background(), orgID, &repoID, "DELETE_ME")
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestActionSecretRepository_ListByRepo(t *testing.T) {
	db := newActionSecretTestDB(t)
	enc := newTestActionSecretEncryptor()
	repo := repository.NewActionSecretRepository(db, enc)

	orgID := createTestOrganization(t, db, "list-repo-org")
	userID := createTestUser(t, db, "list-repo-user")
	repoID := createTestRepositoryRecord(t, db, orgID, userID, "demo")

	secret := &entity.ActionSecret{
		OrganizationID: orgID,
		RepositoryID:   repoID,
		Name:           "REPO_LIST_SECRET",
		KeyID:          "repo-key",
		EncryptedValue: string([]byte("repo-value")),
	}
	if _, err := repo.Upsert(context.Background(), secret); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	secrets, err := repo.ListByRepo(context.Background(), orgID, repoID)
	if err != nil {
		t.Fatalf("ListByRepo: %v", err)
	}
	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(secrets))
	}
	if secrets[0].Name != "REPO_LIST_SECRET" {
		t.Fatalf("unexpected secret name: %q", secrets[0].Name)
	}
	if secrets[0].KeyID != "repo-key" {
		t.Fatalf("expected key id repo-key, got %q", secrets[0].KeyID)
	}
}

func TestActionSecretRepository_ListByOrg(t *testing.T) {
	db := newActionSecretTestDB(t)
	enc := newTestActionSecretEncryptor()
	repo := repository.NewActionSecretRepository(db, enc)

	orgID := createTestOrganization(t, db, "list-org-org")
	insertOrgActionSecret(t, db, enc, orgID, "ORG_LIST_SECRET", "selected", []byte("org-value"))

	secrets, err := repo.ListByOrg(context.Background(), orgID)
	if err != nil {
		t.Fatalf("ListByOrg: %v", err)
	}
	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(secrets))
	}
	if secrets[0].Name != "ORG_LIST_SECRET" {
		t.Fatalf("unexpected secret name: %q", secrets[0].Name)
	}
	if secrets[0].Visibility != "selected" {
		t.Fatalf("expected visibility selected, got %q", secrets[0].Visibility)
	}
}

func TestActionSecretRepository_GetSelectedRepositories(t *testing.T) {
	db := newActionSecretTestDB(t)
	enc := newTestActionSecretEncryptor()
	repo := repository.NewActionSecretRepository(db, enc)

	orgID := createTestOrganization(t, db, "selected-get-org")
	userID := createTestUser(t, db, "selected-get-user")
	repoID := createTestRepositoryRecord(t, db, orgID, userID, "demo")
	otherRepoID := createTestRepositoryRecord(t, db, orgID, userID, "other")

	secretID := insertOrgActionSecret(t, db, enc, orgID, "SELECTED_GET", "selected", []byte("selected-value"))
	if err := repo.SetSelectedRepositories(context.Background(), orgID, secretID, []uuid.UUID{repoID, otherRepoID}); err != nil {
		t.Fatalf("SetSelectedRepositories: %v", err)
	}

	got, err := repo.GetSelectedRepositories(context.Background(), orgID, secretID)
	if err != nil {
		t.Fatalf("GetSelectedRepositories: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 selected repositories, got %d", len(got))
	}
}

func TestActionSecretRepository_SetSelectedRepositories_RejectsOtherOrg(t *testing.T) {
	db := newActionSecretTestDB(t)
	enc := newTestActionSecretEncryptor()
	repo := repository.NewActionSecretRepository(db, enc)

	orgID := createTestOrganization(t, db, "owner-org")
	otherOrgID := createTestOrganization(t, db, "attacker-org")
	userID := createTestUser(t, db, "owner-user")
	repoID := createTestRepositoryRecord(t, db, orgID, userID, "demo")

	secretID := insertOrgActionSecret(t, db, enc, orgID, "PROTECTED", "selected", []byte("protected-value"))

	err := repo.SetSelectedRepositories(context.Background(), otherOrgID, secretID, []uuid.UUID{repoID})
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for cross-org write, got %v", err)
	}

	_, err = repo.GetSelectedRepositories(context.Background(), otherOrgID, secretID)
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for cross-org read, got %v", err)
	}
}

func TestActionSecretRepository_ListForWorkflowIncludesOrgAll(t *testing.T) {
	db := newActionSecretTestDB(t)
	enc := newTestActionSecretEncryptor()
	repo := repository.NewActionSecretRepository(db, enc)

	orgID := createTestOrganization(t, db, "workflow-org")
	userID := createTestUser(t, db, "workflow-user")
	repoID := createTestRepositoryRecord(t, db, orgID, userID, "demo")

	innerRepo := []byte("repo-aes-value")
	repoSecret := &entity.ActionSecret{
		OrganizationID: orgID,
		RepositoryID:   repoID,
		Name:           "REPO_SECRET",
		EncryptedValue: string(innerRepo),
	}
	if _, err := repo.Upsert(context.Background(), repoSecret); err != nil {
		t.Fatalf("Upsert repo secret: %v", err)
	}

	orgSecret := &entity.ActionSecret{
		OrganizationID: orgID,
		Name:           "ORG_SECRET",
		EncryptedValue: string([]byte("org-aes-value")),
	}
	if _, err := repo.Upsert(context.Background(), orgSecret); err != nil {
		t.Fatalf("Upsert org secret: %v", err)
	}

	secrets, err := repo.ListForWorkflow(context.Background(), orgID, repoID)
	if err != nil {
		t.Fatalf("ListForWorkflow: %v", err)
	}

	names := make(map[string]string, len(secrets))
	orderedNames := make([]string, 0, len(secrets))
	for _, secret := range secrets {
		names[secret.Name] = secret.EncryptedValue
		orderedNames = append(orderedNames, secret.Name)
	}
	if _, ok := names["REPO_SECRET"]; !ok {
		t.Fatal("expected repo secret in workflow list")
	}
	if _, ok := names["ORG_SECRET"]; !ok {
		t.Fatal("expected org secret with visibility=all in workflow list")
	}
	if names["REPO_SECRET"] != string(innerRepo) {
		t.Fatalf("expected decrypted repo value, got %q", names["REPO_SECRET"])
	}
	if len(orderedNames) != 2 || orderedNames[0] != "ORG_SECRET" || orderedNames[1] != "REPO_SECRET" {
		t.Fatalf("expected deterministic name ordering, got %v", orderedNames)
	}
}

func TestActionSecretRepository_ListForWorkflowExcludesSelectedWhenNotAllowed(t *testing.T) {
	db := newActionSecretTestDB(t)
	enc := newTestActionSecretEncryptor()
	repo := repository.NewActionSecretRepository(db, enc)

	orgID := createTestOrganization(t, db, "selected-org")
	userID := createTestUser(t, db, "selected-user")
	repoID := createTestRepositoryRecord(t, db, orgID, userID, "demo")
	otherRepoID := createTestRepositoryRecord(t, db, orgID, userID, "other")

	insertOrgActionSecret(t, db, enc, orgID, "SELECTED_SECRET", "selected", []byte("selected-aes-value"))

	secretID := insertOrgActionSecret(t, db, enc, orgID, "ALLOWED_SELECTED", "selected", []byte("allowed-aes-value"))
	if err := repo.SetSelectedRepositories(context.Background(), orgID, secretID, []uuid.UUID{otherRepoID}); err != nil {
		t.Fatalf("SetSelectedRepositories: %v", err)
	}

	secrets, err := repo.ListForWorkflow(context.Background(), orgID, repoID)
	if err != nil {
		t.Fatalf("ListForWorkflow: %v", err)
	}

	for _, secret := range secrets {
		if secret.Name == "SELECTED_SECRET" {
			t.Fatal("selected org secret should be excluded when repo is not allowed")
		}
		if secret.Name == "ALLOWED_SELECTED" {
			t.Fatal("selected org secret for another repo should be excluded")
		}
	}
}
