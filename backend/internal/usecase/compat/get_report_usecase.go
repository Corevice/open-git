package compat

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type ReportCoverage struct {
	TotalEndpoints int     `json:"total_endpoints"`
	Passing        int     `json:"passing"`
	Failing        int     `json:"failing"`
	Unimplemented  int     `json:"unimplemented"`
	Rate           float64 `json:"rate"`
}

type ReportEndpoint struct {
	Method  string                      `json:"method"`
	Path    string                      `json:"path"`
	Status  string                      `json:"status"`
	Checks  *entity.CompatEndpointChecks `json:"checks,omitempty"`
	Diff    json.RawMessage             `json:"diff,omitempty"`
	LastRun string                      `json:"last_run,omitempty"`
}

type ReportResponse struct {
	GeneratedAt string           `json:"generated_at"`
	Coverage    ReportCoverage   `json:"coverage"`
	Endpoints   []ReportEndpoint `json:"endpoints"`
}

type GetReportUsecase struct {
	repo domainrepo.ICompatRepository
}

func NewGetReportUsecase(repo domainrepo.ICompatRepository) *GetReportUsecase {
	return &GetReportUsecase{repo: repo}
}

func (uc *GetReportUsecase) Execute(ctx context.Context, orgID uuid.UUID) (*ReportResponse, error) {
	runs, err := uc.repo.ListRuns(ctx, orgID, 1)
	if err != nil {
		return nil, err
	}

	if len(runs) == 0 {
		return emptyReportResponse(), nil
	}

	latest := runs[0]
	results, err := uc.repo.ListEndpointResults(ctx, latest.ID)
	if err != nil {
		return nil, err
	}

	endpoints := make([]ReportEndpoint, 0, len(results))
	passing := 0
	failing := 0
	unimplemented := 0
	lastRun := formatReportTime(latest.FinishedAt, latest.StartedAt, latest.CreatedAt)

	for _, result := range results {
		switch result.Status {
		case entity.CompatResultPass:
			passing++
		case entity.CompatResultFail:
			failing++
		case entity.CompatResultUnimplemented:
			unimplemented++
		}

		endpoints = append(endpoints, ReportEndpoint{
			Method:  result.Method,
			Path:    result.Path,
			Status:  result.Status,
			Checks:  result.Checks,
			Diff:    result.Diff,
			LastRun: lastRun,
		})
	}

	total := len(endpoints)
	if total == 0 {
		total = latest.TotalEndpoints
		passing = latest.Passing
		failing = latest.Failing
		unimplemented = latest.Unimplemented
	}

	rate := 0.0
	if total > 0 {
		rate = float64(passing) / float64(total)
	}

	generatedAt := formatReportTime(latest.FinishedAt, latest.StartedAt, latest.CreatedAt)
	if generatedAt == "" {
		generatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	return &ReportResponse{
		GeneratedAt: generatedAt,
		Coverage: ReportCoverage{
			TotalEndpoints: total,
			Passing:        passing,
			Failing:        failing,
			Unimplemented:  unimplemented,
			Rate:           rate,
		},
		Endpoints: endpoints,
	}, nil
}

func emptyReportResponse() *ReportResponse {
	return &ReportResponse{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Coverage: ReportCoverage{
			TotalEndpoints: 0,
			Passing:        0,
			Failing:        0,
			Unimplemented:  0,
			Rate:           0,
		},
		Endpoints: []ReportEndpoint{},
	}
}

func formatReportTime(finishedAt, startedAt *time.Time, createdAt time.Time) string {
	switch {
	case finishedAt != nil:
		return finishedAt.UTC().Format(time.RFC3339)
	case startedAt != nil:
		return startedAt.UTC().Format(time.RFC3339)
	case !createdAt.IsZero():
		return createdAt.UTC().Format(time.RFC3339)
	default:
		return ""
	}
}
