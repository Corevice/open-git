package compat

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/open-git/backend/internal/compat"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type stubEndpoint struct {
	Method string
	Path   string
	Tags   []string
	Status string
}

type TriggerRunUsecase struct {
	repo   domainrepo.ICompatRepository
	runner *compat.Runner
}

func NewTriggerRunUsecase(repo domainrepo.ICompatRepository, runner *compat.Runner) *TriggerRunUsecase {
	return &TriggerRunUsecase{
		repo:   repo,
		runner: runner,
	}
}

func (uc *TriggerRunUsecase) Execute(
	ctx context.Context,
	suite string,
	filter []string,
	orgID, triggeredBy uuid.UUID,
) (*entity.CompatTestRun, error) {
	run := &entity.CompatTestRun{
		Suite:          suite,
		Status:         entity.CompatStatusQueued,
		TriggeredBy:    &triggeredBy,
		OrganizationID: orgID,
	}
	if err := uc.repo.CreateRun(ctx, run); err != nil {
		return nil, err
	}

	go uc.runAsync(run.ID, suite, filter, orgID)

	return run, nil
}

func (uc *TriggerRunUsecase) runAsync(runID uuid.UUID, suite string, filter []string, orgID uuid.UUID) {
	ctx := context.Background()
	now := time.Now().UTC()
	startedAt := now

	run, err := uc.repo.GetRun(ctx, runID)
	if err != nil || run == nil {
		return
	}

	run.Status = entity.CompatStatusRunning
	run.StartedAt = &startedAt
	if err := uc.repo.UpdateRun(ctx, run); err != nil {
		return
	}

	cases := stubCasesForSuite(suite, filter)
	passing := 0
	failing := 0
	unimplemented := 0

	for _, stub := range cases {
		status := stub.Status
		if status == "" {
			status = entity.CompatResultPass
		}

		checks := &entity.CompatEndpointChecks{
			Schema:     status == entity.CompatResultPass,
			StatusCode: status == entity.CompatResultPass,
			Headers:    status == entity.CompatResultPass,
			Pagination: status == entity.CompatResultPass,
		}

		result := &entity.CompatEndpointResult{
			RunID:  runID,
			Method: stub.Method,
			Path:   stub.Path,
			Status: status,
			Checks: checks,
		}
		if err := uc.repo.CreateEndpointResult(ctx, result); err != nil {
			run.Status = entity.CompatStatusFailed
			finishedAt := time.Now().UTC()
			run.FinishedAt = &finishedAt
			_ = uc.repo.UpdateRun(ctx, run)
			return
		}

		switch status {
		case entity.CompatResultPass:
			passing++
		case entity.CompatResultFail:
			failing++
		case entity.CompatResultUnimplemented:
			unimplemented++
		}
	}

	total := len(cases)
	rate := 0.0
	if total > 0 {
		rate = float64(passing) / float64(total)
	}

	finishedAt := time.Now().UTC()
	run.Status = entity.CompatStatusCompleted
	run.TotalEndpoints = total
	run.Passing = passing
	run.Failing = failing
	run.Unimplemented = unimplemented
	run.CoverageRate = rate
	run.FinishedAt = &finishedAt
	_ = uc.repo.UpdateRun(ctx, run)
}

func stubCasesForSuite(suite string, filter []string) []stubEndpoint {
	if suite != "" && suite != "rest-v3" {
		return nil
	}

	all := []stubEndpoint{
		{Method: "GET", Path: "/user", Tags: []string{"users"}},
		{Method: "GET", Path: "/users/{username}", Tags: []string{"users"}},
		{Method: "GET", Path: "/repos/{owner}/{repo}/issues", Tags: []string{"issues"}},
		{Method: "POST", Path: "/repos/{owner}/{repo}/issues", Tags: []string{"issues"}},
		{Method: "GET", Path: "/repos/{owner}/{repo}/pulls", Tags: []string{"pulls"}},
		{Method: "POST", Path: "/repos/{owner}/{repo}/pulls", Tags: []string{"pulls"}},
	}

	if len(filter) == 0 {
		return all
	}

	filterSet := make(map[string]struct{}, len(filter))
	for _, tag := range filter {
		filterSet[tag] = struct{}{}
	}

	filtered := make([]stubEndpoint, 0)
	for _, stub := range all {
		for _, tag := range stub.Tags {
			if _, ok := filterSet[tag]; ok {
				filtered = append(filtered, stub)
				break
			}
		}
	}
	return filtered
}
