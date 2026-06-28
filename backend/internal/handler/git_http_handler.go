package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/server"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	infragit "github.com/open-git/backend/internal/infrastructure/git"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/middleware"
)

// GitMembershipAccess checks organization read/write permission and resolves roles.
type GitMembershipAccess interface {
	HasReadAccess(ctx context.Context, userID int64, organizationID uuid.UUID) (bool, error)
	HasWriteAccess(ctx context.Context, userID int64, organizationID uuid.UUID) (bool, error)
	GetRole(ctx context.Context, userID int64, organizationID uuid.UUID) (string, error)
}

// ResolvedGitRepository is metadata required to serve Git Smart HTTP for a repo.
type ResolvedGitRepository struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	OwnerID        int64
	Name           string
	Visibility     string
	DiskPath       string
}

// GitRepositoryResolver resolves owner/repo to on-disk bare repository metadata.
type GitRepositoryResolver interface {
	Resolve(ctx context.Context, ownerLogin, repoName string) (*ResolvedGitRepository, error)
}

// GitBranchProtectionRule describes enforcement settings for a protected branch.
type GitBranchProtectionRule struct {
	Protected        bool
	EnforceAdmins    bool
	AllowForcePushes bool
	AllowDeletions   bool
}

// GitBranchProtectionStore reports branch protection settings for push enforcement.
type GitBranchProtectionStore interface {
	GetBranchProtection(ctx context.Context, repositoryID uuid.UUID, branch string) (GitBranchProtectionRule, error)
}

// GitHTTPHandler serves Git Smart HTTP protocol endpoints.
type GitHTTPHandler struct {
	gitRoot      string
	resolver     GitRepositoryResolver
	memberships  GitMembershipAccess
	protections  GitBranchProtectionStore
	authRequired echo.MiddlewareFunc
}

func NewGitHTTPHandler(
	gitRoot string,
	resolver GitRepositoryResolver,
	memberships GitMembershipAccess,
	protections GitBranchProtectionStore,
	authRequired echo.MiddlewareFunc,
) *GitHTTPHandler {
	return &GitHTTPHandler{
		gitRoot:      gitRoot,
		resolver:     resolver,
		memberships:  memberships,
		protections:  protections,
		authRequired: authRequired,
	}
}

func (h *GitHTTPHandler) RegisterRoutes(e *echo.Echo) {
	e.GET("/:owner/:repo.git/info/refs", h.InfoRefs)
	e.POST("/:owner/:repo.git/git-upload-pack", h.UploadPack)
	if h.authRequired != nil {
		e.POST("/:owner/:repo.git/git-receive-pack", h.ReceivePack, h.authRequired)
	} else {
		e.POST("/:owner/:repo.git/git-receive-pack", h.ReceivePack)
	}
}

func (h *GitHTTPHandler) repoPath(owner, repo string) string {
	return filepath.Join(h.gitRoot, owner, repo+".git")
}

func (h *GitHTTPHandler) resolveRepo(c echo.Context) (*ResolvedGitRepository, error) {
	if h.resolver != nil {
		repo, err := h.resolver.Resolve(c.Request().Context(), c.Param("owner"), strings.TrimSuffix(c.Param("repo"), ".git"))
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		if repo.DiskPath == "" {
			repo.DiskPath = h.repoPath(c.Param("owner"), c.Param("repo"))
		}
		return repo, nil
	}

	path := h.repoPath(c.Param("owner"), c.Param("repo"))
	if _, err := os.Stat(path); err != nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}
	return &ResolvedGitRepository{DiskPath: path}, nil
}

