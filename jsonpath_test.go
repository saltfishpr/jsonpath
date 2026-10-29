package jsonpath_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/saltfishpr/jsonpath"
)

type queryCase struct {
	name  string
	expr  string
	data  string
	want  []string
	paths []string
}

func runQueryValuesCases(t *testing.T, tests []queryCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := []byte(tt.data)
			got, err := jsonpath.QueryValues(tt.expr, data)
			if err != nil {
				t.Fatalf("QueryValues(%q): %v", tt.expr, err)
			}
			if got == nil {
				t.Fatal("QueryValues returned nil instead of a result slice")
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

func runQueryNodesCases(t *testing.T, tests []queryCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := []byte(tt.data)
			got, err := jsonpath.QueryNodes(tt.expr, data)
			if err != nil {
				t.Fatalf("QueryNodes(%q): %v", tt.expr, err)
			}
			if got == nil {
				t.Fatal("QueryNodes returned nil instead of a result slice")
			}
			if len(got) != len(tt.want) {
				t.Fatalf("QueryNodes(%q) = %v, want %q", tt.expr, got, tt.want)
			}
			for i, want := range tt.want {
				if string(got[i].Value) != want {
					t.Errorf("result[%d].Value = %q, want %q", i, got[i].Value, want)
				}
			}
			if tt.paths != nil {
				if len(got) != len(tt.paths) {
					t.Fatalf("QueryNodes(%q) = %d paths, want %d", tt.expr, len(got), len(tt.paths))
				}
				for i, want := range tt.paths {
					if got[i].Path != want {
						t.Errorf("result[%d].Path = %q, want %q", i, got[i].Path, want)
					}
				}
			}
			if !bytes.Equal(data, []byte(tt.data)) {
				t.Error("QueryNodes modified the input document")
			}
		})
	}
}

func runQueryCases(t *testing.T, tests []queryCase) {
	t.Helper()
	t.Run("QueryValues", func(t *testing.T) { runQueryValuesCases(t, tests) })
	t.Run("QueryNodes", func(t *testing.T) { runQueryNodesCases(t, tests) })
}

