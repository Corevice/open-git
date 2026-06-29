package userpreferences

import (
	"context"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

var validThemes = map[string]struct{}{
	"light":  {},
	"dark":   {},
	"system": {},
}

type UpdateUserPreferencesUsecase struct {
	prefs domainrepo.IUserPreferencesRepository
}

func NewUpdateUserPreferencesUsecase(prefs domainrepo.IUserPreferencesRepository) *UpdateUserPreferencesUsecase {
	return &UpdateUserPreferencesUsecase{prefs: prefs}
}

func (u *UpdateUserPreferencesUsecase) Execute(ctx context.Context, userID int64, theme string) (*entity.UserPreferences, error) {
	if _, ok := validThemes[theme]; !ok {
		return nil, domain.ErrValidation
	}

	prefs := &entity.UserPreferences{
		UserID: userID,
		Theme:  theme,
	}
	if err := u.prefs.Upsert(ctx, prefs); err != nil {
		return nil, err
	}
	return prefs, nil
}
