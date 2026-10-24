package jsonpath_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/saltfishpr/jsonpath"
)

const (
	queryValuesNumbers = `[0,1,2,3,4]`
	queryValuesRecords = `[{"id":0,"v":null},{"id":1,"v":false},{"id":2,"v":0},{"id":3,"v":""},{"id":4,"v":[]},{"id":5,"v":{}},{"id":6}]`
	queryValuesStore   = `{"limit":10,"books":[{"title":"Go","price":8,"tags":["go","code"]},{"title":"Rust","price":12,"tags":["code"]},{"title":"JSON","price":10,"tags":[]}]}`
)

type queryValuesCase struct {
	name string
	expr string
	data string
	want []string
}

func runQueryValuesCases(t *testing.T, tests []queryValuesCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := []byte(tt.data)
			got, err := jsonpath.QueryValues(tt.expr, data)
			if err != nil {
				t.Fatalf("QueryValues(%q): %v", tt.expr, err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("QueryValues(%q) = %s, want %q", tt.expr, got, tt.want)
			}
			for i, want := range tt.want {
				if string(got[i]) != want {
					t.Errorf("result[%d] = %q, want %q", i, got[i], want)
				}
			}
			if !bytes.Equal(data, []byte(tt.data)) {
				t.Error("QueryValues modified the input document")
			}
		})
	}
}

func TestQueryValues_Root(t *testing.T) {
	runQueryValuesCases(t, []queryValuesCase{
		{"object", `$`, `{"a":1}`, []string{`{"a":1}`}},
		{"array", `$`, queryValuesNumbers, []string{queryValuesNumbers}},
		{"string", `$`, `"hello"`, []string{`"hello"`}},
		{"number", `$`, `-1.25e+2`, []string{`-1.25e+2`}},
		{"true", `$`, `true`, []string{`true`}},
		{"false", `$`, `false`, []string{`false`}},
		{"null", `$`, `null`, []string{`null`}},
		{"empty_object", `$`, `{}`, []string{`{}`}},
		{"empty_array", `$`, `[]`, []string{`[]`}},
	})
}

func TestQueryValues_RawJSON(t *testing.T) {
	runQueryValuesCases(t, []queryValuesCase{
		{"surrounding_whitespace", `$`, " \n { \"a\" : [1, 2] }\r\t", []string{`{ "a" : [1, 2] }`}},
		{"nested_whitespace", `$.a`, `{"a": { "b" : [1,  2] }}`, []string{`{ "b" : [1,  2] }`}},
		{"string_escapes", `$.a`, `{"a":"\u4e2d\n\"\\"}`, []string{`"\u4e2d\n\"\\"`}},
		{"numbers", `$[*]`, `[-0,1.00,1e+2,9007199254740993]`, []string{`-0`, `1.00`, `1e+2`, `9007199254740993`}},
	})
}

func TestQueryValues_NameSelector(t *testing.T) {
	runQueryValuesCases(t, []queryValuesCase{
		{"dot", `$.a.b`, `{"a":{"b":1}}`, []string{`1`}},
		{"brackets", `$['a']["b"]`, `{"a":{"b":1}}`, []string{`1`}},
		{"special_characters", `$['a.b']['x y']['[0]']`, `{"a.b":{"x y":{"[0]":2}}}`, []string{`2`}},
		{"empty", `$['']`, `{"":3}`, []string{`3`}},
		{"unicode", `$.中文`, `{"中文":4}`, []string{`4`}},
		{"escaped_unicode", `$['\u4e2d\u6587']`, `{"中文":4}`, []string{`4`}},
		{"surrogate_pair", `$['\uD83D\uDE00']`, `{"😀":5}`, []string{`5`}},
		{"escaped_quote", `$['it\'s']`, `{"it's":6}`, []string{`6`}},
		{"escaped_backslash", `$['a\\b']`, `{"a\\b":7}`, []string{`7`}},
		{"decoded_document_key", `$.a`, `{"\u0061":8}`, []string{`8`}},
		{"missing", `$.missing`, `{"a":1}`, nil},
		{"case_sensitive", `$.A`, `{"a":1}`, nil},
		{"on_array", `$.a`, `[1,2]`, nil},
		{"on_scalar", `$.a`, `1`, nil},
		{"after_null", `$.a.b`, `{"a":null}`, nil},
		{"numeric_key", `$['0']`, `{"0":"zero"}`, []string{`"zero"`}},
		{"no_unicode_normalization", `$['é']`, `{"e\u0301":1}`, nil},
	})
}

