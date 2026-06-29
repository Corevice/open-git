package docs

import (
	"context"
	"errors"
	"regexp"

	"github.com/open-git/backend/internal/domain"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// ErrInvalidSlug is returned when the requested slug fails validation.
var ErrInvalidSlug = errors.New("invalid slug")

// GetDocSectionUsecase returns a single CONTRIBUTING.md section by slug.
type GetDocSectionUsecase struct {
	tree *GetDocTreeUsecase
}

func NewGetDocSectionUsecase(tree *GetDocTreeUsecase) *GetDocSectionUsecase {
	return &GetDocSectionUsecase{tree: tree}
}

func (u *GetDocSectionUsecase) Execute(ctx context.Context, slug string) (*DocSection, error) {
	if !slugPattern.MatchString(slug) {
		return nil, ErrInvalidSlug
	}

	sections, err := u.tree.Execute(ctx)
	if err != nil {
		return nil, err
	}

	for i := range sections {
		if sections[i].Slug == slug {
			section := sections[i]
			return &section, nil
		}
	}
	return nil, domain.ErrNotFound
}
