package graph

// This file will be automatically regenerated based on the schema, any resolver
// implementations will be copied through when generating and any unknown code
// will be moved or copied to graph/resolver.go. When modifying or deleting this
// file, make sure to move any code to resolver.go.

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/open-git/backend/graph/generated"
	"github.com/open-git/backend/graph/globalid"
	"github.com/open-git/backend/graph/model"
	"github.com/open-git/backend/graph/relay"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	appmiddleware "github.com/open-git/backend/internal/middleware"
	issueusecase "github.com/open-git/backend/internal/usecase/issue"
	prusecase "github.com/open-git/backend/internal/usecase/pr"
	repoUC "github.com/open-git/backend/internal/usecase/repository"
)

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type issueResolver struct{ *Resolver }
type repositoryResolver struct{ *Resolver }
type pullRequestResolver struct{ *Resolver }

type repositoryMeta struct {
	id             uuid.UUID
	organizationID uuid.UUID
	ownerLogin     string
	name           string
}

type issueMeta struct {
	authorID       uuid.UUID
	labels         []entity.Label
	repositoryID   uuid.UUID
	organizationID uuid.UUID
	number         int
}

type pullRequestMeta struct {
	authorID       uuid.UUID
	repositoryID   uuid.UUID
	organizationID uuid.UUID
	number         int
}

var (
	repositoryMetaByGlobalID  sync.Map
	issueMetaByGlobalID       sync.Map
	pullRequestMetaByGlobalID sync.Map
)

func setRepositoryMeta(globalID string, meta repositoryMeta) {
	repositoryMetaByGlobalID.Store(globalID, meta)
}

func repositoryMetaFor(globalID string) (repositoryMeta, bool) {
	value, ok := repositoryMetaByGlobalID.Load(globalID)
	if !ok {
		return repositoryMeta{}, false
	}
	meta, ok := value.(repositoryMeta)
	return meta, ok
}

func setIssueMeta(globalID string, meta issueMeta) {
	issueMetaByGlobalID.Store(globalID, meta)
}

func issueMetaFor(globalID string) (issueMeta, bool) {
	value, ok := issueMetaByGlobalID.Load(globalID)
	if !ok {
		return issueMeta{}, false
	}
	meta, ok := value.(issueMeta)
	return meta, ok
}

func setPullRequestMeta(globalID string, meta pullRequestMeta) {
	pullRequestMetaByGlobalID.Store(globalID, meta)
}

func pullRequestMetaFor(globalID string) (pullRequestMeta, bool) {
	value, ok := pullRequestMetaByGlobalID.Load(globalID)
	if !ok {
		return pullRequestMeta{}, false
	}
	meta, ok := value.(pullRequestMeta)
	return meta, ok
}

func formatDateTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func userURL(login string) string {
	return fmt.Sprintf("https://github.com/%s", login)
}

func userAvatarURL(login string) string {
	return fmt.Sprintf("https://github.com/%s.png?size=40", login)
}

func repositoryURL(ownerLogin, name string) string {
	return fmt.Sprintf("https://github.com/%s/%s", ownerLogin, name)
}

func mapEntityUser(user *entity.User) *model.User {
	if user == nil {
		return nil
	}
	globalID := globalid.Encode(globalid.TypeUser, user.ID)
	return &model.User{
		ID:        globalID,
		Login:     user.Login,
		AvatarURL: userAvatarURL(user.Login),
		URL:       userURL(user.Login),
		Name:      optionalString(user.Name),
		Email:     optionalString(user.Email),
		Bio:       optionalString(user.Bio),
		CreatedAt: formatDateTime(user.CreatedAt),
		UpdatedAt: formatDateTime(user.UpdatedAt),
	}
}

