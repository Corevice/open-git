package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type BenchmarkStatus string

const (
	StatusCompleted BenchmarkStatus = "completed"
	StatusPartial   BenchmarkStatus = "partial"
	StatusFailed    BenchmarkStatus = "failed"
	StatusTimeout   BenchmarkStatus = "timeout"
	StatusInvalid   BenchmarkStatus = "invalid"
)

type SLOResult string

const (
	SLOPass    SLOResult = "pass"
	SLOFail    SLOResult = "fail"
	SLOSkipped SLOResult = "skipped"
)

type PerfMetrics struct {
	P50Ms         int
	P95Ms         int
	P99Ms         int
	ThroughputRPS float64
	ErrorRate     float64
	TotalRequests int64
}

func (m *PerfMetrics) Validate() error {
	if m.P50Ms < 0 {
		return errors.New("p50_ms must be non-negative")
	}
	if m.P95Ms < 0 {
		return errors.New("p95_ms must be non-negative")
	}
	if m.P99Ms < 0 {
		return errors.New("p99_ms must be non-negative")
	}
	if m.P50Ms > m.P95Ms {
		return errors.New("p50_ms must be less than or equal to p95_ms")
	}
	if m.P95Ms > m.P99Ms {
		return errors.New("p95_ms must be less than or equal to p99_ms")
	}
	if m.ThroughputRPS < 0 {
		return errors.New("throughput_rps must be non-negative")
	}
	if m.ErrorRate < 0 || m.ErrorRate > 1 {
		return errors.New("error_rate must be between 0.0 and 1.0")
	}
	if m.TotalRequests < 0 {
		return errors.New("total_requests must be non-negative")
	}
	return nil
}

type PerfRegressionResult struct {
	VsBaseline string
	Flagged    bool
	DeltaPct   float64
}

type PerfBenchmark struct {
	ID           uuid.UUID
	ScenarioName string
	Environment  string
	Status       BenchmarkStatus
	SLOResult    SLOResult
	StartedAt    time.Time
	FinishedAt   *time.Time
	GitSHA       string
	Metrics      PerfMetrics
	Regression   *PerfRegressionResult
	CreatedAt    time.Time
}
