package secret

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type DeleteActionSecretUsecase struct {
	secretRepo   domainrepo.IActionSecretRepository
	auditLogRepo AuditLogWriter
}

func NewDeleteActionSecretUsecase(
	secretRepo domainrepo.IActionSecretRepository,
	auditLogRepo AuditLogWriter,
) *DeleteActionSecretUsecase {
	return &DeleteActionSecretUsecase{
		secretRepo:   secretRepo,
		auditLogRepo: auditLogRepo,
	}
}

func (uc *DeleteActionSecretUsecase) Execute(
	ctx context.Context,
	orgID uuid.UUID,
	repoID *uuid.UUID,
	actorID uuid.UUID,
	name string,
) error {
	if err := uc.secretRepo.Delete(ctx, orgID, repoID, name); err != nil {
		return err
	}

	metadata, err := json.Marshal(map[string]any{
		"name":  name,
		"scope": secretScope(repoID),
	})
	if err != nil {
		return err
	}

	return uc.auditLogRepo.InsertAuditLog(
		ctx,
		orgID,
		actorID,
		"secret.delete",
		"secret",
		uuid.Nil,
		metadata,
	)
}