func TestQueryValues_IndexSelector(t *testing.T) {
	runQueryValuesCases(t, []queryValuesCase{
		{"first", `$[0]`, queryValuesNumbers, []string{`0`}},
		{"last", `$[-1]`, queryValuesNumbers, []string{`4`}},
		{"negative_first", `$[-5]`, queryValuesNumbers, []string{`0`}},
		{"out_of_bounds", `$[5]`, queryValuesNumbers, nil},
		{"negative_out_of_bounds", `$[-6]`, queryValuesNumbers, nil},
		{"empty_array", `$[0]`, `[]`, nil},
		{"on_object", `$[0]`, `{"0":1}`, nil},
		{"nested", `$[1][0]`, `[[1],[2,3]]`, []string{`2`}},
	})
}

func TestQueryValues_Wildcard(t *testing.T) {
	runQueryValuesCases(t, []queryValuesCase{
		{"array", `$[*]`, `[1,null,false,"x",[],{}]`, []string{`1`, `null`, `false`, `"x"`, `[]`, `{}`}},
		{"object_order", `$.*`, `{"z":1,"a":2,"m":3}`, []string{`1`, `2`, `3`}},
		{"bracket_object", `$[*]`, `{"z":1,"a":2}`, []string{`1`, `2`}},
		{"empty_array", `$[*]`, `[]`, nil},
		{"empty_object", `$.*`, `{}`, nil},
		{"null", `$.*`, `null`, nil},
		{"string", `$[*]`, `"abc"`, nil},
		{"chained", `$[*][*]`, `[[1,2],[],[3]]`, []string{`1`, `2`, `3`}},
		{"heterogeneous", `$[*].a`, `[{"a":1},null,2,{}, {"a":false}]`, []string{`1`, `false`}},
	})
}

func TestQueryValues_MultipleSelectors(t *testing.T) {
	runQueryValuesCases(t, []queryValuesCase{
		{"index_order_and_duplicates", `$[3,0,3,-1]`, queryValuesNumbers, []string{`3`, `0`, `3`, `4`}},
		{"name_order_and_duplicates", `$['b','a','b','missing']`, `{"a":1,"b":2}`, []string{`2`, `1`, `2`}},
		{"mixed_selectors", `$[0,2:4,*]`, queryValuesNumbers, []string{`0`, `2`, `3`, `0`, `1`, `2`, `3`, `4`}},
		{"each_input", `$[*][1,0]`, `[[1,2],[3,4]]`, []string{`2`, `1`, `4`, `3`}},
	})
}

