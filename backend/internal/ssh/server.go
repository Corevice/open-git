package ssh

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	gliderssh "github.com/gliderlabs/ssh"
	"github.com/google/uuid"
	gossh "golang.org/x/crypto/ssh"

	"github.com/open-git/backend/internal/handler"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type contextKey string

const contextKeyUserID contextKey = "sshUserID"

var gitSSHCommandPattern = regexp.MustCompile(`^git-(upload-pack|receive-pack)\s+'?/?([^/]+)/([^'\s]+?)'?\.?$`)

type gitSSHCommand struct {
	packType   string
	ownerLogin string
	repoName   string
}

type SSHServer struct {
	gitRoot     string
	resolver    handler.GitRepositoryResolver
	keys        domainrepo.ISSHKeyRepository
	memberships handler.GitMembershipAccess
	protections handler.GitBranchProtectionStore
}

func NewSSHServer(
	gitRoot string,
	resolver handler.GitRepositoryResolver,
	keys domainrepo.ISSHKeyRepository,
	memberships handler.GitMembershipAccess,
	protections handler.GitBranchProtectionStore,
) *SSHServer {
	return &SSHServer{
		gitRoot:     gitRoot,
		resolver:    resolver,
		keys:        keys,
		memberships: memberships,
		protections: protections,
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

func (s *SSHServer) Start(addr string, signer gossh.Signer) error {
	srv := &gliderssh.Server{
		Addr:             addr,
		PublicKeyHandler: s.publicKeyHandler,
		Handler:          s.handleSession,
	}
	srv.AddHostKey(signer)
	return srv.ListenAndServe()
}

func (s *SSHServer) publicKeyHandler(ctx gliderssh.Context, key gliderssh.PublicKey) (*gliderssh.Permissions, error) {
	publicKey := strings.TrimSpace(string(gossh.MarshalAuthorizedKey(key)))
	stored, err := s.keys.FindByPublicKey(ctx, publicKey)
	if err != nil {
		return nil, fmt.Errorf("lookup ssh key: %w", err)
	}
	if stored == nil {
		return nil, gliderssh.ErrKeyRejected
	}

	keyID := stored.ID
	go func() {
		_ = s.keys.UpdateLastUsed(context.Background(), keyID)
	}()

	ctx.SetValue(contextKeyUserID, stored.UserID)
	return nil, nil
}

func (s *SSHServer) handleSession(sess gliderssh.Session) {
	ctx := sess.Context()

	rawCmd := strings.Join(sess.Command(), " ")
	parsed, err := parseGitSSHCommand(rawCmd)
	if err != nil {
		_, _ = fmt.Fprintf(sess, "ERR %v\n", err)
		_ = sess.Exit(1)
		return
	}

	resolved, err := s.resolver.Resolve(ctx, parsed.ownerLogin, parsed.repoName)
	if err != nil {
		_, _ = fmt.Fprintf(sess, "ERR %v\n", err)
		_ = sess.Exit(1)
		return
	}
	if resolved.DiskPath == "" {
		resolved.DiskPath = filepath.Join(s.gitRoot, parsed.ownerLogin, parsed.repoName+".git")
	}

	switch parsed.packType {
	case "upload-pack":
		if err := s.handleGitUploadPack(sess, resolved.DiskPath); err != nil {
			_, _ = fmt.Fprintf(sess, "ERR %v\n", err)
			_ = sess.Exit(1)
		}
	case "receive-pack":
		userID, _ := ctx.Value(contextKeyUserID).(uuid.UUID)
		if err := s.handleGitReceivePack(sess, resolved, userID); err != nil {
			_, _ = fmt.Fprintf(sess, "ERR %v\n", err)
			_ = sess.Exit(1)
		}
	default:
		_, _ = fmt.Fprintf(sess, "unknown command: %s\n", parsed.packType)
		_ = sess.Exit(1)
	}
}

func (s *SSHServer) handleGitUploadPack(sess gliderssh.Session, repoPath string) error {
	ctx := sess.Context()
	cmd := exec.CommandContext(ctx, "git-upload-pack", repoPath)
	cmd.Stdin = sess
	cmd.Stdout = sess
	cmd.Stderr = sess.Stderr()
	return cmd.Run()
}

func (s *SSHServer) handleGitReceivePack(sess gliderssh.Session, resolved *handler.ResolvedGitRepository, userID uuid.UUID) error {
	if err := s.ensureWriteAccess(sess.Context(), userID, resolved); err != nil {
		return err
	}

	ctx := sess.Context()
	cmd := exec.CommandContext(ctx, "git-receive-pack", resolved.DiskPath)
	cmd.Stdin = sess
	cmd.Stdout = sess
	cmd.Stderr = sess.Stderr()
	return cmd.Run()
}

func (s *SSHServer) ensureWriteAccess(ctx context.Context, userID uuid.UUID, repo *handler.ResolvedGitRepository) error {
	if userID != uuid.Nil && userID == repo.OrganizationID {
		return nil
	}
	if s.memberships == nil {
		return fmt.Errorf("write access required")
	}
	ok, err := s.memberships.HasWriteAccess(ctx, repo.OwnerID, repo.OrganizationID)
	if err != nil {
		return fmt.Errorf("check permissions: %w", err)
	}
	if !ok {
		return fmt.Errorf("write access required")
	}
	return nil
}
