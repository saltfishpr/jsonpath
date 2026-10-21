package jsonpath_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/saltfishpr/jsonpath"
)

// queryCase describes a single JSONPath query expectation.
type queryCase struct {
	name, expr, data string
	want             []string
}

func runQueryCases(t *testing.T, cases []queryCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values, err := jsonpath.QueryValues(tc.expr, []byte(tc.data))
			if err != nil {
				t.Fatal(err)
			}
			got := make([]string, len(values))
			for i, v := range values {
				got[i] = string(v)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestQuery_Numbers(t *testing.T) {
	runQueryCases(t, []queryCase{
		{"int_beyond_float64", "$[?@ == 9007199254740993]", "[9007199254740992,9007199254740993]", []string{"9007199254740993"}},
		{"decimal_equivalent_forms", "$[?@ == 0.1]", "[0.10,1e-1,0.10000000000000001]", []string{"0.10", "1e-1"}},
		{"negative_zero_equals_zero", "$[?@ == 0]", "[-0,0,-0.0,0e999999999999999999999]", []string{"-0", "0", "-0.0", "0e999999999999999999999"}},
		{"huge_positive_exponent", "$[?@ > 9e999998]", "[1e999999,9e999998,1e999998]", []string{"1e999999"}},
		{"huge_negative_exponent", "$[?@ < 1e-999999999999999999999]", "[1e-9999999999999999999999,0,-1]", []string{"1e-9999999999999999999999", "0", "-1"}},
		{"integer_equivalent_forms", "$[?@ == 100]", "[1e+2,100.0,1000e-1]", []string{"1e+2", "100.0", "1000e-1"}},
	})
}

func TestQuery_Equality(t *testing.T) {
	runQueryCases(t, []queryCase{
		{"missing_equals_missing", "$[?@.a == @.b]", "[{}, {\"a\":null}, {\"a\":null,\"b\":null}]", []string{"{}", "{\"a\":null,\"b\":null}"}},
		{"deep_equal_across_number_forms", "$[?@.a == @.b]", `[{"a":{"x":[1,2]},"b":{"x":[1.0,2]}},{"a":[1,2],"b":[2,1]}]`, []string{`{"a":{"x":[1,2]},"b":{"x":[1.0,2]}}`}},
	})
}

func TestQuery_Children(t *testing.T) {
	runQueryCases(t, []queryCase{
		{"wildcard_preserves_source_order", "$.*", `{"z": 1,"a": 2}`, []string{"1", "2"}},
	})
}

func TestQuery_Functions(t *testing.T) {
	runQueryCases(t, []queryCase{
		{"length_counts_code_points", "$[?length(@) == 2]", `["😀a","e\u0301","abc"]`, []string{`"😀a"`, `"e\u0301"`}},
	})
}

func TestQuery_RawJSON(t *testing.T) {
	runQueryCases(t, []queryCase{
		{"preserves_whitespace_and_escapes", "$.x", ` {"x": { "n": 1.2300e+04, "s":"\u0061" }} `, []string{`{ "n": 1.2300e+04, "s":"\u0061" }`}},
		{"preserves_null_literal", "$.x", `{"x":null}`, []string{"null"}},
	})
}

func TestQuery_Slice(t *testing.T) {
	runQueryCases(t, []queryCase{
		{"negative_step", "$[::-2]", `[0,1,2,3,4]`, []string{"4", "2", "0"}},
		{"zero_step_empty", "$[::0]", `[0,1,2]`, []string{}},
		{"step_exceeds_length", "$[::9007199254740991]", `[0,1,2]`, []string{"0"}},
	})
}

// regexCase describes a single regex-function expectation. `search` picks
// between match() (anchored) and search() (unanchored).
type regexCase struct {
	name, pattern, text string
	search, want        bool
}

func runRegexCases(t *testing.T, cases []regexCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal([]string{tc.text})
			if err != nil {
				t.Fatal(err)
			}
			pattern, _ := json.Marshal(tc.pattern)
			fn := "match"
			if tc.search {
				fn = "search"
			}
			expr := "$[?" + fn + "(@," + string(pattern) + ")]"
			values, err := jsonpath.QueryValues(expr, data)
			if err != nil {
				t.Fatalf("static: %v", err)
			}
			if (len(values) == 1) != tc.want {
				t.Fatalf("static: got %s, want match %v", values, tc.want)
			}
			// The pattern supplied by the document takes the dynamic compilation path.
			document, _ := json.Marshal([]map[string]string{{"text": tc.text, "pattern": tc.pattern}})
			dynamic, err := jsonpath.QueryValues("$[?"+fn+"(@.text,@.pattern)]", document)
			if err != nil {
				t.Fatal(err)
			}
			if (len(dynamic) == 1) != tc.want {
				t.Fatal("dynamic pattern disagrees")
			}
		})
	}
}

