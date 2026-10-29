// Package parse checks JSONPath lexical and grammatical rules and builds an AST.
// It preserves source positions but leaves function resolution and type checking
// to the compiler.
package parse

import (
	"fmt"
	"strconv"

	"github.com/saltfishpr/jsonpath/internal/ast"
)

// Error identifies a lexical or grammatical failure in the original query.
type Error struct {
	// Pos locates the token or character at which parsing failed.
	Pos ast.Pos
	// Msg describes the failure without a location prefix.
	Msg string
}

// Error returns the diagnostic with a one-based line and rune column.
func (e *Error) Error() string {
	return fmt.Sprintf("parse error at line %d col %d: %s", e.Pos.Line, e.Pos.Col, e.Msg)
}

// Parse accepts a complete query beginning with $, without leading or trailing
// whitespace. Invalid syntax returns an *[Error] and a nil tree. A successful
// result still requires semantic validation of function calls and types.
func Parse(src string) (*ast.Query, error) {
	toks, err := lex(src)
	if err != nil {
		return nil, err
	}
	if toks[0].Kind == tkWhitespace {
		return nil, &Error{Pos: toks[1].Pos, Msg: "whitespace before root identifier is not allowed"}
	}
	p := &parser{toks: toks}
	q, err := p.parseRootQuery()
	if err != nil {
		return nil, err
	}
	if p.peek().Kind != tkEOF {
		return nil, p.errorf(p.peek().Pos, "unexpected %s after query", describe(p.peek()))
	}
	if p.toks[p.pos-1].Kind == tkWhitespace {
		return nil, p.errorf(p.peek().Pos, "trailing whitespace is not allowed")
	}
	return q, nil
}

type parser struct {
	toks []token
	pos  int
}

func (p *parser) peek() token { return p.toks[p.pos] }
func (p *parser) advance() token {
	t := p.peek()
	p.pos++
	// One token holds an entire whitespace run. Keep it in toks so callers
	// can reject whitespace at grammar boundaries that require adjacency.
	if p.pos < len(p.toks) && p.peek().Kind == tkWhitespace {
		p.pos++
	}
	return t
}

func (p *parser) expect(k tokenKind) (token, error) {
	t := p.peek()
	if t.Kind != k {
		return t, p.errorf(t.Pos, "expected %s, got %s", k, describe(t))
	}
	p.advance()
	return t, nil
}

func (p *parser) errorf(pos ast.Pos, format string, args ...interface{}) error {
	return &Error{Pos: pos, Msg: fmt.Sprintf(format, args...)}
}

func describe(t token) string {
	switch t.Kind {
	case tkEOF:
		return "end of input"
	case tkName, tkInt, tkFloat, tkString:
		return fmt.Sprintf("%s %q", t.Kind, t.Value)
	}
	return t.Kind.String()
}

func (p *parser) parseRootQuery() (*ast.Query, error) {
	t, err := p.expect(tkRoot)
	if err != nil {
		return nil, err
	}
	segs, err := p.parseSegments()
	if err != nil {
		return nil, err
	}
	return &ast.Query{IsRoot: true, Segments: segs, Pos: t.Pos}, nil
}

func (p *parser) parseSegments() ([]ast.Segment, error) {
	var segs []ast.Segment
	for {
		switch p.peek().Kind {
		case tkDot:
			seg, err := p.parseDotSegment()
			if err != nil {
				return nil, err
			}
			segs = append(segs, seg)
		case tkDotDot:
			seg, err := p.parseDescendantSegment()
			if err != nil {
				return nil, err
			}
			segs = append(segs, seg)
		case tkLBracket:
			seg, err := p.parseBracketSegment(ast.SegChild)
			if err != nil {
				return nil, err
			}
			segs = append(segs, seg)
		default:
			return segs, nil
		}
	}
}

func (p *parser) parseDotSegment() (ast.Segment, error) {
	dot := p.advance()
	t := p.peek()
	if p.toks[p.pos-1].Kind == tkWhitespace {
		return ast.Segment{}, p.errorf(t.Pos, "whitespace after dot selector is not allowed")
	}
	switch t.Kind {
	case tkStar:
		p.advance()
		return ast.Segment{
			Kind:      ast.SegChild,
			Selectors: []ast.Selector{&ast.WildcardSelector{Pos: t.Pos}},
			Pos:       dot.Pos,
		}, nil
	case tkName, tkTrue, tkFalse, tkNull:
		// Literal keywords are member names in dot notation.
		p.advance()
		return ast.Segment{
			Kind:      ast.SegChild,
			Selectors: []ast.Selector{&ast.NameSelector{Name: t.Value, Raw: t.Raw, Pos: t.Pos}},
			Pos:       dot.Pos,
		}, nil
	}
	return ast.Segment{}, p.errorf(t.Pos, "expected member name or '*' after '.'")
}

