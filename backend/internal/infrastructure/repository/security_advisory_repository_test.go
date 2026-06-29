package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

func seedSecurityAdvisoryFixtures(t *testing.T, db *sqlx.DB) (orgID, repoID uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	orgID = uuid.New()
	ownerID := uuid.New()
	repoID = uuid.New()

	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("exec %q: %v", query, err)
		}
	}

	exec(`INSERT INTO organizations (id, login, name) VALUES (?, ?, ?)`, orgID, "sec-org", "Security Org")
	exec(`INSERT INTO users (id, login, email, password_hash) VALUES (?, ?, ?, ?)`, ownerID, "owner", "owner@example.com", "hash")
	exec(`INSERT INTO repositories (id, organization_id, owner_id, name) VALUES (?, ?, ?, ?)`, repoID, orgID, ownerID, "demo")

	return orgID, repoID
}

func TestSecurityAdvisoryRepository_CreateGetByGHSAPID(t *testing.T) {
	db := openTestDB(t)
	orgID, repoID := seedSecurityAdvisoryFixtures(t, db)
	repo := repository.NewSecurityAdvisoryRepository(db)
	ctx := context.Background()

	advisory := &entity.SecurityAdvisory{
		OrganizationID:   orgID,
		RepositoryID:     &repoID,
		GHSAPID:          "GHSA-test-1234",
		CVEID:            "CVE-2024-0001",
		Severity:         entity.AdvisorySeverityHigh,
		Summary:          "Test summary",
		Description:      "Test description",
		AffectedPackage:  "lodash",
		AffectedVersions: "<4.17.21",
		PatchedVersions:  "4.17.21",
		State:            entity.AdvisoryStateOpen,
	}

	if err := repo.Create(ctx, advisory); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if advisory.ID == uuid.Nil {
		t.Fatal("expected advisory ID to be assigned")
	}

	got, err := repo.GetByGHSAPID(ctx, orgID, "GHSA-test-1234")
	if err != nil {
		t.Fatalf("GetByGHSAPID: %v", err)
	}
	if got == nil {
		t.Fatal("expected advisory, got nil")
	}
	if got.GHSAPID != advisory.GHSAPID || got.Severity != entity.AdvisorySeverityHigh {
		t.Fatalf("unexpected advisory: %+v", got)
	}
	if got.RepositoryID == nil || *got.RepositoryID != repoID {
		t.Fatalf("expected repository_id %v, got %+v", repoID, got.RepositoryID)
	}
}

