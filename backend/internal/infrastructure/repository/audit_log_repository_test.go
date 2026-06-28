package repository_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/database"
	"github.com/open-git/backend/internal/infrastructure/repository"
)

type mockAuditLogRepository struct {
	orgID   uuid.UUID
	action  string
	page    int
	perPage int
	offset  int
}

func (m *mockAuditLogRepository) Create(_ context.Context, _ *entity.AuditLog) error {
	return nil
}

func (m *mockAuditLogRepository) InsertAuditLog(
	_ context.Context,
	_, _ uuid.UUID,
	_, _ string,
	_ uuid.UUID,
	_ json.RawMessage,
) error {
	return nil
}

func (m *mockAuditLogRepository) List(_ context.Context, orgID uuid.UUID, action string, page, perPage int) ([]*entity.AuditLog, int, error) {
	m.orgID = orgID
	m.action = action
	m.page = page
	m.perPage = perPage
	m.offset = (page - 1) * perPage
	return nil, 0, nil
}

func TestMockAuditLogRepository_ListContract(t *testing.T) {
	orgID := uuid.New()

	tests := []struct {
		name           string
		action         string
		page           int
		perPage        int
		wantAction     string
		wantOffset     int
	}{
		{
			name:       "no action filter first page",
			action:     "",
			page:       1,
			perPage:    30,
			wantAction: "",
			wantOffset: 0,
		},
		{
			name:       "action filter second page",
			action:     "repo.destroy",
			page:       2,
			perPage:    10,
			wantAction: "repo.destroy",
			wantOffset: 10,
		},
		{
			name:       "action filter third page custom perPage",
			action:     "org.add_member",
			page:       3,
			perPage:    25,
			wantAction: "org.add_member",
			wantOffset: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var repo domainrepo.IAuditLogRepository = &mockAuditLogRepository{}
			mock := repo.(*mockAuditLogRepository)

			_, _, err := repo.List(context.Background(), orgID, tt.action, tt.page, tt.perPage)
			if err != nil {
				t.Fatalf("List: %v", err)
			}

			if mock.orgID != orgID {
				t.Fatalf("orgID: got %v, want %v", mock.orgID, orgID)
			}
			if mock.action != tt.wantAction {
				t.Fatalf("action: got %q, want %q", mock.action, tt.wantAction)
			}
			if mock.page != tt.page {
				t.Fatalf("page: got %d, want %d", mock.page, tt.page)
			}
			if mock.perPage != tt.perPage {
				t.Fatalf("perPage: got %d, want %d", mock.perPage, tt.perPage)
			}
			if mock.offset != tt.wantOffset {
				t.Fatalf("offset: got %d, want %d", mock.offset, tt.wantOffset)
			}
		})
	}
}

func newAuditLogTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := database.RunMigrations(db, "sqlite", "../../../migrations"); err != nil {
		_ = db.Close()
		t.Fatalf("run migrations: %v", err)
	}
	if _, err := db.Exec(`ALTER TABLE audit_logs ADD COLUMN ip_address TEXT NOT NULL DEFAULT ''`); err != nil {
		_ = db.Close()
		t.Fatalf("add ip_address column: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return sqlx.NewDb(db, "sqlite3")
}

func TestListByOrg_FiltersAndPaginates(t *testing.T) {
	db := newAuditLogTestDB(t)
	var repo domainrepo.AuditLogRepository = repository.NewAuditLogRepository(db)

	orgA := createTestOrganization(t, db, "audit-org-a")
	orgB := createTestOrganization(t, db, "audit-org-b")
	actorA := createTestUser(t, db, "audit-actor-a")
	actorB := createTestUser(t, db, "audit-actor-b")

	orgAActions := []string{
		"settings.update",
		"settings.update",
		"settings.update",
		"member.invite",
		"member.invite",
	}
	for i, action := range orgAActions {
		log := &entity.AuditLog{
			OrganizationID: orgA,
			ActorID:        actorA,
			ActorLogin:     "audit-actor-a",
			Action:         action,
			TargetType:     "organization",
			TargetID:       uuid.New().String(),
			CreatedAt:      time.Date(2025, 6, 1, 10, i, 0, 0, time.UTC),
		}
		if err := repo.Create(context.Background(), log); err != nil {
			t.Fatalf("Create orgA log %d: %v", i, err)
		}
	}

	for i := 0; i < 3; i++ {
		log := &entity.AuditLog{
			OrganizationID: orgB,
			ActorID:        actorB,
			ActorLogin:     "audit-actor-b",
			Action:         "repo.delete",
			TargetType:     "repository",
			TargetID:       uuid.New().String(),
			CreatedAt:      time.Date(2025, 6, 2, 10, i, 0, 0, time.UTC),
		}
		if err := repo.Create(context.Background(), log); err != nil {
			t.Fatalf("Create orgB log %d: %v", i, err)
		}
	}

	ctx := context.Background()

	logs, total, err := repo.ListByOrg(ctx, domainrepo.AuditLogListOpts{
		OrgID:   orgA,
		Page:    1,
		PerPage: 100,
	})
	if err != nil {
		t.Fatalf("ListByOrg orgA: %v", err)
	}
	if total != 5 {
		t.Fatalf("expected total 5 for orgA, got %d", total)
	}
	if len(logs) != 5 {
		t.Fatalf("expected 5 logs for orgA, got %d", len(logs))
	}
	for _, log := range logs {
		if log.OrganizationID != orgA {
			t.Fatalf("expected orgA log, got org %s", log.OrganizationID)
		}
	}

	logs, total, err = repo.ListByOrg(ctx, domainrepo.AuditLogListOpts{
		OrgID:   orgA,
		Page:    1,
		PerPage: 2,
	})
	if err != nil {
		t.Fatalf("ListByOrg paginated: %v", err)
	}
	if total != 5 {
		t.Fatalf("expected total 5 with pagination, got %d", total)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs per page, got %d", len(logs))
	}

	logs, total, err = repo.ListByOrg(ctx, domainrepo.AuditLogListOpts{
		OrgID:  orgA,
		Action: "settings.update",
	})
	if err != nil {
		t.Fatalf("ListByOrg action filter: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected total 3 for settings.update, got %d", total)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 logs for settings.update, got %d", len(logs))
	}
	for _, log := range logs {
		if log.Action != "settings.update" {
			t.Fatalf("expected action settings.update, got %q", log.Action)
		}
	}
}

func TestAuditLogRowToEntity(t *testing.T) {
	id := uuid.New()
	orgID := uuid.New()
	actorID := uuid.New()
	createdAt := time.Date(2025, 1, 15, 9, 24, 0, 0, time.UTC)

	tests := []struct {
		name    string
		row     repository.AuditLogRow
		want    *entity.AuditLog
		wantErr bool
	}{
		{
			name: "parses UUIDs and metadata JSON",
			row: repository.AuditLogRow{
				ID:             id.String(),
				OrganizationID: orgID.String(),
				ActorID:        actorID.String(),
				ActorLogin:     "octocat",
				Action:         "repo.destroy",
				TargetType:     "repository",
				TargetID:       "repo-123",
				Metadata:       `{"visibility":"private"}`,
				CreatedAt:      createdAt,
			},
			want: &entity.AuditLog{
				ID:             id,
				OrganizationID: orgID,
				ActorID:        actorID,
				ActorLogin:     "octocat",
				Action:         "repo.destroy",
				TargetType:     "repository",
				TargetID:       "repo-123",
				Metadata:       map[string]any{"visibility": "private"},
				CreatedAt:      createdAt,
			},
		},
		{
			name: "empty metadata",
			row: repository.AuditLogRow{
				ID:             id.String(),
				OrganizationID: orgID.String(),
				ActorID:        actorID.String(),
				ActorLogin:     "bot",
				Action:         "token.create",
				TargetType:     "token",
				TargetID:       "tok-1",
				Metadata:       "",
				CreatedAt:      createdAt,
			},
			want: &entity.AuditLog{
				ID:             id,
				OrganizationID: orgID,
				ActorID:        actorID,
				ActorLogin:     "bot",
				Action:         "token.create",
				TargetType:     "token",
				TargetID:       "tok-1",
				Metadata:       nil,
				CreatedAt:      createdAt,
			},
		},
		{
			name: "invalid actor UUID",
			row: repository.AuditLogRow{
				ID:             id.String(),
				OrganizationID: orgID.String(),
				ActorID:        "not-a-uuid",
				ActorLogin:     "octocat",
				Action:         "repo.destroy",
				TargetType:     "repository",
				TargetID:       "repo-123",
				Metadata:       `{}`,
				CreatedAt:      createdAt,
			},
			wantErr: true,
		},
		{
			name: "invalid metadata JSON",
			row: repository.AuditLogRow{
				ID:             id.String(),
				OrganizationID: orgID.String(),
				ActorID:        actorID.String(),
				ActorLogin:     "octocat",
				Action:         "repo.destroy",
				TargetType:     "repository",
				TargetID:       "repo-123",
				Metadata:       `{invalid`,
				CreatedAt:      createdAt,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repository.AuditLogRowToEntity(tt.row)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("AuditLogRowToEntity: %v", err)
			}
			if got.ID != tt.want.ID || got.OrganizationID != tt.want.OrganizationID || got.ActorID != tt.want.ActorID {
				t.Fatalf("unexpected IDs: %+v", got)
			}
			if got.ActorLogin != tt.want.ActorLogin || got.Action != tt.want.Action {
				t.Fatalf("unexpected actor/action: %+v", got)
			}
			if got.TargetType != tt.want.TargetType || got.TargetID != tt.want.TargetID {
				t.Fatalf("unexpected target: %+v", got)
			}
			if !got.CreatedAt.Equal(tt.want.CreatedAt) {
				t.Fatalf("createdAt: got %v, want %v", got.CreatedAt, tt.want.CreatedAt)
			}
			if tt.want.Metadata == nil {
				if got.Metadata != nil {
					t.Fatalf("metadata: got %v, want nil", got.Metadata)
				}
				return
			}
			if got.Metadata["visibility"] != tt.want.Metadata["visibility"] {
				t.Fatalf("metadata: got %v, want %v", got.Metadata, tt.want.Metadata)
			}
		})
	}
}