func (p *parser) parseDescendantSegment() (ast.Segment, error) {
	dot := p.advance()
	t := p.peek()
	if p.toks[p.pos-1].Kind == tkWhitespace {
		return ast.Segment{}, p.errorf(t.Pos, "whitespace after dot selector is not allowed")
	}
	switch t.Kind {
	case tkStar:
		p.advance()
		return ast.Segment{
			Kind:      ast.SegDescendant,
			Selectors: []ast.Selector{&ast.WildcardSelector{Pos: t.Pos}},
			Pos:       dot.Pos,
		}, nil
	case tkName, tkTrue, tkFalse, tkNull:
		p.advance()
		return ast.Segment{
			Kind:      ast.SegDescendant,
			Selectors: []ast.Selector{&ast.NameSelector{Name: t.Value, Raw: t.Raw, Pos: t.Pos}},
			Pos:       dot.Pos,
		}, nil
	case tkLBracket:
		seg, err := p.parseBracketSegment(ast.SegDescendant)
		if err != nil {
			return ast.Segment{}, err
		}
		seg.Pos = dot.Pos
		return seg, nil
	}
	return ast.Segment{}, p.errorf(t.Pos, "expected member name, '*' or '[' after '..'")
}

func (p *parser) parseBracketSegment(kind ast.SegmentKind) (ast.Segment, error) {
	lb, err := p.expect(tkLBracket)
	if err != nil {
		return ast.Segment{}, err
	}
	var sels []ast.Selector
	for {
		sel, err := p.parseSelector()
		if err != nil {
			return ast.Segment{}, err
		}
		sels = append(sels, sel)
		if p.peek().Kind == tkComma {
			p.advance()
			continue
		}
		break
	}
	if _, err := p.expect(tkRBracket); err != nil {
		return ast.Segment{}, err
	}
	return ast.Segment{Kind: kind, Selectors: sels, Pos: lb.Pos}, nil
}

func (p *parser) parseSelector() (ast.Selector, error) {
	t := p.peek()
	switch t.Kind {
	case tkStar:
		p.advance()
		return &ast.WildcardSelector{Pos: t.Pos}, nil
	case tkString:
		p.advance()
		return &ast.NameSelector{Name: t.Value, Raw: t.Raw, Pos: t.Pos}, nil
	case tkQuestion:
		return p.parseFilterSelector()
	case tkInt:
		return p.parseIndexOrSlice()
	case tkColon:
		return p.parseSlice(nil, t.Pos)
	}
	return nil, p.errorf(t.Pos, "expected selector, got %s", describe(t))
}

func (p *parser) parseFilterSelector() (ast.Selector, error) {
	q := p.advance()
	expr, err := p.parseLogicalExpr()
	if err != nil {
		return nil, err
	}
	return &ast.FilterSelector{Expr: expr, Pos: q.Pos}, nil
}

func (p *parser) parseIndexOrSlice() (ast.Selector, error) {
	tok := p.advance()
	n, err := parseIndexInteger(tok)
	if err != nil {
		return nil, err
	}
	if p.peek().Kind == tkColon {
		start := n
		return p.parseSlice(&start, tok.Pos)
	}
	return &ast.IndexSelector{Index: n, Pos: tok.Pos}, nil
}

// The caller has consumed start, if present, and left the first colon pending.
// Keep omitted bounds as nil because their defaults depend on step direction.
func (p *parser) parseSlice(start *int64, pos ast.Pos) (ast.Selector, error) {
	if _, err := p.expect(tkColon); err != nil {
		return nil, err
	}
	var end *int64
	var step *int64
	if p.peek().Kind == tkInt {
		t := p.advance()
		n, err := parseIndexInteger(t)
		if err != nil {
			return nil, err
		}
		end = &n
	}
	if p.peek().Kind == tkColon {
		p.advance()
		if p.peek().Kind == tkInt {
			t := p.advance()
			n, err := parseIndexInteger(t)
			if err != nil {
				return nil, err
			}
			step = &n
		}
	}
	return &ast.SliceSelector{Start: start, End: end, Step: step, Pos: pos}, nil
}