func mapDomainUser(user *domain.User) *model.User {
	if user == nil {
		return nil
	}
	id := appmiddleware.Int64ToUUID(user.ID)
	globalID := globalid.Encode(globalid.TypeUser, id)
	now := formatDateTime(user.CreatedAt)
	return &model.User{
		ID:        globalID,
		Login:     user.Login,
		AvatarURL: userAvatarURL(user.Login),
		URL:       userURL(user.Login),
		Email:     optionalString(user.Email),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func mapEntityOrganization(org *entity.Organization) *model.Organization {
	if org == nil {
		return nil
	}
	globalID := globalid.Encode(globalid.TypeOrganization, org.ID)
	createdAt := formatDateTime(org.CreatedAt)
	return &model.Organization{
		ID:          globalID,
		Login:       org.Login,
		Name:        optionalString(org.Name),
		URL:         userURL(org.Login),
		AvatarURL:   userAvatarURL(org.Login),
		Description: optionalString(org.Description),
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}
}

func mapDomainOrganization(org *domain.Organization) *model.Organization {
	if org == nil {
		return nil
	}
	id := appmiddleware.Int64ToUUID(org.ID)
	globalID := globalid.Encode(globalid.TypeOrganization, id)
	createdAt := formatDateTime(org.CreatedAt)
	return &model.Organization{
		ID:        globalID,
		Login:     org.Login,
		Name:      optionalString(org.Name),
		URL:       userURL(org.Login),
		AvatarURL: userAvatarURL(org.Login),
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
}

func mapEntityRepository(repo *entity.Repository) *model.Repository {
	if repo == nil {
		return nil
	}
	globalID := globalid.Encode(globalid.TypeRepository, repo.ID)
	createdAt := formatDateTime(repo.CreatedAt)
	owner := mapEntityUser(&entity.User{
		ID:    repo.OwnerID,
		Login: repo.OwnerLogin,
	})
	mapped := &model.Repository{
		ID:            globalID,
		Name:          repo.Name,
		NameWithOwner: repo.OwnerLogin + "/" + repo.Name,
		URL:           repositoryURL(repo.OwnerLogin, repo.Name),
		Description:   optionalString(repo.Description),
		IsPrivate:     repo.Visibility == entity.VisibilityPrivate,
		Owner:         owner,
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}
	setRepositoryMeta(globalID, repositoryMeta{
		id:             repo.ID,
		organizationID: repo.OrganizationID,
		ownerLogin:     repo.OwnerLogin,
		name:           repo.Name,
	})
	return mapped
}

func mapIssueState(state string) model.IssueState {
	switch strings.ToLower(state) {
	case "closed":
		return model.IssueStateClosed
	default:
		return model.IssueStateOpen
	}
}

func mapPullRequestState(state string) model.PullRequestState {
	switch strings.ToLower(state) {
	case "closed":
		return model.PullRequestStateClosed
	case "merged":
		return model.PullRequestStateMerged
	default:
		return model.PullRequestStateOpen
	}
}

func mapMergeableState(state string) model.MergeableState {
	switch strings.ToLower(state) {
	case entity.MergeableStateClean:
		return model.MergeableStateMergeable
	case entity.MergeableStateDirty:
		return model.MergeableStateConflicting
	default:
		return model.MergeableStateUnknown
	}
}

func mapEntityIssue(issue *entity.Issue, repo *model.Repository) *model.Issue {
	if issue == nil {
		return nil
	}
	globalID := globalid.Encode(globalid.TypeIssue, issue.ID)
	mapped := &model.Issue{
		ID:         globalID,
		Number:     issue.Number,
		Title:      issue.Title,
		Body:       optionalString(issue.Body),
		State:      mapIssueState(issue.State),
		Repository: repo,
		CreatedAt:  formatDateTime(issue.CreatedAt),
		UpdatedAt:  formatDateTime(issue.UpdatedAt),
	}
	if issue.ClosedAt != nil {
		closedAt := formatDateTime(*issue.ClosedAt)
		mapped.ClosedAt = &closedAt
	}
	setIssueMeta(globalID, issueMeta{
		authorID:       issue.AuthorID,
		labels:         issue.Labels,
		repositoryID:   issue.RepositoryID,
		organizationID: issue.OrganizationID,
		number:         issue.Number,
	})
	return mapped
}

func mapEntityPullRequest(pr *entity.PullRequest, repo *model.Repository) *model.PullRequest {
	if pr == nil {
		return nil
	}
	globalID := globalid.Encode(globalid.TypePullRequest, pr.ID)
	mapped := &model.PullRequest{
		ID:             globalID,
		Number:         pr.Number,
		Title:          pr.Title,
		Body:           optionalString(pr.Body),
		State:          mapPullRequestState(pr.State),
		Repository:     repo,
		HeadRefName:    pr.HeadRef,
		BaseRefName:    pr.BaseRef,
		Mergeable:      mapMergeableState(pr.MergeableState),
		MergeableState: mapMergeableState(pr.MergeableState),
		CreatedAt:      formatDateTime(pr.CreatedAt),
		UpdatedAt:      formatDateTime(pr.UpdatedAt),
	}
	setPullRequestMeta(globalID, pullRequestMeta{
		authorID:       pr.AuthorID,
		repositoryID:   pr.RepositoryID,
		organizationID: pr.OrganizationID,
		number:         pr.Number,
	})
	return mapped
}

func mapEntityLabel(label *entity.Label) *model.Label {
	if label == nil {
		return nil
	}
	createdAt := formatDateTime(label.CreatedAt)
	return &model.Label{
		ID:          globalid.Encode(globalid.TypeLabel, label.ID),
		Name:        label.Name,
		Color:       label.Color,
		Description: optionalString(label.Description),
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}
}

func mapEntityComment(comment *entity.Comment) *model.IssueComment {
	if comment == nil {
		return nil
	}
	return &model.IssueComment{
		ID:        globalid.Encode(globalid.TypeIssueComment, comment.ID),
		Body:      comment.Body,
		CreatedAt: formatDateTime(comment.CreatedAt),
		UpdatedAt: formatDateTime(comment.UpdatedAt),
	}
}

func issueStatesToFilter(states []model.IssueState) string {
	if len(states) == 0 {
		return ""
	}
	if len(states) == 1 {
		switch states[0] {
		case model.IssueStateOpen:
			return "open"
		case model.IssueStateClosed:
			return "closed"
		}
	}
	return ""
}

func pullRequestStatesToFilter(states []model.PullRequestState) string {
	if len(states) == 0 {
		return "open"
	}
	if len(states) == 1 {
		switch states[0] {
		case model.PullRequestStateOpen:
			return "open"
		case model.PullRequestStateClosed:
			return "closed"
		case model.PullRequestStateMerged:
			return "merged"
		}
	}
	return "all"
}

func filterIssuesAfterCursor(issues []*entity.Issue, createdAt time.Time, id uuid.UUID) []*entity.Issue {
	filtered := make([]*entity.Issue, 0, len(issues))
	for _, issue := range issues {
		if issue == nil {
			continue
		}
		if issue.CreatedAt.After(createdAt) || (issue.CreatedAt.Equal(createdAt) && issue.ID.String() > id.String()) {
			filtered = append(filtered, issue)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].CreatedAt.Equal(filtered[j].CreatedAt) {
			return filtered[i].ID.String() < filtered[j].ID.String()
		}
		return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
	})
	return filtered
}

func filterPullRequestsAfterCursor(prs []*entity.PullRequest, createdAt time.Time, id uuid.UUID) []*entity.PullRequest {
	filtered := make([]*entity.PullRequest, 0, len(prs))
	for _, pr := range prs {
		if pr == nil {
			continue
		}
		if pr.CreatedAt.After(createdAt) || (pr.CreatedAt.Equal(createdAt) && pr.ID.String() > id.String()) {
			filtered = append(filtered, pr)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].CreatedAt.Equal(filtered[j].CreatedAt) {
			return filtered[i].ID.String() < filtered[j].ID.String()
		}
		return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
	})
	return filtered
}

func buildIssueConnection(issues []*entity.Issue, repo *model.Repository, first *int, after *string, total int) (*model.IssueConnection, error) {
	hasPrev := after != nil && *after != ""
	pageSize := 30
	if first != nil && *first > 0 {
		pageSize = *first
	}

	if after != nil && *after != "" {
		cursorTime, cursorID, err := relay.DecodeCursor(*after)
		if err != nil {
			return nil, err
		}
		issues = filterIssuesAfterCursor(issues, cursorTime, cursorID)
	}

	fetched := len(issues)
	hasNext := fetched > pageSize
	if hasNext {
		issues = issues[:pageSize]
	}

	edges := make([]*model.IssueEdge, 0, len(issues))
	nodes := make([]*model.Issue, 0, len(issues))
	for _, issue := range issues {
		node := mapEntityIssue(issue, repo)
		cursor := relay.EncodeCursor(issue.ID, issue.CreatedAt)
		edges = append(edges, &model.IssueEdge{Cursor: cursor, Node: node})
		nodes = append(nodes, node)
	}

	pageInfo := relay.BuildPageInfo(fetched, first, nil, hasPrev)
	pageInfo.HasNextPage = hasNext
	if len(edges) > 0 {
		start := edges[0].Cursor
		end := edges[len(edges)-1].Cursor
		pageInfo.StartCursor = &start
		pageInfo.EndCursor = &end
	}

	return &model.IssueConnection{
		Edges:      edges,
		Nodes:      nodes,
		PageInfo:   &pageInfo,
		TotalCount: total,
	}, nil
}

func buildPullRequestConnection(prs []*entity.PullRequest, repo *model.Repository, first *int, after *string, total int) (*model.PullRequestConnection, error) {
	hasPrev := after != nil && *after != ""
	pageSize := 30
	if first != nil && *first > 0 {
		pageSize = *first
	}

	if after != nil && *after != "" {
		cursorTime, cursorID, err := relay.DecodeCursor(*after)
		if err != nil {
			return nil, err
		}
		prs = filterPullRequestsAfterCursor(prs, cursorTime, cursorID)
	}

	fetched := len(prs)
	hasNext := fetched > pageSize
	if hasNext {
		prs = prs[:pageSize]
	}

	edges := make([]*model.PullRequestEdge, 0, len(prs))
	nodes := make([]*model.PullRequest, 0, len(prs))
	for _, pr := range prs {
		node := mapEntityPullRequest(pr, repo)
		cursor := relay.EncodeCursor(pr.ID, pr.CreatedAt)
		edges = append(edges, &model.PullRequestEdge{Cursor: cursor, Node: node})
		nodes = append(nodes, node)
	}

	pageInfo := relay.BuildPageInfo(fetched, first, nil, hasPrev)
	pageInfo.HasNextPage = hasNext
	if len(edges) > 0 {
		start := edges[0].Cursor
		end := edges[len(edges)-1].Cursor
		pageInfo.StartCursor = &start
		pageInfo.EndCursor = &end
	}

	return &model.PullRequestConnection{
		Edges:      edges,
		Nodes:      nodes,
		PageInfo:   &pageInfo,
		TotalCount: total,
	}, nil
}

func (r *issueResolver) Author(ctx context.Context, obj *model.Issue) (model.Actor, error) {
	meta, ok := issueMetaFor(obj.ID)
	if !ok || meta.authorID == uuid.Nil {
		return nil, nil
	}
	loaders := LoadersFromContext(ctx)
	if loaders == nil {
		return nil, nil
	}
	user, err := loaders.UserByID.Load(ctx, meta.authorID)
	if err != nil {
		return nil, err
	}
	return mapEntityUser(user), nil
}

func (r *issueResolver) Labels(ctx context.Context, obj *model.Issue, first *int, after *string, last *int, before *string) (*model.LabelConnection, error) {
	if err := relay.ValidateFirst(first); err != nil {
		return nil, err
	}
	meta, ok := issueMetaFor(obj.ID)
	if !ok || len(meta.labels) == 0 {
		empty := &model.LabelConnection{
			PageInfo:   &model.PageInfo{},
			TotalCount: 0,
		}
		return empty, nil
	}

	loaders := LoadersFromContext(ctx)
	if loaders == nil {
		return nil, nil
	}

	labelIDs := make([]uuid.UUID, 0, len(meta.labels))
	for _, label := range meta.labels {
		labelIDs = append(labelIDs, label.ID)
	}

	loaded, errs := loaders.LabelByID.LoadMany(ctx, labelIDs)
	labels := make([]*model.Label, 0, len(loaded))
	for i, label := range loaded {
		if errs[i] != nil || label == nil {
			continue
		}
		labels = append(labels, mapEntityLabel(label))
	}

	limit := len(labels)
	if first != nil && *first > 0 && *first < limit {
		limit = *first
	}
	labels = labels[:limit]

	edges := make([]*model.LabelEdge, 0, len(labels))
	for _, label := range labels {
		cursor := label.ID
		edges = append(edges, &model.LabelEdge{Cursor: cursor, Node: label})
	}

	pageInfo := relay.BuildPageInfo(len(labels), first, last, after != nil && *after != "")
	return &model.LabelConnection{
		Edges:      edges,
		Nodes:      labels,
		PageInfo:   &pageInfo,
		TotalCount: len(meta.labels),
	}, nil
}

func (r *issueResolver) Comments(ctx context.Context, obj *model.Issue, first *int, after *string, last *int, before *string) (*model.IssueCommentConnection, error) {
	if err := relay.ValidateFirst(first); err != nil {
		return nil, err
	}
	_, ok := issueMetaFor(obj.ID)
	if !ok || r.CommentRepo == nil {
		return &model.IssueCommentConnection{PageInfo: &model.PageInfo{}}, nil
	}

	_, issueID, err := globalid.Decode(obj.ID)
	if err != nil {
		return nil, err
	}

	perPage := 30
	if first != nil && *first > 0 {
		perPage = *first
	}
	comments, total, err := r.CommentRepo.ListByIssue(ctx, issueID, 1, perPage)
	if err != nil {
		return nil, err
	}

	edges := make([]*model.IssueCommentEdge, 0, len(comments))
	nodes := make([]*model.IssueComment, 0, len(comments))
	for _, comment := range comments {
		node := mapEntityComment(comment)
		cursor := relay.EncodeCursor(comment.ID, comment.CreatedAt)
		edges = append(edges, &model.IssueCommentEdge{Cursor: cursor, Node: node})
		nodes = append(nodes, node)
	}

	pageInfo := relay.BuildPageInfo(len(comments), first, last, after != nil && *after != "")
	return &model.IssueCommentConnection{
		Edges:      edges,
		Nodes:      nodes,
		PageInfo:   &pageInfo,
		TotalCount: total,
	}, nil
}

func (r *pullRequestResolver) Author(ctx context.Context, obj *model.PullRequest) (model.Actor, error) {
	meta, ok := pullRequestMetaFor(obj.ID)
	if !ok || meta.authorID == uuid.Nil {
		return nil, nil
	}
	loaders := LoadersFromContext(ctx)
	if loaders == nil {
		return nil, nil
	}
	user, err := loaders.UserByID.Load(ctx, meta.authorID)
	if err != nil {
		return nil, err
	}
	return mapEntityUser(user), nil
}

func (r *repositoryResolver) Issue(ctx context.Context, obj *model.Repository, number int) (*model.Issue, error) {
	if r.GetIssueUC == nil {
		return nil, apperror.ErrNotFound
	}
	meta, ok := repositoryMetaFor(obj.ID)
	if !ok {
		return nil, apperror.ErrNotFound
	}
	issue, err := r.GetIssueUC.Execute(ctx, issueusecase.GetIssueInput{
		OrganizationID: meta.organizationID,
		RepositoryID:   meta.id,
		IssueNumber:    number,
	})
	if err != nil {
		return nil, err
	}
	return mapEntityIssue(issue, obj), nil
}

func (r *repositoryResolver) Issues(ctx context.Context, obj *model.Repository, first *int, after *string, last *int, before *string, states []model.IssueState, labels []string) (*model.IssueConnection, error) {
	if err := relay.ValidateFirst(first); err != nil {
		return nil, err
	}
	if r.ListIssuesUC == nil {
		return nil, apperror.ErrNotFound
	}
	meta, ok := repositoryMetaFor(obj.ID)
	if !ok {
		return nil, apperror.ErrNotFound
	}

	pageSize := 30
	if first != nil && *first > 0 {
		pageSize = *first
	}

	output, err := r.ListIssuesUC.Execute(ctx, issueusecase.ListIssuesInput{
		OrganizationID: meta.organizationID,
		RepositoryID:   meta.id,
		State:          issueStatesToFilter(states),
		Labels:         labels,
		Page:           1,
		PerPage:        pageSize + 1,
	})
	if err != nil {
		return nil, err
	}

	return buildIssueConnection(output.Issues, obj, first, after, output.Total)
}

func (r *repositoryResolver) PullRequest(ctx context.Context, obj *model.Repository, number int) (*model.PullRequest, error) {
	meta, ok := repositoryMetaFor(obj.ID)
	if !ok || r.PullRequestRepo == nil {
		return nil, apperror.ErrNotFound
	}
	pr, err := r.PullRequestRepo.GetByNumber(ctx, meta.id, number)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, apperror.ErrNotFound
	}
	return mapEntityPullRequest(pr, obj), nil
}

