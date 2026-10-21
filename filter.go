package jsonpath

import "strings"

type expressionResult struct {
	typ     exprType
	value   *jsonNode
	nodes   []*jsonNode
	logical bool
}

func (c *evalContext) logical(e *expression, n *jsonNode) bool {
	r := c.expression(e, n)
	if r.typ == nodesType {
		return len(r.nodes) > 0
	}
	return r.logical
}
func (c *evalContext) value(e *expression, n *jsonNode) *jsonNode {
	r := c.expression(e, n)
	if r.typ == nodesType {
		if len(r.nodes) == 1 {
			return r.nodes[0]
		}
		return nil
	}
	return r.value
}
func (c *evalContext) expression(e *expression, n *jsonNode) expressionResult {
	r := expressionResult{typ: valueType}
	if c.err != nil {
		return r
	}
	switch e.kind {
	case 'v':
		r.value = e.literal
	case 'q':
		r.typ = nodesType
		r.nodes = c.path(e.path, n)
	case '(':
		r.typ = logicalType
		r.logical = c.logical(e.left, n)
	case '!':
		r.typ = logicalType
		r.logical = !c.logical(e.left, n)
	case 'b':
		r.typ = logicalType
		if e.name == "&&" {
			r.logical = c.logical(e.left, n) && c.logical(e.right, n)
		} else {
			r.logical = c.logical(e.left, n) || c.logical(e.right, n)
		}
	case 'c':
		r.typ = logicalType
		a, b := c.value(e.left, n), c.value(e.right, n)
		equal := equalValues(a, b)
		less, greater := false, false
		if a != nil && b != nil && a.kind == b.kind {
			var cmp int
			switch a.kind {
			case numberKind:
				cmp = compareNumber(a.number, b.number)
			case stringKind:
				cmp = strings.Compare(a.text, b.text)
			}
			less = cmp < 0
			greater = cmp > 0
		}
		switch e.name {
		case "==":
			r.logical = equal
		case "!=":
			r.logical = !equal
		case "<":
			r.logical = less
		case ">":
			r.logical = greater
		case "<=":
			r.logical = less || equal
		case ">=":
			r.logical = greater || equal
		}
	case 'f':
		return c.function(e, n)
	}
	return r
}
func equalValues(a, b *jsonNode) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.kind != b.kind {
		return false
	}
	switch a.kind {
	case nullKind:
		return true
	case boolKind:
		return a.boolean == b.boolean
	case stringKind:
		return a.text == b.text
	case numberKind:
		return compareNumber(a.number, b.number) == 0
	case arrayKind:
		if len(a.children) != len(b.children) {
			return false
		}
		for i, x := range a.children {
			if !equalValues(x, b.children[i]) {
				return false
			}
		}
		return true
	case objectKind:
		if len(a.members) != len(b.members) {
			return false
		}
		for k, x := range a.members {
			y, ok := b.members[k]
			if !ok || !equalValues(x, y) {
				return false
			}
		}
		return true
	}
	return false
}
