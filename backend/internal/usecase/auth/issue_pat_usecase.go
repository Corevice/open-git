package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/Corevice/open-git/backend/internal/domain"
	"github.com/Corevice/open-git/backend/internal/repository"
)

type IssuePATInput struct {
	UserID    int64
	Scopes    []string
	ExpiresAt *time.Time
}

type IssuePATOutput struct {
	Token  string
	Record *domain.AccessToken
}

type IssuePATUsecase struct {
	tokens repository.IAccessTokenRepository
}

func NewIssuePATUsecase(tokens repository.IAccessTokenRepository) *IssuePATUsecase {
	return &IssuePATUsecase{tokens: tokens}
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (u *IssuePATUsecase) Execute(ctx context.Context, input IssuePATInput) (*IssuePATOutput, error) {
	buf := make([]byte, 40)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	raw := hex.EncodeToString(buf)

	token := &domain.AccessToken{
		UserID:    input.UserID,
		TokenHash: hashToken(raw),
		Scopes:    input.Scopes,
		ExpiresAt: input.ExpiresAt,
		CreatedAt: time.Now().UTC(),
	}

	if err := u.tokens.Create(ctx, token); err != nil {
		return nil, err
	}

	return &IssuePATOutput{Token: raw, Record: token}, nil
}
