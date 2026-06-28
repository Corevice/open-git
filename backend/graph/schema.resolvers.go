package graph

// This file will be automatically regenerated based on the schema, any resolver
// implementations will be copied through when generating and any unknown code
// will be moved or copied to graph/resolver.go. When modifying or deleting this
// file, make sure to move any code to resolver.go.

import (
	"context"
	"fmt"

	"github.com/open-git/backend/graph/generated"
	"github.com/open-git/backend/graph/model"
)

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }

// CreateIssue is the resolver for the createIssue field.
func (r *mutationResolver) CreateIssue(ctx context.Context, input model.CreateIssueInput) (*model.CreateIssuePayload, error) {
	panic(fmt.Errorf("not implemented: CreateIssue - createIssue"))
}

// UpdateIssue is the resolver for the updateIssue field.
func (r *mutationResolver) UpdateIssue(ctx context.Context, input model.UpdateIssueInput) (*model.UpdateIssuePayload, error) {
	panic(fmt.Errorf("not implemented: UpdateIssue - updateIssue"))
}

// CloseIssue is the resolver for the closeIssue field.
func (r *mutationResolver) CloseIssue(ctx context.Context, input model.CloseIssueInput) (*model.CloseIssuePayload, error) {
	panic(fmt.Errorf("not implemented: CloseIssue - closeIssue"))
}

// ReopenIssue is the resolver for the reopenIssue field.
func (r *mutationResolver) ReopenIssue(ctx context.Context, input model.ReopenIssueInput) (*model.ReopenIssuePayload, error) {
	panic(fmt.Errorf("not implemented: ReopenIssue - reopenIssue"))
}

// AddComment is the resolver for the addComment field.
func (r *mutationResolver) AddComment(ctx context.Context, input model.AddCommentInput) (*model.AddCommentPayload, error) {
	panic(fmt.Errorf("not implemented: AddComment - addComment"))
}

// CreatePullRequest is the resolver for the createPullRequest field.
func (r *mutationResolver) CreatePullRequest(ctx context.Context, input model.CreatePullRequestInput) (*model.CreatePullRequestPayload, error) {
	panic(fmt.Errorf("not implemented: CreatePullRequest - createPullRequest"))
}

// MergePullRequest is the resolver for the mergePullRequest field.
func (r *mutationResolver) MergePullRequest(ctx context.Context, input model.MergePullRequestInput) (*model.MergePullRequestPayload, error) {
	panic(fmt.Errorf("not implemented: MergePullRequest - mergePullRequest"))
}

// ClosePullRequest is the resolver for the closePullRequest field.
func (r *mutationResolver) ClosePullRequest(ctx context.Context, input model.ClosePullRequestInput) (*model.ClosePullRequestPayload, error) {
	panic(fmt.Errorf("not implemented: ClosePullRequest - closePullRequest"))
}

// AddLabelsToLabelable is the resolver for the addLabelsToLabelable field.
func (r *mutationResolver) AddLabelsToLabelable(ctx context.Context, input model.AddLabelsToLabelableInput) (*model.AddLabelsToLabelablePayload, error) {
	panic(fmt.Errorf("not implemented: AddLabelsToLabelable - addLabelsToLabelable"))
}

// RemoveLabelsFromLabelable is the resolver for the removeLabelsFromLabelable field.
func (r *mutationResolver) RemoveLabelsFromLabelable(ctx context.Context, input model.RemoveLabelsFromLabelableInput) (*model.RemoveLabelsFromLabelablePayload, error) {
	panic(fmt.Errorf("not implemented: RemoveLabelsFromLabelable - removeLabelsFromLabelable"))
}

// Viewer is the resolver for the viewer field.
func (r *queryResolver) Viewer(ctx context.Context) (*model.User, error) {
	panic(fmt.Errorf("not implemented: Viewer - viewer"))
}

// Node is the resolver for the node field.
func (r *queryResolver) Node(ctx context.Context, id string) (model.Node, error) {
	panic(fmt.Errorf("not implemented: Node - node"))
}

// Repository is the resolver for the repository field.
func (r *queryResolver) Repository(ctx context.Context, owner string, name string) (*model.Repository, error) {
	panic(fmt.Errorf("not implemented: Repository - repository"))
}

// Organization is the resolver for the organization field.
func (r *queryResolver) Organization(ctx context.Context, login string) (*model.Organization, error) {
	panic(fmt.Errorf("not implemented: Organization - organization"))
}

// User is the resolver for the user field.
func (r *queryResolver) User(ctx context.Context, login string) (*model.User, error) {
	panic(fmt.Errorf("not implemented: User - user"))
}
