package actions

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type CreateRegistrationTokenUsecase struct {
	tokenRepo domainrepo.IRunnerRegistrationTokenRepository
}

func NewCreateRegistrationTokenUsecase(
	tokenRepo domainrepo.IRunnerRegistrationTokenRepository,
) *CreateRegistrationTokenUsecase {
	return &CreateRegistrationTokenUsecase{tokenRepo: tokenRepo}
}

func hashRegistrationToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (uc *CreateRegistrationTokenUsecase) Execute(
	ctx context.Context,
	orgID uuid.UUID,
	actorRole string,
) (*entity.RunnerRegistrationToken, string, error) {
	if actorRole != entity.RoleAdmin {
		return nil, "", domain.ErrForbidden
	}

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, "", err
	}
	raw := hex.EncodeToString(buf)

	now := time.Now().UTC()
	token := &entity.RunnerRegistrationToken{
		ID:             uuid.New(),
		OrganizationID: orgID,
		TokenHash:      hashRegistrationToken(raw),
		ExpiresAt:      now.Add(time.Hour),
	}

	if err := uc.tokenRepo.Create(ctx, token); err != nil {
		return nil, "", err
	}

	return token, raw, nil
}
