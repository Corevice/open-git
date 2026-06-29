package repository_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/entity"
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
				IPAddress:      "192.168.1.10",
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
				IPAddress:      "192.168.1.10",
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
			if got.IPAddress != tt.want.IPAddress {
				t.Fatalf("ipAddress: got %q, want %q", got.IPAddress, tt.want.IPAddress)
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

func TestAuditLogRepository_IPAddressRoundTrip(t *testing.T) {
	db := openTestDB(t)
	setupAuditLogSearchColumns(t, db)
	orgID, actorID := seedAuditLogSearchFixtures(t, db)
	repo := repository.NewAuditLogRepository(db)
	ctx := context.Background()

	logID := uuid.New()
	log := &entity.AuditLog{
		ID:             logID,
		OrganizationID: orgID,
		ActorID:        actorID,
		ActorLogin:     "alice",
		Action:         "settings.update",
		TargetType:     "system_setting",
		TargetID:       "site.name",
		IPAddress:      "203.0.113.42",
		Metadata:       map[string]any{"key": "site.name"},
		CreatedAt:      time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
	}

	if err := repo.Create(ctx, log); err != nil {
		t.Fatalf("Create: %v", err)
	}

	logs, _, err := repo.List(ctx, orgID, "settings.update", 1, 100)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	var found *entity.AuditLog
	for _, entry := range logs {
		if entry.ID == logID {
			found = entry
			break
		}
	}
	if found == nil {
		t.Fatalf("expected log %v in list (len=%d)", logID, len(logs))
	}
	if found.IPAddress != "203.0.113.42" {
		t.Fatalf("IPAddress: got %q, want %q", found.IPAddress, "203.0.113.42")
	}
}

func TestAuditLogRepository_CreateIPAddressValidation(t *testing.T) {
	db := openTestDB(t)
	setupAuditLogSearchColumns(t, db)
	orgID, actorID := seedAuditLogSearchFixtures(t, db)
	repo := repository.NewAuditLogRepository(db)
	ctx := context.Background()

	base := &entity.AuditLog{
		OrganizationID: orgID,
		ActorID:        actorID,
		ActorLogin:     "alice",
		Action:         "settings.update",
		TargetType:     "system_setting",
		TargetID:       "site.name",
		CreatedAt:      time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name      string
		ipAddress string
		wantErr   bool
	}{
		{name: "empty ip address allowed", ipAddress: "", wantErr: false},
		{name: "valid ipv4", ipAddress: "203.0.113.42", wantErr: false},
		{name: "invalid ip format rejected", ipAddress: "not-an-ip", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := *base
			entry.ID = uuid.New()
			entry.IPAddress = tt.ipAddress

			err := repo.Create(ctx, &entry)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
		})
	}
}