func (r *repositoryResolver) PullRequests(ctx context.Context, obj *model.Repository, first *int, after *string, last *int, before *string, states []model.PullRequestState) (*model.PullRequestConnection, error) {
	if err := relay.ValidateFirst(first); err != nil {
		return nil, err
	}
	meta, ok := repositoryMetaFor(obj.ID)
	if !ok || r.PullRequestRepo == nil {
		return nil, apperror.ErrNotFound
	}

	pageSize := 30
	if first != nil && *first > 0 {
		pageSize = *first
	}

	prs, total, err := r.PullRequestRepo.ListByRepo(ctx, meta.id, domainrepo.ListPullRequestsFilter{
		State:   pullRequestStatesToFilter(states),
		Page:    1,
		PerPage: pageSize + 1,
	})
	if err != nil {
		return nil, err
	}

	return buildPullRequestConnection(prs, obj, first, after, total)
}

func derefStr(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func mutationActorID(ctx context.Context) (uuid.UUID, error) {
	viewer, ok := ViewerFromContext(ctx)
	if !ok || viewer == nil {
		return uuid.Nil, domain.ErrUnauthorized
	}
	return viewer.ID, nil
}

func loadRepositoryByID(ctx context.Context, r *Resolver, repoID uuid.UUID) (*entity.Repository, error) {
	loaders := LoadersFromContext(ctx)
	if loaders != nil {
		repo, err := loaders.RepositoryByID.Load(ctx, repoID)
		if err != nil {
			return nil, err
		}
		if repo == nil {
			return nil, apperror.ErrNotFound
		}
		return repo, nil
	}
	return nil, apperror.ErrNotFound
}

func loadLabelByID(ctx context.Context, r *Resolver, labelID uuid.UUID) (*entity.Label, error) {
	loaders := LoadersFromContext(ctx)
	if loaders != nil {
		label, err := loaders.LabelByID.Load(ctx, labelID)
		if err != nil {
			return nil, err
		}
		if label == nil {
			return nil, apperror.ErrNotFound
		}
		return label, nil
	}
	return nil, apperror.ErrNotFound
}

func issueStateString(state model.IssueState) string {
	switch state {
	case model.IssueStateClosed:
		return "closed"
	default:
		return "open"
	}
}

func mergeMethodString(method *model.PullRequestMergeMethod) string {
	if method == nil {
		return "merge"
	}
	switch *method {
	case model.PullRequestMergeMethodSquash:
		return "squash"
	case model.PullRequestMergeMethodRebase:
		return "rebase"
	default:
		return "merge"
	}
}

type labelableTarget struct {
	repositoryID   uuid.UUID
	organizationID uuid.UUID
	issueNumber    int
	isIssue        bool
	issue          *entity.Issue
	pullRequest    *entity.PullRequest
}

func (r *mutationResolver) resolveLabelable(ctx context.Context, globalID string) (*labelableTarget, error) {
	typeName, id, err := globalid.Decode(globalID)
	if err != nil {
		return nil, err
	}

	switch typeName {
	case globalid.TypeIssue:
		if r.IssueRepo == nil {
			return nil, apperror.ErrNotFound
		}
		issue, err := r.IssueRepo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if issue == nil || issue.State == "deleted" {
			return nil, apperror.ErrNotFound
		}
		return &labelableTarget{
			repositoryID:   issue.RepositoryID,
			organizationID: issue.OrganizationID,
			issueNumber:    issue.Number,
			isIssue:        true,
			issue:          issue,
		}, nil
	case globalid.TypePullRequest:
		if r.PullRequestRepo == nil {
			return nil, apperror.ErrNotFound
		}
		pr, err := r.PullRequestRepo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if pr == nil {
			return nil, apperror.ErrNotFound
		}
		return &labelableTarget{
			repositoryID:   pr.RepositoryID,
			organizationID: pr.OrganizationID,
			issueNumber:    pr.Number,
			isIssue:        false,
			pullRequest:    pr,
		}, nil
	default:
		return nil, apperror.ErrNotFound
	}
}

func (r *mutationResolver) refreshLabelable(ctx context.Context, target *labelableTarget) (model.Labelable, error) {
	if target.isIssue {
		if r.IssueRepo == nil {
			return mapIssue(target.issue), nil
		}
		issue, err := r.IssueRepo.GetByID(ctx, target.issue.ID)
		if err != nil {
			return nil, err
		}
		if issue == nil {
			return nil, apperror.ErrNotFound
		}
		return mapIssue(issue), nil
	}
	if r.PullRequestRepo == nil {
		return mapPullRequest(target.pullRequest), nil
	}
	pr, err := r.PullRequestRepo.GetByID(ctx, target.pullRequest.ID)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, apperror.ErrNotFound
	}
	return mapPullRequest(pr), nil
}

