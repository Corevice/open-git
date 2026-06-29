package handler

import (
	"encoding/base64"
	"strconv"

	"github.com/google/uuid"
)

func NodeID(typeName, id string) string {
	return base64.StdEncoding.EncodeToString([]byte(typeName + ":" + id))
}

func UserNodeID(id int64) string { return NodeID("User", strconv.FormatInt(id, 10)) }

func RepoNodeID(id uuid.UUID) string { return NodeID("Repository", id.String()) }

func IssueNodeID(id uuid.UUID) string { return NodeID("Issue", id.String()) }

func PRNodeID(id uuid.UUID) string { return NodeID("PullRequest", id.String()) }
