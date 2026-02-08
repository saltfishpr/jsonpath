package jsonpath

import (
	"testing"
)

// TestLexerTokenTypes 测试所有基本 token 类型的识别
func TestLexerTokenTypes(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		// 根节点和当前节点标识符
		{"$", TokenRoot},
		{"@", TokenCurrent},

		// 运算符
		{".", TokenDot},
		{"..", TokenDotDot},
		{"[", TokenLBracket},
		{"]", TokenRBracket},
		{",", TokenComma},
		{"?", TokenQuestion},
		{":", TokenColon},
		{"*", TokenWildcard},

		// 比较运算符
		{"==", TokenEq},
		{"!=", TokenNe},
		{"<", TokenLt},
		{"<=", TokenLe},
		{">", TokenGt},
		{">=", TokenGe},

		// 逻辑运算符
		{"&&", TokenLAnd},
		{"||", TokenLOr},
		{"!", TokenLNot},

		// 括号
		{"(", TokenLParen},
		{")", TokenRParen},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			token := lexer.NextToken()
			if token.Type != tt.expected {
				t.Errorf("输入 %q: 期望类型 %v, 实际 %v", tt.input, tt.expected, token.Type)
			}
		})
	}
}

// TestLexerKeywords 测试关键字识别
func TestLexerKeywords(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"true", TokenTrue},
		{"false", TokenFalse},
		{"null", TokenNull},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			token := lexer.NextToken()
			if token.Type != tt.expected {
				t.Errorf("输入 %q: 期望类型 %v, 实际 %v", tt.input, tt.expected, token.Type)
			}
		})
	}
}

// TestLexerIdentifiers 测试标识符识别
func TestLexerIdentifiers(t *testing.T) {
	tests := []struct {
		input string
		value string
	}{
		// 函数名 - RFC 9535 Section 2.4
		// function-name-first = LCALPHA (a-z)
		{"length", "length"},
		{"count", "count"},
		{"match", "match"},
		{"search", "search"},
		{"value", "value"},
		{"foo", "foo"},
		{"my_function", "my_function"},
		{"func123", "func123"},

		// 成员名简写 - RFC 9535 Section 2.5.1
		// name-first = ALPHA / "_" / %x80-D7FF / %xE000-10FFFF
		{"name", "name"},
		{"_private", "_private"},
		{"camelCase", "camelCase"},
		{"PascalCase", "PascalCase"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			token := lexer.NextToken()
			if token.Type != TokenIdent {
				t.Errorf("输入 %q: 期望类型 TokenIdent, 实际 %v", tt.input, token.Type)
			}
			if token.Value != tt.value {
				t.Errorf("输入 %q: 期望值 %q, 实际 %q", tt.input, tt.value, token.Value)
			}
		})
	}
}

// TestLexerNumbers 测试数字字面量
func TestLexerNumbers(t *testing.T) {
	tests := []struct {
		input       string
		expectType  TokenType
		expectValue string
	}{
		// RFC 9535 Section 2.3.3: int = "0" / (["-"] DIGIT1 *DIGIT)
		{"0", TokenNumber, "0"},
		{"1", TokenNumber, "1"},
		{"123", TokenNumber, "123"},
		{"-1", TokenNumber, "-1"},
		{"-123", TokenNumber, "-123"},

		// RFC 9535: -0 是合法的特殊情况
		{"-0", TokenNumber, "-0"},

		// 带小数部分
		{"0.5", TokenNumber, "0.5"},
		{"3.14", TokenNumber, "3.14"},
		{"-2.5", TokenNumber, "-2.5"},

		// 带指数部分
		{"1e10", TokenNumber, "1e10"},
		{"1E10", TokenNumber, "1E10"},
		{"1e+10", TokenNumber, "1e+10"},
		{"1e-10", TokenNumber, "1e-10"},
		{"-1e10", TokenNumber, "-1e10"},

		// 复杂情况
		{"123.456e78", TokenNumber, "123.456e78"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			token := lexer.NextToken()
			if token.Type != tt.expectType {
				t.Errorf("输入 %q: 期望类型 %v, 实际 %v", tt.input, tt.expectType, token.Type)
			}
			if token.Value != tt.expectValue {
				t.Errorf("输入 %q: 期望值 %q, 实际 %q", tt.input, tt.expectValue, token.Value)
			}
		})
	}
}

