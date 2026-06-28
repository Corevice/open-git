package perf

import (
	"fmt"

	"github.com/open-git/backend/internal/domain/entity"
)

func DetectRegression(current entity.PerfMetrics, baseline *entity.PerfBenchmark, maxPct float64) *entity.PerfRegressionResult {
	if baseline == nil {
		return nil
	}

	p95DeltaPct := (float64(current.P95Ms)-float64(baseline.Metrics.P95Ms)) / float64(baseline.Metrics.P95Ms) * 100
	flagged := p95DeltaPct > maxPct

	return &entity.PerfRegressionResult{
		VsBaseline: fmt.Sprintf("%+.1f%%", p95DeltaPct),
		Flagged:    flagged,
		DeltaPct:   p95DeltaPct,
	}
}
