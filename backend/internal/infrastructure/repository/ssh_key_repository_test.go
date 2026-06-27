package repository_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/ssh"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

const migration003SSHKeys = `
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    login TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE ssh_keys (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    title TEXT NOT NULL,
    fingerprint TEXT NOT NULL UNIQUE,
    public_key TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

const testAuthorizedKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOzRANdrmNo46uGr2ky5ETd7ObwPSeqqxgc/K27LwS1P test@example.com"

func newSSHKeyTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec(migration003SSHKeys); err != nil {
		_ = db.Close()
		t.Fatalf("apply migration 003 schema: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return sqlx.NewDb(db, "sqlite3")
}

func insertTestUser(t *testing.T, db *sqlx.DB, id uuid.UUID) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO users (id, login, email, password_hash, created_at) VALUES (?, ?, ?, ?, ?)`,
		id.String(), "user-"+id.String()[:8], id.String()+"@example.com", "hash", time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
}

func TestSSHKeyCreate_ComputesFingerprint(t *testing.T) {
	db := newSSHKeyTestDB(t)
	repo := repository.NewSSHKeyRepository(db)

	userID := uuid.New()
	insertTestUser(t, db, userID)

	key := &entity.SSHKey{
		UserID:    userID,
		Title:     "laptop",
		PublicKey: testAuthorizedKey,
	}
	if err := repo.Create(context.Background(), key); err != nil {
		t.Fatalf("Create: %v", err)
	}

	pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(testAuthorizedKey))
	if err != nil {
		t.Fatalf("ParseAuthorizedKey: %v", err)
	}
	expected := ssh.FingerprintSHA256(pubKey)
	if key.Fingerprint != expected {
		t.Fatalf("fingerprint = %q, want %q", key.Fingerprint, expected)
	}
	if !strings.HasPrefix(key.Fingerprint, "SHA256:") {
		t.Fatalf("fingerprint %q is not SHA256 format", key.Fingerprint)
	}
}

func TestFindByFingerprint_Found(t *testing.T) {
	db := newSSHKeyTestDB(t)
	repo := repository.NewSSHKeyRepository(db)

	userID := uuid.New()
	insertTestUser(t, db, userID)

	key := &entity.SSHKey{
		UserID:    userID,
		Title:     "work",
		PublicKey: testAuthorizedKey,
	}
	if err := repo.Create(context.Background(), key); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.FindByFingerprint(context.Background(), key.Fingerprint)
	if err != nil {
		t.Fatalf("FindByFingerprint: %v", err)
	}
	if got == nil {
		t.Fatal("expected key, got nil")
	}
	if got.ID != key.ID || got.Fingerprint != key.Fingerprint {
		t.Fatalf("unexpected key: %+v", got)
	}
}

func TestFindByFingerprint_NotFound(t *testing.T) {
	db := newSSHKeyTestDB(t)
	repo := repository.NewSSHKeyRepository(db)

	got, err := repo.FindByFingerprint(context.Background(), "SHA256:nonexistent")
	if err != nil {
		t.Fatalf("FindByFingerprint: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestDelete_WrongUser(t *testing.T) {
	db := newSSHKeyTestDB(t)
	repo := repository.NewSSHKeyRepository(db)

	ownerID := uuid.New()
	otherUserID := uuid.New()
	insertTestUser(t, db, ownerID)
	insertTestUser(t, db, otherUserID)

	key := &entity.SSHKey{
		UserID:    ownerID,
		Title:     "home",
		PublicKey: testAuthorizedKey,
	}
	if err := repo.Create(context.Background(), key); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(context.Background(), key.ID, otherUserID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, err := repo.FindByFingerprint(context.Background(), key.Fingerprint)
	if err != nil {
		t.Fatalf("FindByFingerprint: %v", err)
	}
	if got == nil {
		t.Fatal("expected key to remain after wrong-user delete")
	}
}
