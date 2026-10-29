package document

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
)

func mustParse(t testing.TB, src string) *Node {
	t.Helper()
	n, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse(%q): %v", src, err)
	}
	return n
}

// TestParseScalars checks decoded values, preserved source text, and location
// defaults for standalone scalars.
func TestParseScalars(t *testing.T) {
	type number struct {
		digits   string
		exponent int64
		negative bool
	}
	tests := []struct {
		name    string
		src     string
		kind    Kind
		boolean bool
		str     string
		number  *number // Nil omits assertions about the internal decimal representation.
		len     int
	}{
		{name: "null", src: "null", kind: KindNull, len: -1},
		{name: "true", src: "true", kind: KindBool, boolean: true, len: -1},
		{name: "false", src: "false", kind: KindBool, len: -1},
		{name: "string", src: `"abc"`, kind: KindString, str: "abc", len: 3},
		{name: "empty string", src: `""`, kind: KindString, str: "", len: 0},
		{name: "unicode string", src: `"中😀"`, kind: KindString, str: "中😀", len: 2},
		{name: "escaped string", src: `"a\nb\u00e9"`, kind: KindString, str: "a\nbé", len: 4},
		{name: "zero", src: "0", kind: KindNumber, number: &number{}, len: -1},
		{name: "negative zero", src: "-0", kind: KindNumber, number: &number{}, len: -1},
		{
			name: "fraction and exponent", src: "1.2300e+02", kind: KindNumber,
			number: &number{digits: "123"}, len: -1,
		},
		{
			name: "negative fraction", src: "-1.5", kind: KindNumber,
			number: &number{digits: "15", exponent: -1, negative: true}, len: -1,
		},
		{
			name: "leading zeros of fraction", src: "0.000123", kind: KindNumber,
			number: &number{digits: "123", exponent: -6}, len: -1,
		},
		{
			name: "trailing zeros", src: "100", kind: KindNumber,
			number: &number{digits: "1", exponent: 2}, len: -1,
		},
		{
			name: "long integer keeps all digits", src: "1234567890123456789012345678901234567890", kind: KindNumber,
			number: &number{
				digits:   "123456789012345678901234567890123456789",
				exponent: 1,
			},
			len: -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := mustParse(t, tt.src)
			if got := n.Kind(); got != tt.kind {
				t.Fatalf("Kind() = %v, want %v", got, tt.kind)
			}
			if got := string(n.Raw()); got != tt.src {
				t.Errorf("Raw() = %q, want %q", got, tt.src)
			}
			if got := n.String(); got != tt.str {
				t.Errorf("String() = %q, want %q", got, tt.str)
			}
			if got := n.Bool(); got != tt.boolean {
				t.Errorf("Bool() = %t, want %t", got, tt.boolean)
			}
			if tt.number != nil {
				got := n.Number()
				if got.digits != tt.number.digits {
					t.Errorf("Number().digits = %q, want %q", got.digits, tt.number.digits)
				}
				if got.exponent != tt.number.exponent {
					t.Errorf("Number().exponent = %d, want %d", got.exponent, tt.number.exponent)
				}
				if got.negative != tt.number.negative {
					t.Errorf("Number().negative = %t, want %t", got.negative, tt.number.negative)
				}
				if got.bigExponent != nil {
					t.Errorf("Number().bigExponent = %v, want nil", got.bigExponent)
				}
			}
			if got := n.Len(); got != tt.len {
				t.Errorf("Len() = %d, want %d", got, tt.len)
			}
			if got := n.Array(); got != nil {
				t.Errorf("Array() = %v, want nil", got)
			}
			if got := n.Members(); got != nil {
				t.Errorf("Members() = %v, want nil", got)
			}
			if got := n.Get("x"); got != nil {
				t.Errorf("Get on non-object = %v, want nil", got)
			}
			if got := n.Parent(); got != nil {
				t.Errorf("Parent() = %v, want nil for root", got)
			}
			if got := n.Index(); got != -1 {
				t.Errorf("Index() = %d, want -1", got)
			}
			if got := n.Name(); got != "" {
				t.Errorf("Name() = %q, want empty", got)
			}
		})
	}
}

