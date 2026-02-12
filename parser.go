package jsonpath

import (
	"fmt"
)

// Parse parses a JSONPath expression string and returns an AST
func Parse(path string) (*Query, error) {
	lexer := NewLexer(path)
	p := &Parser{
		lexer: lexer,
	}
	p.advance()
	p.advance()
	return p.parseQuery()
}

// Parser parses JSONPath expressions into an AST
type Parser struct {
	lexer *Lexer
	curr  Token
	peek  Token
}

func (p *Parser) advance() {
	p.curr = p.peek
	p.peek = p.lexer.NextToken()
}

func (p *Parser) expectToken(tokenType TokenType) error {
	if p.curr.Type != tokenType {
		return fmt.Errorf("except %s, got %s(%q)", tokenType, p.curr.Type, p.curr.Value)
	}
	return nil
}

// parseQuery parses a complete JSONPath query
// jsonpath-query = root-identifier segments
func (p *Parser) parseQuery() (*Query, error) {
	query := &Query{}

	// Must start with root identifier $
	if err := p.expectToken(TokenRoot); err != nil {
		return nil, err
	}
	p.advance()

	for p.curr.Type != TokenEOF {
		segment, err := p.parseSegment()
		if err != nil {
			return nil, err
		}
		query.Segments = append(query.Segments, segment)
	}

	return query, nil
}

// parseSegment parses a segment (child or descendant)
// segment = child-segment / descendant-segment
func (p *Parser) parseSegment() (*Segment, error) {
	switch p.curr.Type {
	case TokenDotDot:
		p.advance()
		return p.parseDescendantSegment()

	case TokenDot:
		p.advance()
		return p.parseDotSegment()

	case TokenLBracket:
		return p.parseBracketSegment(ChildSegment)

	default:
		return nil, fmt.Errorf("unexpected token %s(%q), expected '.' or '..'", p.curr.Type, p.curr.Value)
	}
}

// parseDescendantSegment parses descendant segments (..name or ..[...])
// descendant-segment = ".." name-segment / "..[" selectors "]"
func (p *Parser) parseDescendantSegment() (*Segment, error) {
	segment := &Segment{Type: DescendantSegment}

	switch p.curr.Type {
	case TokenLBracket:
		return p.parseBracketSegment(DescendantSegment)

	case TokenWildcard:
		segment.Selectors = []*Selector{{Type: WildcardSelector}}
		p.advance()
		return segment, nil

	case TokenIdent, TokenNull, TokenTrue, TokenFalse:
		name := p.curr.Value
		segment.Selectors = []*Selector{{
			Type: NameSelector,
			Name: name,
		}}
		p.advance()
		return segment, nil

	default:
		return nil, fmt.Errorf("unexpected token %s(%q) after '..' at position %d", p.curr.Type, p.curr.Value, p.curr.Pos)
	}
}

// parseDotSegment parses dot notation segments (.name or .*)
func (p *Parser) parseDotSegment() (*Segment, error) {
	segment := &Segment{Type: ChildSegment}

	switch p.curr.Type {
	case TokenWildcard:
		segment.Selectors = []*Selector{{Type: WildcardSelector}}
		p.advance()
		return segment, nil

	case TokenIdent, TokenNull, TokenTrue, TokenFalse:
		name := p.curr.Value
		segment.Selectors = []*Selector{{
			Type: NameSelector,
			Name: name,
		}}
		p.advance()
		return segment, nil

	default:
		return nil, fmt.Errorf("unexpected token %s(%q) after '.' at position %d", p.curr.Type, p.curr.Value, p.curr.Pos)
	}
}

// parseBracketSegment parses bracket-notation segments
// bracketed-selection = "[" S selector *(S "," S selector) S "]"
func (p *Parser) parseBracketSegment(segType SegmentType) (*Segment, error) {
	if err := p.expectToken(TokenLBracket); err != nil {
		return nil, err
	}
	p.advance()

	segment := &Segment{Type: segType}

	selectors, err := p.parseSelectors()
	if err != nil {
		return nil, err
	}
	segment.Selectors = selectors

	if err := p.expectToken(TokenRBracket); err != nil {
		return nil, err
	}
	p.advance()

	return segment, nil
}

func (p *Parser) parseSelectors() ([]*Selector, error) {
	var selectors []*Selector

	sel, err := p.parseSelector()
	if err != nil {
		return nil, err
	}
	selectors = append(selectors, sel)

	for p.curr.Type == TokenComma {
		p.advance()
		sel, err := p.parseSelector()
		if err != nil {
			return nil, err
		}
		selectors = append(selectors, sel)
	}

	return selectors, nil
}

