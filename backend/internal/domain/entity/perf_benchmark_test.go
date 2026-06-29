package entity_test

import (
	"testing"

	"github.com/open-git/backend/internal/domain/entity"
)

func TestPerfMetricsValidate(t *testing.T) {
	tests := []struct {
		name    string
		metrics entity.PerfMetrics
		wantErr bool
	}{
		{
			name: "valid monotonic values",
			metrics: entity.PerfMetrics{
				P50Ms:         80,
				P95Ms:         250,
				P99Ms:         600,
				ThroughputRPS: 1200,
				ErrorRate:     0.001,
				TotalRequests: 360000,
			},
			wantErr: false,
		},
		{
			name: "fails when P50 > P95",
			metrics: entity.PerfMetrics{
				P50Ms:         300,
				P95Ms:         250,
				P99Ms:         600,
				ThroughputRPS: 1200,
				ErrorRate:     0.001,
				TotalRequests: 360000,
			},
			wantErr: true,
		},
		{
			name: "fails when P95 > P99",
			metrics: entity.PerfMetrics{
				P50Ms:         80,
				P95Ms:         700,
				P99Ms:         600,
				ThroughputRPS: 1200,
				ErrorRate:     0.001,
				TotalRequests: 360000,
			},
			wantErr: true,
		},
		{
			name: "fails when ErrorRate < 0",
			metrics: entity.PerfMetrics{
				P50Ms:         80,
				P95Ms:         250,
				P99Ms:         600,
				ThroughputRPS: 1200,
				ErrorRate:     -0.001,
				TotalRequests: 360000,
			},
			wantErr: true,
		},
		{
			name: "fails when ErrorRate > 1",
			metrics: entity.PerfMetrics{
				P50Ms:         80,
				P95Ms:         250,
				P99Ms:         600,
				ThroughputRPS: 1200,
				ErrorRate:     1.001,
				TotalRequests: 360000,
			},
			wantErr: true,
		},
		{
			name: "fails when P50Ms < 0",
			metrics: entity.PerfMetrics{
				P50Ms:         -1,
				P95Ms:         250,
				P99Ms:         600,
				ThroughputRPS: 1200,
				ErrorRate:     0.001,
				TotalRequests: 360000,
			},
			wantErr: true,
		},
		{
			name: "passes for boundary values",
			metrics: entity.PerfMetrics{
				P50Ms:         100,
				P95Ms:         100,
				P99Ms:         100,
				ThroughputRPS: 0,
				ErrorRate:     0,
				TotalRequests: 0,
			},
			wantErr: false,
		},
		{
			name: "passes when ErrorRate is 1",
			metrics: entity.PerfMetrics{
				P50Ms:         100,
				P95Ms:         100,
				P99Ms:         100,
				ThroughputRPS: 0,
				ErrorRate:     1,
				TotalRequests: 0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metrics.Validate()
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
