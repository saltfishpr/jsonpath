package jsonpath

import (
	"testing"
)

// TestParserBasicQueries 测试基本查询解析
func TestParserBasicQueries(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *Query)
	}{
		{
			name:    "root only",
			input:   "$",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				if len(q.Segments) != 0 {
					t.Errorf("expected 0 segments, got %d", len(q.Segments))
				}
			},
		},
		{
			name:    "child segment dot notation",
			input:   "$.name",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				if len(q.Segments) != 1 {
					t.Fatalf("expected 1 segment, got %d", len(q.Segments))
				}
				seg := q.Segments[0]
				if seg.Type != ChildSegment {
					t.Errorf("expected ChildSegment, got %v", seg.Type)
				}
				if len(seg.Selectors) != 1 {
					t.Fatalf("expected 1 selector, got %d", len(seg.Selectors))
				}
				if seg.Selectors[0].Kind != NameSelector {
					t.Errorf("expected NameSelector, got %v", seg.Selectors[0].Kind)
				}
				if seg.Selectors[0].Name != "name" {
					t.Errorf("expected name 'name', got %q", seg.Selectors[0].Name)
				}
			},
		},
		{
			name:    "wildcard selector",
			input:   "$.*",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				if len(q.Segments) != 1 {
					t.Fatalf("expected 1 segment, got %d", len(q.Segments))
				}
				if q.Segments[0].Selectors[0].Kind != WildcardSelector {
					t.Errorf("expected WildcardSelector, got %v", q.Segments[0].Selectors[0].Kind)
				}
			},
		},
		{
			name:    "multiple child segments",
			input:   "$.store.book",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				if len(q.Segments) != 2 {
					t.Fatalf("expected 2 segments, got %d", len(q.Segments))
				}
				if q.Segments[0].Selectors[0].Name != "store" {
					t.Errorf("expected 'store', got %q", q.Segments[0].Selectors[0].Name)
				}
				if q.Segments[1].Selectors[0].Name != "book" {
					t.Errorf("expected 'book', got %q", q.Segments[1].Selectors[0].Name)
				}
			},
		},
		{
			name:    "bracket notation name",
			input:   "$['name']",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				if len(q.Segments) != 1 {
					t.Fatalf("expected 1 segment, got %d", len(q.Segments))
				}
				if q.Segments[0].Selectors[0].Kind != NameSelector {
					t.Errorf("expected NameSelector, got %v", q.Segments[0].Selectors[0].Kind)
				}
				if q.Segments[0].Selectors[0].Name != "name" {
					t.Errorf("expected name 'name', got %q", q.Segments[0].Selectors[0].Name)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, query)
			}
		})
	}
}

// TestParserIndexSelectors 测试索引选择器解析
func TestParserIndexSelectors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *Query)
	}{
		{
			name:    "positive index",
			input:   "$[0]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Kind != IndexSelector {
					t.Errorf("expected IndexSelector, got %v", sel.Kind)
				}
				if sel.Index != 0 {
					t.Errorf("expected index 0, got %d", sel.Index)
				}
			},
		},
		{
			name:    "negative index",
			input:   "$[-1]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Index != -1 {
					t.Errorf("expected index -1, got %d", sel.Index)
				}
			},
		},
		{
			name:    "multiple indices",
			input:   "$[0,1,2]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				if len(q.Segments[0].Selectors) != 3 {
					t.Fatalf("expected 3 selectors, got %d", len(q.Segments[0].Selectors))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, query)
			}
		})
	}
}

