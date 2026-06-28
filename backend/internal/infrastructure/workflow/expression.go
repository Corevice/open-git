package workflow

import (
	"fmt"
	"strings"
	"unicode"
)

// ExprNode represents a node in a GitHub Actions expression AST.
type ExprNode struct {
	Kind    string     `json:"kind"`
	Value   string     `json:"value,omitempty"`
	Fn      string     `json:"fn,omitempty"`
	Args    []ExprNode `json:"args,omitempty"`
	Op      string     `json:"op,omitempty"`
	Left    *ExprNode  `json:"left,omitempty"`
	Right   *ExprNode  `json:"right,omitempty"`
	Operand *ExprNode  `json:"operand,omitempty"`
}

type exprParser struct {
	input string
	pos   int
	line  int
	col   int
}

var knownFunctions = map[string]bool{
	"contains":    true,
	"startsWith":  true,
	"endsWith":    true,
	"format":      true,
	"join":        true,
	"toJSON":      true,
	"fromJSON":    true,
	"hashFiles":   true,
	"success":     true,
	"failure":     true,
	"always":      true,
	"cancelled":   true,
}

// ParseExpression parses a GitHub Actions expression string into AST nodes.
func ParseExpression(raw string) ([]ExprNode, []Diagnostic) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	p := &exprParser{input: raw, line: 1, col: 1}
	node, diags := p.parseOr()
	if p.pos < len(p.input) {
		remaining := strings.TrimSpace(p.input[p.pos:])
		if remaining != "" {
			diags = append(diags, Diagnostic{
				Line:     p.line,
				Col:      p.col,
				Severity: "error",
				Message:  fmt.Sprintf("unexpected trailing input: %q", remaining),
			})
		}
	}
	if node == nil {
		return nil, diags
	}
	return []ExprNode{*node}, diags
}

const maxExpressionNestDepth = 10

// ExtractExpressions scans s for ${{ ... }} spans and parses each expression.
func ExtractExpressions(s string, baseLine int) ([]ExprNode, []Diagnostic) {
	var nodes []ExprNode
	var diags []Diagnostic
	i := 0
	for i < len(s) {
		start := strings.Index(s[i:], "${{")
		if start < 0 {
			break
		}
		start += i
		innerStart := start + len("${{")
		end := innerStart
		depth := 1
		for end < len(s) {
			if end+len("${{") <= len(s) && s[end:end+len("${{")] == "${{" {
				depth++
				if depth > maxExpressionNestDepth {
					diags = append(diags, Diagnostic{
						Line:     baseLine,
						Col:      1,
						Severity: "error",
						Message:  fmt.Sprintf("expression nesting exceeds maximum depth of %d", maxExpressionNestDepth),
					})
					return nodes, diags
				}
				end += len("${{")
				continue
			}
			if end+1 < len(s) && s[end:end+2] == "}}" {
				depth--
				if depth == 0 {
					inner := strings.TrimSpace(s[innerStart:end])
					exprNodes, exprDiags := ParseExpression(inner)
					for j := range exprDiags {
						exprDiags[j].Line += baseLine - 1
					}
					diags = append(diags, exprDiags...)
					nodes = append(nodes, exprNodes...)
					end += 2
					break
				}
				end += 2
				continue
			}
			end++
		}
		if depth != 0 {
			diags = append(diags, Diagnostic{
				Line:     baseLine,
				Col:      1,
				Severity: "error",
				Message:  "unclosed expression: missing '}}'",
			})
			break
		}
		i = end
	}
	return nodes, diags
}

func (p *exprParser) parseOr() (*ExprNode, []Diagnostic) {
	left, diags := p.parseAnd()
	for p.match("||") {
		right, d := p.parseAnd()
		diags = append(diags, d...)
		left = &ExprNode{Kind: "binary", Op: "||", Left: left, Right: right}
	}
	return left, diags
}

func (p *exprParser) parseAnd() (*ExprNode, []Diagnostic) {
	left, diags := p.parseEquality()
	for p.match("&&") {
		right, d := p.parseEquality()
		diags = append(diags, d...)
		left = &ExprNode{Kind: "binary", Op: "&&", Left: left, Right: right}
	}
	return left, diags
}

func (p *exprParser) parseEquality() (*ExprNode, []Diagnostic) {
	left, diags := p.parseComparison()
	for {
		var op string
		switch {
		case p.match("=="):
			op = "=="
		case p.match("!="):
			op = "!="
		default:
			return left, diags
		}
		right, d := p.parseComparison()
		diags = append(diags, d...)
		left = &ExprNode{Kind: "binary", Op: op, Left: left, Right: right}
	}
}

func (p *exprParser) parseComparison() (*ExprNode, []Diagnostic) {
	left, diags := p.parseUnary()
	for {
		var op string
		switch {
		case p.match("<="):
			op = "<="
		case p.match(">="):
			op = ">="
		case p.match("<"):
			op = "<"
		case p.match(">"):
			op = ">"
		default:
			return left, diags
		}
		right, d := p.parseUnary()
		diags = append(diags, d...)
		left = &ExprNode{Kind: "binary", Op: op, Left: left, Right: right}
	}
}

func (p *exprParser) parseUnary() (*ExprNode, []Diagnostic) {
	if p.match("!") {
		operand, diags := p.parseUnary()
		return &ExprNode{Kind: "unary", Op: "!", Operand: operand}, diags
	}
	return p.parsePrimary()
}

