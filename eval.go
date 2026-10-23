package jsonpath

import (
	"github.com/saltfishpr/jsonpath/internal/ast"
	"github.com/saltfishpr/jsonpath/internal/document"
	"github.com/saltfishpr/jsonpath/internal/iregexp"
)

func (q *Query) run(doc *Document) []*document.Node {
	ev := &evaluator{root: doc.root, literals: q.literals, regexes: q.regexes}
	return ev.query(q.query, doc.root)
}

type evaluator struct {
	root     *document.Node
	literals map[*ast.Literal]document.Value
	regexes  map[string]*iregexp.Regexp
}

func (e *evaluator) query(q *ast.Query, current *document.Node) []*document.Node {
	if q.IsRoot {
		current = e.root
	}
	nodes := []*document.Node{current}
	for _, seg := range q.Segments {
		nodes = e.segment(seg, nodes)
	}
	return nodes
}

func (e *evaluator) segment(seg ast.Segment, inputs []*document.Node) []*document.Node {
	var out []*document.Node
	for _, in := range inputs {
		switch seg.Kind {
		case ast.SegChild:
			for _, sel := range seg.Selectors {
				out = e.selector(sel, in, out)
			}
		case ast.SegDescendant:
			out = e.descendants(seg.Selectors, in, out)
		}
	}
	return out
}

// descendants 按前序访问节点（包含自身），先应用全部 selector，再访问子节点。
func (e *evaluator) descendants(selectors []ast.Selector, in *document.Node, out []*document.Node) []*document.Node {
	for _, sel := range selectors {
		out = e.selector(sel, in, out)
	}
	for _, child := range in.Children {
		out = e.descendants(selectors, child, out)
	}
	return out
}

func (e *evaluator) selector(sel ast.Selector, in *document.Node, out []*document.Node) []*document.Node {
	switch s := sel.(type) {
	case *ast.NameSelector:
		if obj, ok := in.Value.(document.Object); ok {
			if i, exists := obj.MemberIndex(s.Name); exists {
				out = append(out, in.Children[i])
			}
		}
	case *ast.WildcardSelector:
		out = append(out, in.Children...)
	case *ast.IndexSelector:
		if _, ok := in.Value.(document.Array); ok {
			idx := s.Index
			length := int64(len(in.Children))
			if idx < 0 {
				idx += length
			}
			if idx >= 0 && idx < length {
				out = append(out, in.Children[idx])
			}
		}
	case *ast.SliceSelector:
		out = e.sliceSelector(s, in, out)
	case *ast.FilterSelector:
		for _, child := range in.Children {
			if e.evalLogical(s.Expr, child) {
				out = append(out, child)
			}
		}
	}
	return out
}

func (e *evaluator) sliceSelector(s *ast.SliceSelector, in *document.Node, out []*document.Node) []*document.Node {
	if _, ok := in.Value.(document.Array); !ok {
		return out
	}
	step := int64(1)
	if s.Step != nil {
		step = *s.Step
	}
	if step == 0 {
		return out
	}

	length := int64(len(in.Children))
	lower, upper := int64(0), length
	start, end := int64(0), length
	if step < 0 {
		lower, upper = -1, length-1
		start, end = length-1, -1
	}
	normalize := func(index int64) int64 {
		if index < 0 {
			index += length
		}
		if index < lower {
			return lower
		}
		if index > upper {
			return upper
		}
		return index
	}
	if s.Start != nil {
		start = normalize(*s.Start)
	}
	if s.End != nil {
		end = normalize(*s.End)
	}

	if step > 0 {
		for i := start; i < end; {
			out = append(out, in.Children[i])
			// 在相加前判断是否到达边界，避免大步长导致整数溢出。
			if step >= end-i {
				break
			}
			i += step
		}
	} else {
		for i := start; i > end; {
			out = append(out, in.Children[i])
			if step <= end-i {
				break
			}
			i += step
		}
	}
	return out
}
