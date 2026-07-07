package workflow

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// EvalContext supplies the named contexts an expression can reference
// (github, env, secrets, matrix, runner, job, strategy, ...) plus the status
// needed by the success()/failure()/always()/cancelled() functions.
//
// Contexts maps a context name to its key/value pairs, e.g.
// {"github": {"sha": "abc", "ref_name": "main"}, "matrix": {"os": "linux"}}.
type EvalContext struct {
	Contexts  map[string]map[string]string
	Failed    bool // a previous step/job in scope has failed
	Cancelled bool
}

func (c *EvalContext) lookup(path string) (exprValue, bool) {
	parts := strings.SplitN(path, ".", 2)
	if len(parts) != 2 || c == nil {
		return exprValue{}, false
	}
	ctx, ok := c.Contexts[parts[0]]
	if !ok {
		return exprValue{}, false
	}
	v, ok := ctx[parts[1]]
	if !ok {
		return exprValue{}, false
	}
	return stringValue(v), true
}

type valueType int

const (
	tNull valueType = iota
	tString
	tNumber
	tBool
)

type exprValue struct {
	t valueType
	s string
	n float64
	b bool
}

func nullValue() exprValue            { return exprValue{t: tNull} }
func stringValue(s string) exprValue  { return exprValue{t: tString, s: s} }
func numberValue(n float64) exprValue { return exprValue{t: tNumber, n: n} }
func boolValue(b bool) exprValue      { return exprValue{t: tBool, b: b} }

// truthy applies GitHub Actions truthiness: null, false, 0, NaN and "" are
// false; everything else is true.
func (v exprValue) truthy() bool {
	switch v.t {
	case tNull:
		return false
	case tBool:
		return v.b
	case tNumber:
		return v.n != 0 && !math.IsNaN(v.n)
	case tString:
		return v.s != ""
	}
	return false
}

// asString renders a value the way GitHub interpolates it into strings.
func (v exprValue) asString() string {
	switch v.t {
	case tNull:
		return ""
	case tBool:
		if v.b {
			return "true"
		}
		return "false"
	case tNumber:
		if v.n == math.Trunc(v.n) && !math.IsInf(v.n, 0) {
			return strconv.FormatInt(int64(v.n), 10)
		}
		return strconv.FormatFloat(v.n, 'g', -1, 64)
	default:
		return v.s
	}
}

func (v exprValue) asNumber() (float64, bool) {
	switch v.t {
	case tNumber:
		return v.n, true
	case tBool:
		if v.b {
			return 1, true
		}
		return 0, true
	case tNull:
		return 0, true
	case tString:
		if v.s == "" {
			return 0, true
		}
		n, err := strconv.ParseFloat(strings.TrimSpace(v.s), 64)
		if err != nil {
			return 0, false
		}
		return n, true
	}
	return 0, false
}

// EvaluateExpression evaluates a parsed expression AST against ctx.
func EvaluateExpression(node ExprNode, ctx *EvalContext) (exprValue, error) {
	switch node.Kind {
	case "literal":
		return literalValue(node.Value), nil
	case "context":
		return evalContextPath(node.Value, ctx), nil
	case "unary":
		if node.Op == "!" {
			if node.Operand == nil {
				return nullValue(), fmt.Errorf("missing operand for '!'")
			}
			operand, err := EvaluateExpression(*node.Operand, ctx)
			if err != nil {
				return nullValue(), err
			}
			return boolValue(!operand.truthy()), nil
		}
		return nullValue(), fmt.Errorf("unsupported unary operator %q", node.Op)
	case "binary":
		return evalBinary(node, ctx)
	case "call":
		return evalCall(node, ctx)
	default:
		return nullValue(), fmt.Errorf("unsupported expression node %q", node.Kind)
	}
}

func literalValue(raw string) exprValue {
	switch raw {
	case "true":
		return boolValue(true)
	case "false":
		return boolValue(false)
	}
	if n, err := strconv.ParseFloat(raw, 64); err == nil {
		return numberValue(n)
	}
	return stringValue(raw)
}