// parseSelector parses a single selector
// selector = name-selector / wildcard-selector / index-selector / slice-selector / filter-selector
func (p *Parser) parseSelector() (*Selector, error) {
	switch p.curr.Type {
	case TokenString:
		sel := &Selector{
			Type: NameSelector,
			Name: p.curr.Value,
		}
		p.advance()
		return sel, nil

	case TokenWildcard:
		sel := &Selector{Type: WildcardSelector}
		p.advance()
		return sel, nil

	case TokenNumber:
		if p.peek.Type == TokenColon {
			return p.parseSliceSelector()
		}
		return p.parseIndexSelector()

	case TokenColon:
		return p.parseSliceSelector()

	case TokenQuestion:
		return p.parseFilterSelector()

	default:
		return nil, fmt.Errorf("unexpected token %s(%q) in selector at position %d", p.curr.Type, p.curr.Value, p.curr.Pos)
	}
}

func (p *Parser) parseIndexSelector() (*Selector, error) {
	if err := p.expectToken(TokenNumber); err != nil {
		return nil, err
	}

	index, err := parseInteger(p.curr.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid index %q at position %d: %w", p.curr.Value, p.curr.Pos, err)
	}

	p.advance()
	return &Selector{
		Type:  IndexSelector,
		Index: index,
	}, nil
}

// parseSliceSelector parses array slice selectors (start:end:step)
func (p *Parser) parseSliceSelector() (*Selector, error) {
	slice := &SliceParams{}

	// Parse start (optional)
	if p.curr.Type == TokenNumber {
		start, err := parseInteger(p.curr.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid slice start %q: %w", p.curr.Value, err)
		}
		slice.Start = &start
		p.advance()
	}

	// Expect colon
	if err := p.expectToken(TokenColon); err != nil {
		return nil, err
	}
	p.advance()

	// Parse end (optional)
	if p.curr.Type == TokenNumber {
		end, err := parseInteger(p.curr.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid slice end %q: %w", p.curr.Value, err)
		}
		slice.End = &end
		p.advance()
	}

	// Parse step (optional)
	if p.curr.Type == TokenColon {
		p.advance()
		if p.curr.Type != TokenNumber {
			return nil, fmt.Errorf("expected step number after ':', got %s(%q)", p.curr.Type, p.curr.Value)
		}
		step, err := parseInteger(p.curr.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid slice step %q: %w", p.curr.Value, err)
		}
		slice.Step = &step
		p.advance()
	}

	return &Selector{Type: SliceSelector, Slice: slice}, nil
}

// parseFilterSelector parses filter selectors (?<logical-expr>)
// filter-selector = "?" S logical-expr
func (p *Parser) parseFilterSelector() (*Selector, error) {
	if err := p.expectToken(TokenQuestion); err != nil {
		return nil, err
	}
	p.advance()

	expr, err := p.parseLogicalExpr()
	if err != nil {
		return nil, err
	}

	return &Selector{Type: FilterSelector, Filter: expr}, nil
}

// parseLogicalExpr parses a logical expression
// logical-expr = logical-or-expr
func (p *Parser) parseLogicalExpr() (*FilterExpr, error) {
	return p.parseLogicalOrExpr()
}

// parseLogicalOrExpr parses logical-or expressions
// logical-or-expr = logical-and-expr *(S "||" S logical-and-expr)
func (p *Parser) parseLogicalOrExpr() (*FilterExpr, error) {
	left, err := p.parseLogicalAndExpr()
	if err != nil {
		return nil, err
	}

	for p.curr.Type == TokenLOr {
		p.advance()
		right, err := p.parseLogicalAndExpr()
		if err != nil {
			return nil, err
		}
		left = &FilterExpr{
			Type:  FilterLogicalOr,
			Left:  left,
			Right: right,
		}
	}

	return left, nil
}

// parseLogicalAndExpr parses logical-and expressions
// logical-and-expr = basic-expr *(S "&&" S basic-expr)
func (p *Parser) parseLogicalAndExpr() (*FilterExpr, error) {
	left, err := p.parseBasicExpr()
	if err != nil {
		return nil, err
	}

	for p.curr.Type == TokenLAnd {
		p.advance()
		right, err := p.parseBasicExpr()
		if err != nil {
			return nil, err
		}
		left = &FilterExpr{
			Type:  FilterLogicalAnd,
			Left:  left,
			Right: right,
		}
	}

	return left, nil
}

