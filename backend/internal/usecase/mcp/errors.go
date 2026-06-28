package mcp

import "errors"

var (
	ErrMCPRunConflict       = errors.New("mcp: verification already running")
	ErrMCPPlanLimitExceeded = errors.New("mcp: monthly run limit reached")
)
