package runner

import (
	"context"
)

type OfficialRunnerAdapter struct{}

func NewOfficialRunnerAdapter() *OfficialRunnerAdapter {
	return &OfficialRunnerAdapter{}
}

func (a *OfficialRunnerAdapter) Execute(_ context.Context, _ RunnerJobPayload) error {
	return nil
}

func (a *OfficialRunnerAdapter) Cancel(_ context.Context, _ string) error {
	return nil
}
