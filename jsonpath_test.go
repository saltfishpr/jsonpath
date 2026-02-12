package jsonpath

import "testing"

func TestResult_Array(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantLen int
	}{
		{"空数组", "[]", 0},
		{"简单数组", `[1,2,3]`, 3},
		{"嵌套数组", `[[1,2],[3,4]]`, 2},
		{"字符串数组", `["a","b"]`, 2},
		{"混合数组", `[1,"a",null,true]`, 4},
		{"带空格", `[ 1 , 2 , 3 ]`, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := parseValue(tt.json)
			got := r.Array()
			if len(got) != tt.wantLen {
				t.Errorf("Array() len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestResult_Map(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantLen int
	}{
		{"空对象", `{}`, 0},
		{"简单对象", `{"a":1,"b":2}`, 2},
		{"嵌套对象", `{"a":{"b":1}}`, 1},
		{"带空格", `{ "a" : 1 , "b" : 2 }`, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := parseValue(tt.json)
			got := r.Map()
			if len(got) != tt.wantLen {
				t.Errorf("Map() len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}
