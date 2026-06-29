package workflow

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain/entity"
)

type ListRunsInput struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
	Status         string
	Branch         string
	Event          string
	Actor          string
	Page           int
	PerPage        int
}

type ListRunsOutput struct {
	Runs    []*entity.WorkflowRun
	Total   int
	Page    int
	PerPage int
}

type ListRunsFilter struct {
	OrganizationID uuid.UUID
	RepositoryID   uuid.UUID
}

type listRunsRepository interface {
	List(ctx context.Context, filter ListRunsFilter) ([]*entity.WorkflowRun, int, error)
}

type ListRunsUsecase struct {
	runRepo listRunsRepository
}

func NewListRunsUsecase(runRepo listRunsRepository) *ListRunsUsecase {
	return &ListRunsUsecase{runRepo: runRepo}
}

func (uc *ListRunsUsecase) Execute(ctx context.Context, input ListRunsInput) (*ListRunsOutput, error) {
	page := input.Page
	if page < 1 {
		page = 1
	}
	perPage := input.PerPage
	if perPage < 1 {
		perPage = 30
	}
	if perPage > 100 {
		perPage = 100
	}

	runs, _, err := uc.runRepo.List(ctx, ListRunsFilter{
		OrganizationID: input.OrganizationID,
		RepositoryID:   input.RepositoryID,
	})
	if err != nil {
		return nil, err
	}

	filtered := filterRuns(runs, input.Status, input.Branch, input.Event, input.Actor)
	total := len(filtered)

	return &ListRunsOutput{
		Runs:    paginateRuns(filtered, page, perPage),
		Total:   total,
		Page:    page,
		PerPage: perPage,
	}, nil
}

func paginateRuns(runs []*entity.WorkflowRun, page, perPage int) []*entity.WorkflowRun {
	start := (page - 1) * perPage
	if start >= len(runs) {
		return []*entity.WorkflowRun{}
	}
	end := start + perPage
	if end > len(runs) {
		end = len(runs)
	}
	return runs[start:end]
}

func filterRuns(runs []*entity.WorkflowRun, status, branch, event, actor string) []*entity.WorkflowRun {
	if status == "" && branch == "" && event == "" && actor == "" {
		return runs
	}

	filtered := make([]*entity.WorkflowRun, 0, len(runs))
	for _, run := range runs {
		if status != "" && !matchesRunStatus(run, status) {
			continue
		}
		if branch != "" && runHeadBranch(run) != branch {
			continue
		}
		if event != "" && runEvent(run) != event {
			continue
		}
		if actor != "" && !strings.EqualFold(runActorLogin(run), actor) {
			continue
		}
		filtered = append(filtered, run)
	}
	return filtered
}

func matchesRunStatus(run *entity.WorkflowRun, status string) bool {
	switch status {
	case "success":
		return run.Conclusion == entity.WorkflowConclusionSuccess || run.Status == "success"
	case "failure":
		return run.Conclusion == "failure" || run.Status == "failure"
	case "cancelled":
		return run.Conclusion == "cancelled" || run.Status == "cancelled"
	default:
		return run.Status == status || run.Conclusion == status
	}
}

func runHeadBranch(run *entity.WorkflowRun) string {
	return run.HeadBranch
}

func runEvent(run *entity.WorkflowRun) string {
	return run.Event
}

func runActorLogin(run *entity.WorkflowRun) string {
	return run.ActorLogin
}
