package jsonpath

import (
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

// TokenType 表示 token 的类型
type TokenType int

// Token 类型常量
const (
	TokenIllegal TokenType = iota
	TokenEOF

	// 标识符
	TokenRoot    // $  - 根节点标识符
	TokenCurrent // @  - 当前节点标识符

	// 运算符
	TokenDot      // .  - 点号（子段简写）
	TokenDotDot   // .. - 双点号（后代段）
	TokenLBracket // [  - 左方括号
	TokenRBracket // ]  - 右方括号
	TokenComma    // ,  - 逗号
	TokenQuestion // ?  - 问号（过滤器）
	TokenColon    // :  - 冒号（切片）
	TokenWildcard // * - 通配符

	// 比较运算符
	TokenEq // ==  - 等于
	TokenNe // !=  - 不等于
	TokenLt // <   - 小于
	TokenLe // <=  - 小于等于
	TokenGt // >   - 大于
	TokenGe // >=  - 大于等于

	// 逻辑运算符
	TokenLAnd // &&  - 逻辑与
	TokenLOr  // ||  - 逻辑或
	TokenLNot // !   - 逻辑非

	// 括号
	TokenLParen // ( - 左圆括号
	TokenRParen // ) - 右圆括号

	// 字面量
	TokenIdent  // 标识符/名称/函数名
	TokenNumber // 数字
	TokenString // 字符串
	TokenTrue   // true
	TokenFalse  // false
	TokenNull   // null
)

// String 返回 TokenType 的字符串表示
func (t TokenType) String() string {
	switch t {
	case TokenIllegal:
		return "ILLEGAL"
	case TokenEOF:
		return "EOF"
	case TokenRoot:
		return "$"
	case TokenCurrent:
		return "@"
	case TokenDot:
		return "."
	case TokenDotDot:
		return ".."
	case TokenLBracket:
		return "["
	case TokenRBracket:
		return "]"
	case TokenComma:
		return ","
	case TokenQuestion:
		return "?"
	case TokenColon:
		return ":"
	case TokenWildcard:
		return "*"
	case TokenEq:
		return "=="
	case TokenNe:
		return "!="
	case TokenLt:
		return "<"
	case TokenLe:
		return "<="
	case TokenGt:
		return ">"
	case TokenGe:
		return ">="
	case TokenLAnd:
		return "&&"
	case TokenLOr:
		return "||"
	case TokenLNot:
		return "!"
	case TokenLParen:
		return "("
	case TokenRParen:
		return ")"
	case TokenIdent:
		return "identifier"
	case TokenNumber:
		return "number"
	case TokenString:
		return "string"
	case TokenTrue:
		return "true"
	case TokenFalse:
		return "false"
	case TokenNull:
		return "null"
	default:
		return "unknown"
	}
}

// Token 表示一个词法单元
type Token struct {
	Type  TokenType // token 类型
	Value string    // 原始字符串值
	Pos   int       // 在输入中的位置
}

// Lexer 词法分析器
type Lexer struct {
	input string // 输入字符串
	pos   int    // 当前读取位置
	width int    // 最后一个 rune 的宽度
}

// NewLexer 创建一个新的词法分析器
func NewLexer(input string) *Lexer {
	return &Lexer{input: input}
}

// NextToken 读取并返回下一个 token
func (l *Lexer) NextToken() Token {
	l.skipWhitespace()

	pos := l.pos
	r := l.next()

	if r == utf8.RuneError {
		return Token{Type: TokenEOF, Pos: pos}
	}

	switch r {
	case '$':
		return Token{Type: TokenRoot, Value: "$", Pos: pos}
	case '@':
		return Token{Type: TokenCurrent, Value: "@", Pos: pos}
	case '.':
		if l.peek() == '.' {
			l.next()
			return Token{Type: TokenDotDot, Value: "..", Pos: pos}
		}
		return Token{Type: TokenDot, Value: ".", Pos: pos}
	case '[':
		return Token{Type: TokenLBracket, Value: "[", Pos: pos}
	case ']':
		return Token{Type: TokenRBracket, Value: "]", Pos: pos}
	case ',':
		return Token{Type: TokenComma, Value: ",", Pos: pos}
	case ':':
		return Token{Type: TokenColon, Value: ":", Pos: pos}
	case '?':
		return Token{Type: TokenQuestion, Value: "?", Pos: pos}
	case '*':
		return Token{Type: TokenWildcard, Value: "*", Pos: pos}
	case '(':
		return Token{Type: TokenLParen, Value: "(", Pos: pos}
	case ')':
		return Token{Type: TokenRParen, Value: ")", Pos: pos}
	case '!':
		if l.peek() == '=' {
			l.next()
			return Token{Type: TokenNe, Value: "!=", Pos: pos}
		}
		return Token{Type: TokenLNot, Value: "!", Pos: pos}
	case '=':
		if l.peek() == '=' {
			l.next()
			return Token{Type: TokenEq, Value: "==", Pos: pos}
		}
		return Token{Type: TokenIllegal, Value: "=", Pos: pos}
	case '<':
		if l.peek() == '=' {
			l.next()
			return Token{Type: TokenLe, Value: "<=", Pos: pos}
		}
		return Token{Type: TokenLt, Value: "<", Pos: pos}
	case '>':
		if l.peek() == '=' {
			l.next()
			return Token{Type: TokenGe, Value: ">=", Pos: pos}
		}
		return Token{Type: TokenGt, Value: ">", Pos: pos}
	case '&':
		if l.peek() == '&' {
			l.next()
			return Token{Type: TokenLAnd, Value: "&&", Pos: pos}
		}
		return Token{Type: TokenIllegal, Value: string(r), Pos: pos}
	case '|':
		if l.peek() == '|' {
			l.next()
			return Token{Type: TokenLOr, Value: "||", Pos: pos}
		}
		return Token{Type: TokenIllegal, Value: string(r), Pos: pos}
	case '"', '\'':
		l.backup()
		return l.readString()
	}

	// 数字：以 - 或数字开头
	if r == '-' || unicode.IsDigit(r) {
		l.backup()
		return l.readNumber()
	}

	// 标识符/关键字/函数名
	if isNameFirst(r) {
		l.backup()
		return l.readIdent()
	}

	return Token{Type: TokenIllegal, Value: string(r), Pos: pos}
}

// skipWhitespace 跳过空白字符
func (l *Lexer) skipWhitespace() {
	for {
		r := l.peek()
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			return
		}
		l.next()
	}
}

