package userpreferences

import (
	"context"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type GetUserPreferencesUsecase struct {
	prefs domainrepo.IUserPreferencesRepository
}

func NewGetUserPreferencesUsecase(prefs domainrepo.IUserPreferencesRepository) *GetUserPreferencesUsecase {
	return &GetUserPreferencesUsecase{prefs: prefs}
}

func (u *GetUserPreferencesUsecase) Execute(ctx context.Context, userID int64) (*entity.UserPreferences, error) {
	prefs, err := u.prefs.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if prefs == nil {
		return &entity.UserPreferences{UserID: userID, Theme: "system"}, nil
	}
	return prefs, nil
}