// TestQuerySelection checks selection order, duplicate retention, and exact source
// preservation through both public result formats.
func TestQuerySelection(t *testing.T) {
	runQueryCases(t, []queryCase{
		{
			name: "root preserves internal whitespace and number spelling",
			expr: "$", data: " \r\n\t{ \"n\": 1.2300e+02, \"s\": \"\\u4e2d\" } \n",
			want: []string{`{ "n": 1.2300e+02, "s": "\u4e2d" }`}, paths: []string{"$"},
		},
		{
			name: "member preserves escaped string",
			expr: "$.s", data: `{"s":"\u4e2d\/\"\\\b\f\n\r\t\uD83D\uDE00"}`,
			want: []string{`"\u4e2d\/\"\\\b\f\n\r\t\uD83D\uDE00"`}, paths: []string{"$['s']"},
		},
		{
			name: "union preserves order and duplicates",
			expr: "$[2,0,-1,0]", data: `[null,true,false]`,
			want: []string{"false", "null", "false", "null"}, paths: []string{"$[2]", "$[0]", "$[2]", "$[0]"},
		},
		{
			name: "each input retains all union results",
			expr: "$[*][1,0,1]", data: `[[0,1],[2,3]]`,
			want:  []string{"1", "0", "1", "3", "2", "3"},
			paths: []string{"$[0][1]", "$[0][0]", "$[0][1]", "$[1][1]", "$[1][0]", "$[1][1]"},
		},
		{
			name: "descendant follows nested objects and arrays",
			expr: "$..v", data: `{"v":0,"items":[{"v":1},{"nested":{"v":2}}]}`,
			want: []string{"0", "1", "2"}, paths: []string{"$['v']", "$['items'][0]['v']", "$['items'][1]['nested']['v']"},
		},
		{
			name: "filter object members by existence even for false and null",
			expr: "$[?@.flag].flag", data: `{"a":{"flag":false},"b":{"flag":null},"c":{}}`,
			want: []string{"false", "null"}, paths: []string{"$['a']['flag']", "$['b']['flag']"},
		},
		{
			name: "root reference in filter",
			expr: "$.items[?@.n >= $.min].n", data: `{"min":2,"items":[{"n":1},{"n":2},{"n":3}]}`,
			want: []string{"2", "3"}, paths: []string{"$['items'][1]['n']", "$['items'][2]['n']"},
		},
		{
			name: "positive slice clamps bounds",
			expr: "$[-99:99:2]", data: `[0,1,2,3,4]`,
			want: []string{"0", "2", "4"}, paths: []string{"$[0]", "$[2]", "$[4]"},
		},
		{
			name: "negative slice clamps bounds",
			expr: "$[99:-99:-2]", data: `[0,1,2,3,4]`,
			want: []string{"4", "2", "0"}, paths: []string{"$[4]", "$[2]", "$[0]"},
		},
		{
			name: "largest interoperable positive step",
			expr: "$[::9007199254740991]", data: `[0,1,2]`, want: []string{"0"}, paths: []string{"$[0]"},
		},
		{
			name: "largest interoperable negative step",
			expr: "$[::-9007199254740991]", data: `[0,1,2]`, want: []string{"2"}, paths: []string{"$[2]"},
		},
		{name: "missing member", expr: "$.missing", data: `{}`},
		{name: "missing intermediate member", expr: "$.missing.child", data: `{}`},
		{name: "out of range indexes", expr: "$[-4,3]", data: `[0,1,2]`},
		{name: "zero step", expr: "$[::0]", data: `[0,1,2]`},
		{name: "reverse slice with explicit negative end", expr: "$[:-1:-1]", data: `[0,1,2]`},
		{name: "slice on object", expr: "$[:]", data: `{"a":1}`},
		{name: "wildcard on scalar", expr: "$[*]", data: `42`},
		{name: "filter on scalar", expr: "$[?@]", data: `true`},
		{name: "wildcard on empty array", expr: "$[*]", data: `[]`},
		{name: "wildcard on empty object", expr: "$[*]", data: `{}`},
	})
	for _, data := range []string{"null", "true", "false", "-0", `""`, "[]", "{}"} {
		t.Run("root "+data, func(t *testing.T) {
			runQueryCases(t, []queryCase{{name: "scalar or empty container", expr: "$", data: data, want: []string{data}, paths: []string{"$"}}})
		})
	}
}

// TestQueryNormalizedPaths checks canonical escaping and paths deeper than the
// initial ancestor buffer.
func TestQueryNormalizedPaths(t *testing.T) {
	const depth = 40
	runQueryCases(t, []queryCase{
		{
			name: "escaped and Unicode member names",
			expr: `$["'\\\b\f\n\r\t\u0000\u001f中😀\"/"][""][0]`,
			data: `{"'\\\b\f\n\r\t\u0000\u001f中😀\"/":{"": [7]}}`,
			want: []string{"7"}, paths: []string{`$['\'\\\b\f\n\r\t\u0000\u001f中😀"/'][''][0]`},
		},
		{
			name: "path exceeds initial ancestor capacity",
			expr: "$" + strings.Repeat("[0]", depth), data: strings.Repeat("[", depth) + "7" + strings.Repeat("]", depth),
			want: []string{"7"}, paths: []string{"$" + strings.Repeat("[0]", depth)},
		},
	})
}