// TestParserSliceSelectors 测试切片选择器解析
func TestParserSliceSelectors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *Selector)
	}{
		{
			name:    "start:end",
			input:   "$[0:5]",
			wantErr: false,
			check: func(t *testing.T, s *Selector) {
				if s.Kind != SliceSelector {
					t.Errorf("expected SliceSelector, got %v", s.Kind)
				}
				if s.Slice.Start == nil || *s.Slice.Start != 0 {
					t.Errorf("expected start 0")
				}
				if s.Slice.End == nil || *s.Slice.End != 5 {
					t.Errorf("expected end 5")
				}
			},
		},
		{
			name:    "start:end:step",
			input:   "$[0:10:2]",
			wantErr: false,
			check: func(t *testing.T, s *Selector) {
				if s.Slice.Start == nil || *s.Slice.Start != 0 {
					t.Errorf("expected start 0")
				}
				if s.Slice.End == nil || *s.Slice.End != 10 {
					t.Errorf("expected end 10")
				}
				if s.Slice.Step == nil || *s.Slice.Step != 2 {
					t.Errorf("expected step 2")
				}
			},
		},
		{
			name:    "only end",
			input:   "$[:5]",
			wantErr: false,
			check: func(t *testing.T, s *Selector) {
				if s.Slice.Start != nil {
					t.Errorf("expected nil start")
				}
				if s.Slice.End == nil || *s.Slice.End != 5 {
					t.Errorf("expected end 5")
				}
			},
		},
		{
			name:    "only start",
			input:   "$[5:]",
			wantErr: false,
			check: func(t *testing.T, s *Selector) {
				if s.Slice.Start == nil || *s.Slice.Start != 5 {
					t.Errorf("expected start 5")
				}
				if s.Slice.End != nil {
					t.Errorf("expected nil end")
				}
			},
		},
		{
			name:    "reverse slice",
			input:   "$[::-1]",
			wantErr: false,
			check: func(t *testing.T, s *Selector) {
				if s.Slice.Start != nil {
					t.Errorf("expected nil start")
				}
				if s.Slice.End != nil {
					t.Errorf("expected nil end")
				}
				if s.Slice.Step == nil || *s.Slice.Step != -1 {
					t.Errorf("expected step -1")
				}
			},
		},
		{
			name:    "negative indices",
			input:   "$[-3:-1]",
			wantErr: false,
			check: func(t *testing.T, s *Selector) {
				if s.Slice.Start == nil || *s.Slice.Start != -3 {
					t.Errorf("expected start -3")
				}
				if s.Slice.End == nil || *s.Slice.End != -1 {
					t.Errorf("expected end -1")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				sel := query.Segments[0].Selectors[0]
				tt.check(t, sel)
			}
		})
	}
}

// TestParserDescendantSegment 测试后代段解析
func TestParserDescendantSegment(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *Query)
	}{
		{
			name:    "descendant name",
			input:   "$..author",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				if len(q.Segments) != 1 {
					t.Fatalf("expected 1 segment, got %d", len(q.Segments))
				}
				if q.Segments[0].Type != DescendantSegment {
					t.Errorf("expected DescendantSegment, got %v", q.Segments[0].Type)
				}
				if q.Segments[0].Selectors[0].Kind != NameSelector {
					t.Errorf("expected NameSelector, got %v", q.Segments[0].Selectors[0].Kind)
				}
				if q.Segments[0].Selectors[0].Name != "author" {
					t.Errorf("expected name 'author', got %q", q.Segments[0].Selectors[0].Name)
				}
			},
		},
		{
			name:    "descendant wildcard",
			input:   "$..*",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				if q.Segments[0].Type != DescendantSegment {
					t.Errorf("expected DescendantSegment")
				}
				if q.Segments[0].Selectors[0].Kind != WildcardSelector {
					t.Errorf("expected WildcardSelector")
				}
			},
		},
		{
			name:    "descendant bracket notation",
			input:   "$..[price]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				if q.Segments[0].Type != DescendantSegment {
					t.Errorf("expected DescendantSegment")
				}
				if q.Segments[0].Selectors[0].Kind != NameSelector {
					t.Errorf("expected NameSelector")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, query)
			}
		})
	}
}