func parseIndexInteger(t token) (int64, error) {
	if t.Value == "-0" {
		return 0, (&Error{Pos: t.Pos, Msg: "-0 is not a valid integer selector"})
	}
	n, err := strconv.ParseInt(t.Value, 10, 64)
	if err != nil {
		return 0, &Error{Pos: t.Pos, Msg: fmt.Sprintf("invalid integer %q", t.Value)}
	}
	const maxExact = int64(1<<53 - 1)
	if n < -maxExact || n > maxExact {
		return 0, &Error{Pos: t.Pos, Msg: "integer is outside the I-JSON exact range"}
	}
	return n, nil
}

func (p *parser) parseLogicalExpr() (ast.LogicalExpr, error) {
	e, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	return toLogical(e)
}

func (p *parser) parseOr() (ast.Expr, error) {
	first, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	if p.peek().Kind != tkOr {
		return first, nil
	}
	lfirst, err := toLogical(first)
	if err != nil {
		return nil, err
	}
	ops := []ast.LogicalExpr{lfirst}
	pos := lfirst.Position()
	for p.peek().Kind == tkOr {
		p.advance()
		rhs, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		lrhs, err := toLogical(rhs)
		if err != nil {
			return nil, err
		}
		ops = append(ops, lrhs)
	}
	return &ast.OrExpr{Operands: ops, Pos: pos}, nil
}

func (p *parser) parseAnd() (ast.Expr, error) {
	first, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	if p.peek().Kind != tkAnd {
		return first, nil
	}
	lfirst, err := toLogical(first)
	if err != nil {
		return nil, err
	}
	ops := []ast.LogicalExpr{lfirst}
	pos := lfirst.Position()
	for p.peek().Kind == tkAnd {
		p.advance()
		rhs, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		lrhs, err := toLogical(rhs)
		if err != nil {
			return nil, err
		}
		ops = append(ops, lrhs)
	}
	return &ast.AndExpr{Operands: ops, Pos: pos}, nil
}

func (p *parser) parseUnary() (ast.Expr, error) {
	if p.peek().Kind == tkNot {
		t := p.advance()
		if p.peek().Kind == tkNot {
			return nil, p.errorf(p.peek().Pos, "logical NOT cannot be repeated without parentheses")
		}
		inner, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		if _, isComparison := inner.(*ast.ComparisonExpr); isComparison {
			return nil, p.errorf(t.Pos, "negated comparison must be parenthesized")
		}
		linner, err := toLogical(inner)
		if err != nil {
			return nil, err
		}
		return &ast.NotExpr{Operand: linner, Pos: t.Pos}, nil
	}
	return p.parseComparisonOrAtom()
}

func (p *parser) parseComparisonOrAtom() (ast.Expr, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	if op, ok := cmpOpOf(p.peek().Kind); ok {
		pos := p.advance().Pos
		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		lc, err := toComparable(left)
		if err != nil {
			return nil, err
		}
		rc, err := toComparable(right)
		if err != nil {
			return nil, err
		}
		return &ast.ComparisonExpr{Op: op, Left: lc, Right: rc, Pos: pos}, nil
	}
	return left, nil
}

func toLogical(e ast.Expr) (ast.LogicalExpr, error) {
	switch v := e.(type) {
	case ast.LogicalExpr:
		return v, nil
	case *ast.FilterQuery:
		return &ast.TestExpr{Target: v, Pos: v.Pos}, nil
	case *ast.FunctionCall:
		return &ast.TestExpr{Target: v, Pos: v.Pos}, nil
	}
	return nil, &Error{Pos: e.Position(), Msg: "literal cannot be used as a logical expression"}
}

func cmpOpOf(k tokenKind) (ast.CmpOp, bool) {
	switch k {
	case tkEq:
		return ast.CmpEq, true
	case tkNe:
		return ast.CmpNe, true
	case tkLt:
		return ast.CmpLt, true
	case tkLe:
		return ast.CmpLe, true
	case tkGt:
		return ast.CmpGt, true
	case tkGe:
		return ast.CmpGe, true
	}
	return 0, false
}

