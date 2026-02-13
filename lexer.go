package jsonpath

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

// TokenType is the type of a token
type TokenType int

// Token type constants
const (
	TokenIllegal TokenType = iota
	TokenEOF

	// Identifiers
	TokenRoot    // $  - Root node identifier
	TokenCurrent // @  - Current node identifier

	// Operators
	TokenDot      // .  - Dot (child segment shorthand)
	TokenDotDot   // .. - Double dot (descendant segment)
	TokenLBracket // [  - Left bracket
	TokenRBracket // ]  - Right bracket
	TokenComma    // ,  - Comma
	TokenQuestion // ?  - Question mark (filter)
	TokenColon    // :  - Colon (slice)
	TokenWildcard // * - Wildcard

	// Comparison operators
	TokenEq // ==  - Equal
	TokenNe // !=  - Not equal
	TokenLt // <   - Less than
	TokenLe // <=  - Less than or equal
	TokenGt // >   - Greater than
	TokenGe // >=  - Greater than or equal

	// Logical operators
	TokenLAnd // &&  - Logical and
	TokenLOr  // ||  - Logical or
	TokenLNot // !   - Logical not

	// Parentheses
	TokenLParen // ( - Left parenthesis
	TokenRParen // ) - Right parenthesis

	// Literals
	TokenIdent  // Identifier/name/function name
	TokenNumber // Number
	TokenString // String
	TokenTrue   // true
	TokenFalse  // false
	TokenNull   // null
)

// String returns the string representation of TokenType
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

// Token represents a lexical token
type Token struct {
	Type  TokenType
	Value string
	Pos   int
}

// Lexer tokenizes JSONPath expressions
type Lexer struct {
	input string
	pos   int
	width int
}

// NewLexer creates a new lexer for the input string
func NewLexer(input string) *Lexer {
	return &Lexer{input: input}
}

// NextToken reads and returns the next token from the input
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

	// Number literals must start with - or digit
	if r == '-' || unicode.IsDigit(r) {
		l.backup()
		return l.readNumber()
	}

	if isNameFirst(r) {
		l.backup()
		return l.readIdent()
	}

	return Token{Type: TokenIllegal, Value: string(r), Pos: pos}
}

func (l *Lexer) skipWhitespace() {
	for {
		r := l.peek()
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			return
		}
		l.next()
	}
}

func (l *Lexer) readString() Token {
	pos := l.pos
	var sb strings.Builder

	quote := l.next() // Get quote character (' or ")
	for {
		r := l.next()
		if r == utf8.RuneError {
			// No matching quote found, return illegal token
			return Token{Type: TokenIllegal, Value: l.input[pos:l.pos], Pos: pos}
		}

		if r == quote {
			// Found matching quote, string ends
			break
		}

		if r == '\\' {
			// Handle escape sequence
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
				if l.pos+4 > len(l.input) {
					return Token{Type: TokenIllegal, Value: sb.String(), Pos: pos}
				}
				r1, err := strconv.ParseUint(l.input[l.pos:l.pos+4], 16, 64)
				if err != nil {
					return Token{Type: TokenIllegal, Value: sb.String(), Pos: pos}
				}
				l.pos += 4

				if utf16.IsSurrogate(rune(r1)) {
					// Handle surrogate pair
					if l.pos+6 > len(l.input) || l.input[l.pos:l.pos+2] != "\\u" {
						return Token{Type: TokenIllegal, Value: sb.String(), Pos: pos}
					}
					r2, err := strconv.ParseUint(l.input[l.pos+2:l.pos+6], 16, 64)
					if err != nil {
						return Token{Type: TokenIllegal, Value: sb.String(), Pos: pos}
					}
					sb.WriteRune(utf16.DecodeRune(rune(r1), rune(r2)))
					l.pos += 6
				} else {
					sb.WriteRune(rune(r1))
				}
			default:
				return Token{Type: TokenIllegal, Value: sb.String(), Pos: pos}
			}
		} else {
			sb.WriteRune(r)
		}
	}
	return Token{Type: TokenString, Value: sb.String(), Pos: pos}
}

func (l *Lexer) readNumber() Token {
	pos := l.pos

	if l.peek() == '-' {
		l.next()
		// Must be followed by digit
		if !unicode.IsDigit(l.peek()) {
			return Token{Type: TokenIllegal, Value: l.input[pos:l.pos], Pos: pos}
		}
	}

	// Integer part
	if l.peek() == '0' {
		l.next()
		// Leading zeros not allowed
		if unicode.IsDigit(l.peek()) {
			return Token{Type: TokenIllegal, Value: l.input[pos:l.pos], Pos: pos}
		}
	} else {
		for unicode.IsDigit(l.peek()) {
			l.next()
		}
	}

	// Fraction part
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

	// Exponent part
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
	return isNameFirst(r) || isDigit(r)
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}
