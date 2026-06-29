package relay

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/graph/model"
)

func EncodeCursor(id uuid.UUID, createdAt time.Time) string {
	payload := fmt.Sprintf("cursor:%d:%s", createdAt.UnixNano(), id.String())
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func DecodeCursor(cursor string) (createdAt time.Time, id uuid.UUID, err error) {
	data, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("decode cursor: %w", err)
	}

	parts := strings.SplitN(string(data), ":", 3)
	if len(parts) != 3 || parts[0] != "cursor" {
		return time.Time{}, uuid.Nil, fmt.Errorf("invalid cursor format")
	}

	nanos, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("parse cursor timestamp: %w", err)
	}

	parsed, err := uuid.Parse(parts[2])
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("parse cursor id: %w", err)
	}

	return time.Unix(0, nanos).UTC(), parsed, nil
}

func ValidateFirst(first *int) error {
	if first != nil && *first > 100 {
		return &domain.DomainError{
			Code:    "MAX_NODE_LIMIT_EXCEEDED",
			Message: "Maximum number of nodes exceeded",
		}
	}
	return nil
}

func BuildPageInfo(edges int, first *int, last *int, hasPrev bool) model.PageInfo {
	limit := 30
	if first != nil && *first > 0 {
		limit = *first
	} else if last != nil && *last > 0 {
		limit = *last
	}

	hasNext := edges > limit
	if hasNext {
		edges = limit
	}

	info := model.PageInfo{
		HasNextPage:     hasNext,
		HasPreviousPage: hasPrev,
	}
	if edges > 0 {
		// Callers set cursors on edges; PageInfo cursors are optional for this layer.
		_ = edges
	}
	return info
}
