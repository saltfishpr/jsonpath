package jsonpath

import (
	"testing"
)

// RFC 9535 示例 JSON
const rfcExampleJSON = `{
  "store": {
    "book": [
      {"category": "reference", "author": "Nigel Rees", "title": "Sayings of the Century", "price": 8.95},
      {"category": "fiction", "author": "Evelyn Waugh", "title": "Sword of Honour", "price": 12.99},
      {"category": "fiction", "author": "Herman Melville", "title": "Moby Dick", "isbn": "0-553-21311-3", "price": 8.99},
      {"category": "fiction", "author": "J. R. R. Tolkien", "title": "The Lord of the Rings", "isbn": "0-395-19395-8", "price": 22.99}
    ],
    "bicycle": {"color": "red", "price": 399}
  }
}`

func TestEvalNameSelector(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		query    string
		wantLen  int
		wantVal  string
		wantType JSONType
	}{
		{"根属性", `{"name":"value"}`, "$.name", 1, "value", JSONTypeString},
		{"嵌套属性", `{"a":{"b":"c"}}`, "$.a.b", 1, "c", JSONTypeString},
		{"缺失属性", `{"a":1}`, "$.b", 0, "", JSONTypeNull},
		{"数组后属性", `{"a":[{"b":1}]}`, "$.a[0].b", 1, "1", JSONTypeNumber},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Get(tt.json, tt.query)
			if tt.wantLen == 0 {
				if got.Exists() {
					t.Errorf("Get() should return empty, got %v", got)
				}
				return
			}
			if !got.Exists() {
				t.Errorf("Get() should exist, got empty")
				return
			}
			if got.Type != tt.wantType {
				t.Errorf("Get() Type = %v, want %v", got.Type, tt.wantType)
			}
			if got.String() != tt.wantVal {
				t.Errorf("Get() = %v, want %v", got.String(), tt.wantVal)
			}
		})
	}
}