// CreateIssue is the resolver for the createIssue field.
func (r *mutationResolver) CreateIssue(ctx context.Context, input model.CreateIssueInput) (*model.CreateIssuePayload, error) {
	if err := RequireScope(ctx, ScopeRepo); err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.Title) == "" {
		return nil, apperror.ErrValidation
	}

	_, repoID, err := globalid.Decode(input.RepositoryID)
	if err != nil {
		return nil, err
	}

	repo, err := loadRepositoryByID(ctx, r.Resolver, repoID)
	if err != nil {
		return nil, err
	}

	actorID, err := mutationActorID(ctx)
	if err != nil {
		return nil, err
	}

	if r.CreateIssueUC == nil {
		return nil, apperror.ErrNotFound
	}

	issue, err := r.CreateIssueUC.Execute(ctx, issueusecase.CreateIssueInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repoID,
		ActorID:        actorID,
		Title:          input.Title,
		Body:           derefStr(input.Body),
	})
	if err != nil {
		return nil, err
	}

	return &model.CreateIssuePayload{
		Issue:            mapIssue(issue),
		ClientMutationID: input.ClientMutationID,
	}, nil
}

// UpdateIssue is the resolver for the updateIssue field.
func (r *mutationResolver) UpdateIssue(ctx context.Context, input model.UpdateIssueInput) (*model.UpdateIssuePayload, error) {
	if err := RequireScope(ctx, ScopeRepo); err != nil {
		return nil, err
	}

	_, issueID, err := globalid.Decode(input.ID)
	if err != nil {
		return nil, err
	}
	if r.IssueRepo == nil || r.UpdateIssueUC == nil {
		return nil, apperror.ErrNotFound
	}

	issue, err := r.IssueRepo.GetByID(ctx, issueID)
	if err != nil {
		return nil, err
	}
	if issue == nil || issue.State == "deleted" {
		return nil, apperror.ErrNotFound
	}

	actorID, err := mutationActorID(ctx)
	if err != nil {
		return nil, err
	}

	updateInput := issueusecase.UpdateIssueInput{
		OrganizationID: issue.OrganizationID,
		RepositoryID:   issue.RepositoryID,
		IssueNumber:    issue.Number,
		ActorID:        actorID,
	}
	if input.Title != nil {
		updateInput.Title = input.Title
	}
	if input.Body != nil {
		updateInput.Body = input.Body
	}
	if input.State != nil {
		state := issueStateString(*input.State)
		updateInput.State = &state
	}
	if input.LabelIds != nil {
		names := make([]string, 0, len(input.LabelIds))
		for _, labelGlobalID := range input.LabelIds {
			_, labelID, decodeErr := globalid.Decode(labelGlobalID)
			if decodeErr != nil {
				return nil, decodeErr
			}
			label, loadErr := loadLabelByID(ctx, r.Resolver, labelID)
			if loadErr != nil {
				return nil, loadErr
			}
			if label.RepositoryID != issue.RepositoryID {
				return nil, apperror.ErrNotFound
			}
			names = append(names, label.Name)
		}
		updateInput.LabelNames = names
	}

	updated, err := r.UpdateIssueUC.Execute(ctx, updateInput)
	if err != nil {
		return nil, err
	}

	return &model.UpdateIssuePayload{
		Issue:            mapIssue(updated),
		ClientMutationID: input.ClientMutationID,
	}, nil
}