// TestQueryFunctions exercises builtin type conversions, missing values, and
// constant and runtime pattern handling through the public API.
func TestQueryFunctions(t *testing.T) {
	runQueryCases(t, []queryCase{
		{
			name: "length counts Unicode code points",
			expr: "$[?length(@) == 3]", data: `["中😀a","e\u0301","\uD83D\uDE00",[],{},null,1,true]`,
			want: []string{`"中😀a"`}, paths: []string{"$[0]"},
		},
		{
			name: "length handles containers and Nothing",
			expr: "$[?length(@.v) == 2]", data: `[{"v":[0,1]},{"v":{"a":1,"b":2}},{"v":false},{}]`,
			want: []string{`{"v":[0,1]}`, `{"v":{"a":1,"b":2}}`}, paths: []string{"$[0]", "$[1]"},
		},
		{
			name: "count preserves duplicate nodes",
			expr: "$[?count(@[0,0]) == 2]", data: `[[null],[],{}]`, want: []string{`[null]`}, paths: []string{"$[0]"},
		},
		{
			name: "value requires exactly one node and distinguishes null",
			expr: "$[?value(@[*]) == null]", data: `[[null],[],[null,null],[false]]`,
			want: []string{`[null]`}, paths: []string{"$[0]"},
		},
		{
			name: "nested function results adapt to ValueType",
			expr: "$[?length(value(@[*])) == 2]", data: `[["中😀"],["x","y"],[]]`,
			want: []string{`["中😀"]`}, paths: []string{"$[0]"},
		},
		{
			name: "singular query argument with negative index",
			expr: "$[?length(@.a[-1]) == 1]", data: `[{"a":["xx","中"]},{"a":[]},{"a":false},{}]`,
			want: []string{`{"a":["xx","中"]}`}, paths: []string{"$[0]"},
		},
		{
			name: "singular query stops at missing intermediate member",
			expr: "$[?length(@.a.b) == 1]", data: `[{},{"a":{"b":"x"}}]`,
			want: []string{`{"a":{"b":"x"}}`}, paths: []string{"$[1]"},
		},
		{
			name: "dynamic patterns including repeated invalid patterns",
			expr: "$[?match(@.s,@.p)].id",
			data: `[{"id":0,"s":"aa","p":"a+"},{"id":1,"s":"a","p":"a+"},{"id":2,"s":"a","p":"["},{"id":3,"s":"b","p":"["},{"id":4,"s":1,"p":"a"},{"id":5,"s":"a"},{"id":6,"s":"a","p":true}]`,
			want: []string{"0", "1"}, paths: []string{"$[0]['id']", "$[1]['id']"},
		},
		{
			name: "invalid literal patterns evaluate false",
			expr: `$[?match(@,"[") || search(@,"[")]`, data: `["a","b"]`,
		},
		{
			name: "caret and dollar are literal characters",
			expr: `$[?match(@,"^a$")]`, data: `["a","^a$","x^a$y"]`, want: []string{`"^a$"`}, paths: []string{"$[1]"},
		},
		{
			name: "search matches substring",
			expr: `$[?search(@,"^a$")]`, data: `["a","^a$","x^a$y"]`,
			want: []string{`"^a$"`, `"x^a$y"`}, paths: []string{"$[1]", "$[2]"},
		},
	})
}

