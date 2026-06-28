package dataloader

import (
	"context"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/vikstrous/dataloadgen"

	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

const loadersEchoKey = "graphql_loaders"

type Loaders struct {
	UserByID       *dataloadgen.Loader[uuid.UUID, *entity.User]
	LabelByID      *dataloadgen.Loader[uuid.UUID, *entity.Label]
	MilestoneByID  *dataloadgen.Loader[uuid.UUID, *entity.Milestone]
	RepositoryByID *dataloadgen.Loader[uuid.UUID, *entity.Repository]
}

type byIDLoader[T any] interface {
	GetByID(ctx context.Context, id uuid.UUID) (*T, error)
}

func Middleware(
	userRepo domainrepo.IUserRepository,
	labelRepo domainrepo.ILabelRepository,
	milestoneRepo domainrepo.IMilestoneRepository,
	repoRepo domainrepo.IRepositoryRepository,
) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(loadersEchoKey, newLoaders(userRepo, labelRepo, milestoneRepo, repoRepo))
			return next(c)
		}
	}
}

func FromEcho(c echo.Context) *Loaders {
	value := c.Get(loadersEchoKey)
	loaders, ok := value.(*Loaders)
	if !ok {
		return nil
	}
	return loaders
}

func newLoaders(
	userRepo domainrepo.IUserRepository,
	labelRepo domainrepo.ILabelRepository,
	milestoneRepo domainrepo.IMilestoneRepository,
	repoRepo domainrepo.IRepositoryRepository,
) *Loaders {
	return &Loaders{
		UserByID:       dataloadgen.NewLoader(batchLoad(userRepo)),
		LabelByID:      dataloadgen.NewLoader(batchLoadOptional(labelRepo)),
		MilestoneByID:  dataloadgen.NewLoader(batchLoadOptional(milestoneRepo)),
		RepositoryByID: dataloadgen.NewLoader(batchLoadOptional(repoRepo)),
	}
}

func batchLoad[T any, R byIDLoader[T]](repo R) func(context.Context, []uuid.UUID) ([]*T, []error) {
	return func(ctx context.Context, ids []uuid.UUID) ([]*T, []error) {
		results := make([]*T, len(ids))
		errs := make([]error, len(ids))
		for i, id := range ids {
			item, err := repo.GetByID(ctx, id)
			if err != nil {
				errs[i] = err
				continue
			}
			if item == nil {
				errs[i] = apperror.ErrNotFound
				continue
			}
			results[i] = item
		}
		return results, errs
	}
}

func batchLoadOptional[T any](repo any) func(context.Context, []uuid.UUID) ([]*T, []error) {
	loader, ok := repo.(byIDLoader[T])
	if !ok {
		return func(_ context.Context, ids []uuid.UUID) ([]*T, []error) {
			errs := make([]error, len(ids))
			for i := range ids {
				errs[i] = apperror.ErrNotFound
			}
			return make([]*T, len(ids)), errs
		}
	}
	return batchLoad[T](loader)
}