// CloseIssue is the resolver for the closeIssue field.
func (r *mutationResolver) CloseIssue(ctx context.Context, input model.CloseIssueInput) (*model.CloseIssuePayload, error) {
	if err := RequireScope(ctx, ScopeRepo); err != nil {
		return nil, err
	}

	_, issueID, err := globalid.Decode(input.IssueID)
	if err != nil {
		return nil, err
	}
	if r.IssueRepo == nil {
		return nil, apperror.ErrNotFound
	}

	issue, err := r.IssueRepo.GetByID(ctx, issueID)
	if err != nil {
		return nil, err
	}
	if issue == nil || issue.State == "deleted" {
		return nil, apperror.ErrNotFound
	}

	if issue.State == "closed" {
		return &model.CloseIssuePayload{
			Issue:            mapIssue(issue),
			ClientMutationID: input.ClientMutationID,
		}, nil
	}

	if r.UpdateIssueUC == nil {
		return nil, apperror.ErrNotFound
	}

	actorID, err := mutationActorID(ctx)
	if err != nil {
		return nil, err
	}

	closed := "closed"
	updated, err := r.UpdateIssueUC.Execute(ctx, issueusecase.UpdateIssueInput{
		OrganizationID: issue.OrganizationID,
		RepositoryID: issue.RepositoryID,
		IssueNumber:    issue.Number,
		ActorID:        actorID,
		State:          &closed,
	})
	if err != nil {
		return nil, err
	}

	return &model.CloseIssuePayload{
		Issue:            mapIssue(updated),
		ClientMutationID: input.ClientMutationID,
	}, nil
}

