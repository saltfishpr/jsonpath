package iregexp

import (
	"fmt"
	"math/big"
	"unicode/utf8"
)

// LimitError distinguishes resource limits from invalid I-Regexp syntax.
type LimitError struct {
	// Resource identifies the bounded operation.
	Resource string
	// Limit is the maximum supported nesting depth.
	Limit int
}

// Error describes the resource and its limit.
func (e *LimitError) Error() string { return fmt.Sprintf("%s exceeds %d", e.Resource, e.Limit) }

// Pattern is a compiled immutable I-Regexp, safe for concurrent use.
type Pattern struct{ root *node }
type node struct {
	kind     byte
	children []*node
	class    characterClass
	min, max *big.Int
	nullable bool
}
type characterClass struct {
	negative bool
	parts    []classPart
}
type classPart struct {
	lo, hi   rune
	category string
	negative bool
}
type parser struct {
	r          []rune
	pos, depth int
	err        error
}

func (p *parser) fail() {
	if p.err == nil {
		p.err = fmt.Errorf("invalid I-Regexp at character %d", p.pos)
	}
}
func (p *parser) peek() rune {
	if p.pos == len(p.r) {
		return -1
	}
	return p.r[p.pos]
}
func (p *parser) take(r rune) bool {
	if p.peek() == r {
		p.pos++
		return true
	}
	return false
}

// Compile validates the complete RFC 9485 grammar and compiles its pattern.
// Invalid patterns return an error; callers implementing JSONPath should treat
// syntax errors as false and propagate LimitError separately.
func Compile(s string) (*Pattern, error) {
	if !utf8.ValidString(s) {
		return nil, fmt.Errorf("invalid UTF-8 in I-Regexp")
	}
	p := parser{r: []rune(s)}
	n := p.alternation()
	if p.err == nil && p.pos != len(p.r) {
		p.fail()
	}
	if p.err != nil {
		return nil, p.err
	}
	return &Pattern{n}, nil
}
func (p *parser) alternation() *node {
	n := &node{kind: '|'}
	for {
		child := p.branch()
		n.children = append(n.children, child)
		n.nullable = n.nullable || child.nullable
		if p.err != nil || !p.take('|') {
			break
		}
	}
	if len(n.children) == 1 {
		return n.children[0]
	}
	return n
}
func (p *parser) branch() *node {
	n := &node{kind: 's', nullable: true}
	for p.err == nil && p.peek() != -1 && p.peek() != ')' && p.peek() != '|' {
		child := p.piece()
		n.children = append(n.children, child)
		n.nullable = n.nullable && child.nullable
	}
	if len(n.children) == 1 {
		return n.children[0]
	}
	return n
}
func (p *parser) piece() *node {
	atom := p.atom()
	if p.err != nil {
		return atom
	}
	n := &node{kind: 'r', children: []*node{atom}}
	switch p.peek() {
	case '*':
		p.pos++
		n.min = big.NewInt(0)
	case '+':
		p.pos++
		n.min = big.NewInt(1)
	case '?':
		p.pos++
		n.min = big.NewInt(0)
		n.max = big.NewInt(1)
	case '{':
		p.pos++
		n.min = p.integer()
		n.max = n.min
		if p.take(',') {
			n.max = nil
			if p.peek() >= '0' && p.peek() <= '9' {
				n.max = p.integer()
			}
		}
		if !p.take('}') {
			p.fail()
		}
		if n.min != nil && n.max != nil && n.min.Cmp(n.max) > 0 {
			p.fail()
		}
	default:
		return atom
	}
	if n.min == nil {
		n.min = big.NewInt(0)
	}
	n.nullable = n.min.Sign() == 0 || atom.nullable
	return n
}
func (p *parser) integer() *big.Int {
	start := p.pos
	for p.peek() >= '0' && p.peek() <= '9' {
		p.pos++
	}
	if start == p.pos {
		p.fail()
		return nil
	}
	n := new(big.Int)
	n.SetString(string(p.r[start:p.pos]), 10)
	return n
}
func (p *parser) atom() *node {
	n := &node{kind: 'c'}
	switch r := p.peek(); r {
	case '(':
		p.pos++
		p.depth++
		if p.depth > 1024 {
			p.err = &LimitError{"I-Regexp nesting", 1024}
			return n
		}
		n = p.alternation()
		if !p.take(')') {
			p.fail()
		}
		p.depth--
		return n
	case '.':
		p.pos++
		n.class = characterClass{negative: true, parts: []classPart{{lo: '\n', hi: '\n'}, {lo: '\r', hi: '\r'}}}
	case '[':
		p.pos++
		n.class = p.class()
	case '\\':
		n.class.parts = []classPart{p.escape()}
	case -1, ')', '*', '+', '?', '{', '}', '|', ']':
		p.fail()
	default:
		p.pos++
		n.class.parts = []classPart{{lo: r, hi: r}}
	}
	return n
}
func (p *parser) escape() classPart {
	p.pos++
	r := p.peek()
	if r == -1 {
		p.fail()
		return classPart{}
	}
	p.pos++
	if r == 'p' || r == 'P' {
		if !p.take('{') {
			p.fail()
			return classPart{}
		}
		start := p.pos
		for p.peek() != -1 && p.peek() != '}' {
			p.pos++
		}
		name := string(p.r[start:p.pos])
		if !p.take('}') || !validCategory(name) {
			p.fail()
		}
		return classPart{category: name, negative: r == 'P'}
	}
	switch r {
	case 'n':
		r = '\n'
	case 'r':
		r = '\r'
	case 't':
		r = '\t'
	case '(', ')', '*', '+', '-', '.', '?', '[', '\\', ']', '^', '{', '|', '}':
	default:
		p.fail()
	}
	return classPart{lo: r, hi: r}
}
func validCategory(s string) bool {
	switch s {
	case "L", "Ll", "Lm", "Lo", "Lt", "Lu", "M", "Mc", "Me", "Mn",
		"N", "Nd", "Nl", "No", "P", "Pc", "Pd", "Pe", "Pf", "Pi", "Po", "Ps",
		"Z", "Zl", "Zp", "Zs", "S", "Sc", "Sk", "Sm", "So", "C", "Cc", "Cf", "Cn", "Co":
		return true
	}
	return false
}
func (p *parser) classChar() classPart {
	if p.peek() == '\\' {
		return p.escape()
	}
	r := p.peek()
	if r == -1 || r == '[' || r == ']' || r == '-' {
		p.fail()
		return classPart{}
	}
	p.pos++
	return classPart{lo: r, hi: r}
}
func (p *parser) class() characterClass {
	c := characterClass{negative: p.take('^')}
	if p.take('-') {
		c.parts = append(c.parts, classPart{lo: '-', hi: '-'})
	}
	for p.err == nil && p.peek() != -1 && p.peek() != ']' {
		if p.peek() == '-' && p.pos+1 < len(p.r) && p.r[p.pos+1] == ']' {
			p.pos++
			c.parts = append(c.parts, classPart{lo: '-', hi: '-'})
			break
		}
		a := p.classChar()
		if p.err != nil {
			break
		}
		if p.take('-') {
			if p.peek() == ']' {
				c.parts = append(c.parts, a, classPart{lo: '-', hi: '-'})
				break
			}
			b := p.classChar()
			if a.category != "" || b.category != "" || a.lo > b.lo {
				p.fail()
			}
			a.hi = b.lo
		}
		c.parts = append(c.parts, a)
	}
	if !p.take(']') || len(c.parts) == 0 {
		p.fail()
	}
	return c
}
