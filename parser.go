package jsonpath

import (
	"fmt"
)

// Parse 解析 JSONPath 表达式字符串，返回 AST
func Parse(path string) (*Query, error) {
	lexer := NewLexer(path)
	p := &Parser{
		lexer: lexer,
	}
	p.advance()
	p.advance()
	return p.parseQuery()
}

// Parser JSONPath 语法分析器
type Parser struct {
	lexer *Lexer
	curr  Token
	peek  Token
}

// advance 读取下一个 token
func (p *Parser) advance() {
	p.curr = p.peek
	p.peek = p.lexer.NextToken()
}

// expectToken 期望当前 token 是指定类型，否则返回错误
func (p *Parser) expectToken(tokenType TokenType) error {
	if p.curr.Type != tokenType {
		return fmt.Errorf("except %s, got %s(%q)", tokenType, p.curr.Type, p.curr.Value)
	}
	return nil
}

// parseQuery 解析完整的 JSONPath 查询
// RFC 9535: jsonpath-query = root-identifier segments
func (p *Parser) parseQuery() (*Query, error) {
	query := &Query{}

	// 必须以根标识符 $ 开始
	if err := p.expectToken(TokenRoot); err != nil {
		return nil, err
	}

	for p.curr.Type != TokenEOF {
		segment, err := p.parseSegment()
		if err != nil {
			return nil, err
		}
		query.Segments = append(query.Segments, segment)
	}

	return query, nil
}

// parseSegment 解析一个路径段
func (p *Parser) parseSegment() (*Segment, error) {
	switch p.curr.Type {
	case TokenDotDot:
		p.advance()
		return p.parseDescendantSegment()
	case TokenDot:
		p.advance()
		return p.parseDotSegment()
	case TokenLBracket:
		p.advance()
		return p.parseBracketSegment(ChildSegment)
	default:
		return nil, fmt.Errorf("unexpected token %s(%q), expected '.' or '..'", p.curr.Type, p.curr.Value)
	}
}

// parseDescendantSegment 解析后代段 ..name 或 ..[*]
// RFC 9535: descendant-segment = ".." name-segment / "..[" selectors "]"
//
//	name-segment = "." member-name-shorthand / "[" name-selector "]"
//	member-name-shorthand = *identifier / null / true / false
func (p *Parser) parseDescendantSegment() (*Segment, error) {
	segment := &Segment{Type: DescendantSegment}

	switch p.curr.Type {
	case TokenLBracket:
		// ..[selectors] 形式
		p.advance()
		selectors, err := p.parseSelectors()
		if err != nil {
			return nil, err
		}
		segment.Selectors = selectors
		return segment, nil

	case TokenWildcard:
		// ..* 是 ..[*] 的简写
		segment.Selectors = []*Selector{{Kind: WildcardSelector}}
		p.advance()
		return segment, nil

	case TokenIdent, TokenNull, TokenTrue, TokenFalse:
		// ..name 是 ..['name'] 的简写
		name := p.curr.Value
		segment.Selectors = []*Selector{{
			Kind: NameSelector,
			Name: name,
		}}
		p.advance()
		return segment, nil

	default:
		return nil, fmt.Errorf("unexpected token %s(%q) after '..' at position %d", p.curr.Type, p.curr.Value, p.curr.Pos)
	}
}

// parseDotSegment 解析子段的点简写形式 .name 或 .*
// RFC 9535: child-segment-shorthand = "." ( wildcard / member-name-shorthand )
func (p *Parser) parseDotSegment() (*Segment, error) {
	segment := &Segment{Type: ChildSegment}

	switch p.curr.Type {
	case TokenWildcard:
		// .* 是 [*] 的简写
		segment.Selectors = []*Selector{{Kind: WildcardSelector}}
		p.advance()
		return segment, nil

	case TokenIdent, TokenNull, TokenTrue, TokenFalse:
		// .name 是 ['name'] 的简写
		name := p.curr.Value
		segment.Selectors = []*Selector{{
			Kind: NameSelector,
			Name: name,
		}}
		p.advance()
		return segment, nil

	default:
		return nil, fmt.Errorf("unexpected token %s(%q) after '.' at position %d", p.curr.Type, p.curr.Value, p.curr.Pos)
	}
}

