package jsonpath

import (
	"fmt"
	"strings"
)

type valueKind uint8

const (
	nullKind valueKind = iota
	boolKind
	numberKind
	stringKind
	arrayKind
	objectKind
)

type jsonNode struct {
	kind       valueKind
	start, end int
	text       string
	boolean    bool
	number     decimal
	children   []*jsonNode
	members    map[string]*jsonNode
	name       string
	index      int
	parent     *jsonNode
}

// Document is an immutable parsed JSON document. Create it with Parse. Its zero
// value is not usable. Methods on Query may read a Document concurrently.
type Document struct {
	source string
	root   *jsonNode
}

// Parse copies and parses exactly one JSON value. It rejects duplicate member
// names, malformed Unicode and nesting deeper than 1024 containers.
func Parse(data []byte) (*Document, error) {
	p := documentParser{s: string(data)}
	root, err := p.value(nil, 0)
	if err != nil {
		return nil, err
	}
	p.skip()
	if p.pos != len(p.s) {
		return nil, p.fail("unexpected trailing input")
	}
	return &Document{source: p.s, root: root}, nil
}

type documentParser struct {
	s   string
	pos int
}

func (p *documentParser) skip() {
	for p.pos < len(p.s) && space(p.s[p.pos]) {
		p.pos++
	}
}
func (p *documentParser) fail(message string) error { return &DocumentError{p.pos, message} }
func (p *documentParser) value(parent *jsonNode, depth int) (*jsonNode, error) {
	p.skip()
	if p.pos == len(p.s) {
		return nil, p.fail("expected JSON value")
	}
	n := &jsonNode{start: p.pos, parent: parent}
	c := p.s[p.pos]
	switch c {
	case '{', '[':
		if depth >= maxDepth {
			return nil, &LimitError{"JSON nesting", maxDepth}
		}
		p.pos++
		close := byte(']')
		n.kind = arrayKind
		if c == '{' {
			n.kind = objectKind
			n.members = make(map[string]*jsonNode)
			close = '}'
		}
		p.skip()
		if p.pos < len(p.s) && p.s[p.pos] == close {
			p.pos++
			break
		}
		for {
			name := ""
			if c == '{' {
				if p.pos == len(p.s) || p.s[p.pos] != '"' {
					return nil, p.fail("expected object member name")
				}
				var err error
				name, err = readString(p.s, &p.pos)
				if err != nil {
					return nil, p.fail(err.Error())
				}
				if _, ok := n.members[name]; ok {
					return nil, p.fail("duplicate object member name")
				}
				p.skip()
				if p.pos == len(p.s) || p.s[p.pos] != ':' {
					return nil, p.fail("expected colon")
				}
				p.pos++
			}
			child, err := p.value(n, depth+1)
			if err != nil {
				return nil, err
			}
			child.name = name
			child.index = len(n.children)
			n.children = append(n.children, child)
			if c == '{' {
				n.members[name] = child
			}
			p.skip()
			if p.pos == len(p.s) {
				return nil, p.fail("unterminated container")
			}
			if p.s[p.pos] == close {
				p.pos++
				break
			}
			if p.s[p.pos] != ',' {
				return nil, p.fail("expected comma")
			}
			p.pos++
			p.skip()
		}
	case '"':
		n.kind = stringKind
		v, err := readString(p.s, &p.pos)
		if err != nil {
			return nil, p.fail(err.Error())
		}
		n.text = v
	case 't', 'f', 'n':
		word := "null"
		if c == 't' {
			word = "true"
			n.kind = boolKind
			n.boolean = true
		}
		if c == 'f' {
			word = "false"
			n.kind = boolKind
		}
		if !strings.HasPrefix(p.s[p.pos:], word) {
			return nil, p.fail("invalid literal")
		}
		p.pos += len(word)
	default:
		n.kind = numberKind
		if !scanNumber(p.s, &p.pos) {
			return nil, p.fail("invalid JSON value")
		}
		n.number = parseDecimal(p.s[n.start:p.pos])
	}
	n.end = p.pos
	return n, nil
}

func validDocument(d *Document) error {
	if d == nil || d.root == nil {
		return fmt.Errorf("jsonpath: uninitialized document")
	}
	return nil
}
