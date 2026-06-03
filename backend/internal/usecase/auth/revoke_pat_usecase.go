package auth

import (
	"context"

	"github.com/Corevice/open-git/backend/internal/repository"
)

type RevokePATUsecase struct {
	tokens repository.IAccessTokenRepository
}

func NewRevokePATUsecase(tokens repository.IAccessTokenRepository) *RevokePATUsecase {
	return &RevokePATUsecase{tokens: tokens}
}

func (u *RevokePATUsecase) Execute(ctx context.Context, userID, tokenID int64) error {
	return u.tokens.Revoke(ctx, tokenID, userID)
}