// parseBracketSegment 解析括号表示法 [selector1, selector2, ...]
// RFC 9535: child-segment = "[" selectors "]"
//
//	descendant-segment = ".." "[" selectors "]"
func (p *Parser) parseBracketSegment(segType SegmentType) (*Segment, error) {
	segment := &Segment{Type: segType}

	selectors, err := p.parseSelectors()
	if err != nil {
		return nil, err
	}
	segment.Selectors = selectors

	// 期望闭合括号
	if err := p.expectToken(TokenRBracket); err != nil {
		return nil, fmt.Errorf("expected ']', got %s(%q) at position %d", p.curr.Type, p.curr.Value, p.curr.Pos)
	}
	p.advance()

	return segment, nil
}

// parseSelectors 解析逗号分隔的选择器列表
// RFC 9535: selectors = selector ["," selectors]
func (p *Parser) parseSelectors() ([]*Selector, error) {
	var selectors []*Selector

	// 解析第一个选择器
	sel, err := p.parseSelector()
	if err != nil {
		return nil, err
	}
	selectors = append(selectors, sel)

	// 解析后续选择器
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

// parseSelector 解析单个选择器
// RFC 9535: selector = name-selector / wildcard-selector / index-selector /
//
//	slice-selector / filter-selector
func (p *Parser) parseSelector() (*Selector, error) {
	switch p.curr.Type {
	case TokenString:
		// 名称选择器 'name' 或 "name"
		sel := &Selector{
			Kind: NameSelector,
			Name: p.curr.Value,
		}
		p.advance()
		return sel, nil

	case TokenWildcard:
		// 通配符选择器 *
		sel := &Selector{Kind: WildcardSelector}
		p.advance()
		return sel, nil

	case TokenQuestion:
		// 过滤器选择器 ?<logical-expr>
		return p.parseFilterSelector()

	case TokenNumber, TokenColon:
		// 索引选择器或切片选择器
		return p.parseIndexOrSliceSelector()

	default:
		return nil, fmt.Errorf("unexpected token %s(%q) in selector at position %d", p.curr.Type, p.curr.Value, p.curr.Pos)
	}
}

// parseIndexOrSliceSelector 解析索引或切片选择器
// RFC 9535: index-selector = int
//
//	slice-selector = [start S] ":" S [end S] [":" [S step ]]
func (p *Parser) parseIndexOrSliceSelector() (*Selector, error) {
	// 检查是否是切片（包含冒号）
	if p.curr.Type == TokenColon || (p.curr.Type == TokenNumber && p.peek.Type == TokenColon) {
		return p.parseSliceSelector()
	}

	// 索引选择器
	if p.curr.Type != TokenNumber {
		return nil, fmt.Errorf("expected number or ':', got %s(%q) at position %d", p.curr.Type, p.curr.Value, p.curr.Pos)
	}

	index, err := parseInteger(p.curr.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid index %q at position %d: %w", p.curr.Value, p.curr.Pos, err)
	}

	sel := &Selector{
		Kind:  IndexSelector,
		Index: index,
	}
	p.advance()
	return sel, nil
}

// parseSliceSelector 解析数组切片选择器 start:end:step
// RFC 9535 Section 2.3.4
func (p *Parser) parseSliceSelector() (*Selector, error) {
	slice := &SliceParams{}

	// 解析 start（可选）
	if p.curr.Type == TokenNumber {
		start, err := parseInteger(p.curr.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid slice start %q: %w", p.curr.Value, err)
		}
		slice.Start = &start
		p.advance()
	}

	// 期望冒号
	if err := p.expectToken(TokenColon); err != nil {
		return nil, err
	}
	p.advance()

	// 解析 end（可选）
	if p.curr.Type == TokenNumber {
		end, err := parseInteger(p.curr.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid slice end %q: %w", p.curr.Value, err)
		}
		slice.End = &end
		p.advance()
	}

	// 解析 step（可选）
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

	return &Selector{Kind: SliceSelector, Slice: slice}, nil
}

// parseFilterSelector 解析过滤器选择器 ?<logical-expr>
// RFC 9535: filter-selector = "?" S logical-expr
func (p *Parser) parseFilterSelector() (*Selector, error) {
	// 当前 token 应该是 TokenQuestion
	if p.curr.Type != TokenQuestion {
		return nil, fmt.Errorf("expected '?', got %s(%q)", p.curr.Type, p.curr.Value)
	}
	p.advance()

	expr, err := p.parseLogicalExpr()
	if err != nil {
		return nil, err
	}

	return &Selector{Kind: FilterSelector, Filter: expr}, nil
}

// parseLogicalExpr 解析逻辑表达式
// RFC 9535: logical-or-expr = logical-and-expr *(S "||" S logical-and-expr)
//
//	logical-and-expr = basic-expr *(S "&&" S basic-expr)
func (p *Parser) parseLogicalExpr() (*FilterExpr, error) {
	return p.parseLogicalOrExpr()
}

// parseLogicalOrExpr 解析逻辑或表达式
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
			Kind:  FilterLogicalOr,
			Left:  left,
			Right: right,
		}
	}

	return left, nil
}

