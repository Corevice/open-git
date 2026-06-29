package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type AuditLogListOpts struct {
	OrgID   uuid.UUID
	Page    int
	PerPage int
	Action  string
	ActorID *uuid.UUID
	Since   *time.Time
	Until   *time.Time
}

type AuditLogRepository interface {
	IAuditLogFullRepository
	ListByOrg(ctx context.Context, opts AuditLogListOpts) ([]*entity.AuditLog, int64, error)
}
