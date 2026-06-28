package actions_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/usecase/actions"
)

var registerTestOrgID = uuid.MustParse("00000000-0000-0000-0000-000000000060")

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

type mockRunnerRepo struct {
	created *entity.Runner
}

func (m *mockRunnerRepo) Create(_ context.Context, runner *entity.Runner) error {
	copyRunner := *runner
	m.created = &copyRunner
	return nil
}

func (m *mockRunnerRepo) GetByID(context.Context, uuid.UUID) (*entity.Runner, error) {
	return nil, domain.ErrNotFound
}

func (m *mockRunnerRepo) ListByOrg(context.Context, uuid.UUID) ([]*entity.Runner, error) {
	return nil, nil
}

func (m *mockRunnerRepo) UpdateStatus(context.Context, uuid.UUID, string, time.Time) error {
	return nil
}

func (m *mockRunnerRepo) Delete(context.Context, uuid.UUID) error {
	return nil
}

func (m *mockRunnerRepo) FindAvailable(context.Context, uuid.UUID, []string) (*entity.Runner, error) {
	return nil, domain.ErrNotFound
}

var _ domainrepo.IRunnerRepository = (*mockRunnerRepo)(nil)

type mockTokenRepo struct {
	tokens map[string]*entity.RunnerRegistrationToken
	used   map[uuid.UUID]bool
}

func newMockTokenRepo() *mockTokenRepo {
	return &mockTokenRepo{
		tokens: make(map[string]*entity.RunnerRegistrationToken),
		used:   make(map[uuid.UUID]bool),
	}
}

func (m *mockTokenRepo) Create(_ context.Context, token *entity.RunnerRegistrationToken) error {
	copyToken := *token
	m.tokens[token.TokenHash] = &copyToken
	return nil
}

func (m *mockTokenRepo) GetByTokenHash(_ context.Context, hash string) (*entity.RunnerRegistrationToken, error) {
	token, ok := m.tokens[hash]
	if !ok || m.used[token.ID] {
		return nil, domain.ErrNotFound
	}
	copyToken := *token
	return &copyToken, nil
}

func (m *mockTokenRepo) MarkUsed(_ context.Context, id uuid.UUID, usedAt time.Time) error {
	if _, ok := m.tokensByID(id); !ok {
		return domain.ErrNotFound
	}
	if m.used[id] {
		return domain.ErrNotFound
	}
	m.used[id] = true
	return nil
}

func (m *mockTokenRepo) tokensByID(id uuid.UUID) (*entity.RunnerRegistrationToken, bool) {
	for _, token := range m.tokens {
		if token.ID == id {
			return token, true
		}
	}
	return nil, false
}

var _ domainrepo.IRunnerRegistrationTokenRepository = (*mockTokenRepo)(nil)

type mockAuditLogRepo struct {
	logs []*entity.AuditLog
}

func (m *mockAuditLogRepo) Create(_ context.Context, log *entity.AuditLog) error {
	copyLog := *log
	m.logs = append(m.logs, &copyLog)
	return nil
}

func (m *mockAuditLogRepo) List(context.Context, uuid.UUID, string, int, int) ([]*entity.AuditLog, int, error) {
	return nil, 0, nil
}

func (m *mockAuditLogRepo) InsertAuditLog(context.Context, uuid.UUID, uuid.UUID, string, string, uuid.UUID, json.RawMessage) error {
	return nil
}

var _ domainrepo.IAuditLogRepository = (*mockAuditLogRepo)(nil)

func validRegisterRequest(token string) actions.RegisterRunnerRequest {
	return actions.RegisterRunnerRequest{
		RegistrationToken: token,
		Name:              "runner-1",
		Labels:            []string{"self-hosted"},
		OS:                "linux",
		Arch:              "amd64",
		RunnerType:        entity.RunnerTypeAct,
	}
}

