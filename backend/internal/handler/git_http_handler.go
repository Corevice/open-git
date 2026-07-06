package handler

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/pktline"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/open-git/backend/internal/domain/entity"
	infragit "github.com/open-git/backend/internal/infrastructure/git"
	"github.com/open-git/backend/internal/middleware"
	repo "github.com/open-git/backend/internal/repository"
	obs "github.com/open-git/backend/observability"
)

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

// GitMembershipAccess checks organization read/write permission.
type GitMembershipAccess interface {
	HasReadAccess(ctx context.Context, userID int64, organizationID uuid.UUID) (bool, error)
	HasWriteAccess(ctx context.Context, userID int64, organizationID uuid.UUID) (bool, error)
}

// GitBranchProtectionStore reports whether a branch matches a protection rule.
type GitBranchProtectionStore interface {
	IsBranchProtected(ctx context.Context, repositoryID uuid.UUID, branch string) (bool, error)
}

// GitHTTPHandler serves Git Smart HTTP protocol endpoints.
type GitHTTPHandler struct {
	gitRoot       string
	resolver      GitRepositoryResolver
	memberships   GitMembershipAccess
	protections   GitBranchProtectionStore
	collaborators repo.IRepositoryCollaboratorRepository
	authRequired  echo.MiddlewareFunc
	// onPush, when set, is invoked once per branch updated by a successful
	// receive-pack (CI trigger). It must not block or fail the push.
	onPush func(ctx context.Context, repo *ResolvedGitRepository, branch, newSHA string, userID int64)
}

// SetPushListener installs the post-receive callback (e.g. the CI trigger).
func (h *GitHTTPHandler) SetPushListener(fn func(ctx context.Context, repo *ResolvedGitRepository, branch, newSHA string, userID int64)) {
	h.onPush = fn
}

func NewGitHTTPHandler(
	gitRoot string,
	resolver GitRepositoryResolver,
	memberships GitMembershipAccess,
	protections GitBranchProtectionStore,
	collaborators repo.IRepositoryCollaboratorRepository,
	authRequired echo.MiddlewareFunc,
) *GitHTTPHandler {
	return &GitHTTPHandler{
		gitRoot:       gitRoot,
		resolver:      resolver,
		memberships:   memberships,
		protections:   protections,
		collaborators: collaborators,
		authRequired:  authRequired,
	}
}

func (h *GitHTTPHandler) RegisterRoutes(e *echo.Echo) {
	// NOTE: Echo cannot mix a param and a literal in one path segment
	// (":repo.git" would name the param "repo.git" and leave c.Param("repo")
	// empty). Capture the whole segment as :repo and strip the .git suffix in
	// the handler instead.
	e.GET("/:owner/:repo/info/refs", h.InfoRefs)
	e.POST("/:owner/:repo/git-upload-pack", h.UploadPack)
	if h.authRequired != nil {
		e.POST("/:owner/:repo/git-receive-pack", h.ReceivePack, h.authRequired)
	} else {
		e.POST("/:owner/:repo/git-receive-pack", h.ReceivePack)
	}
}

func (h *GitHTTPHandler) repoPath(owner, repo string) string {
	return filepath.Join(h.gitRoot, owner, repo+".git")
}

func (h *GitHTTPHandler) resolveRepo(c echo.Context) (*ResolvedGitRepository, error) {
	owner := c.Param("owner")
	// The route param captures the full last segment (e.g. "demo.git"); strip
	// the .git suffix once and use the cleaned name everywhere.
	repoName := strings.TrimSuffix(c.Param("repo"), ".git")
	if h.resolver != nil {
		repo, err := h.resolver.Resolve(c.Request().Context(), owner, repoName)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
		if repo.DiskPath == "" {
			repo.DiskPath = h.repoPath(owner, repoName)
		}
		return repo, nil
	}

	path := h.repoPath(owner, repoName)
	if _, err := os.Stat(path); err != nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}
	return &ResolvedGitRepository{DiskPath: path}, nil
}

