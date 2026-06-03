package handler

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"

	gitinfra "github.com/Corevice/open-git/backend/internal/infrastructure/git"
	"github.com/Corevice/open-git/backend/internal/middleware"
)

// GitTokenStore validates bearer tokens for git operations.
type GitTokenStore interface {
	// ValidateWriteToken returns the user ID if the raw token has write ("repo") scope, or an error.
	ValidateWriteToken(ctx context.Context, rawToken string) (userID string, err error)
}

// GitHTTPHandler serves Git Smart HTTP protocol endpoints.
type GitHTTPHandler struct {
	db       *sql.DB
	repoRoot string
	tokens   GitTokenStore
}

// NewGitHTTPHandler creates a new GitHTTPHandler.
func NewGitHTTPHandler(db *sql.DB, repoRoot string, tokens GitTokenStore) *GitHTTPHandler {
	return &GitHTTPHandler{db: db, repoRoot: repoRoot, tokens: tokens}
}

// RegisterGitRoutes registers git Smart HTTP routes on e.
// The repoRoot is the filesystem directory where bare repos are stored.
func RegisterGitRoutes(e *echo.Echo, db *sql.DB, repoRoot string, tokens GitTokenStore) {
	h := NewGitHTTPHandler(db, repoRoot, tokens)
	e.GET("/:owner/:repo/info/refs", h.InfoRefs)
	e.POST("/:owner/:repo/git-upload-pack", h.UploadPack)
	e.POST("/:owner/:repo/git-receive-pack", h.ReceivePack)
}

func (h *GitHTTPHandler) repoPath(owner, repo string) string {
	repo = strings.TrimSuffix(repo, ".git")
	return filepath.Join(h.repoRoot, owner, repo+".git")
}

// InfoRefs handles GET /:owner/:repo.git/info/refs?service=git-{upload,receive}-pack
// and returns a service advertisement with the correct Content-Type.
func (h *GitHTTPHandler) InfoRefs(c echo.Context) error {
	service := c.QueryParam("service")
	if service != "git-upload-pack" && service != "git-receive-pack" {
		return echo.NewHTTPError(http.StatusForbidden, "unsupported service")
	}

	// Strip "git-" prefix to get the subcommand (upload-pack / receive-pack).
	subcmd := strings.TrimPrefix(service, "git-")
	ct := fmt.Sprintf("application/x-git-%s-advertisement", subcmd)

	c.Response().Header().Set("Content-Type", ct)
	c.Response().Header().Set("Cache-Control", "no-cache")

	// Write the pkt-line service announcement.
	header := fmt.Sprintf("# service=%s\n", service)
	writePktLine(c.Response(), header)
	writeFlushPkt(c.Response())

	repoPath := h.repoPath(c.Param("owner"), c.Param("repo"))
	cmd := exec.CommandContext(c.Request().Context(),
		"git", subcmd, "--stateless-rpc", "--advertise-refs", repoPath)
	out, _ := cmd.Output()
	if len(out) > 0 {
		_, _ = c.Response().Write(out)
	}
	return nil
}

// UploadPack handles POST /:owner/:repo.git/git-upload-pack (clone/fetch).
func (h *GitHTTPHandler) UploadPack(c echo.Context) error {
	repoPath := h.repoPath(c.Param("owner"), c.Param("repo"))
	gitinfra.ServeUploadPack(c.Response(), c.Request(), repoPath)
	return nil
}

// ReceivePack handles POST /:owner/:repo.git/git-receive-pack (push).
// It requires write permission and rejects force-pushes to protected branches.
func (h *GitHTTPHandler) ReceivePack(c echo.Context) error {
	// Require authentication with write scope.
	if err := h.requireWriteAuth(c); err != nil {
		return err
	}

	owner := c.Param("owner")
	repoName := strings.TrimSuffix(c.Param("repo"), ".git")

	// Buffer body for inspection and re-use.
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to read request body")
	}

	// Enforce branch protection: reject force-push to protected branches.
	if err := h.checkForcePush(c.Request().Context(), owner, repoName, body); err != nil {
		return err
	}

	// Proxy to git receive-pack.
	repoPath := h.repoPath(owner, repoName)
	r := c.Request().Clone(c.Request().Context())
	r.Body = io.NopCloser(bytes.NewReader(body))
	gitinfra.ServeReceivePack(c.Response(), r, repoPath)
	return nil
}

