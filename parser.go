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
// jsonpath-query = root-identifier segments
func (p *Parser) parseQuery() (*Query, error) {
	query := &Query{}

	// 必须以根标识符 $ 开始
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

// parseSegment 解析一个路径段
// segment = child-segment / descendant-segment
func (p *Parser) parseSegment() (*Segment, error) {
	switch p.curr.Type {
	case TokenDotDot:
		p.advance()
		return p.parseDescendantSegment()

	case TokenDot: // .name / .*
		p.advance()
		return p.parseDotSegment()

	case TokenLBracket: // .[
		return p.parseBracketSegment(ChildSegment)

	default:
		return nil, fmt.Errorf("unexpected token %s(%q), expected '.' or '..'", p.curr.Type, p.curr.Value)
	}
}

// parseDescendantSegment 解析后代段 ..name 或 ..[*]
// descendant-segment = ".." name-segment / "..[" selectors "]"
//
//	name-segment = "." member-name-shorthand / "[" name-selector "]"
//	member-name-shorthand = *identifier / null / true / false
func (p *Parser) parseDescendantSegment() (*Segment, error) {
	segment := &Segment{Type: DescendantSegment}

	switch p.curr.Type {
	case TokenLBracket:
		return p.parseBracketSegment(DescendantSegment)

	case TokenWildcard:
		// ..*
		segment.Selectors = []*Selector{{Type: WildcardSelector}}
		p.advance()
		return segment, nil

	case TokenIdent, TokenNull, TokenTrue, TokenFalse:
		// ..name
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

// parseDotSegment .name / .*
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

// parseBracketSegment 解析括号表示法
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
// selector = name-selector / wildcard-selector / index-selector / slice-selector / filter-selector
func (p *Parser) parseSelector() (*Selector, error) {
	switch p.curr.Type {
	case TokenString:
		// 名称选择器 'name' 或 "name"
		sel := &Selector{
			Type: NameSelector,
			Name: p.curr.Value,
		}
		p.advance()
		return sel, nil

	case TokenWildcard:
		// 通配符选择器 *
		sel := &Selector{Type: WildcardSelector}
		p.advance()
		return sel, nil

	case TokenNumber:
		if p.peek.Type == TokenColon {
			return p.parseSliceSelector()
		}
		// 索引选择器：数字后面是 ] 或 ,
		return p.parseIndexSelector()

	case TokenColon:
		return p.parseSliceSelector()

	case TokenQuestion:
		// 过滤器选择器 ?<logical-expr>
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

// parseSliceSelector 解析数组切片选择器 start:end:step
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

	return &Selector{Type: SliceSelector, Slice: slice}, nil
}

// parseFilterSelector 解析过滤器选择器 ?<logical-expr>
// filter-selector = "?" S logical-expr
func (p *Parser) parseFilterSelector() (*Selector, error) {
	// 当前 token 应该是 TokenQuestion
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

// parseLogicalExpr 解析逻辑表达式
// logical-expr = logical-or-expr
func (p *Parser) parseLogicalExpr() (*FilterExpr, error) {
	return p.parseLogicalOrExpr()
}

// parseLogicalOrExpr 解析逻辑或表达式
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

// parseLogicalAndExpr 解析逻辑与表达式
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

// parseBasicExpr 解析基本表达式
// basic-expr = paren-expr / comparison-expr / test-expr
func (p *Parser) parseBasicExpr() (*FilterExpr, error) {
	// paren-expr: [logical-not-op S] "(" S logical-expr S ")"
	// test-expr: [logical-not-op S] (filter-query / function-expr)

	// 以 ! 开头，需要区分是 paren-expr 还是 test-expr
	if p.curr.Type == TokenLNot {
		// 检查下一个 token 是否是 (
		if p.peek.Type == TokenLParen {
			// paren-expr（带 NOT）
			return p.parseParenExpr()
		}
		// test-expr（带 NOT）
		p.advance() // 消费 !
		test, err := p.parseTestExpr()
		if err != nil {
			return nil, err
		}
		return &FilterExpr{Type: FilterLogicalNot, Operand: &FilterExpr{Type: FilterTest, Test: test}}, nil
	}

	// 以 ( 开头，是 paren-expr
	if p.curr.Type == TokenLParen {
		return p.parseParenExpr()
	}

	// 其他情况：先尝试 comparison-expr，失败则尝试 test-expr
	return p.parseBasicExprWithFallback()
}

// parseBasicExprWithFallback 先尝试比较表达式，失败后尝试测试表达式
func (p *Parser) parseBasicExprWithFallback() (*FilterExpr, error) {
	// 保存当前状态
	savedCurr := p.curr
	savedPeek := p.peek
	savedLexerPos := p.lexer.pos

	// 尝试解析比较表达式
	comp, err := p.parseComparisonExpr()
	if err == nil {
		return &FilterExpr{Type: FilterComparison, Comp: comp}, nil
	}

	// 失败，恢复状态并尝试测试表达式
	p.curr = savedCurr
	p.peek = savedPeek
	p.lexer.pos = savedLexerPos

	test, err := p.parseTestExpr()
	if err != nil {
		return nil, err
	}
	return &FilterExpr{Type: FilterTest, Test: test}, nil
}

// parseParenExpr 解析括号表达式
// paren-expr = [logical-not-op S] "(" S logical-expr S ")"
func (p *Parser) parseParenExpr() (*FilterExpr, error) {
	hasNot := p.curr.Type == TokenLNot
	if hasNot {
		p.advance() // 消费 "!"
	}

	if p.curr.Type != TokenLParen {
		return nil, fmt.Errorf("expected '(' after '!', got %s(%q)", p.curr.Type, p.curr.Value)
	}
	p.advance() // 消费 "("

	expr, err := p.parseLogicalExpr()
	if err != nil {
		return nil, err
	}

	if p.curr.Type != TokenRParen {
		return nil, fmt.Errorf("expected ')' after filter expression, got %s(%q)", p.curr.Type, p.curr.Value)
	}
	p.advance() // 消费 ")"

	if hasNot {
		return &FilterExpr{
			Type:    FilterLogicalNot,
			Operand: expr,
		}, nil
	}

	return &FilterExpr{
		Type:    FilterParen,
		Operand: expr,
	}, nil
}

// parseComparisonExpr 解析比较表达式
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
// comparable = literal / singular-query / function-expr
func (p *Parser) parseComparable() (*Comparable, error) {
	switch p.curr.Type {
	case TokenString, TokenNumber, TokenTrue, TokenFalse, TokenNull:
		// 字面量
		lit, err := p.parseLiteral()
		if err != nil {
			return nil, err
		}
		return &Comparable{Type: ComparableLiteral, Literal: lit}, nil

	case TokenRoot, TokenCurrent:
		// 单值查询
		query, err := p.parseSingularQuery()
		if err != nil {
			return nil, err
		}
		return &Comparable{Type: ComparableSingularQuery, SingularQuery: query}, nil

	case TokenIdent:
		// 可能是函数表达式或单值查询
		if p.peek.Type == TokenLParen {
			// 函数表达式
			fn, err := p.parseFunctionExpr()
			if err != nil {
				return nil, err
			}
			return &Comparable{Type: ComparableFuncExpr, FuncExpr: fn}, nil
		}
		// 降级为字面量（标识符作为字符串字面量）
		lit := &LiteralValue{Type: LiteralString, Value: p.curr.Value}
		p.advance()
		return &Comparable{Type: ComparableLiteral, Literal: lit}, nil

	default:
		return nil, fmt.Errorf("unexpected token %s(%q) in comparable at position %d", p.curr.Type, p.curr.Value, p.curr.Pos)
	}
}

// parseLiteral 解析字面量
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

// parseSingularQuery 解析单值查询
// singular-query = rel-singular-query / abs-singular-query
// rel-singular-query = current-node-identifier singular-query-segments
// abs-singular-query = root-identifier singular-query-segments
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

// parseSingularSegment 解析单值查询段（只支持名称和索引）
func (p *Parser) parseSingularSegment() (*SingularSegment, error) {
	seg := &SingularSegment{}

	switch p.curr.Type {
	case TokenDot:
		// .name
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
			// ['name'] 或 ["name"]
			seg.Type = SingularNameSegment
			seg.Name = p.curr.Value
			p.advance()

			if err := p.expectToken(TokenRBracket); err != nil {
				return nil, err
			}
			p.advance()

			return seg, nil

		case TokenNumber:
			// [0]
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

// parseTestExpr 解析测试表达式（存在性测试）
// test-expr = [logical-not-op S] (filter-query / function-expr)
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
// filter-query = rel-query / jsonpath-query
// rel-query = current-node-identifier segments
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

	// 解析段，在遇到结束条件时停止（RBracket, RParen, Comma, EOF）
	for p.curr.Type == TokenDot || p.curr.Type == TokenDotDot || p.curr.Type == TokenLBracket {
		segment, err := p.parseSegment()
		if err != nil {
			return nil, err
		}
		query.Segments = append(query.Segments, segment)
	}

	return query, nil
}

// parseFunctionExpr 解析函数表达式
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
		// 字面量
		lit, err := p.parseLiteral()
		if err != nil {
			return nil, err
		}
		return &FuncArg{Type: FuncArgLiteral, Literal: lit}, nil

	case TokenRoot, TokenCurrent:
		// 可能是过滤器查询或逻辑表达式
		// 使用回溯策略：先尝试逻辑表达式，失败则尝试过滤器查询
		return p.parseFuncArgRootOrCurrent()

	case TokenLNot, TokenLParen, TokenIdent:
		if p.curr.Type == TokenIdent && p.peek.Type == TokenLParen {
			// 函数表达式
			fn, err := p.parseFunctionExpr()
			if err != nil {
				return nil, err
			}
			return &FuncArg{Type: FuncArgFuncExpr, FuncExpr: fn}, nil
		}
		// 逻辑表达式
		expr, err := p.parseLogicalExpr()
		if err != nil {
			return nil, err
		}
		return &FuncArg{Type: FuncArgLogicalExpr, LogicalExpr: expr}, nil

	default:
		return nil, fmt.Errorf("unexpected token %s(%q) in function argument", p.curr.Type, p.curr.Value)
	}
}

// parseFuncArgRootOrCurrent 解析以 $ 或 @ 开头的函数参数
// 优先过滤器查询，除非后面紧跟运算符（逻辑运算符或比较运算符）
func (p *Parser) parseFuncArgRootOrCurrent() (*FuncArg, error) {
	// 保存当前状态
	savedCurr := p.curr
	savedPeek := p.peek
	savedLexerPos := p.lexer.pos

	// 先尝试解析过滤器查询
	query, err := p.parseFilterQuery()
	if err == nil {
		// 如果下一个 token 是逻辑运算符或比较运算符，则应该解析为逻辑表达式
		if p.isOperator(p.curr.Type) {
			// 恢复状态，重新解析为逻辑表达式
			p.curr = savedCurr
			p.peek = savedPeek
			p.lexer.pos = savedLexerPos

			expr, err := p.parseLogicalExpr()
			if err != nil {
				return nil, err
			}
			return &FuncArg{Type: FuncArgLogicalExpr, LogicalExpr: expr}, nil
		}
		return &FuncArg{Type: FuncArgFilterQuery, FilterQuery: query}, nil
	}

	// 失败，恢复状态并尝试逻辑表达式
	p.curr = savedCurr
	p.peek = savedPeek
	p.lexer.pos = savedLexerPos

	expr, err := p.parseLogicalExpr()
	if err != nil {
		return nil, err
	}
	return &FuncArg{Type: FuncArgLogicalExpr, LogicalExpr: expr}, nil
}

// isOperator 检查是否是运算符（逻辑运算符或比较运算符）
func (p *Parser) isOperator(t TokenType) bool {
	return t == TokenLOr || t == TokenLAnd ||
		t == TokenEq || t == TokenNe ||
		t == TokenLt || t == TokenLe ||
		t == TokenGt || t == TokenGe
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