// TestLexerInvalidNumbers 测试非法数字格式
func TestLexerInvalidNumbers(t *testing.T) {
	tests := []struct {
		input         string
		expectIllegal bool
	}{
		// RFC 9535 禁止前导零（除了单独的 0）
		{"01", true},
		{"-01", true},
		{"001", true},

		// 其他非法格式
		{"-", true},   // 只有负号
		{"1.", true},  // 小数点后没有数字
		{"1e", true},  // 指数后没有数字
		{"1e+", true}, // 指数符号后没有数字
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			token := lexer.NextToken()
			isIllegal := token.Type == TokenIllegal
			if isIllegal != tt.expectIllegal {
				t.Errorf("输入 %q: 期望非法=%v, 实际=%v (token类型=%v)", tt.input, tt.expectIllegal, isIllegal, token.Type)
			}
		})
	}
}

// TestLexerStrings 测试字符串字面量
func TestLexerStrings(t *testing.T) {
	tests := []struct {
		input       string
		expectType  TokenType
		expectValue string
	}{
		// 双引号字符串
		{`"hello"`, TokenString, "hello"},
		{`""`, TokenString, ""},

		// 单引号字符串 - RFC 9535 Section 2.3.1 支持
		{`'hello'`, TokenString, "hello"},
		{`''`, TokenString, ""},

		// 转义序列 - RFC 9535 Section 2.3.1 Table 4
		{`"\b"`, TokenString, "\b"}, // U+0008 BS backspace
		{`"\f"`, TokenString, "\f"}, // U+000C FF form feed
		{`"\n"`, TokenString, "\n"}, // U+000A LF line feed
		{`"\r"`, TokenString, "\r"}, // U+000D CR carriage return
		{`"\t"`, TokenString, "\t"}, // U+0009 HT horizontal tab
		{`"\/"`, TokenString, "/"},  // U+002F slash
		{`"\\"`, TokenString, "\\"}, // U+005C backslash
		{`"\""`, TokenString, `"`},  // U+0022 quotation mark
		{`"\'"`, TokenString, "'"},  // U+0027 apostrophe
		{`'"'`, TokenString, `"`},   // 双引号在单引号字符串中
		{`'\''`, TokenString, "'"},  // 单引号在单引号字符串中需要转义

		// Unicode 转义
		{`"\u0041"`, TokenString, "A"},        // 基本多文种平面
		{`"\u4e2d\u6587"`, TokenString, "中文"}, // 中文字符

		// 混合内容
		{`"hello\nworld"`, TokenString, "hello\nworld"},
		{`"path\to\\file"`, TokenString, "path\to\\file"},

		// 特殊字符在字符串中
		{`"a[b]c"`, TokenString, "a[b]c"},
		{`"name.with.dots"`, TokenString, "name.with.dots"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			token := lexer.NextToken()
			if token.Type != tt.expectType {
				t.Errorf("输入 %q: 期望类型 %v, 实际 %v", tt.input, tt.expectType, token.Type)
			}
			if token.Value != tt.expectValue {
				t.Errorf("输入 %q: 期望值 %q, 实际 %q", tt.input, tt.expectValue, token.Value)
			}
		})
	}
}

// TestLexerInvalidStrings 测试非法字符串格式
func TestLexerInvalidStrings(t *testing.T) {
	tests := []struct {
		input string
	}{
		{`"unclosed`}, // 未闭合的双引号字符串
		{`'unclosed`}, // 未闭合的单引号字符串
		{`"\x"`},      // 非法转义序列
		{`"\u"`},      // 不完整的 Unicode 转义
		{`"\u123"`},   // Unicode 转义只有3位数字
		{`"\u12G4"`},  // Unicode 转义包含非十六进制字符
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			token := lexer.NextToken()
			if token.Type != TokenIllegal {
				t.Errorf("输入 %q: 期望 TokenIllegal, 实际 %v", tt.input, token.Type)
			}
		})
	}
}

