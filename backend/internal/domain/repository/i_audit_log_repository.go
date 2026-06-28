package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type IAuditLogRepository interface {
	Create(ctx context.Context, log *entity.AuditLog) error
	List(ctx context.Context, orgID uuid.UUID, action string, page, perPage int) ([]*entity.AuditLog, int, error)
	InsertAuditLog(
		ctx context.Context,
		orgID, actorID uuid.UUID,
		action, targetType string,
		targetID uuid.UUID,
		metadata json.RawMessage,
	) error
}