// InfoRefs handles GET /:owner/:repo.git/info/refs?service=
func (h *GitHTTPHandler) InfoRefs(c echo.Context) error {
	service := c.QueryParam("service")
	if service != transport.UploadPackService.String() && service != transport.ReceivePackService.String() {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid service"})
	}

	repo, err := h.resolveRepo(c)
	if err != nil {
		return err
	}
	if service == transport.ReceivePackService.String() {
		userID, err := middleware.GetUserID(c)
		if err != nil {
			return err
		}
		if err := h.ensureWriteAccess(c.Request().Context(), userID, repo); err != nil {
			return err
		}
	} else if err := h.ensureReadAccess(c, repo); err != nil {
		return err
	}

	contentType := "application/x-git-upload-pack-advertisement"
	if service == transport.ReceivePackService.String() {
		contentType = "application/x-git-receive-pack-advertisement"
	}
	c.Response().Header().Set("Content-Type", contentType)

	return advertiseRefs(c.Response().Writer, c.Request().Context(), repo.DiskPath, service)
}

func advertiseRefs(w http.ResponseWriter, ctx context.Context, repoPath, service string) error {
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return err
	}
	root := filepath.Dir(abs)
	name := filepath.Base(abs)

	loader := server.NewFilesystemLoader(osfs.New(root))
	svr := server.NewServer(loader)
	ep, err := transport.NewEndpoint(name)
	if err != nil {
		return fmt.Errorf("transport endpoint: %w", err)
	}

	var sess transport.Session
	switch service {
	case transport.UploadPackService.String():
		sess, err = svr.NewUploadPackSession(ep, nil)
	case transport.ReceivePackService.String():
		sess, err = svr.NewReceivePackSession(ep, nil)
	default:
		return fmt.Errorf("unsupported service: %s", service)
	}
	if err != nil {
		return err
	}
	defer func() { _ = sess.Close() }()

	refs, err := sess.AdvertisedReferencesContext(ctx)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, refs)
	return err
}

// UploadPack handles POST /:owner/:repo.git/git-upload-pack
func (h *GitHTTPHandler) UploadPack(c echo.Context) error {
	repo, err := h.resolveRepo(c)
	if err != nil {
		return err
	}
	if err := h.ensureReadAccess(c, repo); err != nil {
		return err
	}
	if err := infragit.ServeUploadPack(c.Response().Writer, c.Request(), repo.DiskPath); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}
	return nil
}

// ReceivePack handles POST /:owner/:repo.git/git-receive-pack
func (h *GitHTTPHandler) ReceivePack(c echo.Context) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	repo, err := h.resolveRepo(c)
	if err != nil {
		return err
	}

	if err := h.ensureWriteAccess(c.Request().Context(), userID, repo); err != nil {
		return err
	}

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid request body"})
	}

	if err := h.rejectProtectedUpdates(c.Request().Context(), userID, repo, body); err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}

	c.Request().Body = io.NopCloser(bytes.NewReader(body))
	if err := infragit.ServeReceivePack(c.Response().Writer, c.Request(), repo.DiskPath); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}
	return nil
}

const gitWWWAuthenticateHeader = `Basic realm="OpenGit"`

func (h *GitHTTPHandler) ensureReadAccess(c echo.Context, repo *ResolvedGitRepository) error {
	if repo.Visibility != entity.VisibilityPrivate {
		return nil
	}
	userID := middleware.UserIDFromContext(c)
	if userID == 0 {
		c.Response().Header().Set("WWW-Authenticate", gitWWWAuthenticateHeader)
		return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
	}
	return h.ensureReadAccessForUser(c.Request().Context(), userID, repo)
}

func (h *GitHTTPHandler) ensureReadAccessForUser(ctx context.Context, userID int64, repo *ResolvedGitRepository) error {
	if repo.OwnerID != 0 && repo.OwnerID == userID {
		return nil
	}
	if h.memberships == nil {
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "read access required"})
	}
	ok, err := h.memberships.HasReadAccess(ctx, userID, repo.OrganizationID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to check permissions"})
	}
	if !ok {
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "read access required"})
	}
	return nil
}

func (h *GitHTTPHandler) ensureWriteAccess(ctx context.Context, userID int64, repo *ResolvedGitRepository) error {
	if repo.OwnerID != 0 && repo.OwnerID == userID {
		return nil
	}
	if h.memberships == nil {
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "write access required"})
	}
	ok, err := h.memberships.HasWriteAccess(ctx, userID, repo.OrganizationID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "failed to check permissions"})
	}
	if !ok {
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "write access required"})
	}
	return nil
}

