package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/database"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func newMembershipTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := database.RunMigrations(db, "sqlite", "../../migrations"); err != nil {
		_ = db.Close()
		t.Fatalf("run migrations: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return sqlx.NewDb(db, "sqlite3")
}

func createTestUser(t *testing.T, db *sqlx.DB, login string) uuid.UUID {
	t.Helper()
	userRepo := repository.NewUserRepository(db)
	user := &entity.User{
		Login:        login,
		Email:        login + "@example.com",
		PasswordHash: "hashed",
	}
	if err := userRepo.Create(context.Background(), user); err != nil {
		t.Fatalf("create user %s: %v", login, err)
	}
	return user.ID
}

func createTestOrganization(t *testing.T, db *sqlx.DB, login string) uuid.UUID {
	t.Helper()
	orgRepo := repository.NewOrganizationRepository(db)
	org := &entity.Organization{
		Login: login,
		Name:  login,
	}
	if err := orgRepo.Create(context.Background(), org); err != nil {
		t.Fatalf("create org %s: %v", login, err)
	}
	return org.ID
}

func TestMembershipRepository_AddGetRole(t *testing.T) {
	db := newMembershipTestDB(t)
	repo := repository.NewMembershipRepository(db)

	orgID := createTestOrganization(t, db, "members-org")
	userID := createTestUser(t, db, "member-user")

	m := &entity.Membership{
		OrganizationID: orgID,
		UserID:         userID,
		Role:           entity.RoleMember,
	}
	if err := repo.Add(context.Background(), m); err != nil {
		t.Fatalf("Add: %v", err)
	}

	role, err := repo.GetRole(context.Background(), orgID, userID)
	if err != nil {
		t.Fatalf("GetRole: %v", err)
	}
	if role != entity.RoleMember {
		t.Fatalf("role = %q, want %q", role, entity.RoleMember)
	}
}

func TestMembershipRepository_ListByOrgCrossOrgIsolation(t *testing.T) {
	db := newMembershipTestDB(t)
	repo := repository.NewMembershipRepository(db)

	orgA := createTestOrganization(t, db, "org-a")
	orgB := createTestOrganization(t, db, "org-b")
	userA := createTestUser(t, db, "user-a")
	userB := createTestUser(t, db, "user-b")

	if err := repo.Add(context.Background(), &entity.Membership{
		OrganizationID: orgA,
		UserID:         userA,
		Role:           entity.RoleOwner,
	}); err != nil {
		t.Fatalf("Add orgA member: %v", err)
	}
	if err := repo.Add(context.Background(), &entity.Membership{
		OrganizationID: orgB,
		UserID:         userB,
		Role:           entity.RoleAdmin,
	}); err != nil {
		t.Fatalf("Add orgB member: %v", err)
	}

	members, err := repo.ListByOrg(context.Background(), orgA, 1, 10)
	if err != nil {
		t.Fatalf("ListByOrg: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member for orgA, got %d", len(members))
	}
	if members[0].UserID != userA {
		t.Fatalf("expected userA in orgA list, got %s", members[0].UserID)
	}
	if members[0].OrganizationID != orgA {
		t.Fatalf("expected orgA scope, got %s", members[0].OrganizationID)
	}
}

func TestMembershipRepository_UpdateRole(t *testing.T) {
	db := newMembershipTestDB(t)
	repo := repository.NewMembershipRepository(db)

	orgID := createTestOrganization(t, db, "update-org")
	userID := createTestUser(t, db, "update-user")

	if err := repo.Add(context.Background(), &entity.Membership{
		OrganizationID: orgID,
		UserID:         userID,
		Role:           entity.RoleMember,
	}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if err := repo.UpdateRole(context.Background(), orgID, userID, entity.RoleAdmin); err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}

	role, err := repo.GetRole(context.Background(), orgID, userID)
	if err != nil {
		t.Fatalf("GetRole: %v", err)
	}
	if role != entity.RoleAdmin {
		t.Fatalf("role = %q, want %q", role, entity.RoleAdmin)
	}
}

func TestMembershipRepository_Remove(t *testing.T) {
	db := newMembershipTestDB(t)
	repo := repository.NewMembershipRepository(db)

	orgID := createTestOrganization(t, db, "remove-org")
	userID := createTestUser(t, db, "remove-user")

	if err := repo.Add(context.Background(), &entity.Membership{
		OrganizationID: orgID,
		UserID:         userID,
		Role:           entity.RoleMember,
	}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if err := repo.Remove(context.Background(), orgID, userID); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	role, err := repo.GetRole(context.Background(), orgID, userID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after remove, got role=%q err=%v", role, err)
	}
}
