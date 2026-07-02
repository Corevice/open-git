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

// RepoAuthorizer decides whether the authenticated SSH user may read (clone/fetch)
// or write (push) a repository. It mirrors the HTTP git handler's access checks
// so org members and collaborators — not just the owner — are handled.
type RepoAuthorizer interface {
	CanRead(ctx context.Context, userID uuid.UUID, ownerLogin, repoName string) (bool, error)
	CanWrite(ctx context.Context, userID uuid.UUID, ownerLogin, repoName string) (bool, error)
}

type SSHServer struct {
	gitRoot    string
	keyStore   repository.ISSHKeyStore
	resolver   GitSSHResolver
	authorizer RepoAuthorizer
	hostKey    gossh.Signer
	server     *gossh.Server
	// onPush, when set, is invoked once per branch updated by a successful
	// receive-pack (CI trigger). Must never fail the push.
	onPush func(ctx context.Context, ownerLogin, repoName, branch, newSHA string)
}

// SetPushListener installs the post-receive callback (e.g. the CI trigger).
func (h *SSHServer) SetPushListener(fn func(ctx context.Context, ownerLogin, repoName, branch, newSHA string)) {
	h.onPush = fn
}

// snapshotBranchHeads returns branch name -> head SHA for the bare repo. Used
// to diff refs around receive-pack, since the SSH path streams the protocol
// straight through git and never sees the ref update commands itself.
func snapshotBranchHeads(ctx context.Context, diskPath string) map[string]string {
	out, err := exec.CommandContext(ctx, "git", "--git-dir", diskPath, "for-each-ref", "refs/heads", "--format=%(objectname) %(refname:short)").Output()
	if err != nil {
		return nil
	}
	heads := make(map[string]string)
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), " ", 2)
		if len(parts) == 2 {
			heads[parts[1]] = parts[0]
		}
	}
	return heads
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
	authorizer RepoAuthorizer,
	hostKey gossh.Signer,
) *SSHServer {
	return &SSHServer{
		gitRoot:    gitRoot,
		keyStore:   keyStore,
		resolver:   resolver,
		authorizer: authorizer,
		hostKey:    hostKey,
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
	// Defense in depth: never let a name traverse out of the git root, even if a
	// future resolver returned an empty disk path for a crafted name.
	if !isSafeRepoSegment(parsed.ownerLogin) || !isSafeRepoSegment(parsed.repoName) {
		_, _ = fmt.Fprintf(s.Stderr(), "ERR invalid repository path\n")
		s.Exit(1)
		return
	}
	if diskPath == "" {
		diskPath = filepath.Join(h.gitRoot, parsed.ownerLogin, parsed.repoName+".git")
	}

	// Authorization: check the authenticated key's user against the repository.
	// receive-pack (push) requires write access; upload-pack (clone/fetch)
	// requires read access. When an authorizer is configured it handles owner,
	// org-member and collaborator permissions (parity with HTTP); otherwise we
	// fall back to an owner-only write check.
	authUserID, _ := ctx.Value(authUserIDContextKey).(uuid.UUID)
	if h.authorizer != nil {
		var allowed bool
		var aerr error
		if parsed.packType == "receive-pack" {
			allowed, aerr = h.authorizer.CanWrite(ctx, authUserID, parsed.ownerLogin, parsed.repoName)
		} else {
			allowed, aerr = h.authorizer.CanRead(ctx, authUserID, parsed.ownerLogin, parsed.repoName)
		}
		if aerr != nil || !allowed {
			_, _ = fmt.Fprintf(s.Stderr(), "ERR access denied\n")
			s.Exit(1)
			return
		}
	} else if parsed.packType == "receive-pack" {
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

	var beforeHeads map[string]string
	if parsed.packType == "receive-pack" && h.onPush != nil {
		beforeHeads = snapshotBranchHeads(ctx, diskPath)
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

	if parsed.packType == "receive-pack" && h.onPush != nil {
		after := snapshotBranchHeads(context.Background(), diskPath)
		for branch, sha := range after {
			if beforeHeads[branch] != sha {
				h.onPush(context.Background(), parsed.ownerLogin, parsed.repoName, branch, sha)
			}
		}
	}
	s.Exit(0)
}

// isSafeRepoSegment rejects owner/repo path segments that could traverse out of
// the git root ("..", empty, or containing a path separator).
func isSafeRepoSegment(s string) bool {
	if s == "" || s == "." || s == ".." {
		return false
	}
	return !strings.Contains(s, "/") && !strings.Contains(s, `\`) && !strings.Contains(s, "..")
}
