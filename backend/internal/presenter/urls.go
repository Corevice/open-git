package presenter

import (
	"encoding/base64"
	"encoding/binary"
	"strconv"

	"github.com/google/uuid"
)

var base64NoPadding = base64.StdEncoding.WithPadding(base64.NoPadding)

func NodeID(resourceType string, id int64) string {
	raw := resourceType + ":" + strconv.FormatInt(id, 10)
	return base64NoPadding.EncodeToString([]byte(raw))
}

func UserAPIURL(apiBase, login string) string {
	return apiBase + "/users/" + login
}

func UserHTMLURL(webBase, login string) string {
	return webBase + "/" + login
}

func RepoAPIURL(apiBase, owner, repo string) string {
	return apiBase + "/repos/" + owner + "/" + repo
}

func RepoHTMLURL(webBase, owner, repo string) string {
	return webBase + "/" + owner + "/" + repo
}

func IssueAPIURL(apiBase, owner, repo string, number int) string {
	return apiBase + "/repos/" + owner + "/" + repo + "/issues/" + strconv.Itoa(number)
}

func IssueHTMLURL(webBase, owner, repo string, number int) string {
	return webBase + "/" + owner + "/" + repo + "/issues/" + strconv.Itoa(number)
}

func CommentAPIURL(apiBase, owner, repo string, id int64) string {
	return apiBase + "/repos/" + owner + "/" + repo + "/issues/comments/" + strconv.FormatInt(id, 10)
}

func OrgAPIURL(apiBase, org string) string {
	return apiBase + "/orgs/" + org
}

func OrgHTMLURL(webBase, org string) string {
	return webBase + "/" + org
}

func UUIDToInt64(id uuid.UUID) int64 {
	return int64(binary.BigEndian.Uint64(id[8:]))
}
