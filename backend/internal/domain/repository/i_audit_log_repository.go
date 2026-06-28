package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type AuditLogSearchInput struct {
	OrganizationID uuid.UUID
	Phrase         string
	Action         string
	After          *time.Time
	Before         *time.Time
	Page           int
	PerPage        int
}

type IAuditLogRepository interface {
	Create(ctx context.Context, log *entity.AuditLog) error
	List(ctx context.Context, orgID uuid.UUID, action string, page, perPage int) ([]*entity.AuditLog, int, error)
	Search(ctx context.Context, input AuditLogSearchInput) ([]*entity.AuditLog, int, error)
	InsertAuditLog(
		ctx context.Context,
		orgID, actorID uuid.UUID,
		action, targetType string,
		targetID uuid.UUID,
		metadata json.RawMessage,
	) error
}