// InfoRefs handles GET /:owner/:repo.git/info/refs?service=
func (h *GitHTTPHandler) InfoRefs(c echo.Context) error {
	service := c.QueryParam("service")
	if service != transport.UploadPackServiceName && service != transport.ReceivePackServiceName {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid service"})
	}

	repo, err := h.resolveRepo(c)
	if err != nil {
		return err
	}
	if service == transport.ReceivePackServiceName {
		// info/refs uses optional auth, so an unauthenticated push probe lands
		// here with no user. Return a Basic-auth challenge (not a bare 401) so
		// git retries with credentials instead of failing immediately.
		userID := middleware.UserIDFromContext(c)
		if userID == 0 {
			c.Response().Header().Set("WWW-Authenticate", gitWWWAuthenticateHeader)
			return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
		}
		if err := h.ensureWriteAccess(c.Request().Context(), userID, repo); err != nil {
			return err
		}
	} else if err := h.ensureReadAccess(c, repo); err != nil {
		return err
	}

	contentType := "application/x-git-upload-pack-advertisement"
	if service == transport.ReceivePackServiceName {
		contentType = "application/x-git-receive-pack-advertisement"
	}
	c.Response().Header().Set("Content-Type", contentType)

	return advertiseRefs(c.Response().Writer, c.Request().Context(), repo.DiskPath, service)
}

func advertiseRefs(w http.ResponseWriter, ctx context.Context, repoPath, service string) error {
	// Smart HTTP requires a service announcement header preceding the refs.
	enc := pktline.NewEncoder(w)
	if err := enc.EncodeString("# service=" + service + "\n"); err != nil {
		return err
	}
	if err := enc.Flush(); err != nil {
		return err
	}
	// The advertisement body itself comes from the git binary; go-git's pure-Go
	// server cannot ingest the thin packs real clients push, so all pack IO goes
	// through git to keep advertisement and pack exchange consistent.
	return infragit.AdvertiseRefsCLI(ctx, w, repoPath, service)
}

// UploadPack handles POST /:owner/:repo.git/git-upload-pack
func (h *GitHTTPHandler) UploadPack(c echo.Context) error {
	result := "success"
	defer func() { obs.RecordGitOperation("upload_pack", result) }()

	repo, err := h.resolveRepo(c)
	if err != nil {
		result = "error"
		return err
	}
	if err := h.ensureReadAccess(c, repo); err != nil {
		result = "error"
		return err
	}

	body, err := gitRequestBody(c.Request())
	if err != nil {
		result = "error"
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid request body"})
	}
	defer func() { _ = body.Close() }()

	c.Response().Header().Set("Content-Type", "application/x-git-upload-pack-result")
	if err := infragit.ServePackCLI(c.Request().Context(), c.Response().Writer, body, repo.DiskPath, transport.UploadPackServiceName); err != nil {
		result = "error"
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}
	return nil
}

// ReceivePack handles POST /:owner/:repo.git/git-receive-pack
func (h *GitHTTPHandler) ReceivePack(c echo.Context) error {
	result := "success"
	defer func() { obs.RecordGitOperation("receive_pack", result) }()

	userID, err := middleware.GetUserID(c)
	if err != nil {
		result = "error"
		return err
	}

	repo, err := h.resolveRepo(c)
	if err != nil {
		result = "error"
		return err
	}

	if err := h.ensureWriteAccess(c.Request().Context(), userID, repo); err != nil {
		result = "error"
		return err
	}

	reader, err := gitRequestBody(c.Request())
	if err != nil {
		result = "error"
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid request body"})
	}
	body, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil {
		result = "error"
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid request body"})
	}

	if err := h.rejectProtectedForcePush(c.Request().Context(), repo, body); err != nil {
		result = "error"
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}

	c.Response().Header().Set("Content-Type", "application/x-git-receive-pack-result")
	if err := infragit.ServePackCLI(c.Request().Context(), c.Response().Writer, bytes.NewReader(body), repo.DiskPath, transport.ReceivePackServiceName); err != nil {
		result = "error"
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}

	h.notifyPush(repo, body, userID)
	return nil
}