func TestQueryValues_Slice(t *testing.T) {
	runQueryValuesCases(t, []queryValuesCase{
		{"all", `$[:]`, queryValuesNumbers, []string{`0`, `1`, `2`, `3`, `4`}},
		{"bounded", `$[1:4]`, queryValuesNumbers, []string{`1`, `2`, `3`}},
		{"omitted_start", `$[:2]`, queryValuesNumbers, []string{`0`, `1`}},
		{"omitted_end", `$[3:]`, queryValuesNumbers, []string{`3`, `4`}},
		{"positive_step", `$[::2]`, queryValuesNumbers, []string{`0`, `2`, `4`}},
		{"negative_bounds", `$[-4:-1]`, queryValuesNumbers, []string{`1`, `2`, `3`}},
		{"clamped_bounds", `$[-99:99]`, queryValuesNumbers, []string{`0`, `1`, `2`, `3`, `4`}},
		{"reverse", `$[::-1]`, queryValuesNumbers, []string{`4`, `3`, `2`, `1`, `0`}},
		{"reverse_step", `$[4:0:-2]`, queryValuesNumbers, []string{`4`, `2`}},
		{"reverse_clamped", `$[99:-99:-2]`, queryValuesNumbers, []string{`4`, `2`, `0`}},
		{"explicit_negative_end", `$[:-1:-1]`, queryValuesNumbers, nil},
		{"zero_step", `$[::0]`, queryValuesNumbers, nil},
		{"equal_bounds", `$[2:2]`, queryValuesNumbers, nil},
		{"wrong_direction", `$[4:1]`, queryValuesNumbers, nil},
		{"reverse_wrong_direction", `$[1:4:-1]`, queryValuesNumbers, nil},
		{"empty_array", `$[::-1]`, `[]`, nil},
		{"on_object", `$[:]`, `{"a":1}`, nil},
	})
}

func TestQueryValues_Descendant(t *testing.T) {
	runQueryValuesCases(t, []queryValuesCase{
		{"name", `$..a`, `{"a":0,"nested":{"a":1},"items":[{"a":2}]}`, []string{`0`, `1`, `2`}},
		{"wildcard", `$..*`, `[[1],2]`, []string{`[1]`, `2`, `1`}},
		{"index", `$..[0]`, `[[1,2],[3,4]]`, []string{`[1,2]`, `1`, `3`}},
		{"slice", `$..[1:]`, `[[1,2],[3,4]]`, []string{`[3,4]`, `2`, `4`}},
		{"duplicates", `$..['a','a']`, `{"a":1,"x":{"a":2}}`, []string{`1`, `1`, `2`, `2`}},
		{"scalar", `$..*`, `42`, nil},
		{"followed_by_child", `$..books[*].title`, queryValuesStore, []string{`"Go"`, `"Rust"`, `"JSON"`}},
	})
}

func TestQueryValues_Filter(t *testing.T) {
	runQueryValuesCases(t, []queryValuesCase{
		{"existence_not_truthiness", `$[?@.v].id`, queryValuesRecords, []string{`0`, `1`, `2`, `3`, `4`, `5`}},
		{"missing", `$[?!@.v].id`, queryValuesRecords, []string{`6`}},
		{"null", `$[?@.v == null].id`, queryValuesRecords, []string{`0`}},
		{"false", `$[?@.v == false].id`, queryValuesRecords, []string{`1`}},
		{"zero", `$[?@.v == 0].id`, queryValuesRecords, []string{`2`}},
		{"empty_string", `$[?@.v == ""].id`, queryValuesRecords, []string{`3`}},
		{"two_missing_values", `$[?@.v == @.missing].id`, queryValuesRecords, []string{`6`}},
		{"missing_not_null", `$[?@.v != null].id`, queryValuesRecords, []string{`1`, `2`, `3`, `4`, `5`, `6`}},
		{"less", `$[?@ < 2]`, queryValuesNumbers, []string{`0`, `1`}},
		{"less_equal", `$[?@ <= 2]`, queryValuesNumbers, []string{`0`, `1`, `2`}},
		{"greater", `$[?@ > 2]`, queryValuesNumbers, []string{`3`, `4`}},
		{"greater_equal", `$[?@ >= 2]`, queryValuesNumbers, []string{`2`, `3`, `4`}},
		{"and", `$[?@ > 0 && @ < 3]`, queryValuesNumbers, []string{`1`, `2`}},
		{"or", `$[?@ == 0 || @ == 4]`, queryValuesNumbers, []string{`0`, `4`}},
		{"precedence", `$[?@ == 0 || @ >= 2 && @ < 4]`, queryValuesNumbers, []string{`0`, `2`, `3`}},
		{"parentheses", `$[?(@ == 0 || @ >= 2) && @ < 3]`, queryValuesNumbers, []string{`0`, `2`}},
		{"negation", `$[?!(@ < 3)]`, queryValuesNumbers, []string{`3`, `4`}},
		{"root_reference", `$.books[?@.price <= $.limit].title`, queryValuesStore, []string{`"Go"`, `"JSON"`}},
		{"negative_index", `$.books[?@.tags[-1] == "code"].title`, queryValuesStore, []string{`"Go"`, `"Rust"`}},
		{"nested_filter", `$.books[?@.tags[?@ == "go"]].title`, queryValuesStore, []string{`"Go"`}},
		{"object_values", `$[?@ >= 2]`, `{"z":3,"a":1,"b":2}`, []string{`3`, `2`}},
		{"scalar", `$[?@ == 1]`, `1`, nil},
		{"empty", `$[?@]`, `[]`, nil},
		{"descendant", `$..[?@.price < 10].title`, queryValuesStore, []string{`"Go"`}},
		{"mixed_union", `$[?@ > 3,0]`, queryValuesNumbers, []string{`4`, `0`}},
	})
}

