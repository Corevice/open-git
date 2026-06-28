package repository_test

import (
	"context"
	"testing"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func TestAddCollaborator_roundtrip(t *testing.T) {
	db := openTestDB(t)
	repo := repository.NewRepositoryCollaboratorRepository(db)

	orgID := createTestOrganization(t, db, "collab-org")
	ownerID := createTestUser(t, db, "collab-owner")
	collaboratorID := createTestUser(t, db, "collab-user")
	repoID := createTestRepositoryRecord(t, db, orgID, ownerID, "demo")

	ctx := context.Background()
	if err := repo.AddCollaborator(ctx, repoID, collaboratorID, entity.CollaboratorPermRead); err != nil {
		t.Fatalf("AddCollaborator: %v", err)
	}

	perm, err := repo.GetPermission(ctx, repoID, collaboratorID)
	if err != nil {
		t.Fatalf("GetPermission: %v", err)
	}
	if perm != entity.CollaboratorPermRead {
		t.Fatalf("permission = %q, want %q", perm, entity.CollaboratorPermRead)
	}

	if err := repo.AddCollaborator(ctx, repoID, collaboratorID, entity.CollaboratorPermWrite); err != nil {
		t.Fatalf("AddCollaborator upsert: %v", err)
	}

	perm, err = repo.GetPermission(ctx, repoID, collaboratorID)
	if err != nil {
		t.Fatalf("GetPermission after upsert: %v", err)
	}
	if perm != entity.CollaboratorPermWrite {
		t.Fatalf("permission after upsert = %q, want %q", perm, entity.CollaboratorPermWrite)
	}
}

func TestGetPermission_notFound_returnsEmpty(t *testing.T) {
	db := openTestDB(t)
	repo := repository.NewRepositoryCollaboratorRepository(db)

	orgID := createTestOrganization(t, db, "perm-org")
	ownerID := createTestUser(t, db, "perm-owner")
	repoID := createTestRepositoryRecord(t, db, orgID, ownerID, "demo")

	perm, err := repo.GetPermission(context.Background(), repoID, ownerID)
	if err != nil {
		t.Fatalf("GetPermission: %v", err)
	}
	if perm != "" {
		t.Fatalf("permission = %q, want empty string", perm)
	}
}

func TestRemoveCollaborator(t *testing.T) {
	db := openTestDB(t)
	repo := repository.NewRepositoryCollaboratorRepository(db)

	orgID := createTestOrganization(t, db, "remove-collab-org")
	ownerID := createTestUser(t, db, "remove-collab-owner")
	collaboratorID := createTestUser(t, db, "remove-collab-user")
	repoID := createTestRepositoryRecord(t, db, orgID, ownerID, "demo")

	ctx := context.Background()
	if err := repo.AddCollaborator(ctx, repoID, collaboratorID, entity.CollaboratorPermAdmin); err != nil {
		t.Fatalf("AddCollaborator: %v", err)
	}

	if err := repo.RemoveCollaborator(ctx, repoID, collaboratorID); err != nil {
		t.Fatalf("RemoveCollaborator: %v", err)
	}

	perm, err := repo.GetPermission(ctx, repoID, collaboratorID)
	if err != nil {
		t.Fatalf("GetPermission after remove: %v", err)
	}
	if perm != "" {
		t.Fatalf("permission after remove = %q, want empty string", perm)
	}
}

func TestListCollaborators(t *testing.T) {
	db := openTestDB(t)
	repo := repository.NewRepositoryCollaboratorRepository(db)

	orgID := createTestOrganization(t, db, "list-collab-org")
	ownerID := createTestUser(t, db, "list-collab-owner")
	userA := createTestUser(t, db, "list-collab-a")
	userB := createTestUser(t, db, "list-collab-b")
	repoID := createTestRepositoryRecord(t, db, orgID, ownerID, "demo")

	ctx := context.Background()
	if err := repo.AddCollaborator(ctx, repoID, userA, entity.CollaboratorPermRead); err != nil {
		t.Fatalf("AddCollaborator userA: %v", err)
	}
	if err := repo.AddCollaborator(ctx, repoID, userB, entity.CollaboratorPermWrite); err != nil {
		t.Fatalf("AddCollaborator userB: %v", err)
	}

	collaborators, err := repo.ListCollaborators(ctx, repoID)
	if err != nil {
		t.Fatalf("ListCollaborators: %v", err)
	}
	if len(collaborators) != 2 {
		t.Fatalf("expected 2 collaborators, got %d", len(collaborators))
	}

	byUser := map[string]string{}
	for _, c := range collaborators {
		if c.RepositoryID != repoID {
			t.Fatalf("unexpected repository_id %s", c.RepositoryID)
		}
		byUser[c.UserID.String()] = c.Permission
	}
	if byUser[userA.String()] != entity.CollaboratorPermRead {
		t.Fatalf("userA permission = %q, want %q", byUser[userA.String()], entity.CollaboratorPermRead)
	}
	if byUser[userB.String()] != entity.CollaboratorPermWrite {
		t.Fatalf("userB permission = %q, want %q", byUser[userB.String()], entity.CollaboratorPermWrite)
	}
}
