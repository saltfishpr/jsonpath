package jsonpath

import (
	"testing"
)

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
