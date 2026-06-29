package entity

import (
	"time"

	"github.com/google/uuid"
)

type PerfSLOThreshold struct {
	ID               uuid.UUID
	ScenarioName     string
	P95MsMax         *int
	P99MsMax         *int
	ErrorRateMax     *float64
	ThroughputRPSMin *int
	RegressionPctMax *float64
	UpdatedAt        time.Time
}