// ReopenIssue is the resolver for the reopenIssue field.
func (r *mutationResolver) ReopenIssue(ctx context.Context, input model.ReopenIssueInput) (*model.ReopenIssuePayload, error) {
	if err := RequireScope(ctx, ScopeRepo); err != nil {
		return nil, err
	}

	_, issueID, err := globalid.Decode(input.IssueID)
	if err != nil {
		return nil, err
	}
	if r.IssueRepo == nil {
		return nil, apperror.ErrNotFound
	}

	issue, err := r.IssueRepo.GetByID(ctx, issueID)
	if err != nil {
		return nil, err
	}
	if issue == nil || issue.State == "deleted" {
		return nil, apperror.ErrNotFound
	}

	if issue.State == "open" {
		return &model.ReopenIssuePayload{
			Issue:            mapIssue(issue),
			ClientMutationID: input.ClientMutationID,
		}, nil
	}

	if r.UpdateIssueUC == nil {
		return nil, apperror.ErrNotFound
	}

	actorID, err := mutationActorID(ctx)
	if err != nil {
		return nil, err
	}

	open := "open"
	updated, err := r.UpdateIssueUC.Execute(ctx, issueusecase.UpdateIssueInput{
		OrganizationID: issue.OrganizationID,
		RepositoryID:   issue.RepositoryID,
		IssueNumber:    issue.Number,
		ActorID:        actorID,
		State:          &open,
	})
	if err != nil {
		return nil, err
	}

	return &model.ReopenIssuePayload{
		Issue:            mapIssue(updated),
		ClientMutationID: input.ClientMutationID,
	}, nil
}

// AddComment is the resolver for the addComment field.
func (r *mutationResolver) AddComment(ctx context.Context, input model.AddCommentInput) (*model.AddCommentPayload, error) {
	if err := RequireScope(ctx, ScopeRepo); err != nil {
		return nil, err
	}

	typeName, subjectID, err := globalid.Decode(input.SubjectID)
	if err != nil {
		return nil, err
	}

	actorID, err := mutationActorID(ctx)
	if err != nil {
		return nil, err
	}

	switch typeName {
	case globalid.TypeIssue:
		if r.IssueRepo == nil || r.CreateCommentUC == nil {
			return nil, apperror.ErrNotFound
		}
		issue, err := r.IssueRepo.GetByID(ctx, subjectID)
		if err != nil {
			return nil, err
		}
		if issue == nil || issue.State == "deleted" {
			return nil, apperror.ErrNotFound
		}

		comment, err := r.CreateCommentUC.Execute(ctx, issueusecase.CreateCommentInput{
			OrganizationID: issue.OrganizationID,
			RepositoryID:   issue.RepositoryID,
			IssueNumber:    issue.Number,
			ActorID:        actorID,
			Body:           input.Body,
		})
		if err != nil {
			return nil, err
		}

		node := mapEntityComment(comment)
		cursor := relay.EncodeCursor(comment.ID, comment.CreatedAt)
		return &model.AddCommentPayload{
			CommentEdge:      &model.IssueCommentEdge{Cursor: cursor, Node: node},
			Comment:          node,
			Subject:          mapIssue(issue),
			ClientMutationID: input.ClientMutationID,
		}, nil
	case globalid.TypePullRequest:
		if r.PullRequestRepo == nil || r.CreateCommentUC == nil {
			return nil, apperror.ErrNotFound
		}
		pr, err := r.PullRequestRepo.GetByID(ctx, subjectID)
		if err != nil {
			return nil, err
		}
		if pr == nil {
			return nil, apperror.ErrNotFound
		}

		comment, err := r.CreateCommentUC.Execute(ctx, issueusecase.CreateCommentInput{
			OrganizationID: pr.OrganizationID,
			RepositoryID:   pr.RepositoryID,
			IssueNumber:    pr.Number,
			ActorID:        actorID,
			Body:           input.Body,
		})
		if err != nil {
			return nil, err
		}

		node := mapEntityComment(comment)
		cursor := relay.EncodeCursor(comment.ID, comment.CreatedAt)
		return &model.AddCommentPayload{
			CommentEdge:      &model.IssueCommentEdge{Cursor: cursor, Node: node},
			Comment:          node,
			Subject:          mapPullRequest(pr),
			ClientMutationID: input.ClientMutationID,
		}, nil
	default:
		return nil, apperror.ErrNotFound
	}
}

// CreatePullRequest is the resolver for the createPullRequest field.
func (r *mutationResolver) CreatePullRequest(ctx context.Context, input model.CreatePullRequestInput) (*model.CreatePullRequestPayload, error) {
	if err := RequireScope(ctx, ScopeRepo); err != nil {
		return nil, err
	}

	_, repoID, err := globalid.Decode(input.RepositoryID)
	if err != nil {
		return nil, err
	}

	repo, err := loadRepositoryByID(ctx, r.Resolver, repoID)
	if err != nil {
		return nil, err
	}

	actorID, err := mutationActorID(ctx)
	if err != nil {
		return nil, err
	}

	if r.CreatePRUC == nil {
		return nil, apperror.ErrNotFound
	}

	pr, err := r.CreatePRUC.Execute(ctx, prusecase.CreatePRInput{
		OrganizationID: repo.OrganizationID,
		RepositoryID:   repoID,
		GitPath:        repo.GitPath,
		ActorID:        actorID,
		Title:          input.Title,
		Body:           derefStr(input.Body),
		HeadRef:        input.HeadRefName,
		BaseRef:        input.BaseRefName,
	})
	if err != nil {
		return nil, err
	}

	return &model.CreatePullRequestPayload{
		PullRequest:      mapPullRequest(pr),
		ClientMutationID: input.ClientMutationID,
	}, nil
}

