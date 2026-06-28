package repository

import (
	"context"

	"github.com/open-git/backend/internal/domain/entity"
)

type IAuditLogRepository interface {
	Create(ctx context.Context, log *entity.AuditLog) error
}
