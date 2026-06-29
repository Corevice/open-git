package entity_test

import (
	"testing"

	"github.com/open-git/backend/internal/domain/entity"
)

func TestLogStreamConstants(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"LogStreamStdout", entity.LogStreamStdout, "stdout"},
		{"LogStreamStderr", entity.LogStreamStderr, "stderr"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.want {
				t.Fatalf("%s = %q, want %q", tt.name, tt.value, tt.want)
			}
		})
	}
}

func TestJobLogLineZeroValueStream(t *testing.T) {
	line := entity.JobLogLine{}
	if line.Stream != "" {
		t.Fatalf("Stream = %q, want empty string", line.Stream)
	}
}
