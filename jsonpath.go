package jsonpath

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// Query is an immutable compiled expression, safe for concurrent evaluation.
// Create it with Compile; its zero value is not usable.
type Query struct {
	source   string
	path     pathExpr
	compiled bool
}

// Node pairs a selected JSON value with its RFC 9535 normalized location.
// Value is an independent copy of the source text, excluding surrounding space.
type Node struct {
	// Value contains the selected JSON text, owned by the caller.
	Value json.RawMessage
	// Path is the RFC 9535 normalized path identifying this node.
	Path string
}

// Compile parses and type-checks an RFC 9535 expression for repeated use.
func Compile(expr string) (*Query, error) {
	if !utf8.ValidString(expr) {
		return nil, &CompileError{0, "invalid UTF-8"}
	}
	p := parser{s: expr}
	if len(expr) == 0 || expr[0] != '$' {
		return nil, &CompileError{0, "expression must start with $"}
	}
	path := p.path()
	if p.err != nil {
		return nil, p.err
	}
	if p.pos != len(expr) {
		return nil, &CompileError{p.pos, "unexpected trailing input"}
	}
	return &Query{source: expr, path: path, compiled: true}, nil
}

// String returns the expression passed to Compile.
func (q *Query) String() string {
	if q == nil {
		return ""
	}
	return q.source
}

// QueryValues compiles expr, parses data and returns selected JSON fragments.
func QueryValues(expr string, data []byte) ([]json.RawMessage, error) {
	q, err := Compile(expr)
	if err != nil {
		return nil, err
	}
	d, err := Parse(data)
	if err != nil {
		return nil, err
	}
	return q.Values(d)
}

// QueryNodes compiles expr, parses data and returns selected fragments and paths.
func QueryNodes(expr string, data []byte) ([]Node, error) {
	q, err := Compile(expr)
	if err != nil {
		return nil, err
	}
	d, err := Parse(data)
	if err != nil {
		return nil, err
	}
	return q.Nodes(d)
}

func (q *Query) run(d *Document) ([]*jsonNode, error) {
	if q == nil || !q.compiled {
		return nil, fmt.Errorf("jsonpath: uninitialized query")
	}
	if err := validDocument(d); err != nil {
		return nil, err
	}
	c := evalContext{root: d.root}
	nodes := c.path(q.path, d.root)
	return nodes, c.err
}

// Values evaluates q against doc without constructing paths. Each fragment is
// independently owned by the caller. No matches returns an empty slice.
func (q *Query) Values(doc *Document) ([]json.RawMessage, error) {
	nodes, err := q.run(doc)
	if err != nil {
		return nil, err
	}
	out := make([]json.RawMessage, len(nodes))
	for i, n := range nodes {
		out[i] = json.RawMessage(doc.source[n.start:n.end])
	}
	return out, nil
}

// Nodes evaluates q against doc, returning independent fragments paired with
// normalized paths. Repeated selections are retained in the result.
func (q *Query) Nodes(doc *Document) ([]Node, error) {
	nodes, err := q.run(doc)
	if err != nil {
		return nil, err
	}
	out := make([]Node, len(nodes))
	for i, n := range nodes {
		out[i] = Node{json.RawMessage(doc.source[n.start:n.end]), normalizedPath(n)}
	}
	return out, nil
}
