package perf_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/usecase/perf"
)

func TestDetectRegression(t *testing.T) {
	baseline := &entity.PerfBenchmark{
		ID:           uuid.New(),
		ScenarioName: "rest-repos-read",
		Metrics: entity.PerfMetrics{
			P95Ms: 200,
		},
		CreatedAt: time.Now().UTC(),
	}

	t.Run("nil baseline returns nil", func(t *testing.T) {
		got := perf.DetectRegression(entity.PerfMetrics{P95Ms: 250}, nil, 20.0)
		if got != nil {
			t.Fatalf("DetectRegression() = %+v, want nil", got)
		}
	})

	t.Run("delta within max not flagged", func(t *testing.T) {
		got := perf.DetectRegression(entity.PerfMetrics{P95Ms: 220}, baseline, 20.0)
		if got == nil {
			t.Fatal("DetectRegression() = nil, want result")
		}
		if got.Flagged {
			t.Fatalf("Flagged = true, want false (delta=%f)", got.DeltaPct)
		}
	})

	t.Run("delta exceeds max flagged", func(t *testing.T) {
		got := perf.DetectRegression(entity.PerfMetrics{P95Ms: 260}, baseline, 20.0)
		if got == nil {
			t.Fatal("DetectRegression() = nil, want result")
		}
		if !got.Flagged {
			t.Fatalf("Flagged = false, want true (delta=%f)", got.DeltaPct)
		}
		if got.VsBaseline != "+30.0%" {
			t.Fatalf("VsBaseline = %q, want +30.0%%", got.VsBaseline)
		}
	})

	t.Run("negative delta improvement not flagged", func(t *testing.T) {
		got := perf.DetectRegression(entity.PerfMetrics{P95Ms: 150}, baseline, 20.0)
		if got == nil {
			t.Fatal("DetectRegression() = nil, want result")
		}
		if got.Flagged {
			t.Fatalf("Flagged = true, want false for improvement (delta=%f)", got.DeltaPct)
		}
		if got.VsBaseline != "-25.0%" {
			t.Fatalf("VsBaseline = %q, want -25.0%%", got.VsBaseline)
		}
	})
}