// TestQueryExactNumberComparison covers precision and exponent boundaries that
// floating-point conversion would lose, plus structural and missing-value equality.
func TestQueryExactNumberComparison(t *testing.T) {
	tests := []struct {
		name, left, op, right string
		want                  bool
	}{
		{"beyond float precision", "9007199254740993", ">", "9007199254740992", true},
		{"equivalent decimal spelling", "1.2300e2", "==", "123", true},
		{"negative zero", "-0.0e999999999999999999999", "==", "0", true},
		{"negative numbers", "-2", "<", "-1", true},
		{"negative versus zero", "-1", "<", "0", true},
		{"positive versus negative", "1", ">", "-1", true},
		{"zero versus positive", "0", "<", "1", true},
		{"positive versus zero", "1", ">", "0", true},
		{"different digit lengths", "1.2", "<", "1.21", true},
		{"different digit lengths reversed", "1.21", ">", "1.2", true},
		{"huge positive exponents", "1e9223372036854775808", "==", "10e9223372036854775807", true},
		{"huge negative exponents", "1e-9223372036854775809", "==", "0.1e-9223372036854775808", true},
		{"huge exponent ordering", "1e999999999999999999999", ">", "9e999999999999999999998", true},
		{"huge versus ordinary exponent", "1e9223372036854775808", ">", "1", true},
		{"ordinary versus huge exponent", "1", "<", "1e9223372036854775808", true},
		{"exponent normalization returns to int64", "0.1e9223372036854775808", "==", "1e9223372036854775807", true},
		{"fraction adjustment underflows int64", "0.1e-9223372036854775808", "<", "1e-9223372036854775808", true},
		{"same magnitude with unequal digits", "12e9223372036854775807", "<", "13e9223372036854775807", true},
		{"huge negative number", "-1e999999999999999999999", "<=", "-1e999999999999999999999", true},
		{"different types are unequal", "1", "==", `"1"`, false},
		{"null equality", "null", "==", "null", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := queryCase{
				name: "comparison", expr: fmt.Sprintf("$[?@.a %s @.b].a", tt.op),
				data: fmt.Sprintf(`[{"a":%s,"b":%s}]`, tt.left, tt.right),
			}
			if tt.want {
				tc.want, tc.paths = []string{tt.left}, []string{"$[0]['a']"}
			}
			runQueryCases(t, []queryCase{tc})
		})
	}
	runQueryCases(t, []queryCase{
		{
			name: "Nothing equals Nothing but differs from null",
			expr: "$[?@.a == @.b]", data: `[{},{"a":null},{"a":null,"b":null}]`,
			want: []string{`{}`, `{"a":null,"b":null}`}, paths: []string{"$[0]", "$[2]"},
		},
		{
			name: "object equality compares names and nested values",
			expr: "$[?@.a == @.b].id",
			data: `[{"id":0,"a":{"x":[1,null]},"b":{"x":[1.0,null]}},{"id":1,"a":{"x":1},"b":{"y":1}},{"id":2,"a":[1],"b":[1,2]},{"id":3,"a":[1],"b":[2]}]`,
			want: []string{"0"}, paths: []string{"$[0]['id']"},
		},
	})
}

