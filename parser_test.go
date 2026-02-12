package jsonpath

import (
	"testing"
)

// TestBasicSyntax 测试基础语法
// RFC 9535 Section 2.1: 每个查询必须以根标识符 $ 开始
func TestBasicSyntax(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *Query)
	}{
		{
			name:    "根标识符 - 只有 $",
			input:   "$",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				if len(q.Segments) != 0 {
					t.Errorf("expected 0 segments, got %d", len(q.Segments))
				}
			},
		},
		{
			name:    "根标识符 - 空格后结束",
			input:   "$   ",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				if len(q.Segments) != 0 {
					t.Errorf("expected 0 segments, got %d", len(q.Segments))
				}
			},
		},
		{
			name:    "不以 $ 开始 - 错误",
			input:   "foo",
			wantErr: true,
		},
		{
			name:    "空字符串 - 错误",
			input:   "",
			wantErr: true,
		},
		{
			name:    "只有空格 - 错误",
			input:   "   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, q)
			}
		})
	}
}

// TestNameSelector 测试名称选择器
// RFC 9535 Section 2.3.1: 名称选择器 'name' 或 "name"
func TestNameSelector(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *Query)
	}{
		// 点表示法
		{
			name:    "点表示法 - 简单名称",
			input:   "$.foo",
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
				sel := seg.Selectors[0]
				if sel.Type != NameSelector {
					t.Errorf("expected NameSelector, got %v", sel.Type)
				}
				if sel.Name != "foo" {
					t.Errorf("expected name 'foo', got '%s'", sel.Name)
				}
			},
		},
		{
			name:    "点表示法 - 多个段",
			input:   "$.foo.bar.baz",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				if len(q.Segments) != 3 {
					t.Fatalf("expected 3 segments, got %d", len(q.Segments))
				}
			},
		},
		{
			name:    "点表示法 - 名称包含下划线",
			input:   "$.foo_bar",
			wantErr: false,
		},
		{
			name:    "点表示法 - 名称以大写字母开头",
			input:   "$.Foo",
			wantErr: false,
		},
		{
			name:    "点表示法 - 保留字 null 作为成员名",
			input:   "$.null",
			wantErr: false,
		},
		{
			name:    "点表示法 - 保留字 true 作为成员名",
			input:   "$.true",
			wantErr: false,
		},
		{
			name:    "点表示法 - 保留字 false 作为成员名",
			input:   "$.false",
			wantErr: false,
		},
		// 括号表示法 - 单引号
		{
			name:    "括号表示法 - 单引号简单名称",
			input:   "$['foo']",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Name != "foo" {
					t.Errorf("expected name 'foo', got '%s'", sel.Name)
				}
			},
		},
		{
			name:    "括号表示法 - 单引号名称包含空格",
			input:   "$['foo bar']",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Name != "foo bar" {
					t.Errorf("expected name 'foo bar', got '%s'", sel.Name)
				}
			},
		},
		{
			name:    "括号表示法 - 单引号名称包含点",
			input:   "$['foo.bar']",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Name != "foo.bar" {
					t.Errorf("expected name 'foo.bar', got '%s'", sel.Name)
				}
			},
		},
		{
			name:    "括号表示法 - 单引号特殊字符",
			input:   "$['@']",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Name != "@" {
					t.Errorf("expected name '@', got '%s'", sel.Name)
				}
			},
		},
		// 括号表示法 - 双引号
		{
			name:    "括号表示法 - 双引号简单名称",
			input:   `$["foo"]`,
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Name != "foo" {
					t.Errorf("expected name 'foo', got '%s'", sel.Name)
				}
			},
		},
		{
			name:    "括号表示法 - 双引号名称包含单引号",
			input:   `$["foo's bar"]`,
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Name != "foo's bar" {
					t.Errorf("expected name \"foo's bar\", got '%s'", sel.Name)
				}
			},
		},
		// 转义序列
		{
			name:    "转义序列 - 反斜杠",
			input:   "$['foo\\\\bar']",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Name != "foo\\bar" {
					t.Errorf("expected name 'foo\\bar', got '%s'", sel.Name)
				}
			},
		},
		{
			name:    "转义序列 - 单引号转义",
			input:   `$["foo\"bar"]`,
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Name != `foo"bar` {
					t.Errorf(`expected name 'foo"bar', got '%s'`, sel.Name)
				}
			},
		},
		{
			name:    "转义序列 - \\b 退格",
			input:   "$['foo\\bbar']",
			wantErr: false,
		},
		{
			name:    "转义序列 - \\f 换页",
			input:   "$['foo\\fbar']",
			wantErr: false,
		},
		{
			name:    "转义序列 - \\n 换行",
			input:   "$['foo\\nbar']",
			wantErr: false,
		},
		{
			name:    "转义序列 - \\r 回车",
			input:   "$['foo\\rbar']",
			wantErr: false,
		},
		{
			name:    "转义序列 - \\t 制表符",
			input:   "$['foo\\tbar']",
			wantErr: false,
		},
		{
			name:    "转义序列 - \\u Unicode",
			input:   "$['foo\\u0020bar']",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Name != "foo bar" {
					t.Errorf("expected name 'foo bar', got '%s'", sel.Name)
				}
			},
		},
		// 多个选择器
		{
			name:    "括号表示法 - 多个名称选择器",
			input:   "$['foo','bar','baz']",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				if len(q.Segments[0].Selectors) != 3 {
					t.Fatalf("expected 3 selectors, got %d", len(q.Segments[0].Selectors))
				}
			},
		},
		{
			name:    "括号表示法 - 多个选择器带空格",
			input:   "$['foo' , 'bar' , 'baz']",
			wantErr: false,
		},
		// 错误情况
		{
			name:    "错误 - 未闭合的字符串",
			input:   "$['foo",
			wantErr: true,
		},
		{
			name:    "错误 - 未闭合的括号",
			input:   "$['foo'",
			wantErr: true,
		},
		{
			name:    "错误 - 空名称选择器",
			input:   "$[]",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, q)
			}
		})
	}
}