func TestQueryValues_Comparison(t *testing.T) {
	runQueryValuesCases(t, []queryValuesCase{
		{"no_type_coercion", `$[?@ == 1]`, `[1,"1",true,null]`, []string{`1`}},
		{"mixed_types_not_ordered", `$[?@ < 2]`, `[1,"1",true,null,[],{}]`, []string{`1`}},
		{"strings", `$[?@ < "b"]`, `["a","b","c",1]`, []string{`"a"`}},
		{"decoded_strings", `$[?@ == "中文"]`, `["\u4e2d\u6587","中文","other"]`, []string{`"\u4e2d\u6587"`, `"中文"`}},
		{"number_representations", `$[?@ == 1]`, `[1,1.0,1e0,2]`, []string{`1`, `1.0`, `1e0`}},
		{"signed_zero", `$[?@ == 0]`, `[-0,0,0.0,-0.0]`, []string{`-0`, `0`, `0.0`, `-0.0`}},
		{"large_integer", `$[?@ > 9007199254740992]`, `[9007199254740992,9007199254740993]`, []string{`9007199254740993`}},
		{"decimal_precision", `$[?@ > 1.0000000000000001]`, `[1.0000000000000001,1.0000000000000002]`, []string{`1.0000000000000002`}},
		{"array_equality", `$[?@.a == @.b].id`, `[{"id":1,"a":[1,null],"b":[1.0,null]},{"id":2,"a":[1,2],"b":[2,1]},{"id":3,"a":[],"b":[1]}]`, []string{`1`}},
		{"object_equality", `$[?@.a == @.b].id`, `[{"id":1,"a":{"x":1,"y":[]},"b":{"y":[],"x":1.0}},{"id":2,"a":{"x":1},"b":{"x":2}},{"id":3,"a":{},"b":{"x":1}}]`, []string{`1`}},
	})
}

func TestQueryValues_NumberPrecision(t *testing.T) {
	runQueryValuesCases(t, []queryValuesCase{
		{"zero_huge_exponent", `$[?@ == 0e-9223372036854775809]`, `[-0.00e+9223372036854775808,0e-9223372036854775809,1e-9223372036854775809]`, []string{`-0.00e+9223372036854775808`, `0e-9223372036854775809`}},
		{"beyond_int64", `$[?@ > 9223372036854775808]`, `[9223372036854775808,9223372036854775809]`, []string{`9223372036854775809`}},
		{"long_significand", `$[?@ == 123456789012345678901234567890.123456789]`, `[123456789012345678901234567890.123456788,123456789012345678901234567890.123456789]`, []string{`123456789012345678901234567890.123456789`}},
		{"float64_overflow", `$[?@ > 1e309]`, `[1e309,2e309]`, []string{`2e309`}},
		{"float64_underflow", `$[?@ > 0 && @ < 2e-400]`, `[0,1e-400,2e-400]`, []string{`1e-400`}},
		{"huge_positive_exponent", `$[?@ > 1e9223372036854775808]`, `[1e9223372036854775808,1e9223372036854775809]`, []string{`1e9223372036854775809`}},
		{"huge_negative_exponent", `$[?@ < 1e-9223372036854775809]`, `[1e-9223372036854775810,1e-9223372036854775809]`, []string{`1e-9223372036854775810`}},
		{"normalized_huge_exponent", `$[?@ == 1e9223372036854775808]`, `[10e9223372036854775807,0.1e9223372036854775809,2e9223372036854775808]`, []string{`10e9223372036854775807`, `0.1e9223372036854775809`}},
		{"negative_numbers", `$[?@ < -9223372036854775808]`, `[-9223372036854775809,-9223372036854775808]`, []string{`-9223372036854775809`}},
		{"document_comparison", `$[?@.a < @.b].a`, `[{"a":1e9223372036854775808,"b":2e9223372036854775808},{"a":2e-9223372036854775809,"b":1e-9223372036854775809}]`, []string{`1e9223372036854775808`}},
	})
}

