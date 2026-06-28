package repository_test

import (
	"context"
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/entity"
)

type mockMCPVerificationRepo struct {
	runs   []*entity.MCPVerificationRun
	checks []*entity.MCPVerificationCheck
}

var _ domainrepo.IMCPVerificationRepository = (*mockMCPVerificationRepo)(nil)

func (m *mockMCPVerificationRepo) CreateRun(_ context.Context, run *entity.MCPVerificationRun) error {
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

func (m *mockMCPVerificationRepo) GetRunByID(_ context.Context, id, orgID uuid.UUID) (*entity.MCPVerificationRun, error) {
	for _, run := range m.runs {
		if run.ID == id && run.OrganizationID == orgID {
			stored := *run
			return &stored, nil
		}
	}
	return nil, nil
}

func (m *mockMCPVerificationRepo) UpdateRun(_ context.Context, run *entity.MCPVerificationRun) error {
	for i, existing := range m.runs {
		if existing.ID == run.ID && existing.OrganizationID == run.OrganizationID {
			stored := *run
			m.runs[i] = &stored
			return nil
		}
	}
	return nil
}

func (m *mockMCPVerificationRepo) DeleteRun(_ context.Context, id, orgID uuid.UUID) error {
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

func (m *mockMCPVerificationRepo) GetLatestRun(_ context.Context, orgID uuid.UUID) (*entity.MCPVerificationRun, error) {
	runs := m.orgRuns(orgID)
	if len(runs) == 0 {
		return nil, nil
	}
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].CreatedAt.After(runs[j].CreatedAt)
	})
	stored := *runs[0]
	return &stored, nil
}

func (m *mockMCPVerificationRepo) ListRuns(_ context.Context, orgID uuid.UUID, page, perPage int) ([]*entity.MCPVerificationRun, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}

	runs := m.orgRuns(orgID)
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].CreatedAt.After(runs[j].CreatedAt)
	})

	total := int64(len(runs))
	offset := (page - 1) * perPage
	if offset >= len(runs) {
		return []*entity.MCPVerificationRun{}, total, nil
	}

	end := offset + perPage
	if end > len(runs) {
		end = len(runs)
	}

	pageRuns := make([]*entity.MCPVerificationRun, 0, end-offset)
	for _, run := range runs[offset:end] {
		stored := *run
		pageRuns = append(pageRuns, &stored)
	}
	return pageRuns, total, nil
}

func (m *mockMCPVerificationRepo) GetActiveRun(_ context.Context, orgID uuid.UUID) (*entity.MCPVerificationRun, error) {
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

func (m *mockMCPVerificationRepo) CountRunsThisMonth(_ context.Context, orgID uuid.UUID) (int64, error) {
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

func (m *mockMCPVerificationRepo) BatchCreateChecks(_ context.Context, checks []*entity.MCPVerificationCheck) error {
	for _, check := range checks {
		if check == nil {
			continue
		}
		if check.ID == uuid.Nil {
			check.ID = uuid.New()
		}
		if check.CreatedAt.IsZero() {
			check.CreatedAt = time.Now().UTC()
		}
		stored := *check
		m.checks = append(m.checks, &stored)
	}
	return nil
}

func (m *mockMCPVerificationRepo) ListChecksByRun(_ context.Context, runID, orgID uuid.UUID) ([]*entity.MCPVerificationCheck, error) {
	results := make([]*entity.MCPVerificationCheck, 0)
	for _, check := range m.checks {
		if check.RunID == runID && check.OrganizationID == orgID {
			stored := *check
			results = append(results, &stored)
		}
	}
	return results, nil
}

func (m *mockMCPVerificationRepo) orgRuns(orgID uuid.UUID) []*entity.MCPVerificationRun {
	runs := make([]*entity.MCPVerificationRun, 0)
	for _, run := range m.runs {
		if run.OrganizationID == orgID {
			runs = append(runs, run)
		}
	}
	return runs
}

func TestCreateRun(t *testing.T) {
	repo := &mockMCPVerificationRepo{}
	orgID := uuid.New()

	run := &entity.MCPVerificationRun{
		OrganizationID:   orgID,
		RepositoryFullName: "octo-org/hello-repo",
		Status:           entity.RunStatusQueued,
		Targets:          json.RawMessage(`["graphql","rest"]`),
	}

	if err := repo.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if len(repo.runs) != 1 {
		t.Fatalf("expected 1 stored run, got %d", len(repo.runs))
	}
	if repo.runs[0].ID != run.ID {
		t.Fatalf("stored run ID mismatch: got %v, want %v", repo.runs[0].ID, run.ID)
	}
	if repo.runs[0].OrganizationID != orgID {
		t.Fatalf("stored organization_id mismatch: got %v, want %v", repo.runs[0].OrganizationID, orgID)
	}
}

func TestGetActiveRunEmpty(t *testing.T) {
	repo := &mockMCPVerificationRepo{}

	got, err := repo.GetActiveRun(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("GetActiveRun: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil run, got %+v", got)
	}
}

func TestGetActiveRunWithQueued(t *testing.T) {
	repo := &mockMCPVerificationRepo{}
	orgID := uuid.New()

	run := &entity.MCPVerificationRun{
		OrganizationID:   orgID,
		RepositoryFullName: "octo-org/hello-repo",
		Status:           entity.RunStatusQueued,
	}
	if err := repo.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	got, err := repo.GetActiveRun(context.Background(), orgID)
	if err != nil {
		t.Fatalf("GetActiveRun: %v", err)
	}
	if got == nil {
		t.Fatal("expected active run, got nil")
	}
	if got.ID != run.ID {
		t.Fatalf("run ID mismatch: got %v, want %v", got.ID, run.ID)
	}
	if got.Status != entity.RunStatusQueued {
		t.Fatalf("status mismatch: got %q, want %q", got.Status, entity.RunStatusQueued)
	}
}

func TestListRunsPagination(t *testing.T) {
	repo := &mockMCPVerificationRepo{}
	orgID := uuid.New()
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 15; i++ {
		createdAt := baseTime.Add(time.Duration(i) * time.Hour)
		run := &entity.MCPVerificationRun{
			OrganizationID:   orgID,
			RepositoryFullName: "octo-org/hello-repo",
			Status:           entity.RunStatusCompleted,
			CreatedAt:        createdAt,
		}
		if err := repo.CreateRun(context.Background(), run); err != nil {
			t.Fatalf("CreateRun: %v", err)
		}
	}

	runs, total, err := repo.ListRuns(context.Background(), orgID, 1, 10)
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if total != 15 {
		t.Fatalf("total: got %d, want 15", total)
	}
	if len(runs) != 10 {
		t.Fatalf("page length: got %d, want 10", len(runs))
	}
	if runs[0].CreatedAt.Before(runs[1].CreatedAt) {
		t.Fatalf("expected created_at DESC order, got %v then %v", runs[0].CreatedAt, runs[1].CreatedAt)
	}
}