// TestWildcardSelector 测试通配符选择器
// RFC 9535 Section 2.3.2: 通配符选择器 *
func TestWildcardSelector(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *Query)
	}{
		{
			name:    "点表示法 - 通配符",
			input:   "$.*",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Type != WildcardSelector {
					t.Errorf("expected WildcardSelector, got %v", sel.Type)
				}
			},
		},
		{
			name:    "括号表示法 - 通配符",
			input:   "$[*]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Type != WildcardSelector {
					t.Errorf("expected WildcardSelector, got %v", sel.Type)
				}
			},
		},
		{
			name:    "多个通配符选择器",
			input:   "$[*,*]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				if len(q.Segments[0].Selectors) != 2 {
					t.Fatalf("expected 2 selectors, got %d", len(q.Segments[0].Selectors))
				}
			},
		},
		{
			name:    "混合选择器 - 名称和通配符",
			input:   "$['foo',*]",
			wantErr: false,
		},
		{
			name:    "混合选择器 - 通配符和名称",
			input:   "$[*, 'foo']",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, q)
			}
		})
	}
}

// TestIndexSelector 测试索引选择器
// RFC 9535 Section 2.3.3: 索引选择器 - 支持负索引
func TestIndexSelector(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *Query)
	}{
		{
			name:    "非负索引 - 0",
			input:   "$[0]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Type != IndexSelector {
					t.Errorf("expected IndexSelector, got %v", sel.Type)
				}
				if sel.Index != 0 {
					t.Errorf("expected index 0, got %d", sel.Index)
				}
			},
		},
		{
			name:    "非负索引 - 正数",
			input:   "$[42]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Index != 42 {
					t.Errorf("expected index 42, got %d", sel.Index)
				}
			},
		},
		{
			name:    "负索引 - -1",
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
			name:    "负索引 - -100",
			input:   "$[-100]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Index != -100 {
					t.Errorf("expected index -100, got %d", sel.Index)
				}
			},
		},
		{
			name:    "多个索引选择器",
			input:   "$[0,1,2]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				if len(q.Segments[0].Selectors) != 3 {
					t.Fatalf("expected 3 selectors, got %d", len(q.Segments[0].Selectors))
				}
			},
		},
		{
			name:    "混合选择器 - 索引和名称",
			input:   "$[0,'foo',-1]",
			wantErr: false,
		},
		// 错误情况
		{
			name:    "错误 - 前导零",
			input:   "$[01]",
			wantErr: true,
		},
		{
			name:    "错误 - 负号后跟前导零",
			input:   "$[-01]",
			wantErr: true,
		},
		{
			name:    "错误 - 只有负号",
			input:   "$[-]",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, q)
			}
		})
	}
}

