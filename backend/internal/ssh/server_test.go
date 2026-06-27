package ssh

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"io"
	"net"
	"sync"
	"testing"

	gliderssh "github.com/gliderlabs/ssh"
	"github.com/Masterminds/semver/v3"
	"github.com/google/uuid"
	gossh "golang.org/x/crypto/ssh"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type stubSSHKeyRepository struct {
	mu              sync.Mutex
	findResult      *entity.SSHKey
	updateLastUsed  bool
	lastUsedKeyID   uuid.UUID
	findByPublicKey func(publicKey string) *entity.SSHKey
}

func (s *stubSSHKeyRepository) FindByPublicKey(_ context.Context, publicKey string) (*entity.SSHKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.findByPublicKey != nil {
		return s.findByPublicKey(publicKey), nil
	}
	return s.findResult, nil
}

func (s *stubSSHKeyRepository) UpdateLastUsed(_ context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateLastUsed = true
	s.lastUsedKeyID = id
	return nil
}

type mockSSHContext struct {
	context.Context
	values map[interface{}]interface{}
}

func newMockSSHContext() *mockSSHContext {
	return &mockSSHContext{
		Context: context.Background(),
		values:  make(map[interface{}]interface{}),
	}
}

func (m *mockSSHContext) SetValue(key, value interface{}) {
	m.values[key] = value
}

func (m *mockSSHContext) Value(key interface{}) interface{} {
	if v, ok := m.values[key]; ok {
		return v
	}
	return m.Context.Value(key)
}

func (m *mockSSHContext) User() string                        { return "" }
func (m *mockSSHContext) SessionID() string                   { return "test-session" }
func (m *mockSSHContext) ClientVersion() *semver.Version      { return nil }
func (m *mockSSHContext) ServerVersion() *semver.Version    { return nil }
func (m *mockSSHContext) RemoteAddr() net.Addr                { return &net.TCPAddr{} }
func (m *mockSSHContext) LocalAddr() net.Addr                 { return &net.TCPAddr{} }
func (m *mockSSHContext) Permissions() *gliderssh.Permissions { return nil }

type mockSession struct {
	ctx      *mockSSHContext
	command  []string
	stdout   bytes.Buffer
	stderr   bytes.Buffer
	exitCode int
	exited   bool
}

func (m *mockSession) Read([]byte) (int, error)  { return 0, io.EOF }
func (m *mockSession) Write(p []byte) (int, error) { return m.stdout.Write(p) }
func (m *mockSession) Close() error              { return nil }
func (m *mockSession) SendRequest(string, bool, []byte) (bool, error) {
	return false, nil
}
func (m *mockSession) Context() gliderssh.Context { return m.ctx }
func (m *mockSession) Permissions() *gliderssh.Permissions {
	return nil
}
func (m *mockSession) User() string         { return "" }
func (m *mockSession) RemoteAddr() net.Addr { return &net.TCPAddr{} }
func (m *mockSession) LocalAddr() net.Addr  { return &net.TCPAddr{} }
func (m *mockSession) Exit(code int) error {
	m.exitCode = code
	m.exited = true
	return nil
}
func (m *mockSession) Stderr() io.Writer { return &m.stderr }
func (m *mockSession) Command() []string { return m.command }
func (m *mockSession) RawCommand() string {
	return ""
}
func (m *mockSession) Subsystem() string { return "" }
func (m *mockSession) Pty() (string, int, int, bool) {
	return "", 0, 0, false
}
func (m *mockSession) WindowSize() (int, int, bool) { return 0, 0, false }
func (m *mockSession) Signers() []gliderssh.Signer  { return nil }
func (m *mockSession) PublicKey() gliderssh.PublicKey {
	return nil
}

func generateTestPublicKey(t *testing.T) (gossh.PublicKey, string) {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	signer, err := gossh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	publicKey := string(gossh.MarshalAuthorizedKey(signer.PublicKey()))
	return signer.PublicKey(), publicKey
}

func TestSSHServer_PublicKeyHandler_RejectsUnregisteredKey(t *testing.T) {
	server := NewSSHServer("", nil, &stubSSHKeyRepository{findResult: nil}, nil, nil)
	pubKey, _ := generateTestPublicKey(t)

	_, err := server.publicKeyHandler(newMockSSHContext(), pubKey)
	if err == nil {
		t.Fatal("expected error for unregistered key")
	}
	if err != gliderssh.ErrKeyRejected {
		t.Fatalf("error = %v, want ErrKeyRejected", err)
	}
}

func TestSSHServer_PublicKeyHandler_AcceptsRegisteredKey(t *testing.T) {
	pubKey, publicKeyLine := generateTestPublicKey(t)
	keyID := uuid.New()
	userID := uuid.New()

	repo := &stubSSHKeyRepository{
		findByPublicKey: func(got string) *entity.SSHKey {
			if got != publicKeyLine {
				t.Fatalf("public key = %q, want %q", got, publicKeyLine)
			}
			return &entity.SSHKey{
				ID:        keyID,
				UserID:    userID,
				PublicKey: publicKeyLine,
			}
		},
	}
	server := NewSSHServer("", nil, repo, nil, nil)

	perms, err := server.publicKeyHandler(newMockSSHContext(), pubKey)
	if err != nil {
		t.Fatalf("publicKeyHandler: %v", err)
	}
	if perms != nil {
		t.Fatalf("permissions = %v, want nil", perms)
	}
}

func TestSSHServer_UnknownCommandReturnsError(t *testing.T) {
	server := NewSSHServer("", nil, &stubSSHKeyRepository{}, nil, nil)
	sess := &mockSession{
		ctx:     newMockSSHContext(),
		command: []string{"git-archive", "'owner/repo.git'"},
	}

	server.handleSession(sess)

	if !sess.exited || sess.exitCode == 0 {
		t.Fatalf("expected non-zero exit, got exited=%v code=%d", sess.exited, sess.exitCode)
	}
	if !bytes.Contains(sess.stdout.Bytes(), []byte("invalid git ssh command")) {
		t.Fatalf("stdout = %q, want error message", sess.stdout.String())
	}
}

var _ domainrepo.ISSHKeyRepository = (*stubSSHKeyRepository)(nil)