func TestRegex_Match(t *testing.T) {
	runRegexCases(t, []regexCase{
		{"caret_is_literal_miss", "^ab.*", "abc", false, false},
		{"caret_is_literal_hit", "^ab.*", "^abc", false, true},
		{"dollar_is_literal_hit", ".*bc$", "abc$", false, true},
		{"dot_excludes_cr", ".", "\r", false, false},
		{"dot_excludes_lf", ".", "\n", false, false},
		{"dot_matches_emoji", ".", "😀", false, true},
		{"empty_pattern_matches_empty", "", "", false, true},
		{"empty_pattern_rejects_nonempty", "", "x", false, false},
		{"substring_requires_full_match_miss", "ab", "zabz", false, false},
		{"alternation_full_match_hit", "a|ab", "ab", false, true},
		{"alternation_in_quantifier_hit", "(ab|a){2,3}", "aab", false, true},
	})
}

func TestRegex_Search(t *testing.T) {
	runRegexCases(t, []regexCase{
		{"empty_pattern_matches_any", "", "x", true, true},
		{"star_matches_empty_substring", "a*", "xyz", true, true},
		{"substring_hit", "ab", "zabz", true, true},
	})
}

func TestRegex_Classes(t *testing.T) {
	runRegexCases(t, []regexCase{
		{"unicode_cn_hit", `\p{Cn}`, "\u0378", false, true},
		{"unicode_c_hit", `\p{C}`, "\u0378", false, true},
		{"unicode_cn_negated_hit", `\P{Cn}`, "a", false, true},
		{"unicode_c_letter_miss", `\p{C}`, "a", false, false},
		{"unicode_co_private_use_hit", `\p{Co}`, "\uE000", false, true},
		{"unicode_l_cjk_hit", `\p{L}+`, "你好", false, true},
		{"letter_or_number_hit", `[\p{L}\p{N}]+`, "字1", false, true},
		{"negated_letter_hit", `[^\p{L}]`, "1", false, true},
		{"range_single_dash_hit", "[--]", "-", false, true},
		{"literal_dash_in_class_hit", "[-a-]", "-", false, true},
		{"escaped_dash_in_class_hit", `[\-a]`, "-", false, true},
		{"range_a_to_c_hit", "[a-c]", "b", false, true},
	})
}

func TestRegex_Quantifiers(t *testing.T) {
	runRegexCases(t, []regexCase{
		{"fixed_count_hit", "a{1001}", strings.Repeat("a", 1001), false, true},
		{"fixed_count_miss", "a{1001}", strings.Repeat("a", 1000), false, false},
		{"huge_optional_quantifier_hit", "(a?){9999999999999999999999}", "aaa", false, true},
		{"huge_optional_quantifier_empty_hit", "(a?){9999999999999999999999}", "", false, true},
		{"huge_required_quantifier_miss", "a{9999999999999999999999}", "aaa", false, false},
		{"optional_quantifier_bounded_miss", "(a?){0,2}", "aaa", false, false},
		{"nested_counted_quantifier_hit", "(a{2,4}){2,4}", "aaaaa", false, true},
		{"nested_star_hit", "(a*)*", "aaa", false, true},
		{"nested_star_requires_full_match", "(a*)*", "aaab", false, false},
		{"optional_quantifier_upper_bound_miss", "(a?){3,4}", "aaaaa", false, false},
	})
}

// TestRegex_Rejected collects patterns that are outside I-Regexp. The regex
// function returns no match rather than surfacing a compile error.
func TestRegex_Rejected(t *testing.T) {
	runRegexCases(t, []regexCase{
		{"perl_class_d", `\d`, "1", false, false},
		{"inline_flags", "(?i)a", "A", false, false},
		{"reversed_counted_range", "a{2,1}", "aa", false, false},
		{"negated_empty_class", "[^]", "x", false, false},
		{"reversed_range", "[z-a]", "b", false, false},
		{"category_cs", `\p{Cs}`, "a", false, false},
		{"script_property", `\p{Greek}`, "α", false, false},
		{"general_category_lc", `\p{LC}`, "A", false, false},
		{"dash_after_range", "[a-b-c]", "a", false, false},
		{"double_star", "a**", "a", false, false},
		{"escape_dollar", `\$`, "$", false, false},
	})
}