// TestSliceSelector 测试数组切片选择器
// RFC 9535 Section 2.3.4: 数组切片选择器 start:end:step
func TestSliceSelector(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *Query)
	}{
		{
			name:    "完整切片 - start:end:step",
			input:   "$[1:5:2]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Type != SliceSelector {
					t.Errorf("expected SliceSelector, got %v", sel.Type)
				}
				if sel.Slice.Start == nil || *sel.Slice.Start != 1 {
					t.Errorf("expected start 1, got %v", sel.Slice.Start)
				}
				if sel.Slice.End == nil || *sel.Slice.End != 5 {
					t.Errorf("expected end 5, got %v", sel.Slice.End)
				}
				if sel.Slice.Step == nil || *sel.Slice.Step != 2 {
					t.Errorf("expected step 2, got %v", sel.Slice.Step)
				}
			},
		},
		{
			name:    "只有冒号 - 默认值",
			input:   "$[:]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Slice.Start != nil {
					t.Errorf("expected nil start, got %v", sel.Slice.Start)
				}
				if sel.Slice.End != nil {
					t.Errorf("expected nil end, got %v", sel.Slice.End)
				}
				if sel.Slice.Step != nil {
					t.Errorf("expected nil step, got %v", sel.Slice.Step)
				}
			},
		},
		{
			name:    "只有 start",
			input:   "$[1:]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Slice.Start == nil || *sel.Slice.Start != 1 {
					t.Errorf("expected start 1, got %v", sel.Slice.Start)
				}
				if sel.Slice.End != nil {
					t.Errorf("expected nil end, got %v", sel.Slice.End)
				}
			},
		},
		{
			name:    "只有 end",
			input:   "$[:5]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Slice.Start != nil {
					t.Errorf("expected nil start, got %v", sel.Slice.Start)
				}
				if sel.Slice.End == nil || *sel.Slice.End != 5 {
					t.Errorf("expected end 5, got %v", sel.Slice.End)
				}
			},
		},
		{
			name:    "start:end",
			input:   "$[1:5]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Slice.Step != nil {
					t.Errorf("expected nil step, got %v", sel.Slice.Step)
				}
			},
		},
		{
			name:    "start:end: 带 step",
			input:   "$[0:10:2]",
			wantErr: false,
		},
		{
			name:    "负数 start",
			input:   "$[-3:]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Slice.Start == nil || *sel.Slice.Start != -3 {
					t.Errorf("expected start -3, got %v", sel.Slice.Start)
				}
			},
		},
		{
			name:    "负数 end",
			input:   "$[:-3]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Slice.End == nil || *sel.Slice.End != -3 {
					t.Errorf("expected end -3, got %v", sel.Slice.End)
				}
			},
		},
		{
			name:    "负数 step",
			input:   "$[5:1:-2]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Slice.Step == nil || *sel.Slice.Step != -2 {
					t.Errorf("expected step -2, got %v", sel.Slice.Step)
				}
			},
		},
		{
			name:    "反向切片 - 只有 step",
			input:   "$[::-1]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Slice.Step == nil || *sel.Slice.Step != -1 {
					t.Errorf("expected step -1, got %v", sel.Slice.Step)
				}
			},
		},
		{
			name:    "带空格",
			input:   "$[ 1 : 5 : 2 ]",
			wantErr: false,
		},
		// 错误情况
		{
			name:    "错误 - 前导零",
			input:   "$[01:05:02]",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, q)
			}
		})
	}
}

