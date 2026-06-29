package docs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// GetDocTreeUsecase loads CONTRIBUTING.md sections from the local filesystem.
type GetDocTreeUsecase struct {
	docsRoot string
}

func NewGetDocTreeUsecase(docsRoot string) *GetDocTreeUsecase {
	return &GetDocTreeUsecase{docsRoot: docsRoot}
}

func (u *GetDocTreeUsecase) contributingPath() string {
	return filepath.Join(u.docsRoot, "CONTRIBUTING.md")
}

// UpdatedAt returns the CONTRIBUTING.md modification time, or zero when absent.
func (u *GetDocTreeUsecase) UpdatedAt() (time.Time, error) {
	info, err := os.Stat(u.contributingPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	return info.ModTime().UTC(), nil
}

func (u *GetDocTreeUsecase) Execute(ctx context.Context) ([]DocSection, error) {
	_ = ctx

	data, err := os.ReadFile(u.contributingPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []DocSection{}, nil
		}
		return nil, err
	}
	return ParseSections(string(data)), nil
}