// TestParseBigExponent checks promotion beyond int64 and demotion when decimal
// normalization brings an exponent back into range.
func TestParseBigExponent(t *testing.T) {
	tests := []struct {
		name         string
		src          string
		wantDigits   string
		wantExponent int64
		wantBig      bool
	}{
		{
			name: "exponent overflows int64", src: "1e999999999999999999999",
			wantDigits: "1", wantBig: true,
		},
		{
			name: "overflow with trailing zero adjustment", src: "10e9223372036854775807",
			wantDigits: "1", wantBig: true,
		},
		{
			name: "negative overflow with fraction adjustment", src: "0.001e-9223372036854775807",
			wantDigits: "1", wantBig: true,
		},
		{
			name: "overflow adjusted back into int64", src: "1.555e9223372036854775810",
			wantDigits: "1555", wantExponent: math.MaxInt64,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			num := mustParse(t, tt.src).Number()
			if num.digits != tt.wantDigits {
				t.Errorf("digits = %q, want %q", num.digits, tt.wantDigits)
			}
			if got := num.bigExponent != nil; got != tt.wantBig {
				t.Fatalf("bigExponent 是否存在 = %t, want %t", got, tt.wantBig)
			}
			if !tt.wantBig && num.exponent != tt.wantExponent {
				t.Errorf("exponent = %d, want %d", num.exponent, tt.wantExponent)
			}
		})
	}
}

// TestNodeConstructors checks scalar payloads and independent node identity.
func TestNodeConstructors(t *testing.T) {
	null := NewNull()
	if null.Kind() != KindNull || !null.IsNull() || null.Index() != -1 {
		t.Errorf("NewNull() = %+v", null)
	}
	tr, fa := NewTrue(), NewFalse()
	if tr.Kind() != KindBool || !tr.Bool() {
		t.Errorf("NewTrue() = %+v", tr)
	}
	if fa.Kind() != KindBool || fa.Bool() {
		t.Errorf("NewFalse() = %+v", fa)
	}
	num := NewNumber("-1.20e3")
	if num.Kind() != KindNumber {
		t.Errorf("NewNumber() Kind() = %v", num.Kind())
	}
	if got := num.Number(); got.digits != "12" || got.exponent != 2 || !got.negative {
		t.Errorf("NewNumber(-1.20e3) = %+v", got)
	}
	str := NewString("中")
	if str.Kind() != KindString || str.String() != "中" || str.Len() != 1 {
		t.Errorf("NewString() = %+v", str)
	}

	// Equal values may later occupy different locations, so their node identity
	// must not be shared.
	if tr == NewTrue() || fa == NewFalse() || null == NewNull() {
		t.Error("构造器返回了共享单例，节点需要独立的 parent/index/name")
	}
}