// parseLogicalAndExpr 解析逻辑与表达式
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
			Kind:  FilterLogicalAnd,
			Left:  left,
			Right: right,
		}
	}

	return left, nil
}

// parseBasicExpr 解析基本表达式
// RFC 9535: basic-expr = paren-expr / comparison-expr / test-expr
func (p *Parser) parseBasicExpr() (*FilterExpr, error) {
	// 检查逻辑非运算符
	negated := false
	if p.curr.Type == TokenLNot {
		negated = true
		p.advance()
	}

	switch p.curr.Type {
	case TokenLParen:
		// 括号表达式
		p.advance()
		expr, err := p.parseLogicalExpr()
		if err != nil {
			return nil, err
		}
		if err := p.expectToken(TokenRParen); err != nil {
			return nil, err
		}
		p.advance()
		result := &FilterExpr{Kind: FilterParen, Operand: expr}
		if negated {
			result = &FilterExpr{Kind: FilterLogicalNot, Operand: result}
		}
		return result, nil

	case TokenIdent:
		// 可能是测试表达式（存在性测试）或函数表达式
		if p.peek.Type == TokenLParen {
			// 函数表达式
			funcExpr, err := p.parseFunctionExpr()
			if err != nil {
				return nil, err
			}
			return &FilterExpr{Kind: FilterTest, Test: &TestExpr{FuncExpr: funcExpr}}, nil
		}
		// 降级为测试表达式（存在性测试）
		fallthrough

	default:
		// 比较表达式或测试表达式
		if p.isComparisonOp(p.peek.Type) {
			comp, err := p.parseComparisonExpr()
			if err != nil {
				return nil, err
			}
			if negated {
				return &FilterExpr{Kind: FilterLogicalNot, Operand: &FilterExpr{Kind: FilterComparison, Comp: comp}}, nil
			}
			return &FilterExpr{Kind: FilterComparison, Comp: comp}, nil
		}

		// 测试表达式（存在性测试）
		test, err := p.parseTestExpr()
		if err != nil {
			return nil, err
		}
		if negated {
			test.Negated = true
		}
		return &FilterExpr{Kind: FilterTest, Test: test}, nil
	}
}

// isComparisonOp 检查是否是比较运算符
func (p *Parser) isComparisonOp(tokenType TokenType) bool {
	switch tokenType {
	case TokenEq, TokenNe, TokenLt, TokenLe, TokenGt, TokenGe:
		return true
	default:
		return false
	}
}

// parseComparisonExpr 解析比较表达式
// RFC 9535: comparison-expr = comparable S comparison-op S comparable
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

// parseComparisonOp 解析比较运算符
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

