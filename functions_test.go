package jsonpath

import (
	"testing"
)

// TestLengthFunction tests the length() function extension
func TestLengthFunction(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		query   string
		wantLen int // expected length of result
	}{
		{
			name:    "string length",
			json:    `["hello", "world", "!"]`,
			query:   `$[?length(@) == 5]`,
			wantLen: 2,
		},
		{
			name:    "array length",
			json:    `[[1, 2], [3, 4, 5], [6]]`,
			query:   `$[?length(@) > 2]`,
			wantLen: 1,
		},
		{
			name:    "object member count",
			json:    `[{"a": 1}, {"a": 1, "b": 2}, {"a": 1, "b": 2, "c": 3}]`,
			query:   `$[?length(@) == 2]`,
			wantLen: 1,
		},
		{
			name:    "number returns nothing",
			json:    `[1, 2, 3]`,
			query:   `$[?length(@) > 0]`,
			wantLen: 0,
		},
		{
			name:    "boolean returns nothing",
			json:    `[true, false]`,
			query:   `$[?length(@) > 0]`,
			wantLen: 0,
		},
		{
			name:    "null returns nothing",
			json:    `[null]`,
			query:   `$[?length(@) > 0]`,
			wantLen: 0,
		},
		{
			name:    "unicode string length",
			json:    `["hello", "世界", "👋"]`,
			query:   `$[?length(@) == 2]`,
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := GetMany(tt.json, tt.query)
			if len(results) != tt.wantLen {
				t.Errorf("length(%q) = %d results, want %d", tt.query, len(results), tt.wantLen)
			}
		})
	}
}

// TestCountFunction tests the count() function extension
func TestCountFunction(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		query   string
		wantLen int // expected length of result
	}{
		{
			name:    "count wildcard selector",
			json:    `[{"a": 1}, {"a": 1, "b": 2}, {"a": 1}]`,
			query:   `$[?count(@.*) == 2]`,
			wantLen: 1,
		},
		{
			name:    "count empty nodelist",
			json:    `[{"a": 1}, {"a": 2}]`,
			query:   `$[?count(@.x) == 0]`,
			wantLen: 2,
		},
		{
			name:    "count single node",
			json:    `[{"a": 1}, {"a": 2}]`,
			query:   `$[?count(@.a) == 1]`,
			wantLen: 2,
		},
		{
			name:    "count with comparison",
			json:    `[[1], [2, 3], [4, 5, 6]]`,
			query:   `$[?count(@[*]) > 1]`,
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := GetMany(tt.json, tt.query)
			if len(results) != tt.wantLen {
				t.Errorf("count(%q) = %d results, want %d", tt.query, len(results), tt.wantLen)
			}
		})
	}
}

// TestMatchFunction tests the match() function extension
func TestMatchFunction(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		query   string
		wantLen int
	}{
		{
			name:    "full match",
			json:    `[{"date": "1974-05-28"}, {"date": "2020-01-15"}]`,
			query:   `$[?match(@.date, "1974-05-..")]`,
			wantLen: 1,
		},
		{
			name:    "pattern match",
			json:    `[{"tz": "Europe/Berlin"}, {"tz": "America/New_York"}]`,
			query:   `$[?match(@.tz, "Europe/.*")]`,
			wantLen: 1,
		},
		{
			name:    "no match",
			json:    `[{"tz": "Asia/Tokyo"}, {"tz": "America/LA"}]`,
			query:   `$[?match(@.tz, "Europe/.*")]`,
			wantLen: 0,
		},
		{
			name:    "non-string first arg returns false",
			json:    `[{"val": 123}, {"val": "hello"}]`,
			query:   `$[?match(@.val, ".*")]`,
			wantLen: 1,
		},
		{
			name:    "non-string second arg returns false",
			json:    `[{"val": "hello"}]`,
			query:   `$[?match(@.val, 123)]`,
			wantLen: 0,
		},
		{
			name:    "invalid regex returns false",
			json:    `[{"val": "hello"}]`,
			query:   `$[?match(@.val, "[invalid")]`,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := GetMany(tt.json, tt.query)
			if len(results) != tt.wantLen {
				t.Errorf("match(%q) = %d results, want %d", tt.query, len(results), tt.wantLen)
			}
		})
	}
}

// TestSearchFunction tests the search() function extension
func TestSearchFunction(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		query   string
		wantLen int
	}{
		{
			name:    "substring match",
			json:    `[{"author": "Alice Bob"}, {"author": "Charlie"}, {"author": "Bob Roberts"}]`,
			query:   `$[?search(@.author, "[BR]ob")]`,
			wantLen: 2,
		},
		{
			name:    "pattern search",
			json:    `[{"text": "hello world"}, {"text": "goodbye"}]`,
			query:   `$[?search(@.text, "world")]`,
			wantLen: 1,
		},
		{
			name:    "no match",
			json:    `[{"text": "hello"}, {"text": "hi"}]`,
			query:   `$[?search(@.text, "xyz")]`,
			wantLen: 0,
		},
		{
			name:    "non-string first arg returns false",
			json:    `[{"val": 123}]`,
			query:   `$[?search(@.val, ".*")]`,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := GetMany(tt.json, tt.query)
			if len(results) != tt.wantLen {
				t.Errorf("search(%q) = %d results, want %d", tt.query, len(results), tt.wantLen)
			}
		})
	}
}

// TestValueFunction tests the value() function extension
func TestValueFunction(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		query   string
		wantLen int
	}{
		{
			name:    "single node returns value",
			json:    `[{"color": "red"}, {"color": "blue"}]`,
			query:   `$[?value(@.color) == "red"]`,
			wantLen: 1,
		},
		{
			name:    "empty nodelist returns nothing",
			json:    `[{"color": "red"}, {"color": "blue"}]`,
			query:   `$[?value(@.missing) == "red"]`,
			wantLen: 0,
		},
		{
			name:    "multiple nodes returns nothing",
			json:    `[{"a":{"color":"red","x":{"color":"blue"}}},{"color":"green"}]`,
			query:   `$[?value(@..color) == "red"]`,
			wantLen: 0,
		},
		{
			name:    "nested descendant with single match",
			json:    `[{"x": {"color": "red"}}, {"x": {"color": "blue"}}]`,
			query:   `$[?value(@.x.color) == "red"]`,
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := GetMany(tt.json, tt.query)
			if len(results) != tt.wantLen {
				t.Errorf("value(%q) = %d results, want %d", tt.query, len(results), tt.wantLen)
			}
		})
	}
}

// TestFunctionErrors tests error handling for function calls
func TestFunctionErrors(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		query   string
		wantLen int
	}{
		{
			name:    "unknown function",
			json:    `[1, 2, 3]`,
			query:   `$[?unknown(@) == 1]`,
			wantLen: 0, // error should result in no matches
		},
		{
			name:    "wrong argument count",
			json:    `["hello", "world"]`,
			query:   `$[?length(@) == 5]`, // correct
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := GetMany(tt.json, tt.query)
			if len(results) != tt.wantLen {
				t.Errorf("error(%q) = %d results, want %d", tt.query, len(results), tt.wantLen)
			}
		})
	}
}