func (p *parser) parsePrimary() (ast.Expr, error) {
	t := p.peek()
	switch t.Kind {
	case tkLParen:
		lp := p.advance()
		inner, err := p.parseLogicalExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(tkRParen); err != nil {
			return nil, err
		}
		return &ast.ParenExpr{Inner: inner, Pos: lp.Pos}, nil
	case tkRoot:
		return p.parseFilterQueryOrSingular(true)
	case tkCurrent:
		return p.parseFilterQueryOrSingular(false)
	case tkString:
		p.advance()
		return &ast.Literal{Value: ast.LiteralValue{Kind: ast.LitString, Str: t.Value, Raw: t.Raw}, Pos: t.Pos}, nil
	case tkInt, tkFloat:
		p.advance()
		kind := ast.LitInt
		if t.Kind == tkFloat {
			kind = ast.LitFloat
		}
		return &ast.Literal{Value: ast.LiteralValue{Kind: kind, Raw: t.Raw}, Pos: t.Pos}, nil
	case tkTrue:
		p.advance()
		return &ast.Literal{Value: ast.LiteralValue{Kind: ast.LitBool, Bool: true, Raw: t.Raw}, Pos: t.Pos}, nil
	case tkFalse:
		p.advance()
		return &ast.Literal{Value: ast.LiteralValue{Kind: ast.LitBool, Bool: false, Raw: t.Raw}, Pos: t.Pos}, nil
	case tkNull:
		p.advance()
		return &ast.Literal{Value: ast.LiteralValue{Kind: ast.LitNull, Raw: t.Raw}, Pos: t.Pos}, nil
	case tkName:
		return p.parseFunctionCall()
	}
	return nil, p.errorf(t.Pos, "unexpected %s in expression", describe(t))
}

// Preserve queries as node lists until the enclosing context establishes
// whether a singular value or an existence test is required.
func (p *parser) parseFilterQueryOrSingular(isRoot bool) (ast.Expr, error) {
	rootTok := p.advance()
	segs, err := p.parseSegments()
	if err != nil {
		return nil, err
	}
	q := &ast.Query{IsRoot: isRoot, Segments: segs, Pos: rootTok.Pos}
	return &ast.FilterQuery{Query: q, Pos: rootTok.Pos}, nil
}

func toComparable(e ast.Expr) (ast.Comparable, error) {
	switch v := e.(type) {
	case *ast.Literal:
		return v, nil
	case *ast.FunctionCall:
		return v, nil
	case *ast.FilterQuery:
		if segs, ok := toSingularSegments(v.Query.Segments); ok {
			return &ast.SingularQuery{IsRoot: v.Query.IsRoot, Segments: segs, Pos: v.Pos}, nil
		}
		return nil, &Error{Pos: v.Pos, Msg: "comparison operand must be a singular query"}
	}
	return nil, &Error{Pos: e.Position(), Msg: "comparison operand must be literal, singular-query or function"}
}

func toSingularSegments(segs []ast.Segment) ([]ast.SingularSegment, bool) {
	out := make([]ast.SingularSegment, 0, len(segs))
	for _, seg := range segs {
		if seg.Kind != ast.SegChild || len(seg.Selectors) != 1 {
			return nil, false
		}
		switch sel := seg.Selectors[0].(type) {
		case *ast.NameSelector:
			name := sel.Name
			out = append(out, ast.SingularSegment{Name: &name, Pos: sel.Pos})
		case *ast.IndexSelector:
			idx := sel.Index
			out = append(out, ast.SingularSegment{Index: &idx, Pos: sel.Pos})
		default:
			return nil, false
		}
	}
	return out, true
}

func (p *parser) parseFunctionCall() (ast.Expr, error) {
	name := p.advance()
	if p.peek().Kind != tkLParen {
		return nil, p.errorf(name.Pos, "unexpected identifier %q", name.Value)
	}
	if p.toks[p.pos-1].Kind == tkWhitespace {
		return nil, p.errorf(p.peek().Pos, "whitespace between function name and '(' is not allowed")
	}
	if err := validateFunctionName(name); err != nil {
		return nil, err
	}
	if _, err := p.expect(tkLParen); err != nil {
		return nil, err
	}
	args, err := p.parseFunctionArguments()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(tkRParen); err != nil {
		return nil, err
	}
	return &ast.FunctionCall{Name: name.Value, Args: args, Pos: name.Pos}, nil
}

func validateFunctionName(name token) error {
	if name.Value[0] < 'a' || name.Value[0] > 'z' {
		return &Error{Pos: name.Pos, Msg: "function names must start with a lowercase ASCII letter"}
	}
	for i := 1; i < len(name.Value); i++ {
		c := name.Value[i]
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return &Error{Pos: name.Pos, Msg: fmt.Sprintf("invalid function name %q", name.Value)}
		}
	}
	return nil
}

func (p *parser) parseFunctionArguments() ([]ast.Expr, error) {
	if p.peek().Kind == tkRParen {
		return nil, nil
	}
	var args []ast.Expr
	for {
		arg, err := p.parseFunctionArg()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		if p.peek().Kind != tkComma {
			return args, nil
		}
		p.advance()
	}
}

func (p *parser) parseFunctionArg() (ast.Expr, error) {
	// Unlike a filter condition, an argument may remain a literal or query;
	// only explicit logical operators require conversion to [ast.LogicalExpr].
	return p.parseOr()
}
