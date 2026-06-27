package ssh

import (
	"context"
	"sync"
	"testing"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type stubHostKeyRepository struct {
	mu           sync.Mutex
	keys         map[string]*entity.HostKey
	createCalled bool
	createCount  int
}

func (s *stubHostKeyRepository) FindByAlgorithm(_ context.Context, algorithm string) (*entity.HostKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, ok := s.keys[algorithm]
	if !ok {
		return nil, nil
	}
	return key, nil
}

func (s *stubHostKeyRepository) Create(_ context.Context, key *entity.HostKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.createCalled = true
	s.createCount++
	s.keys[key.Algorithm] = key
	return nil
}

func TestLoadOrGenerateHostKey_GeneratesOnFirstCall(t *testing.T) {
	repo := &stubHostKeyRepository{keys: make(map[string]*entity.HostKey)}
	ctx := context.Background()

	signer, err := LoadOrGenerateHostKey(ctx, repo, AlgorithmEd25519)
	if err != nil {
		t.Fatalf("LoadOrGenerateHostKey: %v", err)
	}
	if signer == nil {
		t.Fatal("expected non-nil signer")
	}
	if !repo.createCalled {
		t.Fatal("expected repo.Create to be called on first call")
	}
	if repo.createCount != 1 {
		t.Fatalf("createCount = %d, want 1", repo.createCount)
	}
}

func TestLoadOrGenerateHostKey_ReusesExistingKey(t *testing.T) {
	repo := &stubHostKeyRepository{keys: make(map[string]*entity.HostKey)}
	ctx := context.Background()

	first, err := LoadOrGenerateHostKey(ctx, repo, AlgorithmEd25519)
	if err != nil {
		t.Fatalf("first LoadOrGenerateHostKey: %v", err)
	}

	repo.createCalled = false
	second, err := LoadOrGenerateHostKey(ctx, repo, AlgorithmEd25519)
	if err != nil {
		t.Fatalf("second LoadOrGenerateHostKey: %v", err)
	}
	if repo.createCalled {
		t.Fatal("expected repo.Create not to be called on second call")
	}
	if string(first.PublicKey().Marshal()) != string(second.PublicKey().Marshal()) {
		t.Fatal("expected same signer from persisted key")
	}
}

var _ domainrepo.IHostKeyRepository = (*stubHostKeyRepository)(nil)
