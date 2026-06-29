package graph

import (
	"strings"

	"github.com/open-git/backend/graph/globalid"
	"github.com/open-git/backend/graph/model"
	"github.com/open-git/backend/internal/domain/entity"
)

func mapIssue(e *entity.Issue) *model.Issue {
	if e == nil {
		return nil
	}
	globalID := globalid.Encode(globalid.TypeIssue, e.ID)
	repoModel := mapRepository(&entity.Repository{
		ID:             e.RepositoryID,
		OrganizationID: e.OrganizationID,
	})
	mapped := &model.Issue{
		ID:         globalID,
		Number:     e.Number,
		Title:      e.Title,
		Body:       optionalString(e.Body),
		State:      mapIssueState(e.State),
		Repository: repoModel,
		CreatedAt:  formatDateTime(e.CreatedAt),
		UpdatedAt:  formatDateTime(e.UpdatedAt),
	}
	if e.ClosedAt != nil {
		closedAt := formatDateTime(*e.ClosedAt)
		mapped.ClosedAt = &closedAt
	}
	setIssueMeta(globalID, issueMeta{
		authorID:       e.AuthorID,
		labels:         e.Labels,
		repositoryID:   e.RepositoryID,
		organizationID: e.OrganizationID,
		number:         e.Number,
	})
	return mapped
}

func mapPullRequest(e *entity.PullRequest) *model.PullRequest {
	if e == nil {
		return nil
	}
	globalID := globalid.Encode(globalid.TypePullRequest, e.ID)
	repoModel := mapRepository(&entity.Repository{
		ID:             e.RepositoryID,
		OrganizationID: e.OrganizationID,
	})
	mapped := &model.PullRequest{
		ID:             globalID,
		Number:         e.Number,
		Title:          e.Title,
		Body:           optionalString(e.Body),
		State:          mapPullRequestState(e.State),
		Repository:     repoModel,
		HeadRefName:    e.HeadRef,
		BaseRefName:    e.BaseRef,
		Mergeable:      mapMergeableState(e.MergeableState),
		MergeableState: mapMergeableState(e.MergeableState),
		CreatedAt:      formatDateTime(e.CreatedAt),
		UpdatedAt:      formatDateTime(e.UpdatedAt),
	}
	setPullRequestMeta(globalID, pullRequestMeta{
		authorID:       e.AuthorID,
		repositoryID:   e.RepositoryID,
		organizationID: e.OrganizationID,
		number:         e.Number,
	})
	return mapped
}

func mapUser(e *entity.User) *model.User {
	if e == nil {
		return nil
	}
	globalID := globalid.Encode(globalid.TypeUser, e.ID)
	return &model.User{
		ID:        globalID,
		Login:     e.Login,
		AvatarURL: userAvatarURL(e.Login),
		URL:       userURL(e.Login),
		Name:      optionalString(e.Name),
		Email:     optionalString(e.Email),
		Bio:       optionalString(e.Bio),
		CreatedAt: formatDateTime(e.CreatedAt),
		UpdatedAt: formatDateTime(e.UpdatedAt),
	}
}

func mapLabel(e *entity.Label) *model.Label {
	if e == nil {
		return nil
	}
	createdAt := formatDateTime(e.CreatedAt)
	return &model.Label{
		ID:          globalid.Encode(globalid.TypeLabel, e.ID),
		Name:        e.Name,
		Color:       e.Color,
		Description: optionalString(e.Description),
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}
}

func mapRepository(e *entity.Repository) *model.Repository {
	if e == nil {
		return nil
	}
	globalID := globalid.Encode(globalid.TypeRepository, e.ID)
	createdAt := formatDateTime(e.CreatedAt)
	ownerLogin := e.OwnerLogin
	if ownerLogin == "" {
		ownerLogin = "unknown"
	}
	owner := mapUser(&entity.User{
		ID:    e.OwnerID,
		Login: ownerLogin,
	})
	name := e.Name
	if name == "" {
		name = "unknown"
	}
	mapped := &model.Repository{
		ID:            globalID,
		Name:          name,
		NameWithOwner: ownerLogin + "/" + name,
		URL:           repositoryURL(ownerLogin, name),
		Description:   optionalString(e.Description),
		IsPrivate:     strings.EqualFold(e.Visibility, entity.VisibilityPrivate),
		Owner:         owner,
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}
	setRepositoryMeta(globalID, repositoryMeta{
		id:             e.ID,
		organizationID: e.OrganizationID,
		ownerLogin:     ownerLogin,
		name:           name,
	})
	return mapped
}
