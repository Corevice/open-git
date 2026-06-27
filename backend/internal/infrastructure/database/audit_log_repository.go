package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	repo "github.com/open-git/backend/internal/repository"
)

type sqlAuditLogRepository struct {
	db *sql.DB
}

func NewAuditLogRepository(db *sql.DB) repo.IAuditLogRepository {
	return &sqlAuditLogRepository{db: db}
}

func (r *sqlAuditLogRepository) Record(
	ctx context.Context,
	orgID, actorID uuid.UUID,
	action, targetType string,
	targetID uuid.UUID,
	metadata map[string]any,
) error {
	metaJSON := []byte("{}")
	if metadata != nil {
		encoded, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		metaJSON = encoded
	}

	const query = `
		INSERT INTO audit_logs (id, organization_id, actor_id, action, target_type, target_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		fmt.Sprintf("%x", uuid.New()),
		orgID.String(),
		actorID.String(),
		action,
		targetType,
		targetID.String(),
		string(metaJSON),
		time.Now().UTC(),
	)
	return err
}
