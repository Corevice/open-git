package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

type IAuditLogRepository interface {
	InsertAuditLog(ctx context.Context, orgID, actorID uuid.UUID, action, targetType string, targetID uuid.UUID, metadata json.RawMessage) error
}
