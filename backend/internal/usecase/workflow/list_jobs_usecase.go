package workflow

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type WorkflowJob struct {
	ID          uuid.UUID
	RunID       uuid.UUID
	Name        string
	Status      string
	Conclusion  string
	StartedAt   *time.Time
	CompletedAt *time.Time
}

type WorkflowStep struct {
	ID         uuid.UUID
	JobID      uuid.UUID
	Number     int
	Name       string
	Status     string
	Conclusion string
	StartedAt  *time.Time
	CompletedAt *time.Time
}

type JobLog struct {
	ID     uuid.UUID
	JobID  uuid.UUID
	Offset int64
	Chunk  string
}

type ListJobsInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	RunID          uuid.UUID
}

type listJobsRepository interface {
	ListByRunID(ctx context.Context, orgID, runID uuid.UUID) ([]*WorkflowJob, error)
}

type ListJobsUsecase struct {
	jobRepo listJobsRepository
}

func NewListJobsUsecase(jobRepo listJobsRepository) *ListJobsUsecase {
	return &ListJobsUsecase{jobRepo: jobRepo}
}

func (uc *ListJobsUsecase) Execute(ctx context.Context, input ListJobsInput) ([]*WorkflowJob, error) {
	return uc.jobRepo.ListByRunID(ctx, input.OrganizationID, input.RunID)
}

type GetJobInput struct {
	OrganizationID uuid.UUID
	JobID          uuid.UUID
}

type getJobRepository interface {
	GetByID(ctx context.Context, orgID, jobID uuid.UUID) (*WorkflowJob, error)
}

type GetJobUsecase struct {
	jobRepo getJobRepository
}

func NewGetJobUsecase(jobRepo getJobRepository) *GetJobUsecase {
	return &GetJobUsecase{jobRepo: jobRepo}
}

func (uc *GetJobUsecase) Execute(ctx context.Context, input GetJobInput) (*WorkflowJob, error) {
	return uc.jobRepo.GetByID(ctx, input.OrganizationID, input.JobID)
}

type ListStepsInput struct {
	OrganizationID uuid.UUID
	JobID          uuid.UUID
}

type listStepsRepository interface {
	ListByJobID(ctx context.Context, orgID, jobID uuid.UUID) ([]*WorkflowStep, error)
}

type ListStepsUsecase struct {
	stepRepo listStepsRepository
}

func NewListStepsUsecase(stepRepo listStepsRepository) *ListStepsUsecase {
	return &ListStepsUsecase{stepRepo: stepRepo}
}

func (uc *ListStepsUsecase) Execute(ctx context.Context, input ListStepsInput) ([]*WorkflowStep, error) {
	return uc.stepRepo.ListByJobID(ctx, input.OrganizationID, input.JobID)
}

type JobLogRepository interface {
	ListChunks(ctx context.Context, jobID uuid.UUID, afterOffset int64) ([]*JobLog, error)
}

type WorkflowJobRepository interface {
	GetByID(ctx context.Context, orgID, jobID uuid.UUID) (*WorkflowJob, error)
}
