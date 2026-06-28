package security_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	securityusecase "github.com/open-git/backend/internal/usecase/security"
)

type mockAuditLogSearchRepo struct {
	logs  []*entity.AuditLog
	total int
	input domainrepo.AuditLogSearchInput
}

func (m *mockAuditLogSearchRepo) Search(_ context.Context, input domainrepo.AuditLogSearchInput) ([]*entity.AuditLog, int, error) {
	m.input = input
	return m.logs, m.total, nil
}

var _ domainrepo.IAuditLogSearchRepository = (*mockAuditLogSearchRepo)(nil)

func TestSearchAuditLogsUsecase_DateRangeExceeded(t *testing.T) {
	t.Parallel()

	after := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	before := after.Add(91 * 24 * time.Hour)

	uc := securityusecase.NewSearchAuditLogsUsecase(&mockAuditLogSearchRepo{})
	_, err := uc.Execute(context.Background(), securityusecase.SearchAuditLogsInput{
		OrganizationID: uuid.New(),
		After:          &after,
		Before:         &before,
	})

	if !errors.Is(err, securityusecase.ErrDateRangeExceeded) {
		t.Fatalf("expected ErrDateRangeExceeded, got %v", err)
	}
}

func TestSearchAuditLogsUsecase_SuccessDelegation(t *testing.T) {
	t.Parallel()

	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	after := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	before := after.Add(30 * 24 * time.Hour)
	logID := uuid.MustParse("00000000-0000-0000-0000-000000000020")

	repo := &mockAuditLogSearchRepo{
		logs: []*entity.AuditLog{{
			ID:             logID,
			OrganizationID: orgID,
			Action:         "member.add",
		}},
		total: 1,
	}
	uc := securityusecase.NewSearchAuditLogsUsecase(repo)

	output, err := uc.Execute(context.Background(), securityusecase.SearchAuditLogsInput{
		OrganizationID: orgID,
		Phrase:         "alice",
		Action:         "member.add",
		After:          &after,
		Before:         &before,
		Page:           2,
		PerPage:        50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Total != 1 || len(output.Logs) != 1 {
		t.Fatalf("expected 1 log, got total=%d len=%d", output.Total, len(output.Logs))
	}
	if repo.input.OrganizationID != orgID {
		t.Fatalf("organization_id = %v, want %v", repo.input.OrganizationID, orgID)
	}
	if repo.input.Phrase != "alice" || repo.input.Action != "member.add" {
		t.Fatalf("unexpected search filters: phrase=%q action=%q", repo.input.Phrase, repo.input.Action)
	}
	if repo.input.Page != 2 || repo.input.PerPage != 50 {
		t.Fatalf("unexpected pagination: page=%d per_page=%d", repo.input.Page, repo.input.PerPage)
	}
}