// readString 读取字符串字面量（支持单引号和双引号）
func (l *Lexer) readString() Token {
	var sb strings.Builder
	pos := l.pos

	quote := l.next() // 获取引号字符
	for {
		r := l.next()
		if r == utf8.RuneError { // 未闭合的字符串
			return Token{Type: TokenIllegal, Value: sb.String(), Pos: pos}
		}

		if r == quote {
			break
		}

		if r == '\\' {
			// 处理转义序列
			escaped := l.next()
			if escaped == utf8.RuneError {
				return Token{Type: TokenIllegal, Value: sb.String(), Pos: pos}
			}
			switch escaped {
			case 'b':
				sb.WriteRune('\b')
			case 'f':
				sb.WriteRune('\f')
			case 'n':
				sb.WriteRune('\n')
			case 'r':
				sb.WriteRune('\r')
			case 't':
				sb.WriteRune('\t')
			case '/', '\\', '\'', '"':
				sb.WriteRune(escaped)
			case 'u':
				rv := l.readUnicodeEscape()
				if rv == unicode.ReplacementChar {
					return Token{Type: TokenIllegal, Value: sb.String(), Pos: pos}
				}
				sb.WriteRune(rv)
			default:
				return Token{Type: TokenIllegal, Value: sb.String(), Pos: pos}
			}
		} else {
			sb.WriteRune(r)
		}
	}
	return Token{Type: TokenString, Value: sb.String(), Pos: pos}
}

