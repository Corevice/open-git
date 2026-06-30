package ssh

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	gossh "github.com/gliderlabs/ssh"
	"github.com/google/uuid"
	cryptossh "golang.org/x/crypto/ssh"

	"github.com/open-git/backend/internal/repository"
)

var gitSSHCommandPattern = regexp.MustCompile(`^git-(upload-pack|receive-pack)\s+'?/?([^/'\s]+)/([^'\s]+?)'?\.?$`)

type GitSSHResolver interface {
	Resolve(ctx context.Context, ownerLogin, repoName string) (diskPath string, ownerID uuid.UUID, err error)
}

type SSHServer struct {
	gitRoot  string
	keyStore repository.ISSHKeyStore
	resolver GitSSHResolver
	hostKey  gossh.Signer
	server   *gossh.Server
}

type gitSSHCommand struct {
	packType   string
	ownerLogin string
	repoName   string
}


func NewSSHServer(
	gitRoot string,
	keyStore repository.ISSHKeyStore,
	resolver GitSSHResolver,
	hostKey gossh.Signer,
) *SSHServer {
	return &SSHServer{
		gitRoot:  gitRoot,
		keyStore: keyStore,
		resolver: resolver,
		hostKey:  hostKey,
	}
}

func parseGitSSHCommand(raw string) (gitSSHCommand, error) {
	raw = strings.TrimSpace(raw)
	matches := gitSSHCommandPattern.FindStringSubmatch(raw)
	if matches == nil {
		return gitSSHCommand{}, fmt.Errorf("invalid git ssh command: %q", raw)
	}

	repoName := strings.TrimSuffix(matches[3], ".git")
	return gitSSHCommand{
		packType:   matches[1],
		ownerLogin: matches[2],
		repoName:   repoName,
	}, nil
}

func (h *SSHServer) Start(addr string) error {
	h.server = &gossh.Server{
		Addr:             addr,
		PublicKeyHandler: h.authenticateKey,
		Handler:          h.handleSession,
	}
	if h.hostKey != nil {
		h.server.AddHostKey(h.hostKey)
	}
	return h.server.ListenAndServe()
}

func (h *SSHServer) Close() error {
	if h.server == nil {
		return nil
	}
	return h.server.Close()
}

// authUserIDContextKey holds the uuid.UUID of the user that owns the
// authenticated SSH key, so the session handler can enforce authorization.
const authUserIDContextKey = "auth_user_id"

func (h *SSHServer) authenticateKey(ctx gossh.Context, key gossh.PublicKey) bool {
	fingerprint := cryptossh.FingerprintSHA256(key)
	stored, err := h.keyStore.FindByFingerprint(ctx, fingerprint)
	if err != nil || stored == nil {
		return false
	}
	// Remember which user this key belongs to for per-repo authorization.
	ctx.SetValue(authUserIDContextKey, stored.UserID)
	return true
}

func (h *SSHServer) handleSession(s gossh.Session) {
	ctx := s.Context()

	parsed, err := parseGitSSHCommand(strings.Join(s.Command(), " "))
	if err != nil {
		_, _ = fmt.Fprintf(s, "ERR %v\n", err)
		s.Exit(1)
		return
	}

	diskPath, ownerID, err := h.resolver.Resolve(ctx, parsed.ownerLogin, parsed.repoName)
	if err != nil {
		_, _ = fmt.Fprintf(s, "ERR %v\n", err)
		s.Exit(1)
		return
	}
	if diskPath == "" {
		diskPath = filepath.Join(h.gitRoot, parsed.ownerLogin, parsed.repoName+".git")
	}

	// Authorization: pushing (receive-pack) requires the authenticated key to
	// belong to the repository owner. Without this, any user with a registered
	// key could write to any repository. (Org/collaborator push over SSH is not
	// yet supported and must use HTTPS.)
	if parsed.packType == "receive-pack" {
		authUserID, _ := ctx.Value(authUserIDContextKey).(uuid.UUID)
		if authUserID == uuid.Nil || ownerID == uuid.Nil || authUserID != ownerID {
			_, _ = fmt.Fprintf(s.Stderr(), "ERR write access denied\n")
			s.Exit(1)
			return
		}
	}

	// Serve git over SSH by running the real git pack programs wired to the
	// session's streams. This is the standard approach used by git servers and
	// implements the interactive SSH protocol correctly (go-git's pure-Go
	// server side does not fully support receive-pack from modern clients).
	if parsed.packType != "upload-pack" && parsed.packType != "receive-pack" {
		_, _ = fmt.Fprintf(s.Stderr(), "ERR unsupported pack type %q\n", parsed.packType)
		s.Exit(1)
		return
	}

	cmd := exec.CommandContext(ctx, "git", parsed.packType, diskPath)
	cmd.Stdin = s
	cmd.Stdout = s
	cmd.Stderr = s.Stderr()
	if err := cmd.Run(); err != nil {
		_, _ = fmt.Fprintf(s.Stderr(), "ERR %v\n", err)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			s.Exit(exitErr.ExitCode())
			return
		}
		s.Exit(1)
		return
	}
	s.Exit(0)
}
