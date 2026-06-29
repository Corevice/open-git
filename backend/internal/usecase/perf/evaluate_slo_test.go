package perf_test

import (
	"testing"

	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/usecase/perf"
)

func intPtr(v int) *int {
	return &v
}

func float64Ptr(v float64) *float64 {
	return &v
}

func TestEvaluateSLO(t *testing.T) {
	tests := []struct {
		name      string
		metrics   entity.PerfMetrics
		threshold *entity.PerfSLOThreshold
		want      entity.SLOResult
	}{
		{
			name: "p95 within bounds passes",
			metrics: entity.PerfMetrics{
				P50Ms:         80,
				P95Ms:         250,
				P99Ms:         600,
				ThroughputRPS: 1200,
				ErrorRate:     0.001,
			},
			threshold: &entity.PerfSLOThreshold{
				P95MsMax:         intPtr(300),
				P99MsMax:         intPtr(800),
				ErrorRateMax:     float64Ptr(0.005),
				ThroughputRPSMin: intPtr(1000),
			},
			want: entity.SLOPass,
		},
		{
			name: "p95 exceeds fails",
			metrics: entity.PerfMetrics{
				P95Ms:         350,
				P99Ms:         600,
				ThroughputRPS: 1200,
				ErrorRate:     0.001,
			},
			threshold: &entity.PerfSLOThreshold{
				P95MsMax: intPtr(300),
			},
			want: entity.SLOFail,
		},
		{
			name: "error_rate exceeds fails",
			metrics: entity.PerfMetrics{
				P95Ms:         250,
				P99Ms:         600,
				ThroughputRPS: 1200,
				ErrorRate:     0.01,
			},
			threshold: &entity.PerfSLOThreshold{
				ErrorRateMax: float64Ptr(0.005),
			},
			want: entity.SLOFail,
		},
		{
			name: "throughput below min fails",
			metrics: entity.PerfMetrics{
				P95Ms:         250,
				P99Ms:         600,
				ThroughputRPS: 500,
				ErrorRate:     0.001,
			},
			threshold: &entity.PerfSLOThreshold{
				ThroughputRPSMin: intPtr(1000),
			},
			want: entity.SLOFail,
		},
		{
			name: "nil threshold skipped",
			metrics: entity.PerfMetrics{
				P95Ms:         250,
				P99Ms:         600,
				ThroughputRPS: 1200,
				ErrorRate:     0.001,
			},
			threshold: nil,
			want:      entity.SLOSkipped,
		},
		{
			name: "all fields nil in threshold passes",
			metrics: entity.PerfMetrics{
				P95Ms:         250,
				P99Ms:         600,
				ThroughputRPS: 1200,
				ErrorRate:     0.001,
			},
			threshold: &entity.PerfSLOThreshold{},
			want:      entity.SLOPass,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := perf.EvaluateSLO(tt.metrics, tt.threshold)
			if got != tt.want {
				t.Fatalf("EvaluateSLO() = %q, want %q", got, tt.want)
			}
		})
	}
}
