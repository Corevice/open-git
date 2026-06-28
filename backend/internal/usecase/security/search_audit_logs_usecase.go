package security

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

const MaxAuditLogRange = 90 * 24 * time.Hour

var ErrDateRangeExceeded = errors.New("audit log date range must not exceed 90 days")

type SearchAuditLogsInput struct {
	OrganizationID uuid.UUID
	Phrase         string
	Action         string
	After          *time.Time
	Before         *time.Time
	Page           int
	PerPage        int
}

type SearchAuditLogsOutput struct {
	Logs  []*entity.AuditLog
	Total int
}

type SearchAuditLogsUsecase struct {
	auditLogs domainrepo.IAuditLogSearchRepository
}

func NewSearchAuditLogsUsecase(repo domainrepo.IAuditLogSearchRepository) *SearchAuditLogsUsecase {
	return &SearchAuditLogsUsecase{auditLogs: repo}
}

func (uc *SearchAuditLogsUsecase) Execute(ctx context.Context, input SearchAuditLogsInput) (*SearchAuditLogsOutput, error) {
	if err := validateAuditLogDateRange(input.After, input.Before); err != nil {
		return nil, err
	}

	logs, total, err := uc.auditLogs.Search(ctx, domainrepo.AuditLogSearchInput{
		OrganizationID: input.OrganizationID,
		Phrase:         input.Phrase,
		Action:         input.Action,
		After:          input.After,
		Before:         input.Before,
		Page:           input.Page,
		PerPage:        input.PerPage,
	})
	if err != nil {
		return nil, err
	}

	return &SearchAuditLogsOutput{Logs: logs, Total: total}, nil
}

func validateAuditLogDateRange(after, before *time.Time) error {
	if after == nil || before == nil {
		return nil
	}
	if before.Before(*after) {
		return fmt.Errorf("%w: before must be on or after after", apperror.ErrValidation)
	}
	if before.Sub(*after) > MaxAuditLogRange {
		return ErrDateRangeExceeded
	}
	return nil
}
