package entity_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/open-git/backend/internal/domain/entity"
)

func TestCompatStatusConstants(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"CompatStatusQueued", entity.CompatStatusQueued, "queued"},
		{"CompatStatusRunning", entity.CompatStatusRunning, "running"},
		{"CompatStatusCompleted", entity.CompatStatusCompleted, "completed"},
		{"CompatStatusFailed", entity.CompatStatusFailed, "failed"},
		{"CompatResultPass", entity.CompatResultPass, "pass"},
		{"CompatResultFail", entity.CompatResultFail, "fail"},
		{"CompatResultUnimplemented", entity.CompatResultUnimplemented, "unimplemented"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.want {
				t.Fatalf("%s = %q, want %q", tt.name, tt.value, tt.want)
			}
		})
	}
}

func TestCompatTestRunZeroValueStatus(t *testing.T) {
	run := entity.CompatTestRun{}
	if run.Status != "" {
		t.Fatalf("Status = %q, want empty string", run.Status)
	}
}

func TestCompatEndpointChecksJSONSnakeCase(t *testing.T) {
	checks := entity.CompatEndpointChecks{
		Schema:     true,
		StatusCode: false,
		Headers:    true,
		Pagination: false,
	}

	data, err := json.Marshal(checks)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	jsonStr := string(data)
	for _, key := range []string{"schema", "status_code", "headers", "pagination"} {
		if !strings.Contains(jsonStr, `"`+key+`"`) {
			t.Fatalf("expected JSON to contain %q key, got %s", key, jsonStr)
		}
	}
}
