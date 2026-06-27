package domain

import "time"

type Visibility string

const (
	VisibilityPublic   Visibility = "public"
	VisibilityPrivate  Visibility = "private"
	VisibilityInternal Visibility = "internal"
)

type User struct {
	ID           int64
	Login        string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type AccessToken struct {
	ID             int64
	UserID         int64
	Name           string
	TokenHash      string
	TokenLastEight string
	Scopes         []string
	ExpiresAt      *time.Time
	LastUsedAt     *time.Time
	RevokedAt      *time.Time
	CreatedAt      time.Time
}

type Repository struct {
	ID             int64
	OrganizationID int64
	OwnerID        int64
	OwnerLogin     string
	Name           string
	Visibility     Visibility
	DefaultBranch  string
	Description    string
	CreatedAt      time.Time
}

type Organization struct {
	ID        int64
	Login     string
	Name      string
	CreatedAt time.Time
}

type OAuthApp struct {
	ClientID     string
	RedirectURIs []string
}
