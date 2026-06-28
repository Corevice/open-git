package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/domain/entity"
)

type sqlxAuditLogRepository struct {
	db *sqlx.DB
}

var _ domainrepo.IAuditLogRepository = (*sqlxAuditLogRepository)(nil)

func NewAuditLogRepository(db *sqlx.DB) domainrepo.IAuditLogRepository {
	return &sqlxAuditLogRepository{db: db}
}

func (r *sqlxAuditLogRepository) Create(ctx context.Context, log *entity.AuditLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	now := time.Now().UTC()
	if log.CreatedAt.IsZero() {
		log.CreatedAt = now
	}

	metaJSON := []byte("{}")
	if log.Metadata != nil {
		encoded, err := json.Marshal(log.Metadata)
		if err != nil {
			return err
		}
		metaJSON = encoded
	}

	const query = `
		INSERT INTO audit_logs (id, organization_id, actor_id, action, target_type, target_id, metadata, created_at)
		VALUES (:id, :organization_id, :actor_id, :action, :target_type, :target_id, :metadata, :created_at)
	`

	_, err := r.db.NamedExecContext(ctx, query, map[string]any{
		"id":              log.ID,
		"organization_id": log.OrganizationID,
		"actor_id":        log.ActorID,
		"action":          log.Action,
		"target_type":     log.TargetType,
		"target_id":       log.TargetID,
		"metadata":        string(metaJSON),
		"created_at":      log.CreatedAt,
	})
	return err
}
