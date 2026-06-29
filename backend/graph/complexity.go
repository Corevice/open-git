package graph

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/open-git/backend/graph/generated"
	"github.com/open-git/backend/graph/model"
)

const (
	ComplexityLimit = 1000
	DepthLimit      = 10
)

func NewComplexityRoot() generated.ComplexityRoot {
	root := generated.ComplexityRoot{}

	root.Issue.Comments = func(childComplexity int, first *int, _ *string, last *int, _ *string) int {
		return listFieldComplexity(childComplexity, first, last)
	}
	root.Issue.Labels = func(childComplexity int, first *int, _ *string, last *int, _ *string) int {
		return listFieldComplexity(childComplexity, first, last)
	}

	root.Repository.Issues = func(childComplexity int, first *int, _ *string, last *int, _ *string, _ []model.IssueState, _ []string) int {
		return listFieldComplexity(childComplexity, first, last)
	}
	root.Repository.PullRequests = func(childComplexity int, first *int, _ *string, last *int, _ *string, _ []model.PullRequestState) int {
		return listFieldComplexity(childComplexity, first, last)
	}
	root.Repository.Labels = func(childComplexity int, first *int, _ *string, last *int, _ *string) int {
		return listFieldComplexity(childComplexity, first, last)
	}

	return root
}

func listFieldComplexity(childComplexity int, first, last *int) int {
	multiplier := 1
	if first != nil && *first > 0 {
		multiplier = *first
	} else if last != nil && *last > 0 {
		multiplier = *last
	}
	return childComplexity * multiplier
}

type depthLimiter struct {
	maxDepth int
}

func (d depthLimiter) ExtensionName() string {
	return "DepthLimiter"
}

func (d depthLimiter) Validate(_ graphql.ExecutableSchema) error {
	return nil
}

func (d depthLimiter) InterceptOperation(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	opCtx := graphql.GetOperationContext(ctx)
	if opCtx != nil && opCtx.Operation != nil {
		depth := selectionDepth(opCtx.Operation.SelectionSet, 0)
		if depth > d.maxDepth {
			return func(_ context.Context) *graphql.Response {
				return &graphql.Response{
					Errors: gqlerror.List{gqlerror.Errorf("query depth limit exceeded")},
				}
			}
		}
	}
	return next(ctx)
}

type domainDepthError struct {
	limit int
}

func (e *domainDepthError) Error() string {
	return "query depth limit exceeded"
}

func selectionDepth(set ast.SelectionSet, current int) int {
	if len(set) == 0 {
		return current
	}

	maxDepth := current
	for _, selection := range set {
		switch sel := selection.(type) {
		case *ast.Field:
			nextDepth := current + 1
			if childDepth := selectionDepth(sel.SelectionSet, nextDepth); childDepth > maxDepth {
				maxDepth = childDepth
			}
		case *ast.InlineFragment:
			if childDepth := selectionDepth(sel.SelectionSet, current); childDepth > maxDepth {
				maxDepth = childDepth
			}
		case *ast.FragmentSpread:
			continue
		}
	}
	return maxDepth
}

func NewDepthLimiter() graphql.HandlerExtension {
	return depthLimiter{maxDepth: DepthLimit}
}
