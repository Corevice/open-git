package handler

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/labstack/echo/v4"
)

// GitRepositoryRef is the minimal view of a repository the git HTTP handler needs.
type GitRepositoryRef struct {
	ID         int64
	OwnerLogin string
	Name       string
	StoragePath string
	Private    bool
}

// GitBranchProtection mirrors the branch_protections row needed to decide force-push rejection.
type GitBranchProtection struct {
	Pattern          string
	AllowForcePushes bool
}

// GitRepositoryFinder resolves an owner/name pair to a repository.
type GitRepositoryFinder interface {
	FindByOwnerAndName(ctx context.Context, owner, name string) (*GitRepositoryRef, error)
}

// GitPermissionChecker reports whether the user has read or write access on a repository.
type GitPermissionChecker interface {
	HasRead(ctx context.Context, userID int64, repoID int64) (bool, error)
	HasWrite(ctx context.Context, userID int64, repoID int64) (bool, error)
}

// GitBranchProtectionFinder returns protection rules covering a ref name.
type GitBranchProtectionFinder interface {
	FindForRef(ctx context.Context, repoID int64, ref string) (*GitBranchProtection, error)
}

// GitServer abstracts the on-disk git operations so handlers stay test-friendly.
type GitServer interface {
	AdvertiseRefs(w http.ResponseWriter, repoPath, service string) error
	ServeUploadPack(w http.ResponseWriter, r *http.Request, repoPath string) error
	ServeReceivePack(w http.ResponseWriter, r *http.Request, repoPath string) error
	IsForcePush(repoPath, ref, oldOID, newOID string) (bool, error)
}

// GitUserResolver extracts an authenticated user id from the request (0 if anonymous).
type GitUserResolver func(c echo.Context) int64

// GitHTTPHandler implements the Git Smart HTTP protocol endpoints.
type GitHTTPHandler struct {
	repos       GitRepositoryFinder
	permissions GitPermissionChecker
	protections GitBranchProtectionFinder
	git         GitServer
	resolveUser GitUserResolver
}

// NewGitHTTPHandler wires the handler dependencies.
func NewGitHTTPHandler(
	repos GitRepositoryFinder,
	permissions GitPermissionChecker,
	protections GitBranchProtectionFinder,
	git GitServer,
	resolveUser GitUserResolver,
) *GitHTTPHandler {
	if resolveUser == nil {
		resolveUser = func(c echo.Context) int64 {
			v := c.Get("user_id")
			if id, ok := v.(int64); ok {
				return id
			}
			return 0
		}
	}
	return &GitHTTPHandler{
		repos:       repos,
		permissions: permissions,
		protections: protections,
		git:         git,
		resolveUser: resolveUser,
	}
}

// RegisterRoutes mounts the git smart-HTTP routes onto the provided group.
// URLs follow the conventional /:owner/:repo.git/{info/refs,git-upload-pack,git-receive-pack} layout.
func (h *GitHTTPHandler) RegisterRoutes(e *echo.Echo) {
	e.GET("/:owner/:repo/info/refs", h.InfoRefs)
	e.POST("/:owner/:repo/git-upload-pack", h.UploadPack)
	e.POST("/:owner/:repo/git-receive-pack", h.ReceivePack)
}

// InfoRefs implements `GET /:owner/:repo.git/info/refs?service=...`.
func (h *GitHTTPHandler) InfoRefs(c echo.Context) error {
	service := c.QueryParam("service")
	if service != "git-upload-pack" && service != "git-receive-pack" {
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "unsupported service"})
	}
	shortService := strings.TrimPrefix(service, "git-")

	repo, err := h.lookupRepo(c)
	if err != nil {
		return err
	}

	userID := h.resolveUser(c)

	// receive-pack discovery requires write authentication, upload-pack requires read.
	if shortService == "receive-pack" {
		if userID == 0 {
			return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "authentication required"})
		}
		allowed, err := h.permissions.HasWrite(c.Request().Context(), userID, repo.ID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "permission lookup failed"})
		}
		if !allowed {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
		}
	} else if repo.Private {
		if err := h.ensureRead(c, userID, repo); err != nil {
			return err
		}
	}

	if err := h.git.AdvertiseRefs(c.Response().Writer, repo.StoragePath, shortService); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "advertise refs failed"})
	}
	return nil
}

// UploadPack implements `POST /:owner/:repo.git/git-upload-pack` (clone / fetch).
func (h *GitHTTPHandler) UploadPack(c echo.Context) error {
	repo, err := h.lookupRepo(c)
	if err != nil {
		return err
	}

	if repo.Private {
		userID := h.resolveUser(c)
		if err := h.ensureRead(c, userID, repo); err != nil {
			return err
		}
	}

	if err := h.git.ServeUploadPack(c.Response().Writer, c.Request(), repo.StoragePath); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "upload-pack failed"})
	}
	return nil
}

