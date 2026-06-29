package globalid_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/open-git/backend/graph/globalid"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	cases := []struct {
		typeName string
		constant string
	}{
		{globalid.TypeRepository, globalid.TypeRepository},
		{globalid.TypeIssue, globalid.TypeIssue},
		{globalid.TypePullRequest, globalid.TypePullRequest},
		{globalid.TypeUser, globalid.TypeUser},
		{globalid.TypeLabel, globalid.TypeLabel},
		{globalid.TypeOrganization, globalid.TypeOrganization},
		{globalid.TypeIssueComment, globalid.TypeIssueComment},
		{globalid.TypeMilestone, globalid.TypeMilestone},
	}

	for _, tc := range cases {
		t.Run(tc.typeName, func(t *testing.T) {
			encoded := globalid.Encode(tc.constant, id)
			decodedType, decodedID, err := globalid.Decode(encoded)
			require.NoError(t, err)
			require.Equal(t, tc.typeName, decodedType)
			require.Equal(t, id, decodedID)
		})
	}
}

func TestDecodeMalformedBase64(t *testing.T) {
	_, _, err := globalid.Decode("not-valid-base64!!!")
	require.Error(t, err)
}

func TestDecodeMissingColon(t *testing.T) {
	encoded := globalid.Encode("Repository", uuid.New())
	malformed := encoded[:len(encoded)-4] + "XXXX"
	_, _, err := globalid.Decode(malformed)
	require.Error(t, err)
}

func TestDecodeWrongFormat(t *testing.T) {
	_, _, err := globalid.Decode("UmVwb3NpdG9ye")
	require.Error(t, err)
}
