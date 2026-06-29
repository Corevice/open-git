package entity

import (
	"time"

	"github.com/google/uuid"
)

type SystemSetting struct {
	Key       string         `db:"key" json:"key"`
	Value     map[string]any `db:"value" json:"value"`
	UpdatedBy uuid.UUID      `db:"updated_by" json:"updated_by"`
	UpdatedAt time.Time      `db:"updated_at" json:"updated_at"`
}
