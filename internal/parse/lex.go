package parse

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/saltfishpr/jsonpath/internal/ast"
)

type tokenKind uint8

const (
	tkEOF tokenKind = iota
	tkError
	tkWhitespace // 连续的空格、Tab、换行或回车

	tkRoot     // $
	tkCurrent  // @
	tkDot      // .
	tkDotDot   // ..
	tkLBracket // [
	tkRBracket // ]
	tkLParen   // (
	tkRParen   // )
	tkComma    // ,
	tkColon    // :
	tkStar     // *
	tkQuestion // ?

	tkName   // member-name-shorthand 或 函数名
	tkString // 'foo' / "foo"（已解码）
	tkInt
	tkFloat // 带小数或指数
	tkTrue
	tkFalse
	tkNull

	tkEq  // ==
	tkNe  // !=
	tkLt  // <
	tkLe  // <=
	tkGt  // >
	tkGe  // >=
	tkAnd // &&
	tkOr  // ||
	tkNot // !
)

func (k tokenKind) String() string {
	switch k {
	case tkEOF:
		return "EOF"
	case tkError:
		return "ERROR"
	case tkWhitespace:
		return "Whitespace"
	case tkRoot:
		return "$"
	case tkCurrent:
		return "@"
	case tkDot:
		return "."
	case tkDotDot:
		return ".."
	case tkLBracket:
		return "["
	case tkRBracket:
		return "]"
	case tkLParen:
		return "("
	case tkRParen:
		return ")"
	case tkComma:
		return ","
	case tkColon:
		return ":"
	case tkStar:
		return "*"
	case tkQuestion:
		return "?"
	case tkName:
		return "Name"
	case tkString:
		return "String"
	case tkInt:
		return "Int"
	case tkFloat:
		return "Number"
	case tkTrue:
		return "true"
	case tkFalse:
		return "false"
	case tkNull:
		return "null"
	case tkEq:
		return "=="
	case tkNe:
		return "!="
	case tkLt:
		return "<"
	case tkLe:
		return "<="
	case tkGt:
		return ">"
	case tkGe:
		return ">="
	case tkAnd:
		return "&&"
	case tkOr:
		return "||"
	case tkNot:
		return "!"
	}
	return "?"
}

type token struct {
	Kind  tokenKind
	Value string // 字符串/标识符的解码值；数字存原文
	Raw   string // 源串原始片段
	Pos   ast.Pos
}

type lexError struct {
	Pos ast.Pos
	Msg string
}

func (e *lexError) Error() string {
	return fmt.Sprintf("jsonpath: lex error at line %d col %d: %s", e.Pos.Line, e.Pos.Col, e.Msg)
}

// lex 将查询串切分为 token 列表，末尾包含 tkEOF。
func lex(src string) ([]token, error) {
	if !utf8.ValidString(src) {
		return nil, &lexError{Pos: ast.Pos{Line: 1, Col: 1}, Msg: "query is not valid UTF-8"}
	}
	l := &lexer{src: src, line: 1, col: 1}
	for {
		tok, err := l.next()
		if err != nil {
			return l.tokens, err
		}
		l.tokens = append(l.tokens, tok)
		if tok.Kind == tkEOF {
			return l.tokens, nil
		}
	}
}

type lexer struct {
	src    string
	off    int
	line   int
	col    int
	tokens []token
}

func (l *lexer) pos() ast.Pos { return ast.Pos{Offset: l.off, Line: l.line, Col: l.col} }

func (l *lexer) peek() (rune, int) {
	if l.off >= len(l.src) {
		return 0, 0
	}
	r, w := utf8.DecodeRuneInString(l.src[l.off:])
	return r, w
}

func (l *lexer) advance(r rune, w int) {
	l.off += w
	if r == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
}

func (l *lexer) errorf(pos ast.Pos, format string, args ...interface{}) error {
	return &lexError{Pos: pos, Msg: fmt.Sprintf(format, args...)}
}