func TestOwnership_Aliasing(t *testing.T) {
	data := []byte(`{"x":{"s":"original"}}`)
	doc, err := jsonpath.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	for i := range data {
		data[i] = 'x'
	}
	q, err := jsonpath.Compile(`$['x','x']`)
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := q.Nodes(doc)
	if err != nil {
		t.Fatal(err)
	}
	nodes[0].Value[0] = '!'
	if !bytes.Equal(nodes[1].Value, []byte(`{"s":"original"}`)) {
		t.Fatal("duplicate results alias")
	}
}

func TestOwnership_Concurrency(t *testing.T) {
	data := []byte(`{"x":{"s":"original"}}`)
	doc, err := jsonpath.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	q, err := jsonpath.Compile(`$['x','x']`)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				nodes, err := q.Nodes(doc)
				if err != nil {
					t.Error(err)
					return
				}
				if len(nodes) != 2 || nodes[0].Path != "$['x']" || string(nodes[0].Value) != `{"s":"original"}` {
					t.Error("shared document changed")
				}
			}
		}()
	}
	wg.Wait()
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		name, input string
	}{
		{"empty_input", ""},
		{"trailing_content", `{} {}`},
		{"duplicate_key", `{"a":1,"\u0061":2}`},
		{"lone_high_surrogate", `"\uD800"`},
		{"lone_low_surrogate", `"\uDC00"`},
		{"invalid_utf8", "\"\xff\""},
		{"trailing_comma_array", `[1,]`},
		{"missing_value_object", `{"a":}`},
		{"leading_zero_number", `01`},
		{"trailing_dot_number", `1.`},
		{"invalid_literal", `truex`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := jsonpath.Parse([]byte(tc.input))
			var de *jsonpath.DocumentError
			if !errors.As(err, &de) {
				t.Errorf("input %q: expected DocumentError, got %v", tc.input, err)
			}
		})
	}
}

func TestCompileErrors(t *testing.T) {
	cases := []struct {
		name, expr string
	}{
		{"filter_requires_comparison", `$[?true]`},
		{"misplaced_filter", `$[?(@)] == 1`},
		{"length_rejects_nodelist", `$[?length(@.*)>0]`},
		{"nodelist_in_comparison", `$[?@.*==1]`},
		{"double_bang", `$[?!!@]`},
		{"unknown_function", `$[?foo(@)]`},
		{"index_out_of_range", `$[9007199254740992]`},
		{"negative_zero_index", `$[-0]`},
		{"match_arity", `$[?match(@)]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := jsonpath.Compile(tc.expr)
			var ce *jsonpath.CompileError
			if !errors.As(err, &ce) {
				t.Errorf("query %q: expected CompileError, got %v", tc.expr, err)
			}
		})
	}
}

func TestLimits(t *testing.T) {
	t.Run("document_depth", func(t *testing.T) {
		deep := strings.Repeat("[", 1025) + "0" + strings.Repeat("]", 1025)
		_, err := jsonpath.Parse([]byte(deep))
		var limit *jsonpath.LimitError
		if !errors.As(err, &limit) {
			t.Fatalf("expected depth limit, got %v", err)
		}
	})
	t.Run("query_nesting", func(t *testing.T) {
		expr := "$[?" + strings.Repeat("(", 1025) + "@" + strings.Repeat(")", 1025) + "]"
		_, err := jsonpath.Compile(expr)
		var limit *jsonpath.LimitError
		if !errors.As(err, &limit) {
			t.Fatalf("expected query limit, got %v", err)
		}
	})
	t.Run("regex_nesting", func(t *testing.T) {
		pattern, _ := json.Marshal(strings.Repeat("(", 1025) + "a" + strings.Repeat(")", 1025))
		_, err := jsonpath.Compile("$[?match(@," + string(pattern) + ")]")
		var limit *jsonpath.LimitError
		if !errors.As(err, &limit) {
			t.Fatalf("expected regex limit, got %v", err)
		}
	})
}

func TestZeroValue(t *testing.T) {
	t.Run("zero_query_rejected", func(t *testing.T) {
		var q jsonpath.Query
		if _, err := q.Values(nil); err == nil {
			t.Fatal("zero query accepted")
		}
	})
	t.Run("zero_document_rejected", func(t *testing.T) {
		compiled, err := jsonpath.Compile("$")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compiled.Nodes(new(jsonpath.Document)); err == nil {
			t.Fatal("zero document accepted")
		}
	})
}