// TestFilterSelector 测试过滤器选择器
// RFC 9535 Section 2.3.5: 过滤器选择器 ?<logical-expr>
func TestFilterSelector(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *Query)
	}{
		// 存在性测试
		{
			name:    "存在性测试 - 相对查询",
			input:   "$[?@.foo]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Type != FilterSelector {
					t.Errorf("expected FilterSelector, got %v", sel.Type)
				}
				if sel.Filter.Type != FilterTest {
					t.Errorf("expected FilterTest, got %v", sel.Filter.Type)
				}
				if sel.Filter.Test.FilterQuery == nil {
					t.Error("expected FilterQuery")
				}
			},
		},
		{
			name:    "存在性测试 - 嵌套路径",
			input:   "$[?@.foo.bar]",
			wantErr: false,
		},
		{
			name:    "存在性测试 - 绝对查询",
			input:   "$[?$.foo.bar]",
			wantErr: false,
		},
		{
			name:    "存在性测试 - 带括号",
			input:   "$[?( @ . foo )]",
			wantErr: false,
		},
		// 比较表达式
		{
			name:    "比较 - 等于",
			input:   "$[?@.foo == 42]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Filter.Type != FilterComparison {
					t.Errorf("expected FilterComparison, got %v", sel.Filter.Type)
				}
				if sel.Filter.Comp.Op != CompEq {
					t.Errorf("expected CompEq, got %v", sel.Filter.Comp.Op)
				}
			},
		},
		{
			name:    "比较 - 不等于",
			input:   "$[?@.foo != 42]",
			wantErr: false,
		},
		{
			name:    "比较 - 小于",
			input:   "$[?@.foo < 42]",
			wantErr: false,
		},
		{
			name:    "比较 - 小于等于",
			input:   "$[?@.foo <= 42]",
			wantErr: false,
		},
		{
			name:    "比较 - 大于",
			input:   "$[?@.foo > 42]",
			wantErr: false,
		},
		{
			name:    "比较 - 大于等于",
			input:   "$[?@.foo >= 42]",
			wantErr: false,
		},
		// 字面量比较
		{
			name:    "比较 - 数字字面量",
			input:   "$[?@.age > 18]",
			wantErr: false,
		},
		{
			name:    "比较 - 字符串字面量单引号",
			input:   "$[?@.name == 'John']",
			wantErr: false,
		},
		{
			name:    "比较 - 字符串字面量双引号",
			input:   `$[?@.name == "John"]`,
			wantErr: false,
		},
		{
			name:    "比较 - true 字面量",
			input:   "$[?@.active == true]",
			wantErr: false,
		},
		{
			name:    "比较 - false 字面量",
			input:   "$[?@.active == false]",
			wantErr: false,
		},
		{
			name:    "比较 - null 字面量",
			input:   "$[?@.value == null]",
			wantErr: false,
		},
		// 逻辑运算符
		{
			name:    "逻辑与",
			input:   "$[?@.age > 18 && @.age < 65]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Filter.Type != FilterLogicalAnd {
					t.Errorf("expected FilterLogicalAnd, got %v", sel.Filter.Type)
				}
			},
		},
		{
			name:    "逻辑或",
			input:   "$[?@.type == 'A' || @.type == 'B']",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Filter.Type != FilterLogicalOr {
					t.Errorf("expected FilterLogicalOr, got %v", sel.Filter.Type)
				}
			},
		},
		{
			name:    "逻辑非",
			input:   "$[?!@.foo]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Filter.Type != FilterLogicalNot {
					t.Errorf("expected FilterLogicalNot, got %v", sel.Filter.Type)
				}
			},
		},
		{
			name:    "逻辑非 - 比较表达式",
			input:   "$[?!(@.x == 1)]",
			wantErr: false,
		},
		// 括号分组
		{
			name:    "括号 - 简单分组",
			input:   "$[?( @.x == 1 )]",
			wantErr: false,
		},
		{
			name:    "括号 - 复杂逻辑",
			input:   "$[?( @.x == 1 || @.y == 2 ) && @.z == 3]",
			wantErr: false,
		},
		// 运算符优先级
		{
			name:    "优先级 - 括号改变优先级",
			input:   "$[?@.x && ( @.y || @.z )]",
			wantErr: false,
		},
		// 单值查询
		{
			name:    "单值查询 - 嵌套索引",
			input:   "$[?@.items[0].price > 100]",
			wantErr: false,
		},
		{
			name:    "单值查询 - 相对",
			input:   "$[?@.foo[0].bar == 'test']",
			wantErr: false,
		},
		// 函数调用（作为存在性测试或比较）
		{
			name:    "函数 - length",
			input:   "$[?length(@.name) > 5]",
			wantErr: false,
		},
		{
			name:    "函数 - count",
			input:   "$[?count(@.*) > 0]",
			wantErr: false,
		},
		{
			name:    "函数 - match",
			input:   `$[?match(@.name, "^[A-Z")]`,
			wantErr: false,
		},
		{
			name:    "函数 - search",
			input:   `$[?search(@.name, "abc")]`,
			wantErr: false,
		},
		{
			name:    "函数 - value",
			input:   `$[?value(@..x) == 42]`,
			wantErr: false,
		},
		// 嵌套过滤器
		{
			name:    "嵌套过滤器",
			input:   "$[?@[?@.x]]",
			wantErr: false,
		},
		// 错误情况
		{
			name:    "错误 - 未闭合的括号",
			input:   "$[?@.foo == 42",
			wantErr: true,
		},
		{
			name:    "错误 - 空的过滤器",
			input:   "$[?]",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, q)
			}
		})
	}
}

