package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type ListAuditLogsInput struct {
	OrganizationID uuid.UUID
	Action         string
	Page           int
	PerPage        int
}

type ListAuditLogsOutput struct {
	Logs  []*entity.AuditLog
	Total int
}

type ListAuditLogsUsecase struct {
	auditLogs domainrepo.IAuditLogRepository
}

func NewListAuditLogsUsecase(repo domainrepo.IAuditLogRepository) *ListAuditLogsUsecase {
	return &ListAuditLogsUsecase{auditLogs: repo}
}

func (u *ListAuditLogsUsecase) Execute(ctx context.Context, input ListAuditLogsInput) (*ListAuditLogsOutput, error) {
	logs, total, err := u.auditLogs.List(ctx, input.OrganizationID, input.Action, input.Page, input.PerPage)
	if err != nil {
		return nil, err
	}
	return &ListAuditLogsOutput{Logs: logs, Total: total}, nil
}
