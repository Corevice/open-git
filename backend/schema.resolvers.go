package graph

// THIS CODE WILL BE UPDATED WITH SCHEMA CHANGES. PREVIOUS IMPLEMENTATION FOR SCHEMA CHANGES WILL BE KEPT IN THE COMMENT SECTION. IMPLEMENTATION FOR UNCHANGED SCHEMA WILL BE KEPT.

import (
	"context"

	"github.com/open-git/backend/graph/generated"
	"github.com/open-git/backend/graph/model"
)

type Resolver struct{}

// CreateIssue is the resolver for the createIssue field.
func (r *mutationResolver) CreateIssue(ctx context.Context, input model.CreateIssueInput) (*model.CreateIssuePayload, error) {
	panic("not implemented")
}

// UpdateIssue is the resolver for the updateIssue field.
func (r *mutationResolver) UpdateIssue(ctx context.Context, input model.UpdateIssueInput) (*model.UpdateIssuePayload, error) {
	panic("not implemented")
}

// CloseIssue is the resolver for the closeIssue field.
func (r *mutationResolver) CloseIssue(ctx context.Context, input model.CloseIssueInput) (*model.CloseIssuePayload, error) {
	panic("not implemented")
}

// ReopenIssue is the resolver for the reopenIssue field.
func (r *mutationResolver) ReopenIssue(ctx context.Context, input model.ReopenIssueInput) (*model.ReopenIssuePayload, error) {
	panic("not implemented")
}

// AddComment is the resolver for the addComment field.
func (r *mutationResolver) AddComment(ctx context.Context, input model.AddCommentInput) (*model.AddCommentPayload, error) {
	panic("not implemented")
}

// CreatePullRequest is the resolver for the createPullRequest field.
func (r *mutationResolver) CreatePullRequest(ctx context.Context, input model.CreatePullRequestInput) (*model.CreatePullRequestPayload, error) {
	panic("not implemented")
}

// MergePullRequest is the resolver for the mergePullRequest field.
func (r *mutationResolver) MergePullRequest(ctx context.Context, input model.MergePullRequestInput) (*model.MergePullRequestPayload, error) {
	panic("not implemented")
}

// ClosePullRequest is the resolver for the closePullRequest field.
func (r *mutationResolver) ClosePullRequest(ctx context.Context, input model.ClosePullRequestInput) (*model.ClosePullRequestPayload, error) {
	panic("not implemented")
}

// AddLabelsToLabelable is the resolver for the addLabelsToLabelable field.
func (r *mutationResolver) AddLabelsToLabelable(ctx context.Context, input model.AddLabelsToLabelableInput) (*model.AddLabelsToLabelablePayload, error) {
	panic("not implemented")
}

// RemoveLabelsFromLabelable is the resolver for the removeLabelsFromLabelable field.
func (r *mutationResolver) RemoveLabelsFromLabelable(ctx context.Context, input model.RemoveLabelsFromLabelableInput) (*model.RemoveLabelsFromLabelablePayload, error) {
	panic("not implemented")
}

// Viewer is the resolver for the viewer field.
func (r *queryResolver) Viewer(ctx context.Context) (*model.User, error) {
	panic("not implemented")
}

// Node is the resolver for the node field.
func (r *queryResolver) Node(ctx context.Context, id string) (model.Node, error) {
	panic("not implemented")
}

// Repository is the resolver for the repository field.
func (r *queryResolver) Repository(ctx context.Context, owner string, name string) (*model.Repository, error) {
	panic("not implemented")
}

// Organization is the resolver for the organization field.
func (r *queryResolver) Organization(ctx context.Context, login string) (*model.Organization, error) {
	panic("not implemented")
}

// User is the resolver for the user field.
func (r *queryResolver) User(ctx context.Context, login string) (*model.User, error) {
	panic("not implemented")
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }

// !!! WARNING !!!
// The code below was going to be deleted when updating resolvers. It has been copied here so you have
// one last chance to move it out of harms way if you want. There are two reasons this happens:
//  - When renaming or deleting a resolver the old code will be put in here. You can safely delete
//    it when you're done.
//  - You have helper methods in this file. Move them out to keep these resolver files clean.
/*
	type Resolver struct{}
*/