func TestSecurityAdvisoryRepository_UpsertDedup(t *testing.T) {
	db := openTestDB(t)
	orgID, _ := seedSecurityAdvisoryFixtures(t, db)
	repo := repository.NewSecurityAdvisoryRepository(db)
	ctx := context.Background()

	base := &entity.SecurityAdvisory{
		OrganizationID: orgID,
		GHSAPID:        "GHSA-dedup-5678",
		Severity:       entity.AdvisorySeverityCritical,
		Summary:        "first summary",
		State:          entity.AdvisoryStateOpen,
	}

	if err := repo.Upsert(ctx, base); err != nil {
		t.Fatalf("first Upsert: %v", err)
	}

	second := &entity.SecurityAdvisory{
		OrganizationID: orgID,
		GHSAPID:        "GHSA-dedup-5678",
		Severity:       entity.AdvisorySeverityMedium,
		Summary:        "second summary",
		State:          entity.AdvisoryStateOpen,
	}
	if err := repo.Upsert(ctx, second); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM security_advisories WHERE organization_id = ? AND ghsa_id = ?`,
		orgID, "GHSA-dedup-5678",
	).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected COUNT=1, got %d", count)
	}

	got, err := repo.GetByGHSAPID(ctx, orgID, "GHSA-dedup-5678")
	if err != nil {
		t.Fatalf("GetByGHSAPID: %v", err)
	}
	if got == nil || got.Summary != "second summary" {
		t.Fatalf("expected updated summary, got %+v", got)
	}
}

func TestSecurityAdvisoryRepository_ListByOrgFilters(t *testing.T) {
	db := openTestDB(t)
	orgID, _ := seedSecurityAdvisoryFixtures(t, db)
	repo := repository.NewSecurityAdvisoryRepository(db)
	ctx := context.Background()

	entries := []struct {
		ghsaID   string
		severity entity.AdvisorySeverity
		state    entity.AdvisoryState
	}{
		{"GHSA-open-high", entity.AdvisorySeverityHigh, entity.AdvisoryStateOpen},
		{"GHSA-open-low", entity.AdvisorySeverityLow, entity.AdvisoryStateOpen},
		{"GHSA-resolved-high", entity.AdvisorySeverityHigh, entity.AdvisoryStateResolved},
	}

	for _, entry := range entries {
		if err := repo.Create(ctx, &entity.SecurityAdvisory{
			OrganizationID: orgID,
			GHSAPID:        entry.ghsaID,
			Severity:       entry.severity,
			Summary:        entry.ghsaID,
			State:          entry.state,
		}); err != nil {
			t.Fatalf("Create %s: %v", entry.ghsaID, err)
		}
	}

	bySeverity, total, err := repo.ListByOrg(ctx, orgID, "", string(entity.AdvisorySeverityHigh), 1, 10)
	if err != nil {
		t.Fatalf("ListByOrg severity filter: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2 for high severity, got %d", total)
	}
	if len(bySeverity) != 2 {
		t.Fatalf("expected 2 advisories, got %d", len(bySeverity))
	}

	openOnly, total, err := repo.ListByOrg(ctx, orgID, string(entity.AdvisoryStateOpen), "", 1, 10)
	if err != nil {
		t.Fatalf("ListByOrg state filter: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2 for open state, got %d", total)
	}
	for _, adv := range openOnly {
		if adv.State != entity.AdvisoryStateOpen {
			t.Fatalf("expected open state, got %s", adv.State)
		}
	}
}

func TestSecurityAdvisoryRepository_UpdateState(t *testing.T) {
	db := openTestDB(t)
	orgID, _ := seedSecurityAdvisoryFixtures(t, db)
	repo := repository.NewSecurityAdvisoryRepository(db)
	ctx := context.Background()

	if err := repo.Create(ctx, &entity.SecurityAdvisory{
		OrganizationID: orgID,
		GHSAPID:        "GHSA-state-9999",
		Severity:       entity.AdvisorySeverityMedium,
		Summary:        "state test",
		State:          entity.AdvisoryStateOpen,
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	acknowledged, err := repo.UpdateState(ctx, orgID, "GHSA-state-9999", entity.AdvisoryStateAcknowledged, nil)
	if err != nil {
		t.Fatalf("UpdateState acknowledged: %v", err)
	}
	if acknowledged == nil || acknowledged.State != entity.AdvisoryStateAcknowledged {
		t.Fatalf("expected acknowledged state, got %+v", acknowledged)
	}

	resolved, err := repo.UpdateState(ctx, orgID, "GHSA-state-9999", entity.AdvisoryStateResolved, nil)
	if err != nil {
		t.Fatalf("UpdateState resolved: %v", err)
	}
	if resolved == nil || resolved.State != entity.AdvisoryStateResolved {
		t.Fatalf("expected resolved state, got %+v", resolved)
	}

	reason := entity.DismissedReasonNoBandwidth
	if err := repo.Create(ctx, &entity.SecurityAdvisory{
		OrganizationID: orgID,
		GHSAPID:        "GHSA-dismiss-0001",
		Severity:       entity.AdvisorySeverityLow,
		Summary:        "dismiss test",
		State:          entity.AdvisoryStateAcknowledged,
	}); err != nil {
		t.Fatalf("Create dismiss advisory: %v", err)
	}

	dismissed, err := repo.UpdateState(ctx, orgID, "GHSA-dismiss-0001", entity.AdvisoryStateDismissed, &reason)
	if err != nil {
		t.Fatalf("UpdateState dismissed: %v", err)
	}
	if dismissed == nil || dismissed.DismissedReason == nil || *dismissed.DismissedReason != reason {
		t.Fatalf("expected dismissed_reason %q, got %+v", reason, dismissed)
	}
}