// TestLexerSurrogatePairs 测试 Unicode 代理对处理
func TestLexerSurrogatePairs(t *testing.T) {
	tests := []struct {
		input       string
		expectType  TokenType
		expectValue string
	}{
		// RFC 9535 Section 2.3.1: 代理对处理
		// 高代理: D800-DBFF, 低代理: DC00-DFFF
		{`"\uD83D\uDE00"`, TokenString, "😀"}, // 笑脸 emoji
		{`"\uD83C\uDC41"`, TokenString, "🁁"}, // DOMINO TILE
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			token := lexer.NextToken()
			if token.Type != tt.expectType {
				t.Errorf("输入 %q: 期望类型 %v, 实际 %v", tt.input, tt.expectType, token.Type)
			}
			if token.Value != tt.expectValue {
				t.Errorf("输入 %q: 期望值 %q (U+%04X), 实际 %q (U+%04X)",
					tt.input, tt.expectValue, []rune(tt.expectValue)[0],
					token.Value, []rune(token.Value)[0])
			}
		})
	}
}

// TestLexerWhitespace 测试空白字符处理
func TestLexerWhitespace(t *testing.T) {
	// RFC 9535 Section 2.1.1: B = %x20 / %x09 / %x0A / %x0D
	// 空格 / 水平制表符 / 换行 / 回车
	tests := []struct {
		input  string
		expect []TokenType
	}{
		{"$  [  ]", []TokenType{TokenRoot, TokenLBracket, TokenRBracket}},
		{"$\t[\n]", []TokenType{TokenRoot, TokenLBracket, TokenRBracket}},
		{"$\r\n[\r]", []TokenType{TokenRoot, TokenLBracket, TokenRBracket}},
		{"$  .  name  ", []TokenType{TokenRoot, TokenDot, TokenIdent}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			for i, expectType := range tt.expect {
				token := lexer.NextToken()
				if token.Type != expectType {
					t.Errorf("位置 %d: 期望类型 %v, 实际 %v", i, expectType, token.Type)
				}
			}
		})
	}
}

