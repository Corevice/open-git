package repository

import (
	"context"

	"github.com/open-git/backend/internal/domain/entity"
)

type ICommentRepository interface {
	Create(ctx context.Context, comment *entity.Comment) error
}