func (l *lexer) next() (token, error) {
	start := l.pos()
	for l.off < len(l.src) {
		r, w := l.peek()
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			l.advance(r, w)
			continue
		}
		break
	}
	if l.off > start.Offset {
		return token{Kind: tkWhitespace, Raw: l.src[start.Offset:l.off], Pos: start}, nil
	}
	if l.off >= len(l.src) {
		return token{Kind: tkEOF, Pos: start}, nil
	}
	r, w := l.peek()
	switch {
	case r == '$':
		l.advance(r, w)
		return token{Kind: tkRoot, Raw: "$", Pos: start}, nil
	case r == '@':
		l.advance(r, w)
		return token{Kind: tkCurrent, Raw: "@", Pos: start}, nil
	case r == '.':
		l.advance(r, w)
		if r2, w2 := l.peek(); r2 == '.' {
			l.advance(r2, w2)
			return token{Kind: tkDotDot, Raw: "..", Pos: start}, nil
		}
		return token{Kind: tkDot, Raw: ".", Pos: start}, nil
	case r == '[':
		l.advance(r, w)
		return token{Kind: tkLBracket, Raw: "[", Pos: start}, nil
	case r == ']':
		l.advance(r, w)
		return token{Kind: tkRBracket, Raw: "]", Pos: start}, nil
	case r == '(':
		l.advance(r, w)
		return token{Kind: tkLParen, Raw: "(", Pos: start}, nil
	case r == ')':
		l.advance(r, w)
		return token{Kind: tkRParen, Raw: ")", Pos: start}, nil
	case r == ',':
		l.advance(r, w)
		return token{Kind: tkComma, Raw: ",", Pos: start}, nil
	case r == ':':
		l.advance(r, w)
		return token{Kind: tkColon, Raw: ":", Pos: start}, nil
	case r == '*':
		l.advance(r, w)
		return token{Kind: tkStar, Raw: "*", Pos: start}, nil
	case r == '?':
		l.advance(r, w)
		return token{Kind: tkQuestion, Raw: "?", Pos: start}, nil
	case r == '=':
		l.advance(r, w)
		if r2, w2 := l.peek(); r2 == '=' {
			l.advance(r2, w2)
			return token{Kind: tkEq, Raw: "==", Pos: start}, nil
		}
		return token{Kind: tkError, Pos: start}, l.errorf(start, "unexpected '=' (did you mean '==')")
	case r == '!':
		l.advance(r, w)
		if r2, w2 := l.peek(); r2 == '=' {
			l.advance(r2, w2)
			return token{Kind: tkNe, Raw: "!=", Pos: start}, nil
		}
		return token{Kind: tkNot, Raw: "!", Pos: start}, nil
	case r == '<':
		l.advance(r, w)
		if r2, w2 := l.peek(); r2 == '=' {
			l.advance(r2, w2)
			return token{Kind: tkLe, Raw: "<=", Pos: start}, nil
		}
		return token{Kind: tkLt, Raw: "<", Pos: start}, nil
	case r == '>':
		l.advance(r, w)
		if r2, w2 := l.peek(); r2 == '=' {
			l.advance(r2, w2)
			return token{Kind: tkGe, Raw: ">=", Pos: start}, nil
		}
		return token{Kind: tkGt, Raw: ">", Pos: start}, nil
	case r == '&':
		l.advance(r, w)
		if r2, w2 := l.peek(); r2 == '&' {
			l.advance(r2, w2)
			return token{Kind: tkAnd, Raw: "&&", Pos: start}, nil
		}
		return token{Kind: tkError, Pos: start}, l.errorf(start, "unexpected '&' (did you mean '&&')")
	case r == '|':
		l.advance(r, w)
		if r2, w2 := l.peek(); r2 == '|' {
			l.advance(r2, w2)
			return token{Kind: tkOr, Raw: "||", Pos: start}, nil
		}
		return token{Kind: tkError, Pos: start}, l.errorf(start, "unexpected '|' (did you mean '||')")
	case r == '\'' || r == '"':
		return l.lexString(start, r)
	case r == '-' || (r >= '0' && r <= '9'):
		return l.lexNumber(start)
	case isNameFirst(r):
		return l.lexNameOrKeyword(start)
	}
	return token{Kind: tkError, Pos: start}, l.errorf(start, "unexpected character %q", r)
}

func (l *lexer) lexString(start ast.Pos, quote rune) (token, error) {
	_, w := l.peek()
	l.advance(quote, w)

	var sb strings.Builder
	for {
		if l.off >= len(l.src) {
			return token{Kind: tkError, Pos: start}, l.errorf(start, "unterminated string literal")
		}
		r, w := l.peek()
		if r == quote {
			l.advance(r, w)
			raw := l.src[start.Offset:l.off]
			return token{Kind: tkString, Value: sb.String(), Raw: raw, Pos: start}, nil
		}
		if r == '\\' {
			l.advance(r, w)
			if err := l.lexEscape(&sb, quote, start); err != nil {
				return token{Kind: tkError, Pos: start}, err
			}
			continue
		}
		if r < 0x20 {
			return token{Kind: tkError, Pos: l.pos()}, l.errorf(l.pos(), "unescaped control character U+%04X in string", r)
		}
		sb.WriteRune(r)
		l.advance(r, w)
	}
}

func (l *lexer) lexEscape(sb *strings.Builder, quote rune, start ast.Pos) error {
	if l.off >= len(l.src) {
		return l.errorf(start, "unterminated escape in string literal")
	}
	r, w := l.peek()
	l.advance(r, w)
	switch r {
	case 'b':
		sb.WriteByte(0x08)
	case 'f':
		sb.WriteByte(0x0C)
	case 'n':
		sb.WriteByte('\n')
	case 'r':
		sb.WriteByte('\r')
	case 't':
		sb.WriteByte('\t')
	case '/':
		sb.WriteByte('/')
	case '\\':
		sb.WriteByte('\\')
	case '\'':
		if quote != '\'' {
			return l.errorf(l.pos(), "invalid escape '\\'' in double-quoted string")
		}
		sb.WriteByte('\'')
	case '"':
		if quote != '"' {
			return l.errorf(l.pos(), "invalid escape '\\\"' in single-quoted string")
		}
		sb.WriteByte('"')
	case 'u':
		cp, err := l.lexUnicodeEscape()
		if err != nil {
			return err
		}
		sb.WriteRune(cp)
	default:
		return l.errorf(l.pos(), "invalid escape '\\%c'", r)
	}
	return nil
}