// MergePullRequest is the resolver for the mergePullRequest field.
func (r *mutationResolver) MergePullRequest(ctx context.Context, input model.MergePullRequestInput) (*model.MergePullRequestPayload, error) {
	if err := RequireScope(ctx, ScopeRepo); err != nil {
		return nil, err
	}

	_, prID, err := globalid.Decode(input.PullRequestID)
	if err != nil {
		return nil, err
	}
	if r.PullRequestRepo == nil {
		return nil, apperror.ErrNotFound
	}

	pr, err := r.PullRequestRepo.GetByID(ctx, prID)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, apperror.ErrNotFound
	}

	mergeable := mapMergeableState(pr.MergeableState)
	if mergeable != model.MergeableStateMergeable {
		return nil, apperror.ErrValidation
	}

	repo, err := loadRepositoryByID(ctx, r.Resolver, pr.RepositoryID)
	if err != nil {
		return nil, err
	}

	actorID, err := mutationActorID(ctx)
	if err != nil {
		return nil, err
	}

	if r.MergePRUC == nil {
		return nil, apperror.ErrNotFound
	}

	merged, err := r.MergePRUC.Execute(ctx, prusecase.MergePRInput{
		OrganizationID: pr.OrganizationID,
		RepositoryID:   pr.RepositoryID,
		GitPath:        repo.GitPath,
		ActorID:        actorID,
		Number:         pr.Number,
		MergeMethod:    mergeMethodString(input.MergeMethod),
	})
	if err != nil {
		return nil, err
	}

	return &model.MergePullRequestPayload{
		PullRequest:      mapPullRequest(merged),
		ClientMutationID: input.ClientMutationID,
	}, nil
}

// ClosePullRequest is the resolver for the closePullRequest field.
func (r *mutationResolver) ClosePullRequest(ctx context.Context, input model.ClosePullRequestInput) (*model.ClosePullRequestPayload, error) {
	if err := RequireScope(ctx, ScopeRepo); err != nil {
		return nil, err
	}

	_, prID, err := globalid.Decode(input.PullRequestID)
	if err != nil {
		return nil, err
	}
	if r.PullRequestRepo == nil {
		return nil, apperror.ErrNotFound
	}

	pr, err := r.PullRequestRepo.GetByID(ctx, prID)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, apperror.ErrNotFound
	}

	if pr.State == entity.PullRequestStateClosed || pr.State == entity.PullRequestStateMerged {
		return &model.ClosePullRequestPayload{
			PullRequest:      mapPullRequest(pr),
			ClientMutationID: input.ClientMutationID,
		}, nil
	}

	pr.State = entity.PullRequestStateClosed
	if err := r.PullRequestRepo.Update(ctx, pr); err != nil {
		return nil, err
	}

	return &model.ClosePullRequestPayload{
		PullRequest:      mapPullRequest(pr),
		ClientMutationID: input.ClientMutationID,
	}, nil
}

// AddLabelsToLabelable is the resolver for the addLabelsToLabelable field.
func (r *mutationResolver) AddLabelsToLabelable(ctx context.Context, input model.AddLabelsToLabelableInput) (*model.AddLabelsToLabelablePayload, error) {
	if err := RequireScope(ctx, ScopeRepo); err != nil {
		return nil, err
	}
	if r.LabelRepo == nil {
		return nil, apperror.ErrNotFound
	}

	target, err := r.resolveLabelable(ctx, input.LabelableID)
	if err != nil {
		return nil, err
	}
	if !target.isIssue {
		return nil, apperror.ErrNotFound
	}

	for _, labelGlobalID := range input.LabelIds {
		_, labelID, decodeErr := globalid.Decode(labelGlobalID)
		if decodeErr != nil {
			return nil, decodeErr
		}
		label, loadErr := loadLabelByID(ctx, r.Resolver, labelID)
		if loadErr != nil {
			return nil, loadErr
		}
		if label.RepositoryID != target.repositoryID {
			return nil, apperror.ErrNotFound
		}
		if err := r.LabelRepo.AddToIssue(ctx, target.repositoryID, target.issueNumber, labelID); err != nil {
			return nil, err
		}
	}

	labelable, err := r.refreshLabelable(ctx, target)
	if err != nil {
		return nil, err
	}

	return &model.AddLabelsToLabelablePayload{
		Labelable:        labelable,
		ClientMutationID: input.ClientMutationID,
	}, nil
}

// RemoveLabelsFromLabelable is the resolver for the removeLabelsFromLabelable field.
func (r *mutationResolver) RemoveLabelsFromLabelable(ctx context.Context, input model.RemoveLabelsFromLabelableInput) (*model.RemoveLabelsFromLabelablePayload, error) {
	if err := RequireScope(ctx, ScopeRepo); err != nil {
		return nil, err
	}
	if r.LabelRepo == nil {
		return nil, apperror.ErrNotFound
	}

	target, err := r.resolveLabelable(ctx, input.LabelableID)
	if err != nil {
		return nil, err
	}
	if !target.isIssue {
		return nil, apperror.ErrNotFound
	}

	for _, labelGlobalID := range input.LabelIds {
		_, labelID, decodeErr := globalid.Decode(labelGlobalID)
		if decodeErr != nil {
			return nil, decodeErr
		}
		label, loadErr := loadLabelByID(ctx, r.Resolver, labelID)
		if loadErr != nil {
			return nil, loadErr
		}
		if label.RepositoryID != target.repositoryID {
			return nil, apperror.ErrNotFound
		}
		if err := r.LabelRepo.RemoveFromIssue(ctx, target.repositoryID, target.issueNumber, labelID); err != nil {
			return nil, err
		}
	}

	labelable, err := r.refreshLabelable(ctx, target)
	if err != nil {
		return nil, err
	}

	return &model.RemoveLabelsFromLabelablePayload{
		Labelable:        labelable,
		ClientMutationID: input.ClientMutationID,
	}, nil
}