// TestParseContainerStructure checks member order, lookup identity, source
// boundaries, and parent metadata across mixed and empty containers.
func TestParseContainerStructure(t *testing.T) {
	const src = `{"arr":[1,"two",null,{"k":true}],"empty":[],"emptyObj":{}}`
	root := mustParse(t, src)

	if got := root.Kind(); got != KindObject {
		t.Fatalf("root Kind() = %v, want %v", got, KindObject)
	}
	if got, want := string(root.Raw()), src; got != want {
		t.Errorf("root Raw() = %q, want %q", got, want)
	}
	if got := root.Len(); got != 3 {
		t.Errorf("root Len() = %d, want 3", got)
	}

	members := root.Members()
	var names []string
	for _, m := range members {
		names = append(names, m.Name)
	}
	if got, want := strings.Join(names, ","), "arr,empty,emptyObj"; got != want {
		t.Fatalf("member order = %q, want %q", got, want)
	}

	arr := root.Get("arr")
	if arr == nil {
		t.Fatal(`Get("arr") = nil`)
	}
	if got := arr.Kind(); got != KindArray {
		t.Fatalf("arr Kind() = %v, want %v", got, KindArray)
	}
	if got := arr.Len(); got != 4 {
		t.Fatalf("arr Len() = %d, want 4", got)
	}
	if got := string(arr.Raw()); got != `[1,"two",null,{"k":true}]` {
		t.Errorf("arr Raw() = %q", got)
	}
	if got := arr.Name(); got != "arr" {
		t.Errorf("arr Name() = %q, want %q", got, "arr")
	}
	if got := arr.Index(); got != -1 {
		t.Errorf("arr Index() = %d, want -1 for object member", got)
	}
	if got := arr.Parent(); got != root {
		t.Errorf("arr Parent() != root")
	}

	elems := arr.Array()
	wantKinds := []Kind{KindNumber, KindString, KindNull, KindObject}
	for i, e := range elems {
		if got := e.Kind(); got != wantKinds[i] {
			t.Errorf("arr[%d] Kind() = %v, want %v", i, got, wantKinds[i])
		}
		if got := e.Index(); got != int64(i) {
			t.Errorf("arr[%d] Index() = %d, want %d", i, got, i)
		}
		if got := e.Name(); got != "" {
			t.Errorf("arr[%d] Name() = %q, want empty", i, got)
		}
		if got := e.Parent(); got != arr {
			t.Errorf("arr[%d] Parent() != arr", i)
		}
	}
	if got := string(elems[0].Raw()); got != "1" {
		t.Errorf("arr[0] Raw() = %q, want %q", got, "1")
	}
	if got := elems[1].String(); got != "two" {
		t.Errorf("arr[1] String() = %q, want %q", got, "two")
	}
	if got := elems[3].Get("k"); got == nil || !got.Bool() {
		t.Errorf(`arr[3].Get("k") = %v, want true`, got)
	}
	if got := elems[3].Get("missing"); got != nil {
		t.Errorf(`arr[3].Get("missing") = %v, want nil`, got)
	}

	wantRaw := map[string]string{"empty": "[]", "emptyObj": "{}"}
	for _, name := range []string{"empty", "emptyObj"} {
		n := root.Get(name)
		if n == nil {
			t.Fatalf("Get(%q) = nil", name)
		}
		if got := n.Len(); got != 0 {
			t.Errorf("%s Len() = %d, want 0", name, got)
		}
		if got := string(n.Raw()); got != wantRaw[name] {
			t.Errorf("%s Raw() = %q, want %q", name, got, wantRaw[name])
		}
		if got := n.Array(); len(got) != 0 {
			t.Errorf("%s Array() = %v, want empty", name, got)
		}
		if got := n.Members(); len(got) != 0 {
			t.Errorf("%s Members() = %v, want empty", name, got)
		}
		if got := n.Index(); got != -1 {
			t.Errorf("%s Index() = %d, want -1", name, got)
		}
	}
	if got := root.Get("missing"); got != nil {
		t.Errorf(`Get("missing") = %v, want nil`, got)
	}
}

