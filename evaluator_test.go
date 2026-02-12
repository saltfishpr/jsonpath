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
