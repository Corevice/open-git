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

	after, before := normalizeAuditLogDateRange(input.After, input.Before)

	logs, total, err := uc.auditLogs.Search(ctx, domainrepo.AuditLogSearchInput{
		OrganizationID: input.OrganizationID,
		Phrase:         input.Phrase,
		Action:         input.Action,
		After:          after,
		Before:         before,
		Page:           input.Page,
		PerPage:        input.PerPage,
	})
	if err != nil {
		return nil, err
	}

	return &SearchAuditLogsOutput{Logs: logs, Total: total}, nil
}

func normalizeAuditLogDateRange(after, before *time.Time) (*time.Time, *time.Time) {
	if after == nil && before == nil {
		return nil, nil
	}

	now := time.Now().UTC()
	normAfter := after
	normBefore := before
	if normAfter == nil {
		derived := normBefore.Add(-MaxAuditLogRange)
		normAfter = &derived
	}
	if normBefore == nil {
		normBefore = &now
	}
	return normAfter, normBefore
}

func validateAuditLogDateRange(after, before *time.Time) error {
	if after == nil && before == nil {
		return nil
	}

	normAfter, normBefore := normalizeAuditLogDateRange(after, before)
	if normBefore.Before(*normAfter) {
		return fmt.Errorf("%w: before must be on or after after", apperror.ErrValidation)
	}
	if normBefore.Sub(*normAfter) > MaxAuditLogRange {
		return ErrDateRangeExceeded
	}
	return nil
}
