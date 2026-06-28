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
	ID         int64
	UserID     int64
	Note       string
	TokenHash  string
	Scopes     []string
	ExpiresAt  *time.Time
	LastUsedAt *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
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
	ID               string
	ClientID         string
	ClientSecretHash string
	RedirectURIs     []string
	Name             string
	HomepageURL      string
	OwnerType        string
	OwnerUserID      int64
	OrganizationID   int64
	UpdatedAt        time.Time
}

type OAuthAuthorizationCode struct {
	ID          string
	CodeHash    string
	OAuthAppID  string
	UserID      int64
	RedirectURI string
	Scopes      []string
	ExpiresAt   time.Time
	ConsumedAt  *time.Time
	CreatedAt   time.Time
}

type OAuthAccessToken struct {
	ID         string
	TokenHash  string
	OAuthAppID string
	UserID     int64
	Scopes     []string
	RevokedAt  *time.Time
	LastUsedAt *time.Time
	CreatedAt  time.Time
}

type OAuthAuthorization struct {
	ID            string
	OAuthAppID    string
	UserID        int64
	GrantedScopes []string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