// gitRequestBody returns the smart-HTTP request body, transparently
// decompressing it when the client sent Content-Encoding: gzip (git may gzip
// upload-pack negotiation requests).
func gitRequestBody(r *http.Request) (io.ReadCloser, error) {
	if strings.EqualFold(r.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			return nil, err
		}
		return gz, nil
	}
	return r.Body, nil
}

// notifyPush reports each branch updated by the receive-pack request to the
// push listener. Deletions (new hash = zero) are skipped. Failures here must
// never fail the push, so this is best-effort by construction.
func (h *GitHTTPHandler) notifyPush(repo *ResolvedGitRepository, body []byte, userID int64) {
	if h.onPush == nil {
		return
	}
	req := packp.NewReferenceUpdateRequest()
	if err := req.Decode(bytes.NewReader(body)); err != nil {
		return
	}
	for _, cmd := range req.Commands {
		if cmd.New == plumbing.ZeroHash {
			continue
		}
		branch, ok := branchFromRef(cmd.Name.String())
		if !ok {
			continue
		}
		h.onPush(context.Background(), repo, branch, cmd.New.String(), userID)
	}
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
		if h.collaborators != nil {
			perm, err := h.collaborators.GetPermission(ctx, repo.ID, middleware.Int64ToUUID(userID))
			if err == nil && perm != "" {
				return nil
			}
		}
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
		if h.collaborators != nil {
			perm, err := h.collaborators.GetPermission(ctx, repo.ID, middleware.Int64ToUUID(userID))
			if err == nil && (perm == entity.CollaboratorPermWrite || perm == entity.CollaboratorPermAdmin) {
				return nil
			}
		}
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "write access required"})
	}
	return nil
}

func (h *GitHTTPHandler) rejectProtectedForcePush(ctx context.Context, repo *ResolvedGitRepository, body []byte) error {
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
		protected, err := h.protections.IsBranchProtected(ctx, repo.ID, branch)
		if err != nil {
			return err
		}
		if !protected {
			continue
		}
		forced, ferr := isForcePush(grepo, cmd.Old, cmd.New)
		if ferr != nil {
			// We cannot prove this is a force-push (e.g. the packfile that
			// carries the new commit hasn't been processed yet at this point of
			// receive-pack, or the bare repo is shallow). Fall back to allowing
			// the update so that legitimate non-force pushes are not rejected.
			continue
		}
		if forced {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{
				"message": fmt.Sprintf("force-push is not allowed on protected branch: %s", branch),
			})
		}
	}
	return nil
}

func branchFromRef(ref string) (string, bool) {
	const prefix = "refs/heads/"
	if !strings.HasPrefix(ref, prefix) {
		return "", false
	}
	return strings.TrimPrefix(ref, prefix), true
}

// isForcePush reports whether updating a ref from oldHash to newHash is a
// non-fast-forward (force) push. The boolean is only meaningful when err is
// nil; when err is non-nil the caller cannot decide and should not treat the
// update as a force-push (e.g. the receive-pack packfile has not yet been
// written to the bare repo, or the repo is a shallow clone missing ancestors).
func isForcePush(repo *gogit.Repository, oldHash, newHash plumbing.Hash) (bool, error) {
	if oldHash == plumbing.ZeroHash || newHash == plumbing.ZeroHash {
		return false, nil
	}
	oldCommit, err := repo.CommitObject(oldHash)
	if err != nil {
		return false, fmt.Errorf("look up old commit %s: %w", oldHash, err)
	}
	newCommit, err := repo.CommitObject(newHash)
	if err != nil {
		return false, fmt.Errorf("look up new commit %s: %w", newHash, err)
	}
	// A non-force (fast-forward) update requires the old commit to be an
	// ancestor of the new one, i.e. the merge base equals the old commit.
	bases, err := oldCommit.MergeBase(newCommit)
	if err != nil {
		return false, fmt.Errorf("compute merge base: %w", err)
	}
	for _, base := range bases {
		if base.Hash == oldHash {
			return false, nil
		}
	}
	return true, nil
}
