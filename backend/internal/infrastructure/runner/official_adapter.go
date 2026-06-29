package runner

import (
	"context"
	"fmt"
)

type OfficialRunnerAdapter struct{}

func NewOfficialRunnerAdapter() *OfficialRunnerAdapter {
	return &OfficialRunnerAdapter{}
}

func (a *OfficialRunnerAdapter) Execute(_ context.Context, _ ActJobPayload) error {
	return fmt.Errorf("official runner executes via polling - call acquire API")
}

func (a *OfficialRunnerAdapter) Cancel(_ context.Context, _ string) error {
	return nil
}