// TestQueryRegexp checks I-Regexp semantics that differ from Go regexp, including
// valid repetitions beyond backend limits and invalid patterns that must yield false.
func TestQueryRegexp(t *testing.T) {
	tests := []struct {
		name, pattern, text string
		want                bool
	}{
		{"exact repetition with leading zeros", "a{0002}", "aa", true},
		{"bounded repetition", "a{1,3}", "aaa", true},
		{"unbounded repetition", "a{2,}", "aaaa", true},
		{"zero repetitions", "a{0}", "", true},
		{"alternation", "(a|bc){2}", "abc", true},
		{"dot excludes carriage return", ".", "\r", false},
		{"dot excludes newline", ".", "\n", false},
		{"dot includes Unicode", ".", "😀", true},
		{"Unicode category", `\p{L}+`, "中文", true},
		{"negated Unicode category", `\P{N}+`, "中文", true},
		{"leading and trailing hyphens", "[-a-]+", "-aa-", true},
		{"negated character class", "[^a]+", "中文", true},
		{"Unicode category in class", `[\p{L}]+`, "中文", true},
		{"escaped range endpoints match", `[\n-\r]+`, "\n\r", true},
		{"escaped range endpoints", `[\n-\r]+`, "\n\t", false},
		{"escaped characters in class", `[\n\r\t\[\]\\\^\-]+`, "\n\r\t[]\\^-", true},
		{"exact repetition beyond RE2 limit", "a{1001}", strings.Repeat("a", 1001), true},
		{"optional unrolled repetitions", "a{0,1001}", strings.Repeat("a", 1000), true},
		{"unbounded unrolled repetitions", "a{1001,}", strings.Repeat("a", 1002), true},
		{"nested repetition beyond RE2 weight limit", "(a{100}){11}", strings.Repeat("a", 1100), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern, err := json.Marshal(tt.pattern)
			if err != nil {
				t.Fatal(err)
			}
			value, err := json.Marshal(tt.text)
			if err != nil {
				t.Fatal(err)
			}
			tc := queryCase{name: "match", expr: fmt.Sprintf("$[?match(@,%s)]", pattern), data: "[" + string(value) + "]"}
			if tt.want {
				tc.want, tc.paths = []string{string(value)}, []string{"$[0]"}
			}
			runQueryCases(t, []queryCase{tc})
		})
	}
	for _, pattern := range []string{
		"[", "[]", "[a-b-c]", "[[]", "[a-", "[a-]x]", "[a-[]", "[z-a]", `[\p{L}-a]`, `[a-\p{L}]`, `[\p{X}]`,
		`[\q]`, `[a-\q]`, `\`, `\q`, `\pL`, `\p{`, `\p{Invalid}`, `\p{}`, `\p{X}`, `\p{Lx}`,
		"(", ")", "*", "a{}", "a{2", "a{2,", "a{2,x}", "a{3,2}",
	} {
		t.Run("invalid pattern "+pattern, func(t *testing.T) {
			quoted, err := json.Marshal(pattern)
			if err != nil {
				t.Fatal(err)
			}
			runQueryCases(t, []queryCase{{
				name: "invalid I-Regexp is false", expr: fmt.Sprintf("$[?match(@,%s) || search(@,%s)]", quoted, quoted), data: `["a",""]`,
			}})
		})
	}
}

// TestQueryResultSharesInput checks byte identity as well as content so an
// accidental copy cannot satisfy the source-aliasing contract.
func TestQueryResultSharesInput(t *testing.T) {
	data := []byte(`{"v": [ 1.00, "\u4e2d" ]}`)
	want := []byte(`[ 1.00, "\u4e2d" ]`)
	start := bytes.Index(data, want)
	values, err := jsonpath.QueryValues("$.v", data)
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := jsonpath.QueryNodes("$.v", data)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || len(nodes) != 1 {
		t.Fatalf("got %d values and %d nodes, want one of each", len(values), len(nodes))
	}
	if !bytes.Equal(values[0], want) || !bytes.Equal(nodes[0].Value, want) {
		t.Fatalf("results = %s, %s, want %s", values[0], nodes[0].Value, want)
	}
	if &values[0][0] != &data[start] || &nodes[0].Value[0] != &data[start] {
		t.Error("results do not share the input byte slice")
	}
}

type queryErrorCase struct {
	name, expr, data, kind, message string
	offset                          int // Negative values disable only the location assertion.
}

func runQueryErrorCases(t *testing.T, tests []queryErrorCase) {
	t.Helper()
	queries := []struct {
		name string
		run  func(string, []byte) (bool, error)
	}{
		{"QueryValues", func(expr string, data []byte) (bool, error) {
			got, err := jsonpath.QueryValues(expr, data)
			return got == nil, err
		}},
		{"QueryNodes", func(expr string, data []byte) (bool, error) {
			got, err := jsonpath.QueryNodes(expr, data)
			return got == nil, err
		}},
	}
	for _, query := range queries {
		t.Run(query.name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					data := []byte(tt.data)
					nilResult, err := query.run(tt.expr, data)
					if err == nil {
						t.Fatal("expected an error")
					}
					if !nilResult {
						t.Error("query returned results on failure")
					}
					var compileErr *jsonpath.CompileError
					var documentErr *jsonpath.DocumentError
					switch tt.kind {
					case "compile":
						if !errors.As(err, &compileErr) {
							t.Fatalf("error = %T: %v, want *CompileError", err, err)
						}
						if tt.offset >= 0 && compileErr.Offset != tt.offset {
							t.Errorf("offset = %d, want %d", compileErr.Offset, tt.offset)
						}
					case "document":
						if !errors.As(err, &documentErr) {
							t.Fatalf("error = %T: %v, want *DocumentError", err, err)
						}
						if tt.offset >= 0 && documentErr.Offset != tt.offset {
							t.Errorf("offset = %d, want %d", documentErr.Offset, tt.offset)
						}
					case "limit":
						if errors.As(err, &compileErr) || errors.As(err, &documentErr) {
							t.Errorf("implementation limit returned as %T: %v", err, err)
						}
					default:
						t.Fatalf("unknown error kind %q", tt.kind)
					}
					if !strings.Contains(err.Error(), tt.message) {
						t.Errorf("error = %q, want substring %q", err, tt.message)
					}
					if !bytes.Equal(data, []byte(tt.data)) {
						t.Error("query modified the input document on failure")
					}
				})
			}
		})
	}
}

// TestQueryCompileErrors checks error types, byte offsets, and diagnostic
// precedence when more than one part of a request is invalid.
func TestQueryCompileErrors(t *testing.T) {
	runQueryErrorCases(t, []queryErrorCase{
		{name: "empty query", expr: "", data: `null`, kind: "compile", message: "compile error at offset 0", offset: 0},
		{name: "unexpected EOF uses byte offset", expr: "$.中[", data: `null`, kind: "compile", message: "end of input", offset: len("$.中[")},
		{name: "invalid UTF-8", expr: "$['\xff']", data: `null`, kind: "compile", message: "UTF-8", offset: 3},
		{name: "query error precedes document error", expr: "$[", data: `{`, kind: "compile", message: "end of input", offset: 2},
		{name: "leading whitespace", expr: " $", data: `null`, kind: "compile", message: "whitespace", offset: 1},
		{name: "trailing whitespace", expr: "$ ", data: `null`, kind: "compile", message: "whitespace", offset: 2},
		{name: "unknown function", expr: "$[?unknown(@)]", data: `[]`, kind: "compile", message: `unknown function "unknown"`, offset: 3},
		{name: "wrong argument count", expr: "$[?length(@,@) == 0]", data: `[]`, kind: "compile", message: "expected 1 arguments, got 2", offset: 3},
		{name: "logical function passed as value", expr: `$[?length(match(@,"a")) == 0]`, data: `[]`, kind: "compile", message: "expected ValueType, got LogicalType", offset: 10},
		{name: "value function passed as nodes", expr: "$[?count(length(@)) == 0]", data: `[]`, kind: "compile", message: "expected NodesType, got ValueType", offset: 9},
		{name: "nonsingular function argument", expr: "$[?length(@[*]) == 0]", data: `[]`, kind: "compile", message: "expected ValueType or singular query", offset: 10},
		{name: "logical comparison left operand", expr: `$[?match(@,"a") == true]`, data: `[]`, kind: "compile", message: "left operand", offset: -1},
		{name: "logical comparison right operand", expr: `$[?true == match(@,"a")]`, data: `[]`, kind: "compile", message: "right operand", offset: -1},
		{name: "invalid nested filter in disjunction", expr: "$[?@.a || unknown(@)]", data: `[]`, kind: "compile", message: "unknown function", offset: -1},
		{name: "invalid nested filter in conjunction", expr: "$[?@.a && unknown(@)]", data: `[]`, kind: "compile", message: "unknown function", offset: -1},
		{name: "invalid function inside function argument", expr: "$[?length(unknown(@)) == 0]", data: `[]`, kind: "compile", message: "unknown function", offset: -1},
		{name: "invalid nested query", expr: "$[?count(@[?unknown(@)]) == 0]", data: `[]`, kind: "compile", message: "unknown function", offset: -1},
	})
}

// TestQuerySyntaxErrors rejects malformed expressions before document evaluation.
func TestQuerySyntaxErrors(t *testing.T) {
	tests := []struct{ name, expr string }{
		{"token after root", "$@"},
		{"invalid descendant bracket", "$..["},
		{"invalid slice end", "$[1:true]"},
		{"disjunction with value operand", "$[?@.a || 1]"},
		{"conjunction with value operand", "$[?@.a && 1]"},
		{"repeated negation", "$[?!!@.a]"},
		{"negation missing operand", "$[?!]"},
		{"unparenthesized negated comparison", "$[?!@.a == 1]"},
		{"negation of value", "$[?!1]"},
		{"nonsingular right comparison operand", "$[?1 == @[*]]"},
		{"logical right comparison operand", "$[?1 == (@.a || @.b)]"},
		{"unterminated parenthesized expression", "$[?(@.a]"},
		{"invalid filter subquery", "$[?@.]"},
		{"function name followed by whitespace", "$[?length (@) == 0]"},
		{"uppercase function name", "$[?Length(@) == 0]"},
		{"Unicode function name", "$[?len中(@) == 0]"},
		{"unterminated function call", "$[?length(@]"},
		{"missing function argument", "$[?match(@,) ]"},
		{"invalid inequality operator", "$[?@ <> 1]"},
		{"invalid arithmetic operator", "$[?@ + 1]"},
		{"dangling query escape", "$['a\\"},
		{"negative number followed by invalid digit", "$[?@ == -x]"},
	}
	cases := make([]queryErrorCase, 0, len(tests))
	for _, tt := range tests {
		cases = append(cases, queryErrorCase{
			name: tt.name, expr: tt.expr, data: `[]`, kind: "compile", message: "compile error", offset: -1,
		})
	}
	runQueryErrorCases(t, cases)
}

// TestQueryDocumentErrors checks that parser failures retain their input byte
// offsets when exposed as public [jsonpath.DocumentError] values.
func TestQueryDocumentErrors(t *testing.T) {
	tests := []queryErrorCase{
		{name: "empty document", data: "", message: "unexpected end", offset: 0},
		{name: "only whitespace", data: " \n\t", message: "unexpected end", offset: 3},
		{name: "BOM", data: "\xef\xbb\xbf{}", message: "byte order mark", offset: 0},
		{name: "trailing data", data: "{} true", message: "trailing data", offset: 3},
		{name: "unexpected character", data: "+1", message: "unexpected character", offset: 0},
		{name: "invalid literal", data: "tru", message: "invalid literal", offset: 0},
		{name: "missing object key", data: `{1:0}`, message: "string key", offset: 1},
		{name: "missing colon", data: `{"a" 0}`, message: "expected ':'", offset: 5},
		{name: "missing object value", data: `{"a":}`, message: "unexpected character", offset: 5},
		{name: "unterminated object", data: `{"a":0`, message: "unexpected end of input in object", offset: 6},
		{name: "missing object comma", data: `{"a":0 "b":1}`, message: "expected ','", offset: 7},
		{name: "duplicate decoded member", data: `{"a":1,"\u0061":2}`, message: "duplicate", offset: 7},
		{name: "invalid escaped key", data: `{"\q":1}`, message: "invalid escape", offset: 3},
		{name: "unterminated array", data: `[1`, message: "unexpected end of input in array", offset: 2},
		{name: "missing array comma", data: `[1 2]`, message: "expected ','", offset: 3},
		{name: "array trailing comma", data: `[1,]`, message: "unexpected character", offset: 3},
		{name: "unterminated string", data: `"abc`, message: "unterminated string", offset: 4},
		{name: "dangling string escape", data: "\"abc\\", message: "unterminated escape", offset: 4},
		{name: "raw control character", data: "\"a\nb\"", message: "control character", offset: 2},
		{name: "invalid UTF-8 string", data: "\"\xff\"", message: "UTF-8", offset: 1},
		{name: "invalid string escape", data: `"\q"`, message: "invalid escape", offset: 2},
		{name: "short Unicode escape", data: `"\u12"`, message: "short \\u escape", offset: 2},
		{name: "invalid Unicode escape", data: `"\uZZZZ"`, message: "invalid \\u escape", offset: 2},
		{name: "unpaired high surrogate", data: `"\uD800"`, message: "surrogate pair", offset: 7},
		{name: "unpaired low surrogate", data: `"\uDC00"`, message: "surrogate pair", offset: 7},
		{name: "invalid second surrogate escape", data: `"\uD800\uZZZZ"`, message: "invalid \\u escape in surrogate pair", offset: 7},
		{name: "invalid second surrogate value", data: `"\uD800\u0041"`, message: "surrogate pair", offset: 7},
		{name: "minus without digits", data: `-`, message: "invalid number", offset: 1},
		{name: "invalid integer digit", data: `-x`, message: "invalid number", offset: 1},
		{name: "leading zero", data: `01`, message: "trailing data", offset: 1},
		{name: "missing fraction digits", data: `1.`, message: "fraction digits", offset: 2},
		{name: "missing exponent digits", data: `1e+`, message: "exponent digits", offset: 3},
		{name: "array exceeds nesting limit", data: strings.Repeat("[", 10001) + "0" + strings.Repeat("]", 10001), message: "max nesting depth", offset: 10000},
		{name: "object exceeds nesting limit", data: strings.Repeat(`{"a":`, 10001) + "0" + strings.Repeat("}", 10001), message: "max nesting depth", offset: 50000},
	}
	for i := range tests {
		tests[i].expr, tests[i].kind = "$", "document"
	}
	runQueryErrorCases(t, tests)
}

// TestQueryRegexpLimits checks that resource failures propagate through nested
// evaluation without becoming syntax errors, false matches, or partial results.
func TestQueryRegexpLimits(t *testing.T) {
	patterns := []struct{ name, pattern, message string }{
		{"nesting", strings.Repeat("(", 129) + "a" + strings.Repeat(")", 129), "nesting exceeds safety limit"},
		{"expanded size", "a{9000000}", "translated pattern exceeds safety limit"},
		{"expansion multiplication overflow", "(ab){9223372036854775807}", "translated pattern exceeds safety limit"},
		{"minimum count overflow", "a{999999999999999999999}", "quantifier minimum exceeds implementation limit"},
		{"maximum count overflow", "a{1,999999999999999999999}", "quantifier maximum exceeds implementation limit"},
	}
	for _, pattern := range patterns {
		t.Run(pattern.name, func(t *testing.T) {
			quoted, err := json.Marshal(pattern.pattern)
			if err != nil {
				t.Fatal(err)
			}
			runQueryErrorCases(t, []queryErrorCase{
				{name: "literal pattern", expr: fmt.Sprintf("$[?match(@,%s)]", quoted), data: `["a"]`, kind: "limit", message: pattern.message, offset: -1},
				{name: "dynamic pattern after successful result", expr: "$[?match(@.s,@.p)]", data: fmt.Sprintf(`[{"s":"a","p":"a"},{"s":"a","p":%s}]`, quoted), kind: "limit", message: pattern.message, offset: -1},
			})
		})
	}
	pattern := strings.Repeat("(", 129) + "a" + strings.Repeat(")", 129)
	quoted, err := json.Marshal(pattern)
	if err != nil {
		t.Fatal(err)
	}
	item := fmt.Sprintf(`{"s":"a","p":%s}`, quoted)
	tests := []queryErrorCase{
		{name: "object filter", expr: "$[?search(@.s,@.p)]", data: `{"a":` + item + `}`},
		{name: "descendant array", expr: "$..[?search(@.s,@.p)]", data: `[[` + item + `]]`},
		{name: "descendant object", expr: "$..[?search(@.s,@.p)]", data: `{"a":{"b":` + item + `}}`},
		{name: "disjunction", expr: "$[?@.missing || search(@.s,@.p)]", data: `[` + item + `]`},
		{name: "conjunction", expr: "$[?@.s && search(@.s,@.p)]", data: `[` + item + `]`},
		{name: "negation", expr: "$[?!search(@.s,@.p)]", data: `[` + item + `]`},
		{name: "left comparison operand", expr: "$[?count(@[?search(@.s,@.p)]) == 0]", data: `[[` + item + `]]`},
		{name: "right comparison operand", expr: "$[?0 == count(@[?search(@.s,@.p)])]", data: `[[` + item + `]]`},
		{name: "nested function argument", expr: "$[?length(value(@[?search(@.s,@.p)])) == 0]", data: `[[` + item + `]]`},
	}
	for i := range tests {
		tests[i].kind, tests[i].message, tests[i].offset = "limit", "nesting exceeds safety limit", -1
	}
	runQueryErrorCases(t, tests)
}
