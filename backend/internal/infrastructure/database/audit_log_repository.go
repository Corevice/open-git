package database

import (
	"context"
	"database/sql"
	"encoding/json"
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

	var actorLogin string
	if err := r.db.QueryRowContext(ctx, `SELECT login FROM users WHERE id = $1`, actorID.String()).Scan(&actorLogin); err != nil {
		actorLogin = ""
	}

	const query = `
		INSERT INTO audit_logs (id, organization_id, actor_id, actor_login, action, target_type, target_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		uuid.New().String(),
		orgID.String(),
		actorID.String(),
		actorLogin,
		action,
		targetType,
		targetID.String(),
		string(metaJSON),
		time.Now().UTC(),
	)
	return err
}