// ReceivePack implements `POST /:owner/:repo.git/git-receive-pack` (push).
func (h *GitHTTPHandler) ReceivePack(c echo.Context) error {
	userID := h.resolveUser(c)
	if userID == 0 {
		return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "authentication required"})
	}

	repo, err := h.lookupRepo(c)
	if err != nil {
		return err
	}

	allowed, err := h.permissions.HasWrite(c.Request().Context(), userID, repo.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "permission lookup failed"})
	}
	if !allowed {
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"message": "write access required"})
	}

	body, err := readReceivePackBody(c.Request())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid receive-pack body"})
	}

	commands, err := parseReceivePackCommands(body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"message": "invalid pkt-line"})
	}

	for _, cmd := range commands {
		protection, err := h.protections.FindForRef(c.Request().Context(), repo.ID, cmd.Ref)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "protection lookup failed"})
		}
		if protection == nil || protection.AllowForcePushes {
			continue
		}

		forced, err := h.git.IsForcePush(repo.StoragePath, cmd.Ref, cmd.OldOID, cmd.NewOID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "force-push check failed"})
		}
		if forced {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, map[string]string{
				"message": fmt.Sprintf("protected branch %q does not allow force pushes", cmd.Ref),
			})
		}
	}

	// Replay the consumed body so the git binary sees the full request.
	c.Request().Body = io.NopCloser(bytes.NewReader(body))
	c.Request().ContentLength = int64(len(body))
	c.Request().Header.Del("Content-Encoding")

	if err := h.git.ServeReceivePack(c.Response().Writer, c.Request(), repo.StoragePath); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "receive-pack failed"})
	}
	return nil
}

func (h *GitHTTPHandler) lookupRepo(c echo.Context) (*GitRepositoryRef, error) {
	owner := c.Param("owner")
	repoParam := strings.TrimSuffix(c.Param("repo"), ".git")
	if owner == "" || repoParam == "" {
		return nil, echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}
	repo, err := h.repos.FindByOwnerAndName(c.Request().Context(), owner, repoParam)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "repository lookup failed"})
	}
	if repo == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}
	if repo.StoragePath == "" {
		repo.StoragePath = path.Join(owner, repoParam+".git")
	}
	return repo, nil
}

func (h *GitHTTPHandler) ensureRead(c echo.Context, userID int64, repo *GitRepositoryRef) error {
	if userID == 0 {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}
	ok, err := h.permissions.HasRead(c.Request().Context(), userID, repo.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"message": "permission lookup failed"})
	}
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"message": "Not Found"})
	}
	return nil
}

type receivePackCommand struct {
	OldOID string
	NewOID string
	Ref    string
}

func readReceivePackBody(r *http.Request) ([]byte, error) {
	var reader io.ReadCloser = r.Body
	if strings.EqualFold(r.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			return nil, err
		}
		reader = gz
		defer reader.Close()
	}
	return io.ReadAll(reader)
}

// parseReceivePackCommands parses the leading pkt-line commands of a receive-pack body.
// Each command line takes the form: "<old-oid> <new-oid> <ref>\0<capabilities>\n".
// Parsing stops at the flush pkt ("0000"); any trailing pack data is left for the git binary.
func parseReceivePackCommands(body []byte) ([]receivePackCommand, error) {
	commands := make([]receivePackCommand, 0, 1)
	offset := 0
	for offset+4 <= len(body) {
		lenField := string(body[offset : offset+4])
		if lenField == "0000" {
			break
		}
		size, err := parseHex16(lenField)
		if err != nil {
			return nil, err
		}
		if size < 4 || offset+size > len(body) {
			return nil, fmt.Errorf("invalid pkt-line length %d", size)
		}
		payload := body[offset+4 : offset+size]
		offset += size

		// Strip trailing newline.
		payload = bytes.TrimRight(payload, "\n")
		// Capabilities are after a NUL byte on the first command.
		if nul := bytes.IndexByte(payload, 0); nul >= 0 {
			payload = payload[:nul]
		}
		parts := strings.SplitN(string(payload), " ", 3)
		if len(parts) != 3 {
			continue
		}
		commands = append(commands, receivePackCommand{
			OldOID: parts[0],
			NewOID: parts[1],
			Ref:    parts[2],
		})
	}
	return commands, nil
}

func parseHex16(s string) (int, error) {
	var n int
	for _, ch := range s {
		var v int
		switch {
		case ch >= '0' && ch <= '9':
			v = int(ch - '0')
		case ch >= 'a' && ch <= 'f':
			v = int(ch-'a') + 10
		case ch >= 'A' && ch <= 'F':
			v = int(ch-'A') + 10
		default:
			return 0, fmt.Errorf("invalid hex digit %q", ch)
		}
		n = (n << 4) | v
	}
	return n, nil
}