// TestParseNodeInvariants checks that block allocation preserves unique node
// addresses, parent links, lookup identity, and source fragments across a large tree.
func TestParseNodeInvariants(t *testing.T) {
	const items = 1000
	var sb strings.Builder
	sb.WriteString("[")
	for i := 0; i < items; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, `{"k%d":[%d,null,"s%d"]}`, i, i, i)
	}
	sb.WriteString("]")
	src := sb.String()
	root := mustParse(t, src)

	seen := make(map[*Node]bool, items*5)
	wantNodes := 1 + items*5 // Root array plus an object, array, number, null, and string per item.
	var walk func(n *Node, parent *Node)
	walk = func(n *Node, parent *Node) {
		t.Helper()
		if seen[n] {
			t.Fatalf("node %p (kind %v) 被重复分配", n, n.Kind())
		}
		seen[n] = true
		if got := n.Parent(); got != parent {
			t.Fatalf("kind %v 的 Parent() 与容器不一致", n.Kind())
		}
		raw := n.Raw()
		if len(raw) == 0 {
			t.Fatalf("kind %v 的 raw 为空", n.Kind())
		}
		if !strings.Contains(src, string(raw)) {
			t.Fatalf("kind %v 的 raw %q 不是输入子串", n.Kind(), raw)
		}
		switch n.Kind() {
		case KindArray:
			if len(n.Array()) == 0 {
				return
			}
			for i, e := range n.Array() {
				if got := e.Index(); got != int64(i) {
					t.Fatalf("数组元素 Index() = %d, want %d", got, i)
				}
				if got := e.Name(); got != "" {
					t.Fatalf("数组元素 Name() = %q, want empty", got)
				}
				walk(e, n)
			}
		case KindObject:
			for _, m := range n.Members() {
				if got := m.Value.Name(); got != m.Name {
					t.Fatalf("对象成员 Name() = %q, want %q", got, m.Name)
				}
				if got := m.Value.Index(); got != -1 {
					t.Fatalf("对象成员 Index() = %d, want -1", got)
				}
				if got := n.Get(m.Name); got != m.Value {
					t.Fatalf("Get(%q) 未返回同一节点", m.Name)
				}
				walk(m.Value, n)
			}
		}
	}
	walk(root, nil)
	if got := len(seen); got != wantNodes {
		t.Fatalf("节点数 = %d, want %d", got, wantNodes)
	}
}

// TestParseStringsAfterBlockSwitch checks that allocating later string blocks
// cannot overwrite text retained by earlier nodes.
func TestParseStringsAfterBlockSwitch(t *testing.T) {
	var sb strings.Builder
	sb.WriteByte('[')
	const items = 512
	want := make([]string, items)
	for i := 0; i < items; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		// Sizes on both sides of 4 KiB exercise shared and oversized blocks.
		want[i] = strings.Repeat(string(rune('a'+i%26)), i*7%5000)
		sb.WriteString(`"`)
		sb.WriteString(want[i])
		sb.WriteString(`"`)
	}
	sb.WriteByte(']')

	root := mustParse(t, sb.String())
	elems := root.Array()
	if len(elems) != items {
		t.Fatalf("len = %d, want %d", len(elems), items)
	}
	for i, e := range elems {
		if got := e.String(); got != want[i] {
			t.Fatalf("元素 %d String() 长度 = %d, want %d（共享字符串块被覆盖）", i, len(got), len(want[i]))
		}
	}
}

// TestParseEscapesReuseScratch checks that decoded strings survive subsequent
// reuse of the escape buffer, including interleaved strings without escapes.
func TestParseEscapesReuseScratch(t *testing.T) {
	src := `["a\nb","plain","\u4e2d\u6587","long-long-long-long-long","\t\/\\","","\ud83d\ude00","x"]`
	root := mustParse(t, src)
	want := []string{"a\nb", "plain", "中文", "long-long-long-long-long", "\t/\\", "", "😀", "x"}
	elems := root.Array()
	if len(elems) != len(want) {
		t.Fatalf("len = %d, want %d", len(elems), len(want))
	}
	for i, e := range elems {
		if got := e.String(); got != want[i] {
			t.Errorf("元素 %d String() = %q, want %q", i, got, want[i])
		}
	}
}

// TestParseStringDecoding checks Unicode decoding and code-point length while
// requiring [Node.Raw] to preserve the original escape spelling.
func TestParseStringDecoding(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{name: "plain", src: `"abc"`, want: "abc"},
		{name: "empty", src: `""`, want: ""},
		{name: "escaped quote", src: `"a\"b"`, want: `a"b`},
		{name: "escaped backslash", src: `"a\\b"`, want: `a\b`},
		{name: "escaped slash", src: `"a\/b"`, want: "a/b"},
		{name: "short escapes", src: `"\b\f\n\r\t"`, want: "\b\f\n\r\t"},
		{name: "nul escape", src: `"\u0000"`, want: "\x00"},
		{name: "latin1 escape", src: `"\u00e9"`, want: "é"},
		{name: "uppercase hex", src: `"\u00E9"`, want: "é"},
		{name: "bmp escape", src: `"\u4e2d\u6587"`, want: "中文"},
		{name: "surrogate pair", src: `"\ud83d\ude00"`, want: "😀"},
		{name: "two surrogate pairs", src: `"\ud83d\ude00\ud83d\ude00"`, want: "😀😀"},
		{name: "raw utf8", src: `"中😀"`, want: "中😀"},
		{name: "replacement char", src: `"\ufffd"`, want: "\uFFFD"},
		{name: "mixed", src: `"a\u00e9\n😀\"\\"`, want: "aé\n😀\"\\"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := mustParse(t, tt.src)
			if got := n.Kind(); got != KindString {
				t.Fatalf("Kind() = %v, want %v", got, KindString)
			}
			if got := n.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
			if got := n.Raw(); string(got) != tt.src {
				t.Errorf("Raw() = %q, want %q", got, tt.src)
			}
			if got, want := n.Len(), len([]rune(tt.want)); got != want {
				t.Errorf("Len() = %d, want %d", got, want)
			}
		})
	}
}

