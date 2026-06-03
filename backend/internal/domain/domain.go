package domain

import "time"

// Visibility is a type alias for the repository visibility string values.
type Visibility = string

const (
	VisibilityPrivate  Visibility = "private"
	VisibilityInternal Visibility = "internal"
	VisibilityPublic   Visibility = "public"
)

type User struct {
	ID           int64
	Login        string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type AccessToken struct {
	ID        int64
	UserID    int64
	TokenHash string
	Scopes    []string
	ExpiresAt *time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

type Repository struct {
	ID             int64
	OrganizationID int64
	OwnerID        int64
	OwnerLogin     string
	Name           string
	Visibility     string
	DefaultBranch  string
	Description    string
	CreatedAt      time.Time
}

type OAuthApp struct {
	ID           int64
	ClientID     string
	RedirectURIs []string
}