func evalContextPath(path string, ctx *EvalContext) exprValue {
	// Status functions are sometimes written bare (rare); the common contexts
	// are dotted paths resolved against the supplied maps.
	if v, ok := ctx.lookup(path); ok {
		return v
	}
	return nullValue()
}

func evalBinary(node ExprNode, ctx *EvalContext) (exprValue, error) {
	if node.Left == nil || node.Right == nil {
		return nullValue(), fmt.Errorf("binary %q missing operand", node.Op)
	}
	// Short-circuit logical operators.
	if node.Op == "&&" || node.Op == "||" {
		left, err := EvaluateExpression(*node.Left, ctx)
		if err != nil {
			return nullValue(), err
		}
		if node.Op == "&&" {
			if !left.truthy() {
				return left, nil
			}
			return EvaluateExpression(*node.Right, ctx)
		}
		// ||
		if left.truthy() {
			return left, nil
		}
		return EvaluateExpression(*node.Right, ctx)
	}

	left, err := EvaluateExpression(*node.Left, ctx)
	if err != nil {
		return nullValue(), err
	}
	right, err := EvaluateExpression(*node.Right, ctx)
	if err != nil {
		return nullValue(), err
	}

	switch node.Op {
	case "==":
		return boolValue(looseEqual(left, right)), nil
	case "!=":
		return boolValue(!looseEqual(left, right)), nil
	case "<", "<=", ">", ">=":
		return compare(node.Op, left, right), nil
	}
	return nullValue(), fmt.Errorf("unsupported binary operator %q", node.Op)
}

func looseEqual(a, b exprValue) bool {
	// Numbers compare numerically; otherwise GitHub compares strings
	// case-insensitively.
	if a.t == tNumber || b.t == tNumber {
		an, aok := a.asNumber()
		bn, bok := b.asNumber()
		if aok && bok {
			return an == bn
		}
	}
	if a.t == tBool || b.t == tBool {
		return a.truthy() == b.truthy()
	}
	return strings.EqualFold(a.asString(), b.asString())
}

func compare(op string, a, b exprValue) exprValue {
	an, aok := a.asNumber()
	bn, bok := b.asNumber()
	if !aok || !bok {
		return boolValue(false)
	}
	switch op {
	case "<":
		return boolValue(an < bn)
	case "<=":
		return boolValue(an <= bn)
	case ">":
		return boolValue(an > bn)
	case ">=":
		return boolValue(an >= bn)
	}
	return boolValue(false)
}

func evalCall(node ExprNode, ctx *EvalContext) (exprValue, error) {
	args := make([]exprValue, 0, len(node.Args))
	for i := range node.Args {
		v, err := EvaluateExpression(node.Args[i], ctx)
		if err != nil {
			return nullValue(), err
		}
		args = append(args, v)
	}
	switch node.Fn {
	case "success":
		return boolValue(!ctx.Failed && !ctx.Cancelled), nil
	case "failure":
		return boolValue(ctx.Failed), nil
	case "cancelled":
		return boolValue(ctx.Cancelled), nil
	case "always":
		return boolValue(true), nil
	case "contains":
		if len(args) != 2 {
			return nullValue(), fmt.Errorf("contains expects 2 args")
		}
		return boolValue(strings.Contains(strings.ToLower(args[0].asString()), strings.ToLower(args[1].asString()))), nil
	case "startsWith":
		if len(args) != 2 {
			return nullValue(), fmt.Errorf("startsWith expects 2 args")
		}
		return boolValue(strings.HasPrefix(strings.ToLower(args[0].asString()), strings.ToLower(args[1].asString()))), nil
	case "endsWith":
		if len(args) != 2 {
			return nullValue(), fmt.Errorf("endsWith expects 2 args")
		}
		return boolValue(strings.HasSuffix(strings.ToLower(args[0].asString()), strings.ToLower(args[1].asString()))), nil
	case "join":
		// Arrays aren't represented in this simplified value model, so join of a
		// scalar is the scalar itself.
		if len(args) == 0 {
			return stringValue(""), nil
		}
		return stringValue(args[0].asString()), nil
	case "format":
		return formatFn(args)
	case "toJSON":
		if len(args) != 1 {
			return nullValue(), fmt.Errorf("toJSON expects 1 arg")
		}
		return stringValue(strconv.Quote(args[0].asString())), nil
	default:
		return nullValue(), fmt.Errorf("unsupported function %q", node.Fn)
	}
}