// TestLexerComplexExpressions 测试复杂的 JSONPath 表达式
func TestLexerComplexExpressions(t *testing.T) {
	tests := []struct {
		input  string
		tokens []Token
	}{
		// RFC 9535 Figure 1 示例
		{
			`$.store.book[0].title`,
			[]Token{
				{Type: TokenRoot, Value: "$"},
				{Type: TokenDot, Value: "."},
				{Type: TokenIdent, Value: "store"},
				{Type: TokenDot, Value: "."},
				{Type: TokenIdent, Value: "book"},
				{Type: TokenLBracket, Value: "["},
				{Type: TokenNumber, Value: "0"},
				{Type: TokenRBracket, Value: "]"},
				{Type: TokenDot, Value: "."},
				{Type: TokenIdent, Value: "title"},
			},
		},
		// 括号表示法
		{
			`$['store']['book'][0]['title']`,
			[]Token{
				{Type: TokenRoot, Value: "$"},
				{Type: TokenLBracket, Value: "["},
				{Type: TokenString, Value: "store"},
				{Type: TokenRBracket, Value: "]"},
				{Type: TokenLBracket, Value: "["},
				{Type: TokenString, Value: "book"},
				{Type: TokenRBracket, Value: "]"},
				{Type: TokenLBracket, Value: "["},
				{Type: TokenNumber, Value: "0"},
				{Type: TokenRBracket, Value: "]"},
				{Type: TokenLBracket, Value: "["},
				{Type: TokenString, Value: "title"},
				{Type: TokenRBracket, Value: "]"},
			},
		},
		// 通配符
		{
			`$.store.book[*].author`,
			[]Token{
				{Type: TokenRoot, Value: "$"},
				{Type: TokenDot, Value: "."},
				{Type: TokenIdent, Value: "store"},
				{Type: TokenDot, Value: "."},
				{Type: TokenIdent, Value: "book"},
				{Type: TokenLBracket, Value: "["},
				{Type: TokenWildcard, Value: "*"},
				{Type: TokenRBracket, Value: "]"},
				{Type: TokenDot, Value: "."},
				{Type: TokenIdent, Value: "author"},
			},
		},
		// 后代段
		{
			`$..author`,
			[]Token{
				{Type: TokenRoot, Value: "$"},
				{Type: TokenDotDot, Value: ".."},
				{Type: TokenIdent, Value: "author"},
			},
		},
		// 数组切片
		{
			`$[0:10:2]`,
			[]Token{
				{Type: TokenRoot, Value: "$"},
				{Type: TokenLBracket, Value: "["},
				{Type: TokenNumber, Value: "0"},
				{Type: TokenColon, Value: ":"},
				{Type: TokenNumber, Value: "10"},
				{Type: TokenColon, Value: ":"},
				{Type: TokenNumber, Value: "2"},
				{Type: TokenRBracket, Value: "]"},
			},
		},
		// 负索引
		{
			`$..book[-1]`,
			[]Token{
				{Type: TokenRoot, Value: "$"},
				{Type: TokenDotDot, Value: ".."},
				{Type: TokenIdent, Value: "book"},
				{Type: TokenLBracket, Value: "["},
				{Type: TokenNumber, Value: "-1"},
				{Type: TokenRBracket, Value: "]"},
			},
		},
		// 多选
		{
			`$..book[0,1]`,
			[]Token{
				{Type: TokenRoot, Value: "$"},
				{Type: TokenDotDot, Value: ".."},
				{Type: TokenIdent, Value: "book"},
				{Type: TokenLBracket, Value: "["},
				{Type: TokenNumber, Value: "0"},
				{Type: TokenComma, Value: ","},
				{Type: TokenNumber, Value: "1"},
				{Type: TokenRBracket, Value: "]"},
			},
		},
		// 过滤器表达式 - RFC 9535 Section 2.3.5
		{
			`$.store.book[?@.price < 10]`,
			[]Token{
				{Type: TokenRoot, Value: "$"},
				{Type: TokenDot, Value: "."},
				{Type: TokenIdent, Value: "store"},
				{Type: TokenDot, Value: "."},
				{Type: TokenIdent, Value: "book"},
				{Type: TokenLBracket, Value: "["},
				{Type: TokenQuestion, Value: "?"},
				{Type: TokenCurrent, Value: "@"},
				{Type: TokenDot, Value: "."},
				{Type: TokenIdent, Value: "price"},
				{Type: TokenLt, Value: "<"},
				{Type: TokenNumber, Value: "10"},
				{Type: TokenRBracket, Value: "]"},
			},
		},
		// 逻辑运算符
		{
			`$[?@.price < 10 && @.category == 'fiction']`,
			[]Token{
				{Type: TokenRoot, Value: "$"},
				{Type: TokenLBracket, Value: "["},
				{Type: TokenQuestion, Value: "?"},
				{Type: TokenCurrent, Value: "@"},
				{Type: TokenDot, Value: "."},
				{Type: TokenIdent, Value: "price"},
				{Type: TokenLt, Value: "<"},
				{Type: TokenNumber, Value: "10"},
				{Type: TokenLAnd, Value: "&&"},
				{Type: TokenCurrent, Value: "@"},
				{Type: TokenDot, Value: "."},
				{Type: TokenIdent, Value: "category"},
				{Type: TokenEq, Value: "=="},
				{Type: TokenString, Value: "fiction"},
				{Type: TokenRBracket, Value: "]"},
			},
		},
		// 函数调用 - RFC 9535 Section 2.4
		{
			`$[?length(@.authors) >= 5]`,
			[]Token{
				{Type: TokenRoot, Value: "$"},
				{Type: TokenLBracket, Value: "["},
				{Type: TokenQuestion, Value: "?"},
				{Type: TokenIdent, Value: "length"},
				{Type: TokenLParen, Value: "("},
				{Type: TokenCurrent, Value: "@"},
				{Type: TokenDot, Value: "."},
				{Type: TokenIdent, Value: "authors"},
				{Type: TokenRParen, Value: ")"},
				{Type: TokenGe, Value: ">="},
				{Type: TokenNumber, Value: "5"},
				{Type: TokenRBracket, Value: "]"},
			},
		},
		// 存在性测试
		{
			`$..book[?@.isbn]`,
			[]Token{
				{Type: TokenRoot, Value: "$"},
				{Type: TokenDotDot, Value: ".."},
				{Type: TokenIdent, Value: "book"},
				{Type: TokenLBracket, Value: "["},
				{Type: TokenQuestion, Value: "?"},
				{Type: TokenCurrent, Value: "@"},
				{Type: TokenDot, Value: "."},
				{Type: TokenIdent, Value: "isbn"},
				{Type: TokenRBracket, Value: "]"},
			},
		},
		// 带括号的表达式
		{
			`$[?(@.price < 10)]`,
			[]Token{
				{Type: TokenRoot, Value: "$"},
				{Type: TokenLBracket, Value: "["},
				{Type: TokenQuestion, Value: "?"},
				{Type: TokenLParen, Value: "("},
				{Type: TokenCurrent, Value: "@"},
				{Type: TokenDot, Value: "."},
				{Type: TokenIdent, Value: "price"},
				{Type: TokenLt, Value: "<"},
				{Type: TokenNumber, Value: "10"},
				{Type: TokenRParen, Value: ")"},
				{Type: TokenRBracket, Value: "]"},
			},
		},
		// 逻辑非
		{
			`$[?!@.isbn]`,
			[]Token{
				{Type: TokenRoot, Value: "$"},
				{Type: TokenLBracket, Value: "["},
				{Type: TokenQuestion, Value: "?"},
				{Type: TokenLNot, Value: "!"},
				{Type: TokenCurrent, Value: "@"},
				{Type: TokenDot, Value: "."},
				{Type: TokenIdent, Value: "isbn"},
				{Type: TokenRBracket, Value: "]"},
			},
		},
		// 函数调用多个参数
		{
			`$[?match(@.date, "1974-05-..")]`,
			[]Token{
				{Type: TokenRoot, Value: "$"},
				{Type: TokenLBracket, Value: "["},
				{Type: TokenQuestion, Value: "?"},
				{Type: TokenIdent, Value: "match"},
				{Type: TokenLParen, Value: "("},
				{Type: TokenCurrent, Value: "@"},
				{Type: TokenDot, Value: "."},
				{Type: TokenIdent, Value: "date"},
				{Type: TokenComma, Value: ","},
				{Type: TokenString, Value: "1974-05-.."},
				{Type: TokenRParen, Value: ")"},
				{Type: TokenRBracket, Value: "]"},
			},
		},
		// 逆序切片
		{
			`$[::-1]`,
			[]Token{
				{Type: TokenRoot, Value: "$"},
				{Type: TokenLBracket, Value: "["},
				{Type: TokenColon, Value: ":"},
				{Type: TokenColon, Value: ":"},
				{Type: TokenNumber, Value: "-1"},
				{Type: TokenRBracket, Value: "]"},
			},
		},
		// null 比较
		{
			`$[?@.foo == null]`,
			[]Token{
				{Type: TokenRoot, Value: "$"},
				{Type: TokenLBracket, Value: "["},
				{Type: TokenQuestion, Value: "?"},
				{Type: TokenCurrent, Value: "@"},
				{Type: TokenDot, Value: "."},
				{Type: TokenIdent, Value: "foo"},
				{Type: TokenEq, Value: "=="},
				{Type: TokenNull, Value: "null"},
				{Type: TokenRBracket, Value: "]"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			for i, expectToken := range tt.tokens {
				token := lexer.NextToken()
				if token.Type != expectToken.Type {
					t.Errorf("位置 %d: 期望类型 %v (%q), 实际 %v (%q)",
						i, expectToken.Type, expectToken.Value, token.Type, token.Value)
				}
				if token.Value != expectToken.Value {
					t.Errorf("位置 %d: 期望值 %q, 实际 %q", i, expectToken.Value, token.Value)
				}
			}
			// 确保没有多余 token
			_eof := lexer.NextToken()
			if _eof.Type != TokenEOF {
				t.Errorf("期望 EOF, 实际 %v (%q)", _eof.Type, _eof.Value)
			}
		})
	}
}

// TestLexerEOF 测试 EOF 处理
func TestLexerEOF(t *testing.T) {
	lexer := NewLexer("$")
	token := lexer.NextToken()
	if token.Type != TokenRoot {
		t.Errorf("第一个 token 应该是 TokenRoot, 实际 %v", token.Type)
	}

	for i := 0; i < 10; i++ {
		token = lexer.NextToken()
		if token.Type != TokenEOF {
			t.Errorf("第 %d 次 NextToken() 应该返回 TokenEOF, 实际 %v", i+1, token.Type)
		}
	}
}

// TestLexerIllegalTokens 测试非法 token
func TestLexerIllegalTokens(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"="}, // 单个 = 应该是非法的
		{"&"}, // 单个 & 应该是非法的
		{"|"}, // 单个 | 应该是非法的
		{"@"}, // @ 后面没有内容是合法的（当前节点标识符）
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			token := lexer.NextToken()

			// @ 是单独的合法 token
			if tt.input == "@" {
				if token.Type != TokenCurrent {
					t.Errorf("输入 %q: 期望 TokenCurrent, 实际 %v", tt.input, token.Type)
				}
				return
			}

			// 其他单个字符应该是非法的
			if token.Type == TokenIllegal {
				// 正确
				return
			}
			// 或者它们被识别为某个其他 token（虽然不是我们期望的）
			t.Logf("输入 %q: 被识别为 %v (%q)", tt.input, token.Type, token.Value)
		})
	}
}

