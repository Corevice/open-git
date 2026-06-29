package workflow

import (
	"strings"
	"testing"
)

func TestExpr_ContextAccess(t *testing.T) {
	nodes, diags := ParseExpression("github.ref")
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(nodes))
	}
	if nodes[0].Kind != "context" || nodes[0].Value != "github.ref" {
		t.Errorf("got %+v", nodes[0])
	}
}

func TestExpr_FunctionCall(t *testing.T) {
	nodes, diags := ParseExpression("contains(github.ref, 'main')")
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(nodes))
	}
	node := nodes[0]
	if node.Kind != "call" || node.Fn != "contains" {
		t.Errorf("got %+v", node)
	}
	if len(node.Args) != 2 {
		t.Fatalf("args: got %d, want 2", len(node.Args))
	}
	if node.Args[0].Kind != "context" || node.Args[0].Value != "github.ref" {
		t.Errorf("arg0: got %+v", node.Args[0])
	}
	if node.Args[1].Kind != "literal" || node.Args[1].Value != "main" {
		t.Errorf("arg1: got %+v", node.Args[1])
	}
}

func TestExpr_BinaryAnd(t *testing.T) {
	nodes, diags := ParseExpression("success() && env.CI")
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(nodes))
	}
	node := nodes[0]
	if node.Kind != "binary" || node.Op != "&&" {
		t.Errorf("got %+v", node)
	}
	if node.Left == nil || node.Left.Kind != "call" || node.Left.Fn != "success" {
		t.Errorf("left: got %+v", node.Left)
	}
	if node.Right == nil || node.Right.Kind != "context" || node.Right.Value != "env.CI" {
		t.Errorf("right: got %+v", node.Right)
	}
}

func TestExpr_UnaryNot(t *testing.T) {
	nodes, diags := ParseExpression("!cancelled()")
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(nodes))
	}
	node := nodes[0]
	if node.Kind != "unary" || node.Op != "!" {
		t.Errorf("got %+v", node)
	}
	if node.Operand == nil || node.Operand.Kind != "call" || node.Operand.Fn != "cancelled" {
		t.Errorf("operand: got %+v", node.Operand)
	}
}

func TestExpr_InvalidMissingClose(t *testing.T) {
	_, diags := ExtractExpressions("if: ${{ github.ref", 1)
	if len(diags) == 0 {
		t.Fatal("expected diagnostics for unclosed expression")
	}
	found := false
	for _, d := range diags {
		if d.Severity == "error" && strings.Contains(d.Message, "unclosed") {
			found = true
		}
	}
	if !found {
		t.Errorf("got diagnostics: %v", diags)
	}
}

func TestExtractExpressions_MultipleSpans(t *testing.T) {
	s := "echo ${{ github.ref }} and ${{ env.CI }}"
	nodes, diags := ExtractExpressions(s, 1)
	if len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(nodes))
	}
	if nodes[0].Kind != "context" || nodes[0].Value != "github.ref" {
		t.Errorf("node0: got %+v", nodes[0])
	}
	if nodes[1].Kind != "context" || nodes[1].Value != "env.CI" {
		t.Errorf("node1: got %+v", nodes[1])
	}
}
