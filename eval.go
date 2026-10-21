package jsonpath

import "github.com/saltfishpr/jsonpath/internal/iregexp"

type evalContext struct {
	root  *jsonNode
	err   error
	regex map[string]*iregexp.Pattern
}

func (c *evalContext) path(p pathExpr, current *jsonNode) []*jsonNode {
	if p.root {
		current = c.root
	}
	nodes := []*jsonNode{current}
	for _, seg := range p.segments {
		var next []*jsonNode
		for _, n := range nodes {
			if !seg.descendant {
				next = c.selectChildren(next, n, seg.selectors)
				continue
			}
			stack := []*jsonNode{n}
			for len(stack) > 0 && c.err == nil {
				last := len(stack) - 1
				n = stack[last]
				stack = stack[:last]
				next = c.selectChildren(next, n, seg.selectors)
				for i := len(n.children) - 1; i >= 0; i-- {
					stack = append(stack, n.children[i])
				}
			}
		}
		nodes = next
		if c.err != nil {
			break
		}
	}
	return nodes
}

func (c *evalContext) selectChildren(out []*jsonNode, n *jsonNode, selectors []selector) []*jsonNode {
	for _, s := range selectors {
		switch s.kind {
		case 'n':
			if child := n.members[s.name]; child != nil {
				out = append(out, child)
			}
		case '*':
			out = append(out, n.children...)
		case 'i':
			if n.kind == arrayKind {
				i := s.index
				if i < 0 {
					i += int64(len(n.children))
				}
				if i >= 0 && i < int64(len(n.children)) {
					out = append(out, n.children[int(i)])
				}
			}
		case 's':
			if n.kind == arrayKind {
				out = appendSlice(out, n.children, s)
			}
		case '?':
			for _, child := range n.children {
				if c.logical(s.filter, child) {
					out = append(out, child)
				}
				if c.err != nil {
					return out
				}
			}
		}
	}
	return out
}

func appendSlice(out, children []*jsonNode, s selector) []*jsonNode {
	length := int64(len(children))
	step := int64(1)
	if s.step != nil {
		step = *s.step
	}
	if step == 0 {
		return out
	}
	clamp := func(v, lo, hi int64) int64 {
		if v < 0 {
			v += length
		}
		if v < lo {
			return lo
		}
		if v > hi {
			return hi
		}
		return v
	}
	if step > 0 {
		start, end := int64(0), length
		if s.start != nil {
			start = clamp(*s.start, 0, length)
		}
		if s.end != nil {
			end = clamp(*s.end, 0, length)
		}
		for i := start; i < end; {
			out = append(out, children[int(i)])
			if step >= end-i {
				break
			}
			i += step
		}
	} else {
		start, end := length-1, int64(-1)
		if s.start != nil {
			start = clamp(*s.start, -1, length-1)
		}
		if s.end != nil {
			end = clamp(*s.end, -1, length-1)
		}
		for i := start; i > end; {
			out = append(out, children[int(i)])
			if -step >= i-end {
				break
			}
			i += step
		}
	}
	return out
}