func (h *GitHTTPHandler) rejectProtectedUpdates(ctx context.Context, userID int64, repo *ResolvedGitRepository, body []byte) error {
	if h.protections == nil {
		return nil
	}

	req := packp.NewReferenceUpdateRequest()
	if err := req.Decode(bytes.NewReader(body)); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid receive-pack request"})
	}

	grepo, err := gogit.PlainOpen(repo.DiskPath)
	if err != nil {
		return err
	}

	for _, cmd := range req.Commands {
		branch, ok := branchFromRef(cmd.Name.String())
		if !ok {
			continue
		}
		rule, err := h.protections.GetBranchProtection(ctx, repo.ID, branch)
		if err != nil {
			return err
		}
		if !rule.Protected {
			continue
		}
		if h.adminBypassesProtection(ctx, userID, repo.OrganizationID, rule) {
			continue
		}
		if err := h.rejectProtectedDeletion(rule, cmd.New); err != nil {
			return err
		}
		if cmd.New == plumbing.ZeroHash {
			continue
		}
		if isForcePush(grepo, cmd.Old, cmd.New) {
			if err := h.rejectProtectedForcePush(rule, branch); err != nil {
				return err
			}
			continue
		}
		if err := h.rejectProtectedDirectPush(branch); err != nil {
			return err
		}
	}
	return nil
}

func (h *GitHTTPHandler) adminBypassesProtection(
	ctx context.Context,
	userID int64,
	organizationID uuid.UUID,
	rule GitBranchProtectionRule,
) bool {
	if rule.EnforceAdmins {
		return false
	}
	return h.isOrgAdmin(ctx, userID, organizationID)
}

func (h *GitHTTPHandler) isOrgAdmin(ctx context.Context, userID int64, organizationID uuid.UUID) bool {
	if userID == 0 || organizationID == uuid.Nil || h.memberships == nil {
		return false
	}
	role, err := h.memberships.GetRole(ctx, userID, organizationID)
	return err == nil && role == entity.RoleAdmin
}

func (h *GitHTTPHandler) rejectProtectedDirectPush(branch string) error {
	return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{
		"message": fmt.Sprintf("direct push is not allowed on protected branch: %s", branch),
	})
}

func (h *GitHTTPHandler) rejectProtectedForcePush(rule GitBranchProtectionRule, branch string) error {
	if rule.AllowForcePushes {
		return nil
	}
	return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{
		"message": fmt.Sprintf("force-push is not allowed on protected branch: %s", branch),
	})
}

func (h *GitHTTPHandler) rejectProtectedDeletion(rule GitBranchProtectionRule, newRef plumbing.Hash) error {
	if newRef != plumbing.ZeroHash {
		return nil
	}
	if rule.AllowDeletions {
		return nil
	}
	return echo.NewHTTPError(http.StatusForbidden, map[string]string{
		"message": "deletion of protected branch not allowed",
	})
}

func branchFromRef(ref string) (string, bool) {
	const prefix = "refs/heads/"
	if !strings.HasPrefix(ref, prefix) {
		return "", false
	}
	return strings.TrimPrefix(ref, prefix), true
}

func isForcePush(repo *gogit.Repository, oldHash, newHash plumbing.Hash) bool {
	if oldHash == plumbing.ZeroHash || newHash == plumbing.ZeroHash {
		return false
	}

	oldCommit, err := repo.CommitObject(oldHash)
	if err != nil {
		return true
	}
	newCommit, err := repo.CommitObject(newHash)
	if err != nil {
		return true
	}

	mergeBases, err := oldCommit.MergeBase(newCommit)
	if err != nil || len(mergeBases) == 0 {
		return true
	}
	for _, mergeBase := range mergeBases {
		if mergeBase.Hash == oldHash {
			return false
		}
	}
	return true
}