// Viewer is the resolver for the viewer field.
func (r *queryResolver) Viewer(ctx context.Context) (*model.User, error) {
	viewer, ok := ViewerFromContext(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}
	if r.GetCurrentUserUC != nil {
		user, err := r.GetCurrentUserUC.Execute(ctx, appmiddleware.UUIDToInt64(viewer.ID))
		if err != nil {
			return nil, err
		}
		return mapDomainUser(user), nil
	}
	return mapEntityUser(viewer), nil
}

// Node is the resolver for the node field.
func (r *queryResolver) Node(ctx context.Context, id string) (model.Node, error) {
	typeName, nodeID, err := globalid.Decode(id)
	if err != nil {
		return nil, nil
	}

	loaders := LoadersFromContext(ctx)
	switch typeName {
	case globalid.TypeUser:
		if loaders == nil {
			return nil, nil
		}
		user, err := loaders.UserByID.Load(ctx, nodeID)
		if err != nil || user == nil {
			return nil, nil
		}
		return mapEntityUser(user), nil
	case globalid.TypeRepository:
		if loaders == nil {
			return nil, nil
		}
		repo, err := loaders.RepositoryByID.Load(ctx, nodeID)
		if err != nil || repo == nil {
			return nil, nil
		}
		return mapEntityRepository(repo), nil
	case globalid.TypeIssue:
		if r.IssueRepo == nil {
			return nil, nil
		}
		issue, err := r.IssueRepo.GetByID(ctx, nodeID)
		if err != nil || issue == nil {
			return nil, nil
		}
		repoModel := mapEntityRepository(&entity.Repository{
			ID:             issue.RepositoryID,
			OrganizationID: issue.OrganizationID,
		})
		return mapEntityIssue(issue, repoModel), nil
	case globalid.TypePullRequest:
		if r.PullRequestRepo == nil {
			return nil, nil
		}
		pr, err := r.PullRequestRepo.GetByID(ctx, nodeID)
		if err != nil || pr == nil {
			return nil, nil
		}
		repoModel := mapEntityRepository(&entity.Repository{
			ID:             pr.RepositoryID,
			OrganizationID: pr.OrganizationID,
		})
		return mapEntityPullRequest(pr, repoModel), nil
	case globalid.TypeOrganization:
		return nil, nil
	case globalid.TypeLabel:
		if loaders == nil {
			return nil, nil
		}
		label, err := loaders.LabelByID.Load(ctx, nodeID)
		if err != nil || label == nil {
			return nil, nil
		}
		return mapEntityLabel(label), nil
	case globalid.TypeMilestone:
		if loaders == nil {
			return nil, nil
		}
		milestone, err := loaders.MilestoneByID.Load(ctx, nodeID)
		if err != nil || milestone == nil {
			return nil, nil
		}
		return mapEntityMilestone(milestone), nil
	case globalid.TypeIssueComment:
		if r.CommentRepo == nil {
			return nil, nil
		}
		comment, err := r.CommentRepo.GetByID(ctx, nodeID)
		if err != nil || comment == nil {
			return nil, nil
		}
		return mapEntityComment(comment), nil
	default:
		return nil, nil
	}
}

func mapEntityMilestone(milestone *entity.Milestone) *model.Milestone {
	if milestone == nil {
		return nil
	}
	createdAt := formatDateTime(milestone.CreatedAt)
	state := model.MilestoneStateOpen
	if strings.EqualFold(milestone.State, "closed") {
		state = model.MilestoneStateClosed
	}
	mapped := &model.Milestone{
		ID:        globalid.Encode(globalid.TypeMilestone, milestone.ID),
		Number:    milestone.Number,
		Title:     milestone.Title,
		State:     state,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
	if milestone.Description != "" {
		mapped.Description = &milestone.Description
	}
	if milestone.DueOn != nil {
		dueOn := formatDateTime(*milestone.DueOn)
		mapped.DueOn = &dueOn
	}
	if milestone.ClosedAt != nil {
		closedAt := formatDateTime(*milestone.ClosedAt)
		mapped.ClosedAt = &closedAt
	}
	return mapped
}

// Repository is the resolver for the repository field.
func (r *queryResolver) Repository(ctx context.Context, owner string, name string) (*model.Repository, error) {
	if r.GetRepositoryUC == nil {
		return nil, repoUC.ErrNotFound
	}
	input := repoUC.GetRepositoryInput{
		OwnerLogin: owner,
		Name:       name,
	}
	if viewer, ok := ViewerFromContext(ctx); ok {
		input.RequestUserID = viewer.ID
	}

	repo, err := r.GetRepositoryUC.Execute(ctx, input)
	if err != nil {
		if errors.Is(err, repoUC.ErrNotFound) {
			return nil, repoUC.ErrNotFound
		}
		return nil, err
	}
	return mapEntityRepository(repo), nil
}

// Organization is the resolver for the organization field.
func (r *queryResolver) Organization(ctx context.Context, login string) (*model.Organization, error) {
	if r.GetOrgUC == nil {
		return nil, domain.ErrNotFound
	}
	org, err := r.GetOrgUC.Execute(ctx, login)
	if err != nil {
		return nil, err
	}
	return mapDomainOrganization(org), nil
}

// User is the resolver for the user field.
func (r *queryResolver) User(ctx context.Context, login string) (*model.User, error) {
	if r.GetUserByLoginUC == nil {
		return nil, domain.ErrNotFound
	}
	user, err := r.GetUserByLoginUC.Execute(ctx, login)
	if err != nil {
		return nil, err
	}
	return mapDomainUser(user), nil
}
