package relay_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/open-git/backend/graph/relay"
)

func TestEncodeDecodeCursorRoundTrip(t *testing.T) {
	id := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	createdAt := time.Date(2024, 6, 1, 12, 0, 0, 123, time.UTC)

	cursor := relay.EncodeCursor(id, createdAt)
	decodedAt, decodedID, err := relay.DecodeCursor(cursor)
	require.NoError(t, err)
	require.Equal(t, id, decodedID)
	require.True(t, createdAt.Equal(decodedAt))
}

func TestValidateFirstWithinLimit(t *testing.T) {
	first := 100
	require.NoError(t, relay.ValidateFirst(&first))
}

func TestValidateFirstExceedsLimit(t *testing.T) {
	first := 101
	err := relay.ValidateFirst(&first)
	require.Error(t, err)
	require.Contains(t, err.Error(), "MAX_NODE_LIMIT_EXCEEDED")
}

func TestValidateFirstNil(t *testing.T) {
	require.NoError(t, relay.ValidateFirst(nil))
}
