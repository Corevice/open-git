package workflow

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type ListWorkflowRunsFilter struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	Status         string
	Conclusion     string
	Branch         string
	Event          string
	Page           int
	PerPage        int
}

type WorkflowRunRepository interface {
	ListByHeadSHA(ctx context.Context, repoID uuid.UUID, sha string) ([]*entity.WorkflowRun, error)
	ListByRepo(ctx context.Context, filter ListWorkflowRunsFilter) ([]*entity.WorkflowRun, int, error)
	GetByID(ctx context.Context, orgID, runID uuid.UUID) (*entity.WorkflowRun, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status, conclusion string, completedAt *time.Time) error
	Create(ctx context.Context, run *entity.WorkflowRun) error
}

type WorkflowJobRepository interface {
	ListByRunID(ctx context.Context, orgID, runID uuid.UUID) ([]*entity.WorkflowJob, error)
}

type WorkflowArtifact struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	RunID          uuid.UUID
	Name           string
	SizeInBytes    int64
	Expired        bool
}

type ArtifactRepository interface {
	ListByRunID(ctx context.Context, runID, orgID uuid.UUID) ([]*WorkflowArtifact, error)
}
