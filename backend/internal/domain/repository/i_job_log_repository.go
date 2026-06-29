package repository

import (
	"context"

	"github.com/open-git/backend/internal/domain/entity"
)

type JobLogMeta struct {
	JobID          string
	OrganizationID string
	Status         string
	TotalLines     int64
}

type IJobLogRepository interface {
	AppendLines(ctx context.Context, lines []*entity.JobLogLine) error
	ListLines(ctx context.Context, orgID, jobID string, fromLine int64, limit int) ([]*entity.JobLogLine, error)
	CountLines(ctx context.Context, orgID, jobID string) (int64, error)
	SetMeta(ctx context.Context, meta *JobLogMeta) error
	GetMeta(ctx context.Context, orgID, jobID string) (*JobLogMeta, error)
}