// TestParserFilterExpressions 测试过滤器表达式解析
func TestParserFilterExpressions(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *FilterExpr)
	}{
		{
			name:    "simple comparison",
			input:   "$[?@.price < 10]",
			wantErr: false,
			check: func(t *testing.T, e *FilterExpr) {
				if e.Kind != FilterComparison {
					t.Errorf("expected FilterComparison, got %v", e.Kind)
				}
				if e.Comp.Op != CompLt {
					t.Errorf("expected CompLt, got %v", e.Comp.Op)
				}
			},
		},
		{
			name:    "logical and",
			input:   "$[?@.price < 10 && @.category == 'fiction']",
			wantErr: false,
			check: func(t *testing.T, e *FilterExpr) {
				if e.Kind != FilterLogicalAnd {
					t.Errorf("expected FilterLogicalAnd, got %v", e.Kind)
				}
			},
		},
		{
			name:    "logical or",
			input:   "$[?@.price < 10 || @.price > 100]",
			wantErr: false,
			check: func(t *testing.T, e *FilterExpr) {
				if e.Kind != FilterLogicalOr {
					t.Errorf("expected FilterLogicalOr, got %v", e.Kind)
				}
			},
		},
		{
			name:    "logical not",
			input:   "$[?!@.isbn]",
			wantErr: false,
			check: func(t *testing.T, e *FilterExpr) {
				if e.Kind != FilterLogicalNot {
					t.Errorf("expected FilterLogicalNot, got %v", e.Kind)
				}
			},
		},
		{
			name:    "parenthesized expression",
			input:   "$[?(@.price < 10)]",
			wantErr: false,
			check: func(t *testing.T, e *FilterExpr) {
				if e.Kind != FilterParen {
					t.Errorf("expected FilterParen, got %v", e.Kind)
				}
			},
		},
		{
			name:    "existence test",
			input:   "$[?@.isbn]",
			wantErr: false,
			check: func(t *testing.T, e *FilterExpr) {
				if e.Kind != FilterTest {
					t.Errorf("expected FilterTest, got %v", e.Kind)
				}
				if e.Test.FilterQuery == nil {
					t.Errorf("expected FilterQuery")
				}
			},
		},
		{
			name:    "function test",
			input:   "$[?length(@.name) > 5]",
			wantErr: false,
			check: func(t *testing.T, e *FilterExpr) {
				if e.Kind != FilterComparison {
					t.Errorf("expected FilterComparison, got %v", e.Kind)
				}
				if e.Comp.Left.Kind != ComparableFuncExpr {
					t.Errorf("expected function on left side")
				}
				if e.Comp.Left.FuncExpr.Name != "length" {
					t.Errorf("expected function 'length', got %q", e.Comp.Left.FuncExpr.Name)
				}
			},
		},
		{
			name:    "all comparison operators",
			input:   "$[?@.x == 1]",
			wantErr: false,
		},
		{
			name:    "ne operator",
			input:   "$[?@.x != 1]",
			wantErr: false,
		},
		{
			name:    "le operator",
			input:   "$[?@.x <= 1]",
			wantErr: false,
		},
		{
			name:    "gt operator",
			input:   "$[?@.x > 1]",
			wantErr: false,
		},
		{
			name:    "ge operator",
			input:   "$[?@.x >= 1]",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				sel := query.Segments[0].Selectors[0]
				if sel.Kind != FilterSelector {
					t.Fatalf("expected FilterSelector, got %v", sel.Kind)
				}
				tt.check(t, sel.Filter)
			}
		})
	}
}

// TestParserFunctions 测试函数解析
func TestParserFunctions(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *FuncCall)
	}{
		{
			name:    "length function",
			input:   "$[?length(@.name) > 0]",
			wantErr: false,
			check: func(t *testing.T, fn *FuncCall) {
				if fn.Name != "length" {
					t.Errorf("expected function name 'length', got %q", fn.Name)
				}
				if len(fn.Args) != 1 {
					t.Errorf("expected 1 argument, got %d", len(fn.Args))
				}
			},
		},
		{
			name:    "count function",
			input:   "$[?count(@..items) > 0]",
			wantErr: false,
			check: func(t *testing.T, fn *FuncCall) {
				if fn.Name != "count" {
					t.Errorf("expected function name 'count'")
				}
			},
		},
		{
			name:    "match function",
			input:   "$[?match(@.date, '^1974')]",
			wantErr: false,
			check: func(t *testing.T, fn *FuncCall) {
				if fn.Name != "match" {
					t.Errorf("expected function name 'match'")
				}
				if len(fn.Args) != 2 {
					t.Errorf("expected 2 arguments")
				}
			},
		},
		{
			name:    "search function",
			input:   "$[?search(@.text, 'pattern')]",
			wantErr: false,
			check: func(t *testing.T, fn *FuncCall) {
				if fn.Name != "search" {
					t.Errorf("expected function name 'search'")
				}
			},
		},
		{
			name:    "value function",
			input:   "$[?value(@.x) == 42]",
			wantErr: false,
			check: func(t *testing.T, fn *FuncCall) {
				if fn.Name != "value" {
					t.Errorf("expected function name 'value'")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				sel := query.Segments[0].Selectors[0]
				comp := sel.Filter.Comp
				fn := comp.Left.FuncExpr
				tt.check(t, fn)
			}
		})
	}
}