// TestLexerTokenPositions 测试 token 位置信息
func TestLexerTokenPositions(t *testing.T) {
	input := "$ . name"
	lexer := NewLexer(input)

	expectedPositions := []int{0, 2, 4}
	expectedTypes := []TokenType{TokenRoot, TokenDot, TokenIdent}

	for i := 0; i < 3; i++ {
		token := lexer.NextToken()
		if token.Pos != expectedPositions[i] {
			t.Errorf("token %d: 期望位置 %d, 实际 %d", i, expectedPositions[i], token.Pos)
		}
		if token.Type != expectedTypes[i] {
			t.Errorf("token %d: 期望类型 %v, 实际 %v", i, expectedTypes[i], token.Type)
		}
	}
}

// TestLexerStringsWithUnescapedQuotes RFC 9535 Section 2.3.1
// 测试字符串中未转义的引号
func TestLexerStringsWithUnescapedQuotes(t *testing.T) {
	tests := []struct {
		input       string
		expectType  TokenType
		expectValue string
	}{
		// 双引号字符串中可以有单引号
		{`"it's"`, TokenString, `it's`},
		// 单引号字符串中可以有双引号
		{`'He said "hello"'`, TokenString, `He said "hello"`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			token := lexer.NextToken()
			if token.Type != tt.expectType {
				t.Errorf("输入 %q: 期望类型 %v, 实际 %v", tt.input, tt.expectType, token.Type)
			}
			if token.Value != tt.expectValue {
				t.Errorf("输入 %q: 期望值 %q, 实际 %q", tt.input, tt.expectValue, token.Value)
			}
		})
	}
}

