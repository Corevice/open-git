package security_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	securityusecase "github.com/open-git/backend/internal/usecase/security"
)

type mockAuditLogExportEnqueuer struct {
	payload securityusecase.AuditLogExportPayload
	err     error
}

func (m *mockAuditLogExportEnqueuer) EnqueueAuditLogExport(_ context.Context, payload securityusecase.AuditLogExportPayload) error {
	m.payload = payload
	return m.err
}

func TestExportAuditLogsUsecase_RedisNotConfigured(t *testing.T) {
	t.Parallel()

	uc := securityusecase.NewExportAuditLogsUsecase(nil)
	_, err := uc.Execute(context.Background(), securityusecase.ExportAuditLogsInput{
		OrganizationID: uuid.New(),
		ActorID:        uuid.New(),
		Format:         "csv",
	})

	if !errors.Is(err, securityusecase.ErrRedisNotConfigured) {
		t.Fatalf("expected ErrRedisNotConfigured, got %v", err)
	}
}

func TestExportAuditLogsUsecase_DateRangeExceeded(t *testing.T) {
	t.Parallel()

	after := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	before := after.Add(91 * 24 * time.Hour)
	enqueuer := &mockAuditLogExportEnqueuer{}
	uc := securityusecase.NewExportAuditLogsUsecaseWithDeps(enqueuer)

	_, err := uc.Execute(context.Background(), securityusecase.ExportAuditLogsInput{
		OrganizationID: uuid.New(),
		ActorID:        uuid.New(),
		Format:         "csv",
		After:          &after,
		Before:         &before,
	})

	if !errors.Is(err, securityusecase.ErrDateRangeExceeded) {
		t.Fatalf("expected ErrDateRangeExceeded, got %v", err)
	}
}

func TestExportAuditLogsUsecase_EnqueuesNormalizedRange(t *testing.T) {
	t.Parallel()

	before := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	enqueuer := &mockAuditLogExportEnqueuer{}
	uc := securityusecase.NewExportAuditLogsUsecaseWithDeps(enqueuer)

	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000011")

	output, err := uc.Execute(context.Background(), securityusecase.ExportAuditLogsInput{
		OrganizationID: orgID,
		ActorID:        actorID,
		Format:         "json",
		Phrase:         "alice",
		Action:         "member.add",
		Before:         &before,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.JobID == uuid.Nil {
		t.Fatal("expected job id")
	}
	if enqueuer.payload.OrganizationID != orgID.String() {
		t.Fatalf("organization_id = %q, want %q", enqueuer.payload.OrganizationID, orgID)
	}
	if enqueuer.payload.After == nil {
		t.Fatal("expected normalized after bound in export payload")
	}
	expectedAfter := before.Add(-securityusecase.MaxAuditLogRange)
	if !enqueuer.payload.After.Equal(expectedAfter) {
		t.Fatalf("after = %v, want %v", enqueuer.payload.After, expectedAfter)
	}
	if !enqueuer.payload.Before.Equal(before) {
		t.Fatalf("before = %v, want %v", enqueuer.payload.Before, before)
	}
}
