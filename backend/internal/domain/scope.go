package domain

var ValidScopes = map[string]bool{
	"repo":        true,
	"read:org":    true,
	"write:org":   true,
	"admin:org":   true,
	"user":        true,
	"read:user":   true,
	"repo:status": true,
	"delete_repo": true,
}

func IsValidScope(s string) bool {
	return ValidScopes[s]
}