// parseComparable 解析可比较值
// RFC 9535: comparable = literal / singular-query / function-expr
func (p *Parser) parseComparable() (*Comparable, error) {
	switch p.curr.Type {
	case TokenString, TokenNumber, TokenTrue, TokenFalse, TokenNull:
		// 字面量
		lit, err := p.parseLiteral()
		if err != nil {
			return nil, err
		}
		return &Comparable{Kind: ComparableLiteral, Literal: lit}, nil

	case TokenRoot, TokenCurrent:
		// 单值查询
		query, err := p.parseSingularQuery()
		if err != nil {
			return nil, err
		}
		return &Comparable{Kind: ComparableSingularQuery, SingularQuery: query}, nil

	case TokenIdent:
		// 可能是函数表达式或单值查询（成员名简写）
		if p.peek.Type == TokenLParen {
			// 函数表达式
			fn, err := p.parseFunctionExpr()
			if err != nil {
				return nil, err
			}
			return &Comparable{Kind: ComparableFuncExpr, FuncExpr: fn}, nil
		}
		// 降级为字面量（标识符作为字符串字面量）
		lit := &LiteralValue{Type: LiteralTypeString, Value: p.curr.Value}
		p.advance()
		return &Comparable{Kind: ComparableLiteral, Literal: lit}, nil

	default:
		return nil, fmt.Errorf("unexpected token %s(%q) in comparable at position %d", p.curr.Type, p.curr.Value, p.curr.Pos)
	}
}

// parseLiteral 解析字面量
func (p *Parser) parseLiteral() (*LiteralValue, error) {
	switch p.curr.Type {
	case TokenString:
		lit := &LiteralValue{Type: LiteralTypeString, Value: p.curr.Value}
		p.advance()
		return lit, nil
	case TokenNumber:
		lit := &LiteralValue{Type: LiteralTypeNumber, Value: p.curr.Value}
		p.advance()
		return lit, nil
	case TokenTrue:
		lit := &LiteralValue{Type: LiteralTypeTrue, Value: p.curr.Value}
		p.advance()
		return lit, nil
	case TokenFalse:
		lit := &LiteralValue{Type: LiteralTypeFalse, Value: p.curr.Value}
		p.advance()
		return lit, nil
	case TokenNull:
		lit := &LiteralValue{Type: LiteralTypeNull, Value: p.curr.Value}
		p.advance()
		return lit, nil
	default:
		return nil, fmt.Errorf("expected literal, got %s(%q)", p.curr.Type, p.curr.Value)
	}
}

// parseSingularQuery 解析单值查询
// RFC 9535: singular-query = rel-singular-query / abs-singular-query
//
//	rel-singular-query = current-node-identifier singular-query-segments
//	abs-singular-query = root-identifier singular-query-segments
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

	// 解析单值查询段
	for p.curr.Type == TokenDot || p.curr.Type == TokenLBracket {
		seg, err := p.parseSingularSegment()
		if err != nil {
			return nil, err
		}
		query.Segments = append(query.Segments, seg)
	}

	return query, nil
}

// parseSingularSegment 解析单值查询段（只支持名称和索引）
func (p *Parser) parseSingularSegment() (*SingularSegment, error) {
	seg := &SingularSegment{}

	switch p.curr.Type {
	case TokenDot:
		p.advance()
		if p.curr.Type != TokenIdent && p.curr.Type != TokenNull && p.curr.Type != TokenTrue && p.curr.Type != TokenFalse {
			return nil, fmt.Errorf("expected identifier after '.', got %s(%q)", p.curr.Type, p.curr.Value)
		}
		seg.IsIndex = false
		seg.Name = p.curr.Value
		p.advance()
		return seg, nil

	case TokenLBracket:
		p.advance()
		defer p.advance() // 消费闭合括号

		if p.curr.Type == TokenString {
			// 名称选择器 ['name']
			seg.IsIndex = false
			seg.Name = p.curr.Value
			p.advance()
			if p.curr.Type != TokenRBracket {
				return nil, fmt.Errorf("expected ']', got %s(%q)", p.curr.Type, p.curr.Value)
			}
			return seg, nil
		}

		if p.curr.Type == TokenNumber {
			// 索引选择器 [0]
			index, err := parseInteger(p.curr.Value)
			if err != nil {
				return nil, fmt.Errorf("invalid index: %w", err)
			}
			seg.IsIndex = true
			seg.Index = index
			p.advance()
			if p.curr.Type != TokenRBracket {
				return nil, fmt.Errorf("expected ']', got %s(%q)", p.curr.Type, p.curr.Value)
			}
			return seg, nil
		}

		return nil, fmt.Errorf("expected string or number in singular segment, got %s(%q)", p.curr.Type, p.curr.Value)

	default:
		return nil, fmt.Errorf("expected '.' or '[', got %s(%q)", p.curr.Type, p.curr.Value)
	}
}