func TestRegisterRunnerUsecase_LabelWithSpaceRejected(t *testing.T) {
	uc := actions.NewRegisterRunnerUsecase(&mockRunnerRepo{}, newMockTokenRepo(), &mockAuditLogRepo{})

	_, err := uc.Execute(context.Background(), registerTestOrgID, actions.RegisterRunnerRequest{
		RegistrationToken: "tok",
		Name:              "runner-1",
		Labels:            []string{"bad label"},
		OS:                "linux",
		Arch:              "amd64",
		RunnerType:        entity.RunnerTypeAct,
	})
	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestRegisterRunnerUsecase_LabelTooLongRejected(t *testing.T) {
	uc := actions.NewRegisterRunnerUsecase(&mockRunnerRepo{}, newMockTokenRepo(), &mockAuditLogRepo{})

	longLabel := make([]byte, 65)
	for i := range longLabel {
		longLabel[i] = 'a'
	}

	_, err := uc.Execute(context.Background(), registerTestOrgID, actions.RegisterRunnerRequest{
		RegistrationToken: "tok",
		Name:              "runner-1",
		Labels:            []string{string(longLabel)},
		OS:                "linux",
		Arch:              "amd64",
		RunnerType:        entity.RunnerTypeAct,
	})
	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestRegisterRunnerUsecase_ExpiredTokenRejected(t *testing.T) {
	raw := "expired-token"
	tokenRepo := newMockTokenRepo()
	tokenRepo.tokens[hashToken(raw)] = &entity.RunnerRegistrationToken{
		ID:             uuid.New(),
		OrganizationID: registerTestOrgID,
		TokenHash:      hashToken(raw),
		ExpiresAt:      time.Now().UTC().Add(-time.Hour),
	}

	uc := actions.NewRegisterRunnerUsecase(&mockRunnerRepo{}, tokenRepo, &mockAuditLogRepo{})

	_, err := uc.Execute(context.Background(), registerTestOrgID, validRegisterRequest(raw))
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected unauthorized error, got %v", err)
	}
}

func TestRegisterRunnerUsecase_UsedTokenRejected(t *testing.T) {
	raw := "used-token"
	tokenID := uuid.New()
	tokenRepo := newMockTokenRepo()
	tokenRepo.tokens[hashToken(raw)] = &entity.RunnerRegistrationToken{
		ID:             tokenID,
		OrganizationID: registerTestOrgID,
		TokenHash:      hashToken(raw),
		ExpiresAt:      time.Now().UTC().Add(time.Hour),
	}
	tokenRepo.used[tokenID] = true

	uc := actions.NewRegisterRunnerUsecase(&mockRunnerRepo{}, tokenRepo, &mockAuditLogRepo{})

	_, err := uc.Execute(context.Background(), registerTestOrgID, validRegisterRequest(raw))
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected unauthorized error, got %v", err)
	}
}

func TestRegisterRunnerUsecase_ValidRegistrationCreatesOnlineRunner(t *testing.T) {
	raw := "valid-token"
	tokenRepo := newMockTokenRepo()
	tokenRepo.tokens[hashToken(raw)] = &entity.RunnerRegistrationToken{
		ID:             uuid.New(),
		OrganizationID: registerTestOrgID,
		TokenHash:      hashToken(raw),
		ExpiresAt:      time.Now().UTC().Add(time.Hour),
	}
	runnerRepo := &mockRunnerRepo{}
	auditRepo := &mockAuditLogRepo{}

	uc := actions.NewRegisterRunnerUsecase(runnerRepo, tokenRepo, auditRepo)

	runner, err := uc.Execute(context.Background(), registerTestOrgID, validRegisterRequest(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner.Status != entity.RunnerStatusOnline {
		t.Fatalf("expected status online, got %q", runner.Status)
	}
	if runnerRepo.created == nil || runnerRepo.created.Status != entity.RunnerStatusOnline {
		t.Fatalf("expected created runner with online status, got %#v", runnerRepo.created)
	}
	if len(auditRepo.logs) != 1 || auditRepo.logs[0].Action != "runner_registered" {
		t.Fatalf("expected runner_registered audit log, got %#v", auditRepo.logs)
	}
}
