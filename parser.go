package jsonpath

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/saltfishpr/jsonpath/internal/iregexp"
)

type parser struct {
	s          string
	pos, depth int
	err        error
}

func (p *parser) fail(message string) {
	if p.err == nil {
		p.err = &CompileError{p.pos, message}
	}
}

func (p *parser) skip() {
	for p.pos < len(p.s) && space(p.s[p.pos]) {
		p.pos++
	}
}

func (p *parser) take(s string) bool {
	if strings.HasPrefix(p.s[p.pos:], s) {
		p.pos += len(s)
		return true
	}
	return false
}

func (p *parser) enter() bool {
	p.depth++
	if p.depth > maxDepth {
		p.err = &LimitError{"JSONPath nesting", maxDepth}
		return false
	}
	return true
}

func (p *parser) path() pathExpr {
	result := pathExpr{root: p.take("$")}
	if !result.root && !p.take("@") {
		p.fail("expected $ or @")
		return result
	}
	for p.err == nil {
		before := p.pos
		p.skip()
		s := segment{}
		if p.take("..") {
			s.descendant = true
			if p.pos < len(p.s) && p.s[p.pos] == '[' {
				s.selectors = p.bracket()
			} else if p.take("*") {
				s.selectors = []selector{{kind: '*'}}
			} else {
				s.selectors = []selector{{kind: 'n', name: p.member()}}
			}
		} else if p.take(".") {
			if p.take("*") {
				s.selectors = []selector{{kind: '*'}}
			} else {
				s.selectors = []selector{{kind: 'n', name: p.member()}}
			}
		} else if p.pos < len(p.s) && p.s[p.pos] == '[' {
			s.selectors = p.bracket()
		} else {
			p.pos = before
			break
		}
		result.segments = append(result.segments, s)
	}
	return result
}

func nameFirst(r rune) bool {
	return r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= 128
}

func (p *parser) member() string {
	start := p.pos
	for p.pos < len(p.s) {
		r, n := utf8.DecodeRuneInString(p.s[p.pos:])
		if !nameFirst(r) && !(p.pos > start && r >= '0' && r <= '9') {
			break
		}
		p.pos += n
	}
	if p.pos == start {
		p.fail("expected member name")
	}
	return p.s[start:p.pos]
}

