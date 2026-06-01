package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type AuditLogRepository struct {
	db *sqlx.DB
}

func NewAuditLogRepository(db *sqlx.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) InsertAuditLog(
	ctx context.Context,
	orgID, actorID uuid.UUID,
	action, targetType string,
	targetID uuid.UUID,
	metadata json.RawMessage,
) error {
	if metadata == nil {
		metadata = json.RawMessage(`{}`)
	}

	const query = `
		INSERT INTO audit_logs (id, organization_id, actor_id, action, target_type, target_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		uuid.New(),
		orgID,
		actorID,
		action,
		targetType,
		targetID,
		metadata,
		time.Now().UTC(),
	)
	return err
}