// TestDescendantSegment 测试后代段
// RFC 9535 Section 2.5.2: 后代段 ..name 或 ..[*]
func TestDescendantSegment(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *Query)
	}{
		{
			name:    "后代段 - 点表示法",
			input:   "$..foo",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				if len(q.Segments) != 1 {
					t.Fatalf("expected 1 segment, got %d", len(q.Segments))
				}
				seg := q.Segments[0]
				if seg.Type != DescendantSegment {
					t.Errorf("expected DescendantSegment, got %v", seg.Type)
				}
			},
		},
		{
			name:    "后代段 - 通配符点表示法",
			input:   "$..*",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				seg := q.Segments[0]
				if seg.Type != DescendantSegment {
					t.Errorf("expected DescendantSegment, got %v", seg.Type)
				}
			},
		},
		{
			name:    "后代段 - 括号表示法名称",
			input:   "$..['foo']",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				seg := q.Segments[0]
				if seg.Type != DescendantSegment {
					t.Errorf("expected DescendantSegment, got %v", seg.Type)
				}
			},
		},
		{
			name:    "后代段 - 括号表示法通配符",
			input:   "$..[*]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				seg := q.Segments[0]
				if seg.Type != DescendantSegment {
					t.Errorf("expected DescendantSegment, got %v", seg.Type)
				}
			},
		},
		{
			name:    "后代段 - 索引",
			input:   "$..[0]",
			wantErr: false,
		},
		{
			name:    "后代段 - 切片",
			input:   "$..[0:5]",
			wantErr: false,
		},
		{
			name:    "后代段 - 多个选择器",
			input:   "$..['foo','bar']",
			wantErr: false,
		},
		{
			name:    "后代段 - 过滤器",
			input:   "$..[?@.x > 0]",
			wantErr: false,
		},
		{
			name:    "多个后代段",
			input:   "$..foo..bar",
			wantErr: false,
		},
		{
			name:    "子段和后代段混合",
			input:   "$.store..book..title",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				if len(q.Segments) != 3 {
					t.Fatalf("expected 3 segments, got %d", len(q.Segments))
				}
				if q.Segments[0].Type != ChildSegment {
					t.Errorf("expected first segment to be ChildSegment")
				}
				if q.Segments[1].Type != DescendantSegment {
					t.Errorf("expected second segment to be DescendantSegment")
				}
				if q.Segments[2].Type != DescendantSegment {
					t.Errorf("expected third segment to be DescendantSegment")
				}
			},
		},
		// 错误情况
		{
			name:    "错误 - 只有 .. 无选择器",
			input:   "$..",
			wantErr: true,
		},
		{
			name:    "错误 - .. 后面跟着 [ 无内容",
			input:   "$..[]",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, q)
			}
		})
	}
}

