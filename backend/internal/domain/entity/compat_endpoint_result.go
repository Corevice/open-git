package entity

import (
	"encoding/json"

	"github.com/google/uuid"
)

const (
	CompatResultPass          = "pass"
	CompatResultFail          = "fail"
	CompatResultUnimplemented = "unimplemented"
)

type CompatEndpointChecks struct {
	Schema     bool `json:"schema"`
	StatusCode bool `json:"status_code"`
	Headers    bool `json:"headers"`
	Pagination bool `json:"pagination"`
}

type CompatEndpointResult struct {
	ID     uuid.UUID
	RunID  uuid.UUID
	Method string
	Path   string
	Status string
	Checks *CompatEndpointChecks
	Diff   json.RawMessage
}