func TestEvalWildcardSelector(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		query   string
		wantLen int
	}{
		{"对象所有成员", `{"a":1,"b":2}`, "$.*", 2},
		{"数组所有元素", `[1,2,3]`, "$[*]", 3},
		{"嵌套数组", `{"a":[1,2]}`, "$.a[*]", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMany(tt.json, tt.query)
			if len(got) != tt.wantLen {
				t.Errorf("GetMany() len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestEvalIndexSelector(t *testing.T) {
	json := `[1,2,3,4,5]`
	tests := []struct {
		name    string
		query   string
		wantVal string
	}{
		{"正索引", "$[0]", "1"},
		{"正索引中间", "$[2]", "3"},
		{"负索引", "$[-1]", "5"},
		{"负索引中间", "$[-2]", "4"},
		{"超出范围正", "$[10]", ""},
		{"超出范围负", "$[-10]", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Get(json, tt.query)
			if tt.wantVal == "" {
				if got.Exists() {
					t.Errorf("Get() should return empty, got %v", got)
				}
			} else {
				if !got.Exists() {
					t.Errorf("Get() should exist, got empty")
				} else if got.String() != tt.wantVal {
					t.Errorf("Get() = %v, want %v", got.String(), tt.wantVal)
				}
			}
		})
	}
}

func TestEvalSliceSelector(t *testing.T) {
	json := `["a","b","c","d","e","f","g"]`
	tests := []struct {
		name     string
		query    string
		wantVals []string
	}{
		{"基本切片", "$[1:3]", []string{"b", "c"}},
		{"从开头", "$[:3]", []string{"a", "b", "c"}},
		{"到结尾", "$[5:]", []string{"f", "g"}},
		{"全部", "$[:]", []string{"a", "b", "c", "d", "e", "f", "g"}},
		{"带步长", "$[1:5:2]", []string{"b", "d"}},
		{"负步长", "$[5:1:-2]", []string{"f", "d"}},
		{"反转", "$[::-1]", []string{"g", "f", "e", "d", "c", "b", "a"}},
		{"负开始", "$[-2:]", []string{"f", "g"}},
		{"负结束", "$[:-2]", []string{"a", "b", "c", "d", "e"}},
		{"step=0", "$[::0]", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMany(json, tt.query)
			if tt.wantVals == nil {
				if len(got) != 0 {
					t.Errorf("GetMany() len = %d, want 0", len(got))
				}
				return
			}
			if len(got) != len(tt.wantVals) {
				t.Errorf("GetMany() len = %d, want %d", len(got), len(tt.wantVals))
				return
			}
			for i, want := range tt.wantVals {
				if got[i].String() != want {
					t.Errorf("GetMany()[%d] = %v, want %v", i, got[i].String(), want)
				}
			}
		})
	}
}

func TestEvalFilterSelector(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantLen int
	}{
		{"价格小于10", "$.store.book[?@.price < 10]", 2},
		{"有 ISBN", "$.store.book[?@.isbn]", 2},
		{"分类为 fiction", "$.store.book[?@.category == 'fiction']", 3},
		{"价格小于10且fiction", "$.store.book[?@.price < 10 && @.category == 'fiction']", 1},
		{"价格小于10或大于20", "$.store.book[?@.price < 10 || @.price > 20]", 3},
		{"存在 title", "$.store.book[?@.title]", 4},
		{"不存在字段", "$.store.book[?@.missing]", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMany(rfcExampleJSON, tt.query)
			if len(got) != tt.wantLen {
				t.Errorf("GetMany() len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestEvalDescendantSegment(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantLen int
	}{
		{"所有作者", "$..author", 4},
		{"所有 price", "$..price", 5},
		{"所有 book", "$..book", 1},
		{"第三本书", "$..book[2]", 1},
		{"最后一本书", "$..book[-1]", 1},
		{"所有值", "$..*", 27}, // store, book, 4 books, bicycle, 2 bicycle fields, etc.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMany(rfcExampleJSON, tt.query)
			if len(got) != tt.wantLen {
				t.Errorf("GetMany() len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestRFC9535Examples(t *testing.T) {
	tests := []struct {
		query   string
		wantLen int
	}{
		{"$.store.book[*].author", 4},
		{"$..author", 4},
		{"$.store.*", 2},
		{"$..book[2]", 1},
		{"$..book[-1]", 1},
		{"$..book[0,1]", 2},
		{"$..book[:2]", 2},
		{"$..book[?@.isbn]", 2},
		{"$..book[?@.price < 10]", 2},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got := GetMany(rfcExampleJSON, tt.query)
			if len(got) != tt.wantLen {
				t.Errorf("GetMany(%q) len = %d, want %d", tt.query, len(got), tt.wantLen)
			}
		})
	}
}

// TestEvalLengthFunction 测试 length() 函数
func TestEvalLengthFunction(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		query   string
		wantLen int
	}{
		{"数组长度等于3", `["a","b","c"]`, "$[?length(@) == 3]", 1},
		{"数组长度大于2", `["a","b","c","d"]`, "$[?length(@) > 2]", 1},
		{"字符串长度大于5", `["short","longer string"]`, "$[?length(@) > 5]", 1},
		{"对象成员数等于2", `{"a":1,"b":2}`, "$[?length(@) == 2]", 1},
		{"嵌套数组长度", `{"arr":[1,2,3]}`, "$.arr[?length(@) == 3]", 3},
		{"空数组长度0", `[]`, "$[?length(@) == 0]", 0}, // 空数组没有元素可匹配
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMany(tt.json, tt.query)
			if len(got) != tt.wantLen {
				t.Errorf("GetMany(%q) len = %d, want %d", tt.query, len(got), tt.wantLen)
			}
		})
	}
}

// TestEvalCountFunction 测试 count() 函数
func TestEvalCountFunction(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		query   string
		wantLen int
	}{
		{"计数子节点", `{"a": {"x": 1, "y": 2}}`, "$[?count(@.*) == 2]", 1},
		{"计数大于1", `{"a": [1, 2, 3]}`, "$.a[?count(@.*) > 1]", 0}, // 数组元素不是对象
		{"计数数组元素", `{"arr": [1, 2]}`, "$[?count(@.arr[*]) == 2]", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMany(tt.json, tt.query)
			if len(got) != tt.wantLen {
				t.Errorf("GetMany(%q) len = %d, want %d", tt.query, len(got), tt.wantLen)
			}
		})
	}
}