func formatFn(args []exprValue) (exprValue, error) {
	if len(args) == 0 {
		return stringValue(""), nil
	}
	tmpl := args[0].asString()
	var sb strings.Builder
	for i := 0; i < len(tmpl); i++ {
		ch := tmpl[i]
		if ch == '{' && i+1 < len(tmpl) {
			// Escaped '{{'
			if tmpl[i+1] == '{' {
				sb.WriteByte('{')
				i++
				continue
			}
			end := strings.IndexByte(tmpl[i:], '}')
			if end > 0 {
				idxStr := tmpl[i+1 : i+end]
				if n, err := strconv.Atoi(idxStr); err == nil && n+1 < len(args) {
					sb.WriteString(args[n+1].asString())
					i += end
					continue
				}
			}
		}
		if ch == '}' && i+1 < len(tmpl) && tmpl[i+1] == '}' {
			sb.WriteByte('}')
			i++
			continue
		}
		sb.WriteByte(ch)
	}
	return stringValue(sb.String()), nil
}

// InterpolateString replaces every ${{ ... }} span in s with the string form of
// its evaluated result. Spans that fail to parse or evaluate are replaced with
// an empty string (mirroring GitHub, which treats unresolved context access as
// empty) and the first such error is returned for diagnostics.
func InterpolateString(s string, ctx *EvalContext) (string, error) {
	var sb strings.Builder
	var firstErr error
	i := 0
	for i < len(s) {
		start := strings.Index(s[i:], "${{")
		if start < 0 {
			sb.WriteString(s[i:])
			break
		}
		start += i
		sb.WriteString(s[i:start])
		end := findExprEnd(s, start+3)
		if end < 0 {
			// Unclosed; emit the rest verbatim.
			sb.WriteString(s[start:])
			break
		}
		inner := strings.TrimSpace(s[start+3 : end])
		val, err := evaluateInner(inner, ctx)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		sb.WriteString(val)
		i = end + 2
	}
	return sb.String(), firstErr
}

// findExprEnd returns the index of the closing "}}" for a span opened at from,
// accounting for nesting, or -1 if unterminated.
func findExprEnd(s string, from int) int {
	depth := 1
	for j := from; j+1 < len(s); j++ {
		if s[j] == '$' && j+2 < len(s) && s[j+1] == '{' && s[j+2] == '{' {
			depth++
			j += 2
			continue
		}
		if s[j] == '}' && s[j+1] == '}' {
			depth--
			if depth == 0 {
				return j
			}
			j++
		}
	}
	return -1
}

func evaluateInner(inner string, ctx *EvalContext) (string, error) {
	nodes, diags := ParseExpression(inner)
	for _, d := range diags {
		if d.Severity == "error" {
			return "", fmt.Errorf("expression %q: %s", inner, d.Message)
		}
	}
	if len(nodes) == 0 {
		return "", nil
	}
	v, err := EvaluateExpression(nodes[0], ctx)
	if err != nil {
		return "", err
	}
	return v.asString(), nil
}

// EvaluateCondition evaluates an `if:` expression to a boolean. A bare
// expression (no ${{ }}) is evaluated directly; GitHub also allows the wrapped
// form. An empty condition is true.
func EvaluateCondition(cond string, ctx *EvalContext) (bool, error) {
	cond = strings.TrimSpace(cond)
	if cond == "" {
		return true, nil
	}
	if strings.Contains(cond, "${{") {
		start := strings.Index(cond, "${{")
		end := findExprEnd(cond, start+3)
		if end < 0 {
			return false, fmt.Errorf("unterminated if expression")
		}
		cond = strings.TrimSpace(cond[start+3 : end])
	}
	nodes, diags := ParseExpression(cond)
	for _, d := range diags {
		if d.Severity == "error" {
			return false, fmt.Errorf("if %q: %s", cond, d.Message)
		}
	}
	if len(nodes) == 0 {
		return true, nil
	}
	v, err := EvaluateExpression(nodes[0], ctx)
	if err != nil {
		return false, err
	}
	return v.truthy(), nil
}