// requireWriteAuth validates the bearer token and checks for write ("repo") scope.
func (h *GitHTTPHandler) requireWriteAuth(c echo.Context) error {
	raw, ok := middleware.BearerToken(c.Request().Header.Get("Authorization"))
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "authentication required"})
	}
	if h.tokens == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "authentication required"})
	}
	_, err := h.tokens.ValidateWriteToken(c.Request().Context(), raw)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"message": "invalid or insufficient token"})
	}
	return nil
}

// checkForcePush parses the receive-pack body and returns 422 if a force-push
// targets a protected branch.
func (h *GitHTTPHandler) checkForcePush(ctx context.Context, owner, repoName string, body []byte) error {
	updates := parsePktLineUpdates(body)
	for _, u := range updates {
		if isZeroSHA(u.oldSHA) {
			continue // New branch — not a force-push.
		}
		branch := refToBranch(u.ref)
		if branch == "" {
			continue
		}
		protected, err := h.isBranchProtected(ctx, owner, repoName, branch)
		if err != nil || !protected {
			continue
		}
		return echo.NewHTTPError(
			http.StatusUnprocessableEntity,
			map[string]string{
				"message": fmt.Sprintf("force-push to protected branch %q is not allowed", branch),
			},
		)
	}
	return nil
}

// refUpdate holds old/new SHA and ref name from a git receive-pack command.
type refUpdate struct {
	oldSHA string
	newSHA string
	ref    string
}

// parsePktLineUpdates decodes the pkt-line stream in a git-receive-pack body.
func parsePktLineUpdates(data []byte) []refUpdate {
	var updates []refUpdate
	r := bytes.NewReader(data)
	for {
		lenBuf := make([]byte, 4)
		if _, err := io.ReadFull(r, lenBuf); err != nil {
			break
		}
		var pktLen int
		if _, err := fmt.Sscanf(string(lenBuf), "%04x", &pktLen); err != nil {
			break
		}
		if pktLen == 0 {
			break // flush packet
		}
		if pktLen <= 4 {
			break
		}
		payload := make([]byte, pktLen-4)
		if _, err := io.ReadFull(r, payload); err != nil {
			break
		}
		line := strings.TrimRight(string(payload), "\n")
		// Strip capability list after NUL byte (present on first command only).
		if nul := strings.IndexByte(line, 0); nul != -1 {
			line = line[:nul]
		}
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			updates = append(updates, refUpdate{
				oldSHA: fields[0],
				newSHA: fields[1],
				ref:    fields[2],
			})
		}
	}
	return updates
}

func isZeroSHA(sha string) bool {
	if len(sha) != 40 {
		return false
	}
	for _, c := range sha {
		if c != '0' {
			return false
		}
	}
	return true
}

func refToBranch(ref string) string {
	const prefix = "refs/heads/"
	if strings.HasPrefix(ref, prefix) {
		return strings.TrimPrefix(ref, prefix)
	}
	return ""
}

// isBranchProtected checks whether the named branch has a protection rule.
func (h *GitHTTPHandler) isBranchProtected(ctx context.Context, owner, repoName, branch string) (bool, error) {
	if h.db == nil {
		return false, nil
	}
	query := `
		SELECT COUNT(*) FROM branch_protections bp
		JOIN repositories r ON r.id = bp.repository_id
		JOIN users u ON u.id = r.owner_id
		WHERE u.login = ? AND r.name = ? AND bp.pattern = ?
	`
	var count int
	err := h.db.QueryRowContext(ctx, query, owner, repoName, branch).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// writePktLine writes a git protocol pkt-line to w.
func writePktLine(w io.Writer, s string) {
	fmt.Fprintf(w, "%04x%s", len(s)+4, s)
}

// writeFlushPkt writes a git protocol flush packet to w.
func writeFlushPkt(w io.Writer) {
	fmt.Fprint(w, "0000")
}