// TestFunctionExpression 测试函数表达式
// RFC 9535 Section 2.4: 函数扩展
func TestFunctionExpression(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *Query)
	}{
		{
			name:    "函数 - 无参数",
			input:   "$[?func()]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				fn := sel.Filter.Test.FuncExpr
				if fn == nil {
					t.Fatal("expected function expression")
				}
				if fn.Name != "func" {
					t.Errorf("expected function name 'func', got '%s'", fn.Name)
				}
			},
		},
		{
			name:    "函数 - 单个参数字面量",
			input:   "$[?func(42)]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				fn := sel.Filter.Test.FuncExpr
				if len(fn.Args) != 1 {
					t.Fatalf("expected 1 argument, got %d", len(fn.Args))
				}
			},
		},
		{
			name:    "函数 - 多个参数",
			input:   "$[?func(1, 'two', true)]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				fn := sel.Filter.Test.FuncExpr
				if len(fn.Args) != 3 {
					t.Fatalf("expected 3 arguments, got %d", len(fn.Args))
				}
			},
		},
		{
			name:    "函数 - 过滤器查询参数",
			input:   "$[?func(@.x)]",
			wantErr: false,
		},
		{
			name:    "函数 - 逻辑表达式参数",
			input:   "$[?func(@.x == 1)]",
			wantErr: false,
		},
		{
			name:    "函数 - 嵌套函数调用",
			input:   "$[?func1(func2(@.x))]",
			wantErr: false,
		},
		{
			name:    "函数 - 带空格",
			input:   "$[?func( 1 , 2 )]",
			wantErr: false,
		},
		// 标准函数
		{
			name:    "length 函数",
			input:   "$[?length(@.items) >= 3]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				sel := q.Segments[0].Selectors[0]
				if sel.Type != FilterSelector {
					t.Errorf("expected FilterSelector, got %v", sel.Type)
				}
				if sel.Filter.Type != FilterComparison {
					t.Errorf("expected FilterComparison, got %v", sel.Filter.Type)
				}
				comp := sel.Filter.Comp
				if comp.Op != CompGe {
					t.Errorf("expected CompGe, got %v", comp.Op)
				}
				if comp.Left.Type != ComparableFuncExpr {
					t.Errorf("expected left to be ComparableFuncExpr, got %v", comp.Left.Type)
				}
				fn := comp.Left.FuncExpr
				if fn.Name != "length" {
					t.Errorf("expected function name 'length', got '%s'", fn.Name)
				}
				if len(fn.Args) != 1 {
					t.Errorf("expected 1 argument, got %d", len(fn.Args))
				}
				{
					fnArg := fn.Args[0]
					if fnArg.Type != FuncArgFilterQuery {
						t.Errorf("expected FuncArgFilterQuery, got %v", fnArg.Type)
					}
					fq := fnArg.FilterQuery
					if !fq.Relative {
						t.Error("expected relative query (starting with @)")
					}
					if len(fq.Segments) != 1 {
						t.Errorf("expected 1 segment in filter query, got %d", len(fq.Segments))
					}
					seg := fq.Segments[0]
					if len(seg.Selectors) != 1 {
						t.Errorf("expected 1 selector, got %d", len(seg.Selectors))
					}
					sel := seg.Selectors[0]
					if sel.Type != NameSelector {
						t.Errorf("expected NameSelector, got %v", sel.Type)
					}

					if sel.Name != "items" {
						t.Errorf("expected name 'items', got '%s'", sel.Name)
					}
				}
				if comp.Right.Type != ComparableLiteral {
					t.Errorf("expected right to be ComparableLiteral, got %v", comp.Right.Type)
				}
				lit := comp.Right.Literal
				if lit.Type != LiteralNumber {
					t.Errorf("expected LiteralNumber, got %v", lit.Type)
				}
				if lit.Value != "3" {
					t.Errorf("expected value '3', got '%s'", lit.Value)
				}
			},
		},
		{
			name:    "count 函数",
			input:   "$[?count(@..price) == 5]",
			wantErr: false,
		},
		{
			name:    "match 函数",
			input:   `$[?match(@.date, "1974-05-..")]`,
			wantErr: false,
		},
		{
			name:    "search 函数",
			input:   `$[?search(@.author, "[BR]ob")]`,
			wantErr: false,
		},
		{
			name:    "value 函数",
			input:   `$[?value(@..color) == "red"]`,
			wantErr: false,
		},
		// 错误情况
		{
			name:    "错误 - 未闭合的括号",
			input:   "$[?func(]",
			wantErr: true,
		},
		{
			name:    "错误 - 大写函数名",
			input:   "$[?Func()]",
			wantErr: true,
		},
		{
			name:    "错误 - 函数名包含连字符",
			input:   "$[?func-name()]",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, q)
			}
		})
	}
}

