package domain

var validScopes = map[string]bool{
	"repo":        true,
	"read:org":    true,
	"write:org":   true,
	"admin:org":   true,
	"user":        true,
	"read:user":   true,
	"repo:status": true,
	"repo:delete": true,
}

func IsValidScope(s string) bool {
	return validScopes[s]
}