func TestQueryValues_Functions(t *testing.T) {
	runQueryValuesCases(t, []queryValuesCase{
		{"length/string_codepoints", `$[?length(@) == 2]`, `["中文","😀a","abc","e\u0301"]`, []string{`"中文"`, `"😀a"`, `"e\u0301"`}},
		{"length/containers", `$[?length(@) == 2]`, `[[1,2],{"a":1,"b":2},[],{}]`, []string{`[1,2]`, `{"a":1,"b":2}`}},
		{"length/empty_values", `$[?length(@) == 0]`, `["",[],{},null,0,false]`, []string{`""`, `[]`, `{}`}},
		{"length/unsupported_returns_nothing", `$[?length(@.v) == @.missing].id`, queryValuesRecords, []string{`0`, `1`, `2`, `6`}},
		{"count/children", `$.books[?count(@.tags[*]) == 2].title`, queryValuesStore, []string{`"Go"`}},
		{"count/missing", `$[?count(@.v) == 0].id`, queryValuesRecords, []string{`6`}},
		{"count/null_is_node", `$[?count(@.v) == 1].id`, queryValuesRecords, []string{`0`, `1`, `2`, `3`, `4`, `5`}},
		{"count/duplicates", `$[?count(@[0,0]) == 2]`, `[[1],[]]`, []string{`[1]`}},
		{"value/single_node", `$.books[?value(@.tags[*]) == "code"].title`, queryValuesStore, []string{`"Rust"`}},
		{"value/zero_or_multiple_nodes", `$[?value(@.v[*]) == @.missing].id`, `[{"id":0,"v":[]},{"id":1,"v":[1]},{"id":2,"v":[1,2]}]`, []string{`0`, `2`}},
		{"value/null", `$[?value(@[*]) == null]`, `[[null],[],[null,null]]`, []string{`[null]`}},
		{"functions/nested", `$[?length(value(@[*])) == 2]`, `[["中文"],["a"],[],["ab","cd"]]`, []string{`["中文"]`}},
		{"match/full_string", `$[?match(@, "a+")]`, `["a","aa","ba","ab",1,null]`, []string{`"a"`, `"aa"`}},
		{"search/substring", `$[?search(@, "a+")]`, `["a","ba","ab","b",1,null]`, []string{`"a"`, `"ba"`, `"ab"`}},
		{"match/empty_pattern", `$[?match(@, "")]`, `["","a"]`, []string{`""`}},
		{"search/empty_pattern", `$[?search(@, "")]`, `["","a",null]`, []string{`""`, `"a"`}},
		{"match/invalid_pattern", `$[?match(@, "[")]`, `["a",""]`, nil},
		{"search/invalid_pattern", `$[?search(@, "[")]`, `["a",""]`, nil},
		{"match/non_string_pattern", `$[?match(@, 1)]`, `["a"]`, nil},
		{"search/missing_argument_value", `$[?search(@.missing, "a")]`, `[{}]`, nil},
		{"match/dynamic_pattern", `$[?match(@.text, @.pattern)].id`, `[{"id":1,"text":"abc","pattern":"a.*"},{"id":2,"text":"abc","pattern":"["},{"id":3,"text":"abc","pattern":null},{"id":4,"text":"abc"}]`, []string{`1`}},
		{"search/dynamic_pattern", `$[?search(@.text, @.pattern)].id`, `[{"id":1,"text":"abc","pattern":"b"},{"id":2,"text":"abc","pattern":"z"}]`, []string{`1`}},
		{"match/literal_anchors", `$[?match(@, "^a$")]`, `["a","^a$"]`, []string{`"^a$"`}},
	})
}