// TestParserRFCExamples 测试 RFC 9535 中的示例表达式解析
// Table 2: Example JSONPath Expressions
func TestParserRFCExamples(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:  "示例 - $.store.book[*].author",
			input: "$.store.book[*].author",
		},
		{
			name:  "示例 - $..author",
			input: "$..author",
		},
		{
			name:  "示例 - $.store.*",
			input: "$.store.*",
		},
		{
			name:  "示例 - $.store..price",
			input: "$.store..price",
		},
		{
			name:  "示例 - $..book[2]",
			input: "$..book[2]",
		},
		{
			name:  "示例 - $..book[2].author",
			input: "$..book[2].author",
		},
		{
			name:  "示例 - $..book[2].publisher",
			input: "$..book[2].publisher",
		},
		{
			name:  "示例 - $..book[-1]",
			input: "$..book[-1]",
		},
		{
			name:  "示例 - $..book[0,1]",
			input: "$..book[0,1]",
		},
		{
			name:  "示例 - $..book[:2]",
			input: "$..book[:2]",
		},
		{
			name:  "示例 - $..book[?@.isbn]",
			input: "$..book[?@.isbn]",
		},
		{
			name:  "示例 - $..book[?@.price<10]",
			input: "$..book[?@.price<10]",
		},
		{
			name:  "示例 - $..*",
			input: "$..*",
		},
		// Table 5: Name Selector Examples
		{
			name:  "名称选择器 - $.o['j j']",
			input: "$.o['j j']",
		},
		{
			name:  "名称选择器 - $.o['j j']['k.k']",
			input: "$.o['j j']['k.k']",
		},
		{
			name:  "名称选择器 - $.o[\"j j\"][\"k.k\"]",
			input: `$.o["j j"]["k.k"]`,
		},
		{
			name:  "名称选择器 - $[\"'\"][\"@\"]",
			input: `$["'"]["@"]`,
		},
		// Table 9: Array Slice Selector Examples
		{
			name:  "切片 - $[1:3]",
			input: "$[1:3]",
		},
		{
			name:  "切片 - $[5:]",
			input: "$[5:]",
		},
		{
			name:  "切片 - $[1:5:2]",
			input: "$[1:5:2]",
		},
		{
			name:  "切片 - $[5:1:-2]",
			input: "$[5:1:-2]",
		},
		{
			name:  "切片 - $[::-1]",
			input: "$[::-1]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// TestMixedSelectors 测试混合选择器
func TestMixedSelectors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *Query)
	}{
		{
			name:    "名称和索引混合",
			input:   "$['foo',0,'bar',1]",
			wantErr: false,
			check: func(t *testing.T, q *Query) {
				if len(q.Segments[0].Selectors) != 4 {
					t.Fatalf("expected 4 selectors, got %d", len(q.Segments[0].Selectors))
				}
			},
		},
		{
			name:    "名称和切片混合",
			input:   "$['foo',0:5,'bar']",
			wantErr: false,
		},
		{
			name:    "通配符和索引混合",
			input:   "$[*,0,*]",
			wantErr: false,
		},
		{
			name:    "多种选择器混合",
			input:   "$['name',*,0:5,-1,?@.x]",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, q)
			}
		})
	}
}

