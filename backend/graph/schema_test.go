package graph_test

import (
	"os"
	"testing"

	"github.com/open-git/backend/graph"
	"github.com/open-git/backend/graph/generated"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestSchemaParses(t *testing.T) {
	data, err := os.ReadFile("schema.graphqls")
	require.NoError(t, err)

	_, err = gqlparser.LoadSchema(&ast.Source{Name: "schema.graphqls", Input: string(data)})
	require.NoError(t, err)
}

func TestExecutableSchemaBuilds(t *testing.T) {
	srv := generated.NewExecutableSchema(generated.Config{
		Resolvers: &graph.Resolver{},
	})
	require.NotNil(t, srv)

	schema := srv.Schema()
	require.NotNil(t, schema)
	require.Contains(t, schema.Types, "Query")
	require.Contains(t, schema.Types, "PullRequest")

	prType := schema.Types["PullRequest"]
	require.NotNil(t, prType)

	fieldNames := make([]string, len(prType.Fields))
	for i, field := range prType.Fields {
		fieldNames[i] = field.Name
	}
	require.Contains(t, fieldNames, "mergeableState")
	require.NotContains(t, fieldNames, "mergeable")
}