// parseBasicExpr parses basic expressions
// basic-expr = paren-expr / comparison-expr / test-expr
func (p *Parser) parseBasicExpr() (*FilterExpr, error) {
	if p.curr.Type == TokenLNot {
		// Distinguish paren-expr from test-expr
		if p.peek.Type == TokenLParen {
			return p.parseParenExpr()
		}
		p.advance()
		test, err := p.parseTestExpr()
		if err != nil {
			return nil, err
		}
		return &FilterExpr{Type: FilterLogicalNot, Operand: &FilterExpr{Type: FilterTest, Test: test}}, nil
	}

	if p.curr.Type == TokenLParen {
		return p.parseParenExpr()
	}

	return p.parseBasicExprWithFallback()
}

// parseBasicExprWithFallback tries comparison-expr, falls back to test-expr
func (p *Parser) parseBasicExprWithFallback() (*FilterExpr, error) {
	savedCurr := p.curr
	savedPeek := p.peek
	savedLexerPos := p.lexer.pos

	comp, err := p.parseComparisonExpr()
	if err == nil {
		return &FilterExpr{Type: FilterComparison, Comp: comp}, nil
	}

	p.curr = savedCurr
	p.peek = savedPeek
	p.lexer.pos = savedLexerPos

	test, err := p.parseTestExpr()
	if err != nil {
		return nil, err
	}
	return &FilterExpr{Type: FilterTest, Test: test}, nil
}

// parseParenExpr parses parenthesized expressions
// paren-expr = [logical-not-op S] "(" S logical-expr S ")"
func (p *Parser) parseParenExpr() (*FilterExpr, error) {
	hasNot := p.curr.Type == TokenLNot
	if hasNot {
		p.advance()
	}

	if err := p.expectToken(TokenLParen); err != nil {
		return nil, err
	}
	p.advance()

	expr, err := p.parseLogicalExpr()
	if err != nil {
		return nil, err
	}

	if err := p.expectToken(TokenRParen); err != nil {
		return nil, err
	}
	p.advance()

	expr = &FilterExpr{
		Type:    FilterParen,
		Operand: expr,
	}

	if hasNot {
		expr = &FilterExpr{
			Type:    FilterLogicalNot,
			Operand: expr,
		}
	}

	return expr, nil
}

// parseComparisonExpr parses comparison expressions
// comparison-expr = comparable S comparison-op S comparable
func (p *Parser) parseComparisonExpr() (*Comparison, error) {
	left, err := p.parseComparable()
	if err != nil {
		return nil, err
	}

	op, err := p.parseComparisonOp()
	if err != nil {
		return nil, err
	}

	right, err := p.parseComparable()
	if err != nil {
		return nil, err
	}

	return &Comparison{Left: left, Op: op, Right: right}, nil
}

func (p *Parser) parseComparisonOp() (CompOp, error) {
	switch p.curr.Type {
	case TokenEq:
		p.advance()
		return CompEq, nil
	case TokenNe:
		p.advance()
		return CompNe, nil
	case TokenLt:
		p.advance()
		return CompLt, nil
	case TokenLe:
		p.advance()
		return CompLe, nil
	case TokenGt:
		p.advance()
		return CompGt, nil
	case TokenGe:
		p.advance()
		return CompGe, nil
	default:
		return 0, fmt.Errorf("expected comparison operator, got %s(%q)", p.curr.Type, p.curr.Value)
	}
}

// parseComparable parses comparable values
// comparable = literal / singular-query / function-expr
func (p *Parser) parseComparable() (*Comparable, error) {
	switch p.curr.Type {
	case TokenString, TokenNumber, TokenTrue, TokenFalse, TokenNull:
		lit, err := p.parseLiteral()
		if err != nil {
			return nil, err
		}
		return &Comparable{Type: ComparableLiteral, Literal: lit}, nil

	case TokenRoot, TokenCurrent:
		query, err := p.parseSingularQuery()
		if err != nil {
			return nil, err
		}
		return &Comparable{Type: ComparableSingularQuery, SingularQuery: query}, nil

	case TokenIdent:
		if p.peek.Type == TokenLParen {
			fn, err := p.parseFunctionExpr()
			if err != nil {
				return nil, err
			}
			return &Comparable{Type: ComparableFuncExpr, FuncExpr: fn}, nil
		}
		lit := &LiteralValue{Type: LiteralString, Value: p.curr.Value}
		p.advance()
		return &Comparable{Type: ComparableLiteral, Literal: lit}, nil

	default:
		return nil, fmt.Errorf("unexpected token %s(%q) in comparable at position %d", p.curr.Type, p.curr.Value, p.curr.Pos)
	}
}

