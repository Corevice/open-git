package entity

import (
	"time"

	"github.com/google/uuid"
)

type PerfBaseline struct {
	ScenarioName string
	BenchmarkID  uuid.UUID
	SetAt        time.Time
}