func (p *parser) integer() *int64 {
	start := p.pos
	p.take("-")
	digits := p.pos
	for p.pos < len(p.s) && digit(p.s[p.pos]) {
		p.pos++
	}
	if digits == p.pos {
		p.fail("expected integer")
		return nil
	}
	raw := p.s[start:p.pos]
	if raw == "-0" || (p.pos-digits > 1 && p.s[digits] == '0') {
		p.fail("invalid integer")
		return nil
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < -9007199254740991 || n > 9007199254740991 {
		p.fail("integer outside interoperable range")
		return nil
	}
	return &n
}

func (p *parser) bracket() []selector {
	if !p.enter() {
		return nil
	}
	defer func() { p.depth-- }()
	p.pos++
	var out []selector
	for p.err == nil {
		p.skip()
		s := selector{}
		if p.pos == len(p.s) {
			p.fail("unterminated selection")
			break
		}
		switch p.s[p.pos] {
		case '\'', '"':
			s.kind = 'n'
			name, err := readString(p.s, &p.pos)
			if err != nil {
				p.fail(err.Error())
			}
			s.name = name
		case '*':
			p.pos++
			s.kind = '*'
		case '?':
			p.pos++
			p.skip()
			s.kind = '?'
			s.filter = p.or()
			if p.err == nil {
				p.require(s.filter, logicalType)
			}
		default:
			s.kind = 'i'
			var first *int64
			if p.s[p.pos] != ':' {
				first = p.integer()
			}
			p.skip()
			if p.take(":") {
				s.kind = 's'
				s.start = first
				p.skip()
				if p.pos < len(p.s) && (digit(p.s[p.pos]) || p.s[p.pos] == '-') {
					s.end = p.integer()
				}
				p.skip()
				if p.take(":") {
					p.skip()
					if p.pos < len(p.s) && (digit(p.s[p.pos]) || p.s[p.pos] == '-') {
						s.step = p.integer()
					}
				}
			} else if first != nil {
				s.index = *first
			}
		}
		out = append(out, s)
		p.skip()
		if p.take("]") {
			break
		}
		if !p.take(",") {
			p.fail("expected comma or ]")
			break
		}
	}
	return out
}

func (p *parser) or() *expression {
	e := p.and()
	for p.err == nil {
		p.skip()
		if !p.take("||") {
			break
		}
		p.skip()
		e = &expression{kind: 'b', name: "||", left: e, right: p.and(), offset: p.pos}
	}
	return e
}

func (p *parser) and() *expression {
	e := p.basic()
	for p.err == nil {
		p.skip()
		if !p.take("&&") {
			break
		}
		p.skip()
		e = &expression{kind: 'b', name: "&&", left: e, right: p.basic(), offset: p.pos}
	}
	return e
}

func (p *parser) basic() *expression {
	if !p.enter() {
		return &expression{}
	}
	defer func() { p.depth-- }()
	p.skip()
	start := p.pos
	if p.take("!") {
		p.skip()
		e := p.atom()
		return &expression{kind: '!', left: e, offset: start}
	}
	left := p.atom()
	p.skip()
	for _, op := range []string{"==", "!=", "<=", ">=", "<", ">"} {
		if p.take(op) {
			p.skip()
			return &expression{kind: 'c', name: op, left: left, right: p.atom(), offset: start}
		}
	}
	return left
}

func (p *parser) atom() *expression {
	e := &expression{offset: p.pos}
	if p.err != nil {
		return e
	}
	if p.pos == len(p.s) {
		p.fail("expected expression")
		return e
	}
	switch c := p.s[p.pos]; {
	case c == '(':
		p.pos++
		p.skip()
		e.kind = '('
		e.left = p.or()
		p.skip()
		if !p.take(")") {
			p.fail("expected )")
		}
	case c == '$' || c == '@':
		e.kind = 'q'
		e.path = p.path()
	case c == '\'' || c == '"':
		e.kind = 'v'
		s, err := readString(p.s, &p.pos)
		if err != nil {
			p.fail(err.Error())
		}
		e.literal = &jsonNode{kind: stringKind, text: s}
	case c == '-' || digit(c):
		e.kind = 'v'
		start := p.pos
		if !scanNumber(p.s, &p.pos) {
			p.fail("invalid number")
			return e
		}
		e.literal = &jsonNode{kind: numberKind, number: parseDecimal(p.s[start:p.pos])}
	case c >= 'a' && c <= 'z':
		start := p.pos
		for p.pos < len(p.s) {
			c = p.s[p.pos]
			if !(c >= 'a' && c <= 'z' || c == '_' || digit(c)) {
				break
			}
			p.pos++
		}
		word := p.s[start:p.pos]
		if word == "true" || word == "false" || word == "null" {
			e.kind = 'v'
			e.literal = &jsonNode{kind: nullKind}
			if word != "null" {
				e.literal.kind = boolKind
				e.literal.boolean = word == "true"
			}
			return e
		}
		e.kind = 'f'
		e.name = word
		if !p.take("(") {
			p.fail("expected function call")
			return e
		}
		p.skip()
		if p.take(")") {
			return e
		}
		for p.err == nil {
			e.args = append(e.args, p.or())
			p.skip()
			if p.take(")") {
				break
			}
			if !p.take(",") {
				p.fail("expected comma or )")
				break
			}
			p.skip()
		}
	default:
		p.fail("expected literal, query, function or parentheses")
	}
	return e
}

func (p *parser) require(e *expression, want exprType) {
	if p.err != nil {
		return
	}
	got := p.check(e)
	if p.err != nil {
		return
	}
	if got == want {
		return
	}
	if got == nodesType && (want == logicalType || want == valueType && e.kind == 'q' && e.path.singular()) {
		return
	}
	p.err = &CompileError{e.offset, "expression has incompatible type"}
}

func (p *parser) check(e *expression) exprType {
	if p.err != nil {
		return valueType
	}
	switch e.kind {
	case 'v':
		return valueType
	case 'q':
		return nodesType
	case '(', '!':
		p.require(e.left, logicalType)
		return logicalType
	case 'b':
		p.require(e.left, logicalType)
		p.require(e.right, logicalType)
		return logicalType
	case 'c':
		p.require(e.left, valueType)
		p.require(e.right, valueType)
		return logicalType
	case 'f':
		var params []exprType
		ret := valueType
		switch e.name {
		case "length":
			params = []exprType{valueType}
		case "count", "value":
			params = []exprType{nodesType}
		case "match", "search":
			params = []exprType{valueType, valueType}
			ret = logicalType
		default:
			p.err = &CompileError{e.offset, "unknown function: " + e.name}
			return ret
		}
		if len(e.args) != len(params) {
			p.err = &CompileError{e.offset, "wrong number of arguments to " + e.name}
			return ret
		}
		for i, t := range params {
			p.require(e.args[i], t)
		}
		if (e.name == "match" || e.name == "search") && e.args[1].kind == 'v' {
			e.regexConstant = true
			if e.args[1].literal.kind == stringKind {
				re, err := iregexp.Compile(e.args[1].literal.text)
				if err != nil {
					if limit, ok := err.(*iregexp.LimitError); ok {
						p.err = &LimitError{limit.Resource, limit.Limit}
					}
				} else {
					e.regex = re
				}
			}
		}
		return ret
	}
	p.fail("invalid expression")
	return valueType
}