func TestQueryValues_StringEscapes(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		runQueryValuesCases(t, []queryValuesCase{
			{"decoded_surrogate_pair", `$[?@ == "😀"]`, `["\uD83D\uDE00","\ud83d\ude00","😀","other"]`, []string{`"\uD83D\uDE00"`, `"\ud83d\ude00"`, `"😀"`}},
			{"decoded_control_escapes", `$[?@ == "\b\f\n\r\t/\\\""]`, `["\b\f\n\r\t\/\\\"","other"]`, []string{`"\b\f\n\r\t\/\\\""`}},
		})
	})
	t.Run("invalid", func(t *testing.T) {
		tests := []struct {
			name string
			data string
		}{
			{"truncated_unicode_escape", `"\u12`},
			{"unpaired_high_surrogate", `"\uD83D"`},
			{"unpaired_low_surrogate", `"\uDE00"`},
			{"surrogate_followed_by_text", `"\uD83Dabcdef"`},
			{"surrogate_followed_by_other_escape", `"\uD83D\nabcdef"`},
			{"surrogate_followed_by_non_surrogate", `"\uD83D\u0041"`},
			{"reversed_surrogate_pair", `"\uDE00\uD83D"`},
			{"invalid_low_surrogate_hex", `"\uD83D\uZZZZ"`},
			{"unterminated_escape", `"abc\`},
			{"unterminated_escaped_string", `"abc\n`},
			{"control_character_after_escape", "\"\\n\t\""},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := jsonpath.QueryValues(`$`, []byte(tt.data))
				var documentErr *jsonpath.DocumentError
				if !errors.As(err, &documentErr) {
					t.Fatalf("QueryValues error = %v, want *jsonpath.DocumentError", err)
				}
				if got != nil {
					t.Errorf("result = %s, want nil on error", got)
				}
			})
		}
	})
}

func TestQueryValues_InvalidExpression(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"empty", ``},
		{"missing_root", `books`},
		{"current_node_at_root", `@.a`},
		{"trailing_dot", `$.`},
		{"incomplete_descendant", `$..`},
		{"unclosed_bracket", `$[0`},
		{"empty_selector", `$[]`},
		{"trailing_comma", `$[0,]`},
		{"unclosed_string", `$['a]`},
		{"invalid_escape", `$['\x']`},
		{"leading_zero_index", `$[01]`},
		{"fractional_index", `$[1.5]`},
		{"index_outside_exact_integer_range", `$[9007199254740992]`},
		{"negative_index_outside_exact_integer_range", `$[-9007199254740992]`},
		{"slice_start_outside_exact_integer_range", `$[9007199254740992:]`},
		{"slice_end_outside_exact_integer_range", `$[:-9007199254740992]`},
		{"slice_step_outside_exact_integer_range", `$[::9007199254740992]`},
		{"too_many_slice_parts", `$[1:2:3:4]`},
		{"empty_filter", `$[?]`},
		{"incomplete_comparison", `$[?@ ==]`},
		{"assignment", `$[?@ = 1]`},
		{"arithmetic", `$[?@ + 1 == 2]`},
		{"non_singular_comparison", `$[?@[*] == 1]`},
		{"trailing_input", `$ garbage`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := jsonpath.QueryValues(tt.expr, []byte(`{"books":[1,2]}`))
			if err == nil {
				t.Fatalf("QueryValues(%q) = %s, want an error", tt.expr, got)
			}
			if got != nil {
				t.Errorf("result = %s, want nil on error", got)
			}
		})
	}
}