// parseTestExpr 解析测试表达式（存在性测试）
// RFC 9535: test-expr = [logical-not-op S] (filter-query / function-expr)
func (p *Parser) parseTestExpr() (*TestExpr, error) {
	test := &TestExpr{}

	switch p.curr.Type {
	case TokenRoot, TokenCurrent:
		// 过滤器查询（存在性测试）
		query, err := p.parseFilterQuery()
		if err != nil {
			return nil, err
		}
		test.FilterQuery = query
		return test, nil

	case TokenIdent:
		if p.peek.Type == TokenLParen {
			// 函数表达式
			fn, err := p.parseFunctionExpr()
			if err != nil {
				return nil, err
			}
			test.FuncExpr = fn
			return test, nil
		}
		// 标识符作为过滤器查询的起始
		query, err := p.parseFilterQuery()
		if err != nil {
			return nil, err
		}
		test.FilterQuery = query
		return test, nil

	default:
		return nil, fmt.Errorf("expected filter query or function expression, got %s(%q)", p.curr.Type, p.curr.Value)
	}
}

// parseFilterQuery 解析过滤器查询
// RFC 9535: filter-query = rel-query / jsonpath-query
//
//	rel-query = current-node-identifier segments
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
		// 无显式标识符，当作当前节点引用
		query.Relative = true
	}

	// 解析段
	for p.curr.Type == TokenDot || p.curr.Type == TokenDotDot || p.curr.Type == TokenLBracket {
		seg, err := p.parseSegment()
		if err != nil {
			return nil, err
		}
		query.Segments = append(query.Segments, seg)
	}

	return query, nil
}

// parseFunctionExpr 解析函数表达式
// RFC 9535: function-expr = function-name "(" S [function-argument *(S "," S function-argument)] S ")"
func (p *Parser) parseFunctionExpr() (*FuncCall, error) {
	if p.curr.Type != TokenIdent {
		return nil, fmt.Errorf("expected function name, got %s(%q)", p.curr.Type, p.curr.Value)
	}

	name := p.curr.Value
	p.advance()

	if p.curr.Type != TokenLParen {
		return nil, fmt.Errorf("expected '(' after function name, got %s(%q)", p.curr.Type, p.curr.Value)
	}
	p.advance()

	fn := &FuncCall{Name: name, Args: []*FuncArg{}}

	// 解析参数
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

	if p.curr.Type != TokenRParen {
		return nil, fmt.Errorf("expected ')' after function arguments, got %s(%q)", p.curr.Type, p.curr.Value)
	}
	p.advance()

	return fn, nil
}

// parseFuncArg 解析函数参数
// RFC 9535: function-argument = literal / filter-query / logical-expr / function-expr
func (p *Parser) parseFuncArg() (*FuncArg, error) {
	switch p.curr.Type {
	case TokenString, TokenNumber, TokenTrue, TokenFalse, TokenNull:
		// 字面量
		lit, err := p.parseLiteral()
		if err != nil {
			return nil, err
		}
		return &FuncArg{Kind: FuncArgLiteral, Literal: lit}, nil

	case TokenRoot, TokenCurrent:
		// 过滤器查询
		query, err := p.parseFilterQuery()
		if err != nil {
			return nil, err
		}
		return &FuncArg{Kind: FuncArgFilterQuery, FilterQuery: query}, nil

	case TokenLNot, TokenLParen, TokenIdent:
		// 逻辑表达式或函数表达式
		if p.curr.Type == TokenIdent && p.peek.Type == TokenLParen {
			// 函数表达式
			fn, err := p.parseFunctionExpr()
			if err != nil {
				return nil, err
			}
			return &FuncArg{Kind: FuncArgFuncExpr, FuncExpr: fn}, nil
		}
		// 逻辑表达式
		expr, err := p.parseLogicalExpr()
		if err != nil {
			return nil, err
		}
		return &FuncArg{Kind: FuncArgLogicalExpr, LogicalExpr: expr}, nil

	default:
		return nil, fmt.Errorf("unexpected token %s(%q) in function argument", p.curr.Type, p.curr.Value)
	}
}

// parseInteger 解析整数字符串
func parseInteger(s string) (int, error) {
	var i int64
	_, err := fmt.Sscanf(s, "%d", &i)
	if err != nil {
		return 0, err
	}
	return int(i), nil
}
