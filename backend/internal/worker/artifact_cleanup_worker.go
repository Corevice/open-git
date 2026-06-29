package worker

import (
	"context"
	"log/slog"

	"github.com/hibiken/asynq"

	domainrepo "github.com/open-git/backend/internal/domain/repository"
	artifactusecase "github.com/open-git/backend/internal/usecase/artifact"
)

type ArtifactCleanupWorker struct {
	artifactRepo domainrepo.IArtifactRepository
	storage      artifactusecase.ArtifactStorage
	bucket       string
}

func NewArtifactCleanupWorker(
	repo domainrepo.IArtifactRepository,
	storage artifactusecase.ArtifactStorage,
	bucket string,
) *ArtifactCleanupWorker {
	return &ArtifactCleanupWorker{
		artifactRepo: repo,
		storage:      storage,
		bucket:       bucket,
	}
}

func (w *ArtifactCleanupWorker) HandleCleanup(ctx context.Context, task *asynq.Task) error {
	const batchSize = 100
	for {
		artifacts, err := w.artifactRepo.ListExpired(ctx, batchSize)
		if err != nil {
			return err
		}
		if len(artifacts) == 0 {
			break
		}

		for _, artifact := range artifacts {
			if err := w.storage.DeleteObject(ctx, w.bucket, artifact.StorageKey); err != nil {
				slog.Error("delete artifact object", "artifact_id", artifact.ID, "error", err)
				continue
			}
			if err := w.artifactRepo.SoftDelete(ctx, artifact.ID, artifact.OrganizationID); err != nil {
				slog.Error("soft delete artifact", "artifact_id", artifact.ID, "error", err)
			}
		}

		if len(artifacts) < batchSize {
			break
		}
	}
	return nil
}
