package ssh

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	gossh "github.com/gliderlabs/ssh"
	"github.com/google/uuid"
	cryptossh "golang.org/x/crypto/ssh"

	infragit "github.com/open-git/backend/internal/infrastructure/git"
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

type sshResponseWriter struct {
	session gossh.Session
	header  http.Header
	status  int
}

func (w *sshResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *sshResponseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.session.Write(b)
}

func (w *sshResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
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

func (h *SSHServer) authenticateKey(ctx gossh.Context, key gossh.PublicKey) bool {
	fingerprint := cryptossh.FingerprintSHA256(key)
	stored, err := h.keyStore.FindByFingerprint(ctx, fingerprint)
	if err != nil || stored == nil {
		return false
	}
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

	diskPath, _, err := h.resolver.Resolve(ctx, parsed.ownerLogin, parsed.repoName)
	if err != nil {
		_, _ = fmt.Fprintf(s, "ERR %v\n", err)
		s.Exit(1)
		return
	}
	if diskPath == "" {
		diskPath = filepath.Join(h.gitRoot, parsed.ownerLogin, parsed.repoName+".git")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/"+parsed.packType, io.NopCloser(s))
	if err != nil {
		_, _ = fmt.Fprintf(s, "ERR %v\n", err)
		s.Exit(1)
		return
	}

	rw := &sshResponseWriter{session: s}
	switch parsed.packType {
	case "upload-pack":
		err = infragit.ServeUploadPack(rw, req, diskPath)
	case "receive-pack":
		err = infragit.ServeReceivePack(rw, req, diskPath)
	default:
		err = fmt.Errorf("unsupported pack type %q", parsed.packType)
	}
	if err != nil {
		_, _ = fmt.Fprintf(s, "ERR %v\n", err)
		s.Exit(1)
	}
}