func (l *Lexer) readUnicodeEscape() rune {
	if l.pos+4 > len(l.input) {
		return unicode.ReplacementChar
	}
	r1, ok := parseHex4(l.input[l.pos : l.pos+4])
	if !ok {
		return unicode.ReplacementChar
	}
	l.pos += 4

	if utf16.IsSurrogate(r1) {
		if l.pos+6 > len(l.input) || l.input[l.pos:l.pos+2] != "\\u" {
			return unicode.ReplacementChar
		}
		r2, ok := parseHex4(l.input[l.pos+2 : l.pos+6])
		if ok {
			combined := utf16.DecodeRune(r1, r2)
			l.pos += 6
			return combined
		}
		return unicode.ReplacementChar
	}
	return r1
}

func parseHex4(s string) (rune, bool) {
	if len(s) != 4 {
		return 0, false
	}
	var r rune
	for _, c := range []byte(s) {
		r <<= 4
		switch {
		case '0' <= c && c <= '9':
			r += rune(c - '0')
		case 'a' <= c && c <= 'f':
			r += rune(c - 'a' + 10)
		case 'A' <= c && c <= 'F':
			r += rune(c - 'A' + 10)
		default:
			return 0, false
		}
	}
	return r, true
}

// readNumber 读取数字字面量
func (l *Lexer) readNumber() Token {
	pos := l.pos

	// 负号
	if l.peek() == '-' {
		l.next()
		// 负号后必须跟数字
		if !unicode.IsDigit(l.peek()) {
			return Token{Type: TokenIllegal, Value: l.input[pos:l.pos], Pos: pos}
		}
	}

	// 整数部分
	if l.peek() == '0' {
		l.next()
		// 0 后面不能跟数字
		if unicode.IsDigit(l.peek()) {
			return Token{Type: TokenIllegal, Value: l.input[pos:l.pos], Pos: pos}
		}
	} else {
		for unicode.IsDigit(l.peek()) {
			l.next()
		}
	}

	// 小数部分
	if l.peek() == '.' {
		l.next()
		hasDigit := false
		for unicode.IsDigit(l.peek()) {
			hasDigit = true
			l.next()
		}
		if !hasDigit {
			return Token{Type: TokenIllegal, Value: l.input[pos:l.pos], Pos: pos}
		}
	}

	// 指数部分
	if l.peek() == 'e' || l.peek() == 'E' {
		l.next()
		if l.peek() == '+' || l.peek() == '-' {
			l.next()
		}
		hasDigit := false
		for unicode.IsDigit(l.peek()) {
			hasDigit = true
			l.next()
		}
		if !hasDigit {
			return Token{Type: TokenIllegal, Value: l.input[pos:l.pos], Pos: pos}
		}
	}

	return Token{Type: TokenNumber, Value: l.input[pos:l.pos], Pos: pos}
}

// readIdent 读取标识符或关键字
func (l *Lexer) readIdent() Token {
	pos := l.pos

	for isNameChar(l.peek()) {
		l.next()
	}

	value := l.input[pos:l.pos]
	switch value {
	case "true":
		return Token{Type: TokenTrue, Value: value, Pos: pos}
	case "false":
		return Token{Type: TokenFalse, Value: value, Pos: pos}
	case "null":
		return Token{Type: TokenNull, Value: value, Pos: pos}
	default:
		return Token{Type: TokenIdent, Value: value, Pos: pos}
	}
}

func (l *Lexer) next() rune {
	if l.pos >= len(l.input) {
		return utf8.RuneError
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = w
	l.pos += w
	return r
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.input) {
		return utf8.RuneError
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.pos:])
	return r
}

func (l *Lexer) backup() {
	l.pos -= l.width
}

// name-first = ALPHA / "_" / %x80-D7FF / %xE000-10FFFF
func isNameFirst(r rune) bool {
	if r == '_' {
		return true
	}
	return unicode.IsLetter(r)
}

// name-char = name-first / DIGIT
func isNameChar(r rune) bool {
	return isNameFirst(r) || unicode.IsDigit(r)
}

// function-name-first = LCALPHA
func isFunctionNameFirst(r rune) bool {
	return (r >= 'a' && r <= 'z')
}

// function-name-char = function-name-first / DIGIT / "_"
func isFunctionNameChar(r rune) bool {
	return isFunctionNameFirst(r) || (r >= '0' && r <= '9') || r == '_'
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}
