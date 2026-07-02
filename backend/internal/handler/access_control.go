package handler

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/middleware"
)

// accessMembership resolves a user's organization role (RoleOwner/Admin/Member),
// returning domain.ErrNotFound when the user is not a member.
type accessMembership interface {
	GetRole(ctx context.Context, orgID, userID uuid.UUID) (string, error)
}

// accessCollaborator resolves a user's per-repository collaborator permission
// (read/write/admin), or "" when they are not a collaborator.
type accessCollaborator interface {
	GetPermission(ctx context.Context, repoID, userID uuid.UUID) (string, error)
}

// RepoAccess centralizes repository and organization authorization so mutating
// and private-read handlers enforce it consistently. It mirrors the policy used
// by the git HTTP/SSH layer: the owner (including personal-namespace owner),
// organization role, and collaborator permission all grant access.
//
// A nil *RepoAccess authorizes everything; production always wires a real one,
// while unit tests that exercise handler logic in isolation may leave it unset.
type RepoAccess struct {
	memberships   accessMembership
	collaborators accessCollaborator
}

func NewRepoAccess(m accessMembership, c accessCollaborator) *RepoAccess {
	return &RepoAccess{memberships: m, collaborators: c}
}

// repoRef is the minimal set of repository facts authorization needs, letting
// the same logic serve both entity.Repository and the git-layer
// ResolvedGitRepository.
type repoRef struct {
	id          uuid.UUID
	orgID       uuid.UUID
	ownerUserID int64 // individual owner's user id
	private     bool
}

func refFromEntity(repo *entity.Repository) repoRef {
	return repoRef{
		id:          repo.ID,
		orgID:       repo.OrganizationID,
		ownerUserID: middleware.UUIDToInt64(repo.OwnerID),
		private:     repo.Visibility == entity.VisibilityPrivate,
	}
}

func (a *RepoAccess) isOwner(userID int64, r repoRef) bool {
	// The individual owner, or the personal-namespace owner (a personal repo's
	// organization id equals the owner's user id and has no membership row).
	return userID != 0 && (userID == r.ownerUserID || middleware.Int64ToUUID(userID) == r.orgID)
}

func (a *RepoAccess) orgRole(ctx context.Context, orgID uuid.UUID, userID int64) string {
	if a.memberships == nil || userID == 0 {
		return ""
	}
	role, err := a.memberships.GetRole(ctx, orgID, middleware.Int64ToUUID(userID))
	if err != nil {
		return ""
	}
	return role
}

func (a *RepoAccess) collabPerm(ctx context.Context, repoID uuid.UUID, userID int64) string {
	if a.collaborators == nil || userID == 0 {
		return ""
	}
	perm, err := a.collaborators.GetPermission(ctx, repoID, middleware.Int64ToUUID(userID))
	if err != nil {
		return ""
	}
	return perm
}

func (a *RepoAccess) canRead(ctx context.Context, userID int64, r repoRef) bool {
	if a == nil || !r.private {
		return true
	}
	if a.isOwner(userID, r) {
		return true
	}
	if a.orgRole(ctx, r.orgID, userID) != "" {
		return true
	}
	return a.collabPerm(ctx, r.id, userID) != ""
}

func (a *RepoAccess) canWrite(ctx context.Context, userID int64, r repoRef) bool {
	if a == nil {
		return true
	}
	if a.isOwner(userID, r) {
		return true
	}
	if role := a.orgRole(ctx, r.orgID, userID); role == entity.RoleOwner || role == entity.RoleAdmin {
		return true
	}
	perm := a.collabPerm(ctx, r.id, userID)
	return perm == entity.CollaboratorPermWrite || perm == entity.CollaboratorPermAdmin
}

func (a *RepoAccess) canAdmin(ctx context.Context, userID int64, r repoRef) bool {
	if a == nil {
		return true
	}
	if a.isOwner(userID, r) {
		return true
	}
	if role := a.orgRole(ctx, r.orgID, userID); role == entity.RoleOwner || role == entity.RoleAdmin {
		return true
	}
	return a.collabPerm(ctx, r.id, userID) == entity.CollaboratorPermAdmin
}

// CanRead/CanWrite/CanAdmin operate on an entity.Repository.
func (a *RepoAccess) CanRead(ctx context.Context, userID int64, repo *entity.Repository) bool {
	return a.canRead(ctx, userID, refFromEntity(repo))
}

func (a *RepoAccess) CanWrite(ctx context.Context, userID int64, repo *entity.Repository) bool {
	return a.canWrite(ctx, userID, refFromEntity(repo))
}

func (a *RepoAccess) CanAdmin(ctx context.Context, userID int64, repo *entity.Repository) bool {
	return a.canAdmin(ctx, userID, refFromEntity(repo))
}

func forbidden() error {
	return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "Forbidden"})
}

func notFound() error {
	return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
}

// EnsureRead returns a 404 (to avoid disclosing existence) when the caller may
// not read the repository.
func (a *RepoAccess) EnsureRead(c echo.Context, repo *entity.Repository) error {
	if a.canRead(c.Request().Context(), middleware.UserIDFromContext(c), refFromEntity(repo)) {
		return nil
	}
	return notFound()
}

// EnsureWrite returns a 403 when the caller lacks write access.
func (a *RepoAccess) EnsureWrite(c echo.Context, repo *entity.Repository) error {
	if a.canWrite(c.Request().Context(), middleware.UserIDFromContext(c), refFromEntity(repo)) {
		return nil
	}
	return forbidden()
}

// EnsureAdmin returns a 403 when the caller lacks admin access.
func (a *RepoAccess) EnsureAdmin(c echo.Context, repo *entity.Repository) error {
	if a.canAdmin(c.Request().Context(), middleware.UserIDFromContext(c), refFromEntity(repo)) {
		return nil
	}
	return forbidden()
}

// EnsureReadGit authorizes read access to a git-layer ResolvedGitRepository
// (used by the content/blob/commit browsing endpoints).
func (a *RepoAccess) EnsureReadGit(c echo.Context, repo *ResolvedGitRepository) error {
	r := repoRef{
		id:          repo.ID,
		orgID:       repo.OrganizationID,
		ownerUserID: repo.OwnerID,
		private:     repo.Visibility == string(entity.VisibilityPrivate),
	}
	if a.canRead(c.Request().Context(), middleware.UserIDFromContext(c), r) {
		return nil
	}
	return notFound()
}

// EnsureOrgMember returns a 404 when the caller is not a member of the org
// (personal-namespace owner counts as a member of their own namespace).
func (a *RepoAccess) EnsureOrgMember(c echo.Context, orgID uuid.UUID) error {
	if a == nil {
		return nil
	}
	userID := middleware.UserIDFromContext(c)
	if userID != 0 && middleware.Int64ToUUID(userID) == orgID {
		return nil
	}
	if a.orgRole(c.Request().Context(), orgID, userID) != "" {
		return nil
	}
	return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
}

// EnsureOrgAdmin returns a 403 when the caller is not an owner/admin of the org.
func (a *RepoAccess) EnsureOrgAdmin(c echo.Context, orgID uuid.UUID) error {
	if a == nil {
		return nil
	}
	userID := middleware.UserIDFromContext(c)
	if userID != 0 && middleware.Int64ToUUID(userID) == orgID {
		return nil
	}
	role := a.orgRole(c.Request().Context(), orgID, userID)
	if role == entity.RoleOwner || role == entity.RoleAdmin {
		return nil
	}
	return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "Forbidden"})
}