func (p *Parser) parseLiteral() (*LiteralValue, error) {
	switch p.curr.Type {
	case TokenString:
		lit := &LiteralValue{Type: LiteralString, Value: p.curr.Value}
		p.advance()
		return lit, nil
	case TokenNumber:
		lit := &LiteralValue{Type: LiteralNumber, Value: p.curr.Value}
		p.advance()
		return lit, nil
	case TokenTrue:
		lit := &LiteralValue{Type: LiteralTrue, Value: p.curr.Value}
		p.advance()
		return lit, nil
	case TokenFalse:
		lit := &LiteralValue{Type: LiteralFalse, Value: p.curr.Value}
		p.advance()
		return lit, nil
	case TokenNull:
		lit := &LiteralValue{Type: LiteralNull, Value: p.curr.Value}
		p.advance()
		return lit, nil
	default:
		return nil, fmt.Errorf("expected literal, got %s(%q)", p.curr.Type, p.curr.Value)
	}
}

// parseSingularQuery parses singular queries
// singular-query = rel-singular-query / abs-singular-query
func (p *Parser) parseSingularQuery() (*SingularQuery, error) {
	query := &SingularQuery{}

	switch p.curr.Type {
	case TokenRoot:
		query.Relative = false
		p.advance()
	case TokenCurrent:
		query.Relative = true
		p.advance()
	default:
		return nil, fmt.Errorf("expected '$' or '@', got %s(%q)", p.curr.Type, p.curr.Value)
	}

	for p.curr.Type == TokenDot || p.curr.Type == TokenLBracket {
		seg, err := p.parseSingularSegment()
		if err != nil {
			return nil, err
		}
		query.Segments = append(query.Segments, seg)
	}

	return query, nil
}

func (p *Parser) parseSingularSegment() (*SingularSegment, error) {
	seg := &SingularSegment{}

	switch p.curr.Type {
	case TokenDot:
		p.advance()
		switch p.curr.Type {
		case TokenIdent, TokenNull, TokenTrue, TokenFalse:
			seg.Type = SingularNameSegment
			seg.Name = p.curr.Value
			p.advance()
			return seg, nil
		default:
			return nil, fmt.Errorf("expected identifier after '.', got %s(%q)", p.curr.Type, p.curr.Value)
		}

	case TokenLBracket:
		p.advance()
		switch p.curr.Type {
		case TokenString:
			seg.Type = SingularNameSegment
			seg.Name = p.curr.Value
			p.advance()

			if err := p.expectToken(TokenRBracket); err != nil {
				return nil, err
			}
			p.advance()

			return seg, nil

		case TokenNumber:
			index, err := parseInteger(p.curr.Value)
			if err != nil {
				return nil, fmt.Errorf("invalid index %q: %w", p.curr.Value, err)
			}
			seg.Type = SingularIndexSegment
			seg.Index = index
			p.advance()

			if err := p.expectToken(TokenRBracket); err != nil {
				return nil, err
			}
			p.advance()

			return seg, nil

		default:
			return nil, fmt.Errorf("expected string or number in singular query segment, got %s(%q)", p.curr.Type, p.curr.Value)
		}

	default:
		return nil, fmt.Errorf("expected '.' or '[', got %s(%q)", p.curr.Type, p.curr.Value)
	}
}

// parseTestExpr parses test expressions (existence tests)
// test-expr = [logical-not-op S] (filter-query / function-expr)
func (p *Parser) parseTestExpr() (*TestExpr, error) {
	test := &TestExpr{}

	switch p.curr.Type {
	case TokenRoot, TokenCurrent:
		query, err := p.parseFilterQuery()
		if err != nil {
			return nil, err
		}
		test.FilterQuery = query
		return test, nil

	case TokenIdent:
		fn, err := p.parseFunctionExpr()
		if err != nil {
			return nil, err
		}
		test.FuncExpr = fn
		return test, nil

	default:
		return nil, fmt.Errorf("expected filter query or function expression, got %s(%q)", p.curr.Type, p.curr.Value)
	}
}