func (p *exprParser) parsePrimary() (*ExprNode, []Diagnostic) {
	p.skipWhitespace()
	if p.pos >= len(p.input) {
		return nil, []Diagnostic{{Line: p.line, Col: p.col, Severity: "error", Message: "unexpected end of expression"}}
	}

	if p.match("(") {
		node, diags := p.parseOr()
		if !p.match(")") {
			diags = append(diags, Diagnostic{Line: p.line, Col: p.col, Severity: "error", Message: "expected ')'"})
		}
		return node, diags
	}

	if p.peek() == '\'' {
		val, diags := p.parseString('\'')
		return &ExprNode{Kind: "literal", Value: val}, diags
	}
	if p.peek() == '"' {
		return nil, []Diagnostic{{
			Line:     p.line,
			Col:      p.col,
			Severity: "error",
			Message:  "double-quoted string literals are not supported in GitHub Actions expressions",
		}}
	}

	if p.matchKeyword("true") {
		return &ExprNode{Kind: "literal", Value: "true"}, nil
	}
	if p.matchKeyword("false") {
		return &ExprNode{Kind: "literal", Value: "false"}, nil
	}

	if unicode.IsDigit(rune(p.peek())) {
		val := p.parseNumber()
		return &ExprNode{Kind: "literal", Value: val}, nil
	}

	if unicode.IsLetter(rune(p.peek())) || p.peek() == '_' {
		ident := p.parseIdent()
		p.skipWhitespace()
		if p.peek() == '(' {
			return p.parseFunctionCall(ident)
		}
		// Context path: ident(.ident)*
		path := ident
		for p.match(".") {
			part := p.parseIdent()
			path += "." + part
		}
		return &ExprNode{Kind: "context", Value: path}, nil
	}

	return nil, []Diagnostic{{Line: p.line, Col: p.col, Severity: "error", Message: fmt.Sprintf("unexpected character %q", string(p.peek()))}}
}

func (p *exprParser) parseFunctionCall(name string) (*ExprNode, []Diagnostic) {
	var diags []Diagnostic
	if !p.match("(") {
		return &ExprNode{Kind: "context", Value: name}, nil
	}
	var args []ExprNode
	p.skipWhitespace()
	if p.peek() != ')' {
		for {
			arg, d := p.parseOr()
			diags = append(diags, d...)
			if arg != nil {
				args = append(args, *arg)
			}
			if !p.match(",") {
				break
			}
			p.skipWhitespace()
		}
	}
	if !p.match(")") {
		diags = append(diags, Diagnostic{Line: p.line, Col: p.col, Severity: "error", Message: "expected ')' after function arguments"})
	}
	if !knownFunctions[name] {
		diags = append(diags, Diagnostic{Line: p.line, Col: p.col, Severity: "warning", Message: fmt.Sprintf("unknown function %q", name)})
	}
	return &ExprNode{Kind: "call", Fn: name, Args: args}, diags
}

func (p *exprParser) parseString(quote byte) (string, []Diagnostic) {
	p.advance()
	var sb strings.Builder
	for p.pos < len(p.input) {
		ch := p.peek()
		if ch == quote {
			p.advance()
			return sb.String(), nil
		}
		if ch == '\\' && p.pos+1 < len(p.input) {
			p.advance()
			sb.WriteByte(p.peek())
			p.advance()
			continue
		}
		sb.WriteByte(ch)
		p.advance()
	}
	return sb.String(), []Diagnostic{{Line: p.line, Col: p.col, Severity: "error", Message: "unterminated string literal"}}
}

func (p *exprParser) parseNumber() string {
	start := p.pos
	if p.peek() == '.' {
		return ""
	}
	for p.pos < len(p.input) && (unicode.IsDigit(rune(p.peek())) || p.peek() == '.') {
		p.advance()
	}
	return p.input[start:p.pos]
}

func (p *exprParser) parseIdent() string {
	start := p.pos
	for p.pos < len(p.input) {
		ch := p.peek()
		if unicode.IsLetter(rune(ch)) || unicode.IsDigit(rune(ch)) || ch == '_' || ch == '-' {
			p.advance()
			continue
		}
		break
	}
	return p.input[start:p.pos]
}

func (p *exprParser) peek() byte {
	if p.pos >= len(p.input) {
		return 0
	}
	return p.input[p.pos]
}

func (p *exprParser) advance() {
	if p.pos < len(p.input) {
		if p.input[p.pos] == '\n' {
			p.line++
			p.col = 1
		} else {
			p.col++
		}
		p.pos++
	}
}

func (p *exprParser) skipWhitespace() {
	for p.pos < len(p.input) && unicode.IsSpace(rune(p.peek())) {
		p.advance()
	}
}

func (p *exprParser) match(s string) bool {
	p.skipWhitespace()
	if p.pos+len(s) > len(p.input) || p.input[p.pos:p.pos+len(s)] != s {
		return false
	}
	for i := 0; i < len(s); i++ {
		p.advance()
	}
	return true
}

func (p *exprParser) matchKeyword(kw string) bool {
	p.skipWhitespace()
	if p.pos+len(kw) > len(p.input) || p.input[p.pos:p.pos+len(kw)] != kw {
		return false
	}
	if p.pos+len(kw) < len(p.input) {
		next := p.input[p.pos+len(kw)]
		if unicode.IsLetter(rune(next)) || unicode.IsDigit(rune(next)) || next == '_' {
			return false
		}
	}
	for i := 0; i < len(kw); i++ {
		p.advance()
	}
	return true
}
