package mcp_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	mcpusecase "github.com/open-git/backend/internal/usecase/mcp"
)

type mockMCPRepo struct {
	runs                []*entity.MCPVerificationRun
	checks              []*entity.MCPVerificationCheck
	countRunsThisMonth  int64
	countRunsThisMonthSet bool
}

var _ domainrepo.IMCPVerificationRepository = (*mockMCPRepo)(nil)

func (m *mockMCPRepo) CreateRun(_ context.Context, run *entity.MCPVerificationRun) error {
	if run.ID == uuid.Nil {
		run.ID = uuid.New()
	}
	if run.CreatedAt.IsZero() {
		run.CreatedAt = time.Now().UTC()
	}
	stored := *run
	m.runs = append(m.runs, &stored)
	return nil
}

func (m *mockMCPRepo) GetRunByID(_ context.Context, id, orgID uuid.UUID) (*entity.MCPVerificationRun, error) {
	for _, run := range m.runs {
		if run.ID == id && run.OrganizationID == orgID {
			stored := *run
			return &stored, nil
		}
	}
	return nil, nil
}

func (m *mockMCPRepo) UpdateRun(_ context.Context, run *entity.MCPVerificationRun) error {
	for i, existing := range m.runs {
		if existing.ID == run.ID && existing.OrganizationID == run.OrganizationID {
			stored := *run
			m.runs[i] = &stored
			return nil
		}
	}
	return nil
}

func (m *mockMCPRepo) DeleteRun(_ context.Context, id, orgID uuid.UUID) error {
	filtered := make([]*entity.MCPVerificationRun, 0, len(m.runs))
	for _, run := range m.runs {
		if run.ID == id && run.OrganizationID == orgID {
			continue
		}
		filtered = append(filtered, run)
	}
	m.runs = filtered
	return nil
}

func (m *mockMCPRepo) GetLatestRun(_ context.Context, orgID uuid.UUID) (*entity.MCPVerificationRun, error) {
	return nil, nil
}

func (m *mockMCPRepo) ListRuns(_ context.Context, orgID uuid.UUID, page, perPage int) ([]*entity.MCPVerificationRun, int64, error) {
	return nil, 0, nil
}

func (m *mockMCPRepo) GetActiveRun(_ context.Context, orgID uuid.UUID) (*entity.MCPVerificationRun, error) {
	for _, run := range m.runs {
		if run.OrganizationID != orgID {
			continue
		}
		if run.Status == entity.RunStatusQueued || run.Status == entity.RunStatusRunning {
			stored := *run
			return &stored, nil
		}
	}
	return nil, nil
}

func (m *mockMCPRepo) CountRunsThisMonth(_ context.Context, orgID uuid.UUID) (int64, error) {
	if m.countRunsThisMonthSet {
		return m.countRunsThisMonth, nil
	}

	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	var count int64
	for _, run := range m.runs {
		if run.OrganizationID == orgID && !run.CreatedAt.Before(monthStart) {
			count++
		}
	}
	return count, nil
}

func (m *mockMCPRepo) BatchCreateChecks(_ context.Context, checks []*entity.MCPVerificationCheck) error {
	return nil
}

func (m *mockMCPRepo) ListChecksByRun(_ context.Context, runID, orgID uuid.UUID) ([]*entity.MCPVerificationCheck, error) {
	return nil, nil
}

type mockAuditRepo struct {
	entries []*entity.AuditLog
}

var _ domainrepo.IAuditLogRepository = (*mockAuditRepo)(nil)

func (m *mockAuditRepo) Create(_ context.Context, log *entity.AuditLog) error {
	stored := *log
	m.entries = append(m.entries, &stored)
	return nil
}

func (m *mockAuditRepo) List(context.Context, uuid.UUID, string, int, int) ([]*entity.AuditLog, int, error) {
	return nil, 0, nil
}

func (m *mockAuditRepo) InsertAuditLog(context.Context, uuid.UUID, uuid.UUID, string, string, uuid.UUID, []byte) error {
	return nil
}

func newTestAsynqClient(t *testing.T) *asynq.Client {
	t.Helper()
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: "127.0.0.1:6379"})
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestRunVerification_ConflictError(t *testing.T) {
	orgID := uuid.New()
	actorID := uuid.New()
	repo := &mockMCPRepo{
		runs: []*entity.MCPVerificationRun{{
			ID:                 uuid.New(),
			OrganizationID:     orgID,
			RepositoryFullName: "acme/widgets",
			Status:             entity.RunStatusQueued,
			CreatedAt:          time.Now().UTC(),
		}},
	}
	audit := &mockAuditRepo{}
	uc := mcpusecase.NewRunVerificationUsecase(repo, audit, newTestAsynqClient(t))

	_, err := uc.Execute(context.Background(), orgID, actorID, mcpusecase.RunVerificationInput{
		RepositoryFullName: "acme/widgets",
		Targets:            []string{"rest"},
	})
	if err != mcpusecase.ErrMCPRunConflict {
		t.Fatalf("expected ErrMCPRunConflict, got %v", err)
	}
}

func TestRunVerification_PlanLimit(t *testing.T) {
	orgID := uuid.New()
	actorID := uuid.New()
	repo := &mockMCPRepo{
		countRunsThisMonthSet: true,
		countRunsThisMonth:    10,
	}
	audit := &mockAuditRepo{}
	uc := mcpusecase.NewRunVerificationUsecase(repo, audit, newTestAsynqClient(t))

	_, err := uc.Execute(context.Background(), orgID, actorID, mcpusecase.RunVerificationInput{
		RepositoryFullName: "acme/widgets",
		Targets:            []string{"rest"},
	})
	if err != mcpusecase.ErrMCPPlanLimitExceeded {
		t.Fatalf("expected ErrMCPPlanLimitExceeded, got %v", err)
	}
}

func TestRunVerification_Success(t *testing.T) {
	orgID := uuid.New()
	actorID := uuid.New()
	repo := &mockMCPRepo{}
	audit := &mockAuditRepo{}
	uc := mcpusecase.NewRunVerificationUsecase(repo, audit, newTestAsynqClient(t))

	run, err := uc.Execute(context.Background(), orgID, actorID, mcpusecase.RunVerificationInput{
		RepositoryFullName: "acme/widgets",
		Targets:            []string{"rest"},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if run.Status != entity.RunStatusQueued {
		t.Fatalf("expected status queued, got %s", run.Status)
	}
	if len(audit.entries) != 1 {
		t.Fatalf("expected 1 audit log entry, got %d", len(audit.entries))
	}
	if audit.entries[0].Action != "mcp_verification.run" {
		t.Fatalf("expected audit action mcp_verification.run, got %s", audit.entries[0].Action)
	}
}
