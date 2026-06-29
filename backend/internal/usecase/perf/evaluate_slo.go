package perf

import "github.com/open-git/backend/internal/domain/entity"

func EvaluateSLO(metrics entity.PerfMetrics, threshold *entity.PerfSLOThreshold) entity.SLOResult {
	if threshold == nil {
		return entity.SLOSkipped
	}

	if threshold.P95MsMax != nil && metrics.P95Ms > *threshold.P95MsMax {
		return entity.SLOFail
	}
	if threshold.P99MsMax != nil && metrics.P99Ms > *threshold.P99MsMax {
		return entity.SLOFail
	}
	if threshold.ErrorRateMax != nil && metrics.ErrorRate > *threshold.ErrorRateMax {
		return entity.SLOFail
	}
	if threshold.ThroughputRPSMin != nil && metrics.ThroughputRPS < float64(*threshold.ThroughputRPSMin) {
		return entity.SLOFail
	}

	return entity.SLOPass
}