// parseFilterQuery parses filter queries
// filter-query = rel-query / jsonpath-query
func (p *Parser) parseFilterQuery() (*FilterQuery, error) {
	query := &FilterQuery{}

	switch p.curr.Type {
	case TokenRoot:
		query.Relative = false
		p.advance()
	case TokenCurrent:
		query.Relative = true
		p.advance()
	default:
		// No explicit identifier, treat as current node reference
		query.Relative = true
	}

	for p.curr.Type == TokenDot || p.curr.Type == TokenDotDot || p.curr.Type == TokenLBracket {
		segment, err := p.parseSegment()
		if err != nil {
			return nil, err
		}
		query.Segments = append(query.Segments, segment)
	}

	return query, nil
}

// parseFunctionExpr parses function call expressions
// function-expr = function-name "(" S [function-argument *(S "," S function-argument)] S ")"
func (p *Parser) parseFunctionExpr() (*FuncCall, error) {
	if err := p.expectToken(TokenIdent); err != nil {
		return nil, err
	}

	name := p.curr.Value
	if !p.isValidFunctionName(name) {
		return nil, fmt.Errorf("invalid function name %q", name)
	}
	p.advance()

	if err := p.expectToken(TokenLParen); err != nil {
		return nil, err
	}
	p.advance()

	fn := &FuncCall{Name: name, Args: []*FuncArg{}}

	if p.curr.Type != TokenRParen {
		arg, err := p.parseFuncArg()
		if err != nil {
			return nil, err
		}
		fn.Args = append(fn.Args, arg)

		for p.curr.Type == TokenComma {
			p.advance()
			arg, err := p.parseFuncArg()
			if err != nil {
				return nil, err
			}
			fn.Args = append(fn.Args, arg)
		}
	}

	if err := p.expectToken(TokenRParen); err != nil {
		return nil, err
	}
	p.advance()

	return fn, nil
}

func (p *Parser) isValidFunctionName(name string) bool {
	for i, ch := range name {
		if i == 0 && !isFunctionNameFirst(ch) {
			return false
		}
		if !isFunctionNameChar(ch) {
			return false
		}
	}
	return true
}

func (p *Parser) parseFuncArg() (*FuncArg, error) {
	switch p.curr.Type {
	case TokenString, TokenNumber, TokenTrue, TokenFalse, TokenNull:
		lit, err := p.parseLiteral()
		if err != nil {
			return nil, err
		}
		return &FuncArg{Type: FuncArgLiteral, Literal: lit}, nil

	case TokenRoot, TokenCurrent:
		return p.parseFuncArgRootOrCurrent()

	case TokenIdent:
		fn, err := p.parseFunctionExpr()
		if err != nil {
			return nil, err
		}
		return &FuncArg{Type: FuncArgFuncExpr, FuncExpr: fn}, nil

	default:
		return nil, fmt.Errorf("unexpected token %s(%q) in function argument", p.curr.Type, p.curr.Value)
	}
}

func (p *Parser) parseFuncArgRootOrCurrent() (*FuncArg, error) {
	savedCurr := p.curr
	savedPeek := p.peek
	savedLexerPos := p.lexer.pos

	query, err := p.parseFilterQuery()
	if err != nil {
		return nil, err
	}

	switch p.curr.Type {
	case TokenLAnd:
		p.advance()
		right, err := p.parseLogicalExpr()
		if err != nil {
			return nil, err
		}
		return &FuncArg{
			LogicalExpr: &FilterExpr{
				Type: FilterLogicalAnd,
				Left: &FilterExpr{
					Test: &TestExpr{
						FilterQuery: query,
					},
				},
				Right: right,
			},
		}, nil

	case TokenLOr:
		p.advance()
		right, err := p.parseLogicalExpr()
		if err != nil {
			return nil, err
		}
		return &FuncArg{
			LogicalExpr: &FilterExpr{
				Type: FilterLogicalOr,
				Left: &FilterExpr{
					Test: &TestExpr{
						FilterQuery: query,
					},
				},
				Right: right,
			},
		}, nil

	case TokenEq, TokenNe, TokenLt, TokenLe, TokenGt, TokenGe:
		// reparse comparison
		p.curr = savedCurr
		p.peek = savedPeek
		p.lexer.pos = savedLexerPos
		comp, err := p.parseComparisonExpr()
		if err != nil {
			return nil, err
		}
		return &FuncArg{
			LogicalExpr: &FilterExpr{
				Type: FilterComparison,
				Comp: comp,
			},
		}, nil

	}
	return &FuncArg{Type: FuncArgFilterQuery, FilterQuery: query}, nil
}

func parseInteger(s string) (int, error) {
	var i int64
	_, err := fmt.Sscanf(s, "%d", &i)
	if err != nil {
		return 0, err
	}
	return int(i), nil
}