func (l *lexer) lexUnicodeEscape() (rune, error) {
	hi, err := l.lexHex4()
	if err != nil {
		return 0, err
	}
	if hi >= 0xD800 && hi <= 0xDBFF {
		if l.off+2 > len(l.src) || l.src[l.off] != '\\' || l.src[l.off+1] != 'u' {
			return 0, l.errorf(l.pos(), "unpaired high surrogate in \\u escape")
		}
		l.advance('\\', 1)
		l.advance('u', 1)
		lo, err := l.lexHex4()
		if err != nil {
			return 0, err
		}
		if lo < 0xDC00 || lo > 0xDFFF {
			return 0, l.errorf(l.pos(), "invalid low surrogate in \\u escape")
		}
		return 0x10000 + ((hi-0xD800)<<10 | (lo - 0xDC00)), nil
	}
	if hi >= 0xDC00 && hi <= 0xDFFF {
		return 0, l.errorf(l.pos(), "unexpected low surrogate in \\u escape")
	}
	return hi, nil
}

func (l *lexer) lexHex4() (rune, error) {
	if l.off+4 > len(l.src) {
		return 0, l.errorf(l.pos(), "truncated \\u escape")
	}
	var v rune
	for i := 0; i < 4; i++ {
		c := l.src[l.off]
		var d rune
		switch {
		case c >= '0' && c <= '9':
			d = rune(c - '0')
		case c >= 'a' && c <= 'f':
			d = rune(c-'a') + 10
		case c >= 'A' && c <= 'F':
			d = rune(c-'A') + 10
		default:
			return 0, l.errorf(l.pos(), "invalid hex digit %q in \\u escape", c)
		}
		v = v<<4 | d
		l.advance(rune(c), 1)
	}
	return v, nil
}

func (l *lexer) lexNumber(start ast.Pos) (token, error) {
	if r, w := l.peek(); r == '-' {
		l.advance(r, w)
	}
	intStart := l.off
	if r, _ := l.peek(); r == '0' {
		l.advance(r, 1)
	} else if r >= '1' && r <= '9' {
		for {
			r, w := l.peek()
			if r < '0' || r > '9' {
				break
			}
			l.advance(r, w)
		}
	} else {
		return token{Kind: tkError, Pos: start}, l.errorf(l.pos(), "invalid number literal")
	}
	if l.off == intStart {
		return token{Kind: tkError, Pos: start}, l.errorf(start, "invalid number literal")
	}

	isFloat := false
	if r, _ := l.peek(); r == '.' {
		isFloat = true
		l.advance('.', 1)
		cnt := 0
		for {
			r, w := l.peek()
			if r < '0' || r > '9' {
				break
			}
			l.advance(r, w)
			cnt++
		}
		if cnt == 0 {
			return token{Kind: tkError, Pos: start}, l.errorf(start, "fractional part requires at least one digit")
		}
	}
	if r, _ := l.peek(); r == 'e' || r == 'E' {
		isFloat = true
		l.advance(r, 1)
		if r2, _ := l.peek(); r2 == '+' || r2 == '-' {
			l.advance(r2, 1)
		}
		cnt := 0
		for {
			r, w := l.peek()
			if r < '0' || r > '9' {
				break
			}
			l.advance(r, w)
			cnt++
		}
		if cnt == 0 {
			return token{Kind: tkError, Pos: start}, l.errorf(start, "exponent requires at least one digit")
		}
	}

	raw := l.src[start.Offset:l.off]
	if isFloat {
		return token{Kind: tkFloat, Value: raw, Raw: raw, Pos: start}, nil
	}
	return token{Kind: tkInt, Value: raw, Raw: raw, Pos: start}, nil
}

func (l *lexer) lexNameOrKeyword(start ast.Pos) (token, error) {
	for l.off < len(l.src) {
		r, w := l.peek()
		if !isNameChar(r) {
			break
		}
		l.advance(r, w)
	}
	raw := l.src[start.Offset:l.off]
	switch raw {
	case "true":
		return token{Kind: tkTrue, Value: raw, Raw: raw, Pos: start}, nil
	case "false":
		return token{Kind: tkFalse, Value: raw, Raw: raw, Pos: start}, nil
	case "null":
		return token{Kind: tkNull, Value: raw, Raw: raw, Pos: start}, nil
	}
	return token{Kind: tkName, Value: raw, Raw: raw, Pos: start}, nil
}

func isNameFirst(r rune) bool {
	switch {
	case r == '_':
		return true
	case r >= 'A' && r <= 'Z':
		return true
	case r >= 'a' && r <= 'z':
		return true
	case r >= 0x80 && r <= 0xD7FF:
		return true
	case r >= 0xE000 && r <= 0x10FFFF:
		return true
	}
	return false
}

func isNameChar(r rune) bool {
	if isNameFirst(r) {
		return true
	}
	return r >= '0' && r <= '9'
}
