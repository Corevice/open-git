package globalid

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const (
	TypeRepository   = "Repository"
	TypeIssue        = "Issue"
	TypePullRequest  = "PullRequest"
	TypeUser         = "User"
	TypeLabel        = "Label"
	TypeOrganization = "Organization"
	TypeIssueComment = "IssueComment"
	TypeMilestone    = "Milestone"
)

func Encode(typeName string, id uuid.UUID) string {
	payload := typeName + ":" + id.String()
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func Decode(globalID string) (typeName string, id uuid.UUID, err error) {
	data, err := base64.RawURLEncoding.DecodeString(globalID)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("decode global id: %w", err)
	}

	parts := strings.SplitN(string(data), ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", uuid.Nil, fmt.Errorf("invalid global id format")
	}

	parsed, err := uuid.Parse(parts[1])
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("parse global id uuid: %w", err)
	}

	return parts[0], parsed, nil
}
