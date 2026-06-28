package graph_test

import (
	"os"
	"testing"

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