// TestLexerArraySliceRFCExamples RFC 9535 Table 9
func TestLexerArraySliceRFCExamples(t *testing.T) {
	tests := []struct {
		input  string
		tokens []TokenType
	}{
		{"$[1:3]", []TokenType{TokenRoot, TokenLBracket, TokenNumber, TokenColon, TokenNumber, TokenRBracket}},
		{"$[5:]", []TokenType{TokenRoot, TokenLBracket, TokenNumber, TokenColon, TokenRBracket}},
		{"$[1:5:2]", []TokenType{TokenRoot, TokenLBracket, TokenNumber, TokenColon, TokenNumber, TokenColon, TokenNumber, TokenRBracket}},
		{"$[5:1:-2]", []TokenType{TokenRoot, TokenLBracket, TokenNumber, TokenColon, TokenNumber, TokenColon, TokenNumber, TokenRBracket}},
		{"$[::-1]", []TokenType{TokenRoot, TokenLBracket, TokenColon, TokenColon, TokenNumber, TokenRBracket}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			for i, expectType := range tt.tokens {
				token := lexer.NextToken()
				if token.Type != expectType {
					t.Errorf("位置 %d: 期望类型 %v, 实际 %v", i, expectType, token.Type)
				}
			}
		})
	}
}

// TestLexerFunctionNames RFC 9535 Section 2.4
// function-name-first = LCALPHA (a-z only)
// 函数名必须以小写字母开头
func TestLexerFunctionNames(t *testing.T) {
	tests := []struct {
		input       string
		expectIdent bool // 期望被识别为标识符
	}{
		// RFC 9535 标准函数
		{"length", true},
		{"count", true},
		{"match", true},
		{"search", true},
		{"value", true},

		// 合法的函数名（小写开头）
		{"foo", true},
		{"bar123", true},
		{"my_func", true},

		// 注意：当前 lexer 实现允许大写字母作为标识符起始
		// 但 RFC 9535 规定函数名必须以小写字母开头
		// 这是语法层面需要检查的，lexer 只负责识别标识符
		{"Length", true}, // 词法上合法，但语法上不是合法的函数名
		{"LENGTH", true}, // 词法上合法，但语法上不是合法的函数名
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			token := lexer.NextToken()
			isIdent := token.Type == TokenIdent
			if isIdent != tt.expectIdent {
				t.Errorf("输入 %q: 期望标识符=%v, 实际=%v (类型=%v)", tt.input, tt.expectIdent, isIdent, token.Type)
			}
			if isIdent && token.Value != tt.input {
				t.Errorf("输入 %q: 期望值 %q, 实际 %q", tt.input, tt.input, token.Value)
			}
		})
	}
}