// TestParseNumberGrammar separates numeric syntax validity from magnitude and
// requires specific diagnostics for incomplete fractions and exponents.
func TestParseNumberGrammar(t *testing.T) {
	valid := []string{
		"0", "-0", "1", "-1", "1234567890",
		"0.5", "-0.5", "1.000", "1e2", "1E2", "1e+2", "1e-2", "-1.5e-3",
		"1234567890123456789012345678901234567890",
		"1e999999999999999999999", "-1e-999999999999999999999",
	}
	for _, src := range valid {
		t.Run("valid "+src, func(t *testing.T) {
			n := mustParse(t, src)
			if got := n.Kind(); got != KindNumber {
				t.Fatalf("Kind() = %v, want %v", got, KindNumber)
			}
			if got := string(n.Raw()); got != src {
				t.Errorf("Raw() = %q, want %q", got, src)
			}
		})
	}

	invalid := []struct {
		name string
		src  string
	}{
		{name: "leading zero", src: "[01]"},
		{name: "minus only", src: "[-]"},
		{name: "missing fraction digits", src: "[1.]"},
		{name: "missing integer", src: "[.1]"},
		{name: "leading plus", src: "[+1]"},
		{name: "missing exponent digits", src: "[1e]"},
		{name: "sign only exponent", src: "[1e+]"},
		{name: "double dot", src: "[1..2]"},
		{name: "hex literal", src: "[0x10]"},
		{name: "infinity", src: "[Infinity]"},
		{name: "nan", src: "[NaN]"},
		{name: "double exponent", src: "[1e2e3]"},
		{name: "space after sign", src: "[- 1]"},
	}
	for _, tt := range invalid {
		t.Run("invalid "+tt.name, func(t *testing.T) {
			if _, err := Parse([]byte(tt.src)); err == nil {
				t.Fatalf("Parse(%q) = nil error, want syntax error", tt.src)
			}
		})
	}

	for _, tt := range []struct{ src, want string }{
		{src: "[1.]", want: "missing fraction digits"},
		{src: "[1e]", want: "missing exponent digits"},
	} {
		_, err := Parse([]byte(tt.src))
		var syntaxErr *SyntaxError
		if !errors.As(err, &syntaxErr) {
			t.Fatalf("Parse(%q) error = %v, want *SyntaxError", tt.src, err)
		}
		if !strings.Contains(syntaxErr.Message, tt.want) {
			t.Errorf("Parse(%q) message = %q, want contains %q", tt.src, syntaxErr.Message, tt.want)
		}
	}
}

