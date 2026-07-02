package handler

import (
	"strconv"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/middleware"
)

// parseActionsID parses a run/job id path parameter that the Actions API
// exposes as a numeric (int64) value derived from an int64-compatible UUID via
// middleware.UUIDToInt64. It accepts either that numeric form (mapped back with
// Int64ToUUID) or a raw UUID string, so both the UI (numeric) and direct API
// clients work.
func parseActionsID(raw string) (uuid.UUID, bool) {
	if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return middleware.Int64ToUUID(n), true
	}
	if id, err := uuid.Parse(raw); err == nil {
		return id, true
	}
	return uuid.Nil, false
}