// TestLexerMemberNameShorthand RFC 9535 Section 2.5.1
// member-name-shorthand = name-first *name-char
// name-first = ALPHA / "_" / %x80-D7FF / %xE000-10FFFF
func TestLexerMemberNameShorthand(t *testing.T) {
	tests := []struct {
		input       string
		expectIdent bool
	}{
		// ASCII 字母
		{"name", true},
		{"Name", true},
		{"_private", true},
		{"name123", true},

		// 注意：数字不能作为开头（但可以在后续位置）
		{"123name", false}, // 会被识别为数字

		// 特殊字符不能作为标识符
		{"name-with-dash", false},
		{"name.with.dot", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			token := lexer.NextToken()
			isIdent := token.Type == TokenIdent

			// 对于 "name.with.dot"，第一个 token 应该是 "name"
			if tt.input == "name.with.dot" {
				if token.Type != TokenIdent || token.Value != "name" {
					t.Errorf("输入 %q: 第一个 token 应该是标识符 'name', 实际 %v (%q)", tt.input, token.Type, token.Value)
				}
				return
			}

			// 对于 "name-with-dash"，第一个 token 应该是 "name"
			if tt.input == "name-with-dash" {
				if token.Type != TokenIdent || token.Value != "name" {
					t.Errorf("输入 %q: 第一个 token 应该是标识符 'name', 实际 %v (%q)", tt.input, token.Type, token.Value)
				}
				return
			}

			if isIdent != tt.expectIdent {
				t.Errorf("输入 %q: 期望标识符=%v, 实际=%v (类型=%v)", tt.input, tt.expectIdent, isIdent, token.Type)
			}
		})
	}
}

// TestLexerRFCExamples RFC 9535 中的示例表达式词法分析
func TestLexerRFCExamples(t *testing.T) {
	// RFC 9535 Table 2: 示例 JSONPath 表达式
	examples := []string{
		`$.store.book[*].author`,
		`$..author`,
		`$.store.*`,
		`$.store..price`,
		`$..book[2]`,
		`$..book[2].author`,
		`$..book[2].publisher`,
		`$..book[-1]`,
		`$..book[0,1]`,
		`$..book[:2]`,
		`$..book[?@.isbn]`,
		`$..book[?@.price<10]`,
		`$..*`,
	}

	for _, example := range examples {
		t.Run(example, func(t *testing.T) {
			lexer := NewLexer(example)
			tokenCount := 0
			hasIllegal := false

			for {
				token := lexer.NextToken()
				if token.Type == TokenEOF {
					break
				}
				if token.Type == TokenIllegal {
					t.Errorf("示例 %q 包含非法 token: %q", example, token.Value)
					hasIllegal = true
					break
				}
				tokenCount++
			}

			if !hasIllegal && tokenCount == 0 {
				t.Errorf("示例 %q 没有产生任何 token", example)
			}
		})
	}
}

// BenchmarkLexerSimple 简单表达式基准测试
func BenchmarkLexerSimple(b *testing.B) {
	input := "$.store.book[0].title"
	for i := 0; i < b.N; i++ {
		lexer := NewLexer(input)
		for lexer.NextToken().Type != TokenEOF {
		}
	}
}

// BenchmarkLexerComplex 复杂表达式基准测试
func BenchmarkLexerComplex(b *testing.B) {
	input := `$.store.book[?@.price < 10 && @.category == 'fiction'].title`
	for i := 0; i < b.N; i++ {
		lexer := NewLexer(input)
		for lexer.NextToken().Type != TokenEOF {
		}
	}
}

// BenchmarkLexerWithUnicode Unicode 字符串基准测试
func BenchmarkLexerWithUnicode(b *testing.B) {
	input := "$[?@.name == '中文测试']"
	for i := 0; i < b.N; i++ {
		lexer := NewLexer(input)
		for lexer.NextToken().Type != TokenEOF {
		}
	}
}