// TestEvalMatchFunction 测试 match() 函数
func TestEvalMatchFunction(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		query   string
		wantLen int
	}{
		{"匹配日期格式", `["2024-01-01", "2024-13-01", "not-a-date"]`, "$[?match(@, '^\\d{4}-\\d{2}-\\d{2}$')]", 2},
		{"匹配邮箱", `["test@example.com", "invalid", "user@domain.org"]`, "$[?match(@, '^[^@]+@[^@]+$')]", 2},
		{"匹配开头", `["apple", "application", "banana"]`, "$[?match(@, '^app')]", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMany(tt.json, tt.query)
			if len(got) != tt.wantLen {
				t.Errorf("GetMany(%q) len = %d, want %d", tt.query, len(got), tt.wantLen)
			}
		})
	}
}

// TestEvalSearchFunction 测试 search() 函数
func TestEvalSearchFunction(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		query   string
		wantLen int
	}{
		{"搜索数字", `["abc123def", "abcdef", "123"]`, "$[?search(@, '\\d+')]", 2},
		{"搜索子串", `["hello world", "hello", "world"]`, "$[?search(@, 'world')]", 2},
		{"搜索模式", `["test@example.com", "example.org"]`, "$[?search(@, 'example')]", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMany(tt.json, tt.query)
			if len(got) != tt.wantLen {
				t.Errorf("GetMany(%q) len = %d, want %d", tt.query, len(got), tt.wantLen)
			}
		})
	}
}

// TestEvalValueFunction 测试 value() 函数
func TestEvalValueFunction(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		query   string
		wantVal string
	}{
		{"单节点取值", `{"a": [{"b": 1}]}`, "$[?value(@.a[0].b) == 1]", "a"},
		{"多节点返回Nothing", `{"a": [1, 2]}`, "$[?value(@.a[*]) == 1]", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMany(tt.json, tt.query)
			if tt.wantVal == "" {
				if len(got) != 0 {
					t.Errorf("GetMany(%q) should return empty, got %d results", tt.query, len(got))
				}
			} else {
				if len(got) == 0 {
					t.Errorf("GetMany(%q) should return results", tt.query)
				}
			}
		})
	}
}

// TestFunctionComplexQueries 测试复杂的函数查询
func TestFunctionComplexQueries(t *testing.T) {
	json := `{
		"users": [
			{"name": "Alice", "age": 30, "email": "alice@example.com"},
			{"name": "Bob", "age": 25, "email": "bob@test.org"},
			{"name": "Charlie", "age": 35, "email": "charlie@example.com"}
		]
	}`

	tests := []struct {
		name    string
		query   string
		wantLen int
	}{
		{"长度筛选", "$.users[?length(@.name) >= 4]", 2},
		{"年龄比较", "$.users[?length(@.name) > 3 && @.age > 30]", 1},
		{"邮箱匹配", "$.users[?match(@.email, '^[^@]+@example\\\\.com$')]", 2},
		{"邮箱搜索", "$.users[?search(@.email, 'example')]", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMany(json, tt.query)
			if len(got) != tt.wantLen {
				t.Errorf("GetMany(%q) len = %d, want %d\nResults: %v", tt.query, len(got), tt.wantLen, got)
			}
		})
	}
}