func TestQueryValues_CompileError(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"unknown_function", `$[?unknown(@)]`},
		{"too_few_arguments", `$[?match(@)]`},
		{"too_many_arguments", `$[?length(@, @) == 1]`},
		{"nodes_required", `$[?count(1) == 1]`},
		{"value_required", `$[?length(@[*]) == 1]`},
		{"logical_function_as_value", `$[?match(@, "a") == true]`},
		{"value_function_as_test", `$[?length(@)]`},
		{"nested_argument_type", `$[?length(count(1)) == 1]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := jsonpath.QueryValues(tt.expr, []byte(`[]`))
			var compileErr *jsonpath.CompileError
			if !errors.As(err, &compileErr) {
				t.Fatalf("QueryValues(%q) error = %v, want *jsonpath.CompileError", tt.expr, err)
			}
			if got != nil {
				t.Errorf("result = %s, want nil on error", got)
			}
		})
	}
}

func TestQueryValues_InvalidDocument(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"empty", ``},
		{"whitespace_only", " \n\t\r"},
		{"unclosed_object", `{"a":1`},
		{"unclosed_array", `[1,2`},
		{"unclosed_string", `"abc`},
		{"missing_colon", `{"a" 1}`},
		{"missing_value", `{"a":}`},
		{"unquoted_key", `{a:1}`},
		{"object_trailing_comma", `{"a":1,}`},
		{"array_trailing_comma", `[1,]`},
		{"missing_comma", `[1 2]`},
		{"single_quoted_string", `'abc'`},
		{"invalid_escape", `"\x"`},
		{"invalid_unicode_escape", `"\uZZZZ"`},
		{"unescaped_control_character", "\"a\nb\""},
		{"invalid_literal", `undefined`},
		{"incomplete_literal", `tru`},
		{"leading_zero", `01`},
		{"leading_plus", `+1`},
		{"missing_integer", `.1`},
		{"missing_fraction", `1.`},
		{"missing_exponent", `1e+`},
		{"multiple_values", `{} []`},
		{"trailing_garbage", `null x`},
		{"comment", `/* comment */ {}`},
		{"bom", "\xef\xbb\xbf{}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := jsonpath.QueryValues(`$`, []byte(tt.data))
			var documentErr *jsonpath.DocumentError
			if !errors.As(err, &documentErr) {
				t.Fatalf("QueryValues error = %v, want *jsonpath.DocumentError", err)
			}
			if got != nil {
				t.Errorf("result = %s, want nil on error", got)
			}
		})
	}
	t.Run("nil_input", func(t *testing.T) {
		got, err := jsonpath.QueryValues(`$`, nil)
		var documentErr *jsonpath.DocumentError
		if !errors.As(err, &documentErr) || got != nil {
			t.Fatalf("QueryValues(nil) = (%s, %v), want (nil, *jsonpath.DocumentError)", got, err)
		}
	})
	t.Run("invalid_unselected_subtree", func(t *testing.T) {
		got, err := jsonpath.QueryValues(`$.a`, []byte(`{"a":1,"b":[}`))
		var documentErr *jsonpath.DocumentError
		if !errors.As(err, &documentErr) || got != nil {
			t.Fatalf("QueryValues = (%s, %v), want (nil, *jsonpath.DocumentError)", got, err)
		}
	})
}

func TestQueryValues_SharesInput(t *testing.T) {
	data := []byte(`{"value": [1, 2]}`)
	got, err := jsonpath.QueryValues(`$.value`, data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || string(got[0]) != `[1, 2]` {
		t.Fatalf("result = %s, want [[1, 2]]", got)
	}
	// 比较地址来验证共享内存，避免修改调用方应保持只读的输入或结果。
	start := bytes.Index(data, []byte(`[1, 2]`))
	if &got[0][0] != &data[start] {
		t.Error("result does not share the input document's backing array")
	}
}