// TestComplexQueries 测试复杂查询
func TestComplexQueries(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:  "深度嵌套路径",
			input: "$.a.b.c.d.e.f.g",
		},
		{
			name:  "多个连续切片",
			input: "$[0:5][1:3]",
		},
		{
			name:  "复杂过滤器",
			input: `$[?(@.x > 0 && @.y < 10) || (@.z == "test")]`,
		},
		{
			name:  "嵌套过滤器",
			input: "$.items[?@.values[?@.active == true]]",
		},
		{
			name:  "后代段带过滤器",
			input: "$..items[?@.count > 0]",
		},
		{
			name:  "复杂函数调用",
			input: `$[?length(@.name) > 5 && match(@.type, "^[A-Z]")]`,
		},
		{
			name:  "单值查询在比较中",
			input: "$.items[?@.value == $.max]",
		},
		{
			name:  "多个后代段",
			input: "$..a..b..c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// TestErrorCases 测试错误情况
func TestErrorCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "错误 - 不以 $ 开始",
			input:   "foo.bar",
			wantErr: true,
		},
		{
			name:    "错误 - 未闭合的括号",
			input:   "$.foo[",
			wantErr: true,
		},
		{
			name:    "错误 - 多余的右括号",
			input:   "$.foo]",
			wantErr: true,
		},
		{
			name:    "错误 - 未闭合的圆括号",
			input:   "$[?(@.x == 1]",
			wantErr: true,
		},
		{
			name:    "错误 - 无效的运算符",
			input:   "$[?@.x === 1]",
			wantErr: true,
		},
		{
			name:    "错误 - 单个 =",
			input:   "$[?@.x = 1]",
			wantErr: true,
		},
		{
			name:    "错误 - 单个 &",
			input:   "$[?@.x & 1]",
			wantErr: true,
		},
		{
			name:    "错误 - 单个 |",
			input:   "$[?@.x | 1]",
			wantErr: true,
		},
		{
			name:    "错误 - 无效的转义",
			input:   "$['foo\\x']",
			wantErr: true,
		},
		{
			name:    "错误 - 无效的 Unicode 转义",
			input:   "$['foo\\uGGGG']",
			wantErr: true,
		},
		{
			name:    "错误 - . 后面跟着 [",
			input:   "$.[foo]",
			wantErr: true,
		},
		{
			name:    "错误 - . 后面跟着数字",
			input:   "$.0",
			wantErr: true,
		},
		{
			name:    "错误 - . 后面跟着 *",
			input:   "$.*", // 这个应该是合法的
			wantErr: false,
		},
		{
			name:    "错误 - 空字符串",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// TestWhitespace 测试空白字符处理
func TestWhitespace(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:  "空格",
			input: "$ . foo . bar",
		},
		{
			name:  "制表符",
			input: "$\t.\tfoo\t.\tbar",
		},
		{
			name:  "换行符",
			input: "$\n.\nfoo\n.\nbar",
		},
		{
			name:  "回车符",
			input: "$\r.\rfoo\r.\rbar",
		},
		{
			name:  "混合空白",
			input: "$ \t\n.\r foo \t\n",
		},
		{
			name:  "括号内空白",
			input: "$[ 'foo' , 'bar' ]",
		},
		{
			name:  "过滤器内空白",
			input: "$[ ? @.x == 1 && @.y == 2 ]",
		},
		{
			name:  "函数参数空白",
			input: "$[?func( 1 , 'two' , @.x )]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// TestNumericLiterals 测试数字字面量
func TestNumericLiterals(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *Query)
	}{
		{
			name:    "整数",
			input:   "$[?@.x == 42]",
			wantErr: false,
		},
		{
			name:    "负整数",
			input:   "$[?@.x == -42]",
			wantErr: false,
		},
		{
			name:    "零",
			input:   "$[?@.x == 0]",
			wantErr: false,
		},
		{
			name:    "负零",
			input:   "$[?@.x == -0]",
			wantErr: false,
		},
		{
			name:    "小数",
			input:   "$[?@.x == 3.14]",
			wantErr: false,
		},
		{
			name:    "科学计数法 - 小写 e",
			input:   "$[?@.x == 1e10]",
			wantErr: false,
		},
		{
			name:    "科学计数法 - 大写 E",
			input:   "$[?@.x == 1E10]",
			wantErr: false,
		},
		{
			name:    "科学计数法 - 负指数",
			input:   "$[?@.x == 1e-10]",
			wantErr: false,
		},
		{
			name:    "科学计数法 - 正指数",
			input:   "$[?@.x == 1e+10]",
			wantErr: false,
		},
		{
			name:    "组合 - 小数加科学计数法",
			input:   "$[?@.x == 1.5e10]",
			wantErr: false,
		},
		// 错误情况
		{
			name:    "错误 - 只有点",
			input:   "$[?@.x == .]",
			wantErr: true,
		},
		{
			name:    "错误 - 前导零（索引）",
			input:   "$[01]",
			wantErr: true,
		},
		{
			name:    "错误 - 前导零（切片）",
			input:   "$[01:05:02]",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, q)
			}
		})
	}
}

// TestSingularQuery 测试单值查询
func TestSingularQuery(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, *Query)
	}{
		{
			name:    "相对单值查询 - 简单",
			input:   "$[?@.x == @.y]",
			wantErr: false,
		},
		{
			name:    "相对单值查询 - 嵌套",
			input:   "$[?@.x == @.a.b.c]",
			wantErr: false,
		},
		{
			name:    "相对单值查询 - 带索引",
			input:   "$[?@.x == @.items[0]]",
			wantErr: false,
		},
		{
			name:    "绝对单值查询",
			input:   "$[?@.x == $.max.value]",
			wantErr: false,
		},
		{
			name:    "括号表示法单值查询",
			input:   `$[?@.x == @['foo']['bar']]`,
			wantErr: false,
		},
		{
			name:    "混合表示法",
			input:   "$[?@.x == @.a['b'].c]",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, q)
			}
		})
	}
}