// TestParserLiterals 测试字面量解析
func TestParserLiterals(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *LiteralValue)
	}{
		{
			name:    "string literal",
			input:   `$[?@.x == "test"]`,
			wantErr: false,
			check: func(t *testing.T, lit *LiteralValue) {
				if lit.Type != LiteralTypeString {
					t.Errorf("expected LiteralTypeString, got %v", lit.Type)
				}
				if lit.Value != "test" {
					t.Errorf("expected value 'test', got %q", lit.Value)
				}
			},
		},
		{
			name:    "number literal",
			input:   "$[?@.x == 42]",
			wantErr: false,
			check: func(t *testing.T, lit *LiteralValue) {
				if lit.Type != LiteralTypeNumber {
					t.Errorf("expected LiteralTypeNumber, got %v", lit.Type)
				}
				if lit.Value != "42" {
					t.Errorf("expected value '42', got %q", lit.Value)
				}
			},
		},
		{
			name:    "true literal",
			input:   "$[?@.x == true]",
			wantErr: false,
			check: func(t *testing.T, lit *LiteralValue) {
				if lit.Type != LiteralTypeTrue {
					t.Errorf("expected LiteralTypeTrue, got %v", lit.Type)
				}
			},
		},
		{
			name:    "false literal",
			input:   "$[?@.x == false]",
			wantErr: false,
			check: func(t *testing.T, lit *LiteralValue) {
				if lit.Type != LiteralTypeFalse {
					t.Errorf("expected LiteralTypeFalse, got %v", lit.Type)
				}
			},
		},
		{
			name:    "null literal",
			input:   "$[?@.x == null]",
			wantErr: false,
			check: func(t *testing.T, lit *LiteralValue) {
				if lit.Type != LiteralTypeNull {
					t.Errorf("expected LiteralTypeNull, got %v", lit.Type)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				sel := query.Segments[0].Selectors[0]
				lit := sel.Filter.Comp.Right.Literal
				tt.check(t, lit)
			}
		})
	}
}

// TestParserSingularQuery 测试单值查询解析
func TestParserSingularQuery(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *SingularQuery)
	}{
		{
			name:    "relative singular query",
			input:   "$[?@.price < $.maxPrice]",
			wantErr: false,
			check: func(t *testing.T, q *SingularQuery) {
				if !q.Relative {
					t.Errorf("expected relative query")
				}
			},
		},
		{
			name:    "absolute singular query",
			input:   "$[?@.price < $.config.maxPrice]",
			wantErr: false,
			check: func(t *testing.T, q *SingularQuery) {
				if q.Relative {
					t.Errorf("expected absolute query")
				}
			},
		},
		{
			name:    "singular query with index",
			input:   "$[?@.x == @.arr[0]]",
			wantErr: false,
			check: func(t *testing.T, q *SingularQuery) {
				if len(q.Segments) != 2 {
					t.Errorf("expected 2 segments")
				}
				if !q.Segments[1].IsIndex {
					t.Errorf("expected index segment")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				sel := query.Segments[0].Selectors[0]
				sq := sel.Filter.Comp.Right.SingularQuery
				tt.check(t, sq)
			}
		})
	}
}

// TestParserErrors 测试解析错误
func TestParserErrors(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		errContains string
	}{
		{
			name:     "missing root",
			input:    ".name",
			wantErr:  true,
			errContains: "root",
		},
		{
			name:     "unclosed bracket",
			input:    "$[0",
			wantErr:  true,
			errContains: "]",
		},
		{
			name:     "invalid token after dot",
			input:    "$.123",
			wantErr:  true,
		},
		{
			name:     "empty selector",
			input:    "$[]",
			wantErr:  true,
		},
		{
			name:     "unclosed filter",
			input:    "$[?@.x < 10",
			wantErr:  true,
			errContains: "]",
		},
		{
			name:     "invalid comparison",
			input:    "$[?@.x = 10]",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got %q", tt.errContains, err.Error())
				}
			}
		})
	}
}

// TestRFC9535Table2Examples 测试 RFC 9535 Table 2 示例表达式
func TestRFC9535Table2Examples(t *testing.T) {
	examples := []string{
		`$.store.book[*].author`,
		`$..author`,
		`$.store.*`,
		`$.store..price`,
		`$..book[2]`,
		`$..book[2].author`,
		`$..book[2].publisher`,
		`$..book[-1]`,
		`$..book[0,1]`,
		`$..book[:2]`,
		`$..book[?@.isbn]`,
		`$..book[?@.price<10]`,
		`$..*`,
	}

	for _, example := range examples {
		t.Run(example, func(t *testing.T) {
			query, err := Parse(example)
			if err != nil {
				t.Errorf("Parse(%q) failed: %v", example, err)
				return
			}
			if query == nil {
				t.Errorf("Parse(%q) returned nil query", example)
			}
		})
	}
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