// TestParseSyntaxErrors checks rejection, typed diagnostics, and source offsets
// without returning a partial document.
func TestParseSyntaxErrors(t *testing.T) {
	const unknownOffset = -1
	tests := []struct {
		name       string
		src        string
		want       string
		wantOffset int
	}{
		{name: "empty input", src: "", want: "unexpected end of input", wantOffset: 0},
		{name: "whitespace only", src: " \t\r\n", want: "unexpected end of input", wantOffset: 4},
		{name: "byte order mark", src: "\xef\xbb\xbf{}", want: "unexpected byte order mark", wantOffset: 0},
		{name: "trailing data", src: "1 2", want: "unexpected trailing data", wantOffset: 2},
		{name: "unexpected character", src: "@", want: "unexpected character", wantOffset: 0},
		{name: "dangling minus", src: "-", want: "invalid number", wantOffset: 1},
		{name: "bad literal", src: "tru", want: `invalid literal, expected "true"`, wantOffset: 0},
		{name: "unclosed array", src: "[1", want: "unexpected end of input in array", wantOffset: unknownOffset},
		{name: "unclosed object", src: `{"a":1`, want: "unexpected end of input in object", wantOffset: unknownOffset},
		{name: "non-string key", src: "{1:2}", want: "expected string key in object", wantOffset: 1},
		{name: "invalid key escape", src: `{"a\q":1}`, want: `invalid escape \q`, wantOffset: 4},
		{name: "invalid value in object", src: `{"a":@}`, want: "unexpected character", wantOffset: 5},
		{name: "missing colon", src: `{"a" 1}`, want: "expected ':' after object key", wantOffset: 5},
		{name: "missing comma in object", src: `{"a":1 "b":2}`, want: "expected ',' or '}' in object", wantOffset: 7},
		{name: "missing comma in array", src: "[1 2]", want: "expected ',' or ']' in array", wantOffset: 3},
		{name: "duplicate member", src: `{"a":0,"b":1,"a":2}`, want: `duplicate object member "a"`, wantOffset: 13},
		{name: "unterminated string", src: `"abc`, want: "unterminated string literal", wantOffset: 4},
		{name: "control character", src: "\"a\x01b\"", want: "control character in string literal", wantOffset: 2},
		{name: "invalid utf8", src: "\"a\xffb\"", want: "invalid UTF-8 in string literal", wantOffset: 2},
		{name: "invalid escape", src: `"a\qb"`, want: `invalid escape \q`, wantOffset: 3},
		{name: "escape at eof", src: `"a\`, want: "unterminated escape at end of input", wantOffset: 2},
		{name: "short unicode escape", src: `"a\u12"`, want: `short \u escape`, wantOffset: 3},
		{name: "invalid unicode escape", src: `"a\u12g4"`, want: `invalid \u escape`, wantOffset: 3},
		{name: "lone high surrogate", src: `"a\ud83d"`, want: "invalid surrogate pair", wantOffset: unknownOffset},
		{name: "lone low surrogate", src: `"a\ude00"`, want: "invalid surrogate pair", wantOffset: unknownOffset},
		{name: "surrogate pair with bmp rune", src: `"a\ud83d\u0041"`, want: "invalid surrogate pair", wantOffset: unknownOffset},
		{name: "surrogate pair with invalid hex", src: `"a\ud83d\uZZZZ"`, want: `invalid \u escape in surrogate pair`, wantOffset: unknownOffset},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := Parse([]byte(tt.src))
			if err == nil {
				t.Fatalf("Parse(%q) = %v, want syntax error", tt.src, n)
			}
			if n != nil {
				t.Errorf("Parse(%q) = %v, want nil node on error", tt.src, n)
			}
			var syntaxErr *SyntaxError
			if !errors.As(err, &syntaxErr) {
				t.Fatalf("Parse(%q) error type = %T, want *SyntaxError", tt.src, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Parse(%q) error = %q, want contains %q", tt.src, err, tt.want)
			}
			if tt.wantOffset != unknownOffset && syntaxErr.Offset != tt.wantOffset {
				t.Errorf("Parse(%q) offset = %d, want %d", tt.src, syntaxErr.Offset, tt.wantOffset)
			}
		})
	}
}

// TestParseMaxDepth checks the inclusive container limit and the default chosen
// for nonpositive limits.
func TestParseMaxDepth(t *testing.T) {
	nested := func(depth int) []byte {
		return []byte(strings.Repeat("[", depth) + strings.Repeat("]", depth))
	}
	if _, err := ParseWithDepth(nested(3), 3); err != nil {
		t.Fatalf("ParseWithDepth(depth=3, max=3): %v", err)
	}
	_, err := ParseWithDepth(nested(4), 3)
	if err == nil {
		t.Fatal("ParseWithDepth(depth=4, max=3) = nil error, want depth error")
	}
	var syntaxErr *SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("error type = %T, want *SyntaxError", err)
	}
	if want := "exceeded max nesting depth 3"; !strings.Contains(syntaxErr.Message, want) {
		t.Errorf("message = %q, want contains %q", syntaxErr.Message, want)
	}

	if _, err := ParseWithDepth([]byte("[]"), 0); err != nil {
		t.Fatalf("ParseWithDepth(max=0): %v", err)
	}
	if _, err := ParseWithDepth([]byte("[]"), -1); err != nil {
		t.Fatalf("ParseWithDepth(max=-1): %v", err)
	}
	if _, err := ParseWithDepth([]byte(`{"a":{"b":1}}`), 1); err == nil {
		t.Fatal("ParseWithDepth(object depth=2, max=1) = nil error, want depth error")
	}
	if _, err := ParseWithDepth(nested(DefaultMaxDepth), 0); err != nil {
		t.Fatalf("ParseWithDepth(max=%d): %v", DefaultMaxDepth, err)
	}
	if _, err := ParseWithDepth(nested(DefaultMaxDepth+1), 0); err == nil {
		t.Fatalf("ParseWithDepth(depth=%d) = nil error, want depth error", DefaultMaxDepth+1)
	}
}

// TestParseStringHelper checks that the string entry point preserves the same
// root source fragment as the byte entry point.
func TestParseStringHelper(t *testing.T) {
	fromBytes, err := Parse([]byte(`{"a":[1,true]}`))
	if err != nil {
		t.Fatal(err)
	}
	fromString, err := ParseString(`{"a":[1,true]}`)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(fromString.Raw()), string(fromBytes.Raw()); got != want {
		t.Errorf("ParseString Raw() = %q, want %q", got, want)
	}
}

// TestParseRawAliasesInput distinguishes retained source bytes from independently
// stored string and numeric values.
func TestParseRawAliasesInput(t *testing.T) {
	data := []byte(`{"n":111,"s":"text"}`)
	root, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	num := root.Get("n")
	str := root.Get("s")

	// Deliberately violate the caller's immutability requirement to observe
	// which representations alias the input; this is not supported application use.
	for i, c := range data {
		switch {
		case c >= 'a' && c <= 'z':
			data[i] = 'x'
		case c >= '0' && c <= '9':
			data[i] = '9'
		}
	}
	if got := string(num.Raw()); got != "999" {
		t.Errorf("数字 Raw() = %q, want %q（应与输入共享内存）", got, "999")
	}
	if got := string(str.Raw()); got != `"xxxx"` {
		t.Errorf("字符串 Raw() = %q, want %q（应与输入共享内存）", got, `"xxxx"`)
	}
	if got := str.String(); got != "text" {
		t.Errorf("String() = %q, want %q（不应受输入修改影响）", got, "text")
	}
	if got := num.Number().digits; got != "111" {
		t.Errorf("Number().digits = %q, want %q（不应受输入修改影响）", got, "111")
	}
}

// TestParseDoesNotModifyInput checks that decoding leaves the caller's bytes intact.
func TestParseDoesNotModifyInput(t *testing.T) {
	const src = ` {"a":[1,"\\u4e2d",null],"b":{}} `
	data := []byte(src)
	if _, err := Parse(data); err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != src {
		t.Errorf("Parse 修改了输入：%q, want %q", got, src)
	}
}

// TestNodeNilGuards checks the two accessors that explicitly accept nil receivers.
func TestNodeNilGuards(t *testing.T) {
	var n *Node
	if n.IsNull() {
		t.Error("nil IsNull() = true, want false")
	}
	if got := n.Len(); got != -1 {
		t.Errorf("nil Len() = %d, want -1", got)
	}
}

// TestParserGuards invokes internal routines directly to exercise rejection
// paths normally preempted by the string scanner.
func TestParserGuards(t *testing.T) {
	t.Run("parseString 要求双引号", func(t *testing.T) {
		p := &parser{src: []byte("x")}
		if _, err := p.parseString(); err == nil {
			t.Fatal("parseString() = nil error, want expected '\"'")
		} else if !strings.Contains(err.Error(), `expected '"'`) {
			t.Errorf("error = %q, want expected '\"'", err)
		}
	})
	t.Run("unescape 拒绝截断的转义", func(t *testing.T) {
		p := &parser{}
		if _, err := p.unescape([]byte(`a\`), 0); err == nil {
			t.Fatal("unescape() = nil error, want truncated escape sequence")
		} else if !strings.Contains(err.Error(), "truncated escape sequence") {
			t.Errorf("error = %q, want truncated escape sequence", err)
		}
	})
	t.Run("unescape 拒绝控制字符", func(t *testing.T) {
		p := &parser{}
		if _, err := p.unescape([]byte("a\x01"), 0); err == nil {
			t.Fatal("unescape() = nil error, want control character error")
		} else if !strings.Contains(err.Error(), "control character") {
			t.Errorf("error = %q, want control character", err)
		}
	})
	t.Run("unescape 拒绝非法 UTF-8", func(t *testing.T) {
		p := &parser{}
		if _, err := p.unescape([]byte{'a', 0xff}, 0); err == nil {
			t.Fatal("unescape() = nil error, want invalid UTF-8 error")
		} else if !strings.Contains(err.Error(), "invalid UTF-8") {
			t.Errorf("error = %q, want invalid UTF-8", err)
		}
	})
}

// TestKindString checks diagnostic names, including the fallback for unknown kinds.
func TestKindString(t *testing.T) {
	tests := []struct {
		kind Kind
		want string
	}{
		{kind: KindNull, want: "null"},
		{kind: KindBool, want: "bool"},
		{kind: KindNumber, want: "number"},
		{kind: KindString, want: "string"},
		{kind: KindArray, want: "array"},
		{kind: KindObject, want: "object"},
		{kind: Kind(200), want: "unknown"},
	}
	for _, tt := range tests {
		if got := tt.kind.String(); got != tt.want {
			t.Errorf("Kind(%d).String() = %q, want %q", tt.kind, got, tt.want)
		}
	}
}

// TestNumberCmp checks exact decimal ordering and antisymmetry for equivalent
// spellings, large coefficients, and exponents beyond machine integer range.
func TestNumberCmp(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{a: "1", b: "1", want: 0},
		{a: "1", b: "2", want: -1},
		{a: "2", b: "10", want: -1},
		{a: "10", b: "9", want: 1},
		{a: "-1", b: "1", want: -1},
		{a: "-2", b: "-1", want: -1},
		{a: "-0", b: "0", want: 0},
		{a: "1.5", b: "1.50", want: 0},
		{a: "1e2", b: "100", want: 0},
		{a: "0.1", b: "1e-1", want: 0},
		{a: "0.1", b: "0.100001", want: -1},
		{a: "9.99", b: "10", want: -1},
		{a: "1e10", b: "1e11", want: -1},
		{
			a: "123456789012345678901234567890", b: "123456789012345678901234567891",
			want: -1,
		},
		{
			a: "1e999999999999999999999", b: "1e1000000000000000000000",
			want: -1,
		},
		{
			a: "1e999999999999999999999", b: "1e999999999999999999998",
			want: 1,
		},
		{a: "100", b: "1e999999999999999999999", want: -1},
		{a: "-1e999999999999999999999", b: "0", want: -1},
		{a: "1e-999999999999999999999", b: "0", want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.a+" vs "+tt.b, func(t *testing.T) {
			a, b := ParseNumber(tt.a), ParseNumber(tt.b)
			if got := a.Cmp(b); got != tt.want {
				t.Errorf("Cmp(%s, %s) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
			if got, want := b.Cmp(a), -tt.want; got != want {
				t.Errorf("Cmp(%s, %s) = %d, want %d（反对称性）", tt.b, tt.a, got, want)
			}
		})
	}
}
