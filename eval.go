package jsonpath

import (
	"github.com/saltfishpr/jsonpath/internal/ast"
	"github.com/saltfishpr/jsonpath/internal/document"
	"github.com/saltfishpr/jsonpath/internal/iregexp"
)

func (q *Query) run(doc *Document) ([]*document.Node, error) {
	ev := &evaluator{root: doc.root, literals: q.literals, regexes: q.regexes}
	return ev.query(q.query, doc.root)
}

type evaluator struct {
	root     *document.Node
	literals map[*ast.Literal]*document.Node
	regexes  map[string]*iregexp.Regexp
}

func (e *evaluator) query(q *ast.Query, current *document.Node) ([]*document.Node, error) {
	if q.IsRoot {
		current = e.root
	}
	nodes := []*document.Node{current}
	for _, seg := range q.Segments {
		var err error
		nodes, err = e.segment(seg, nodes)
		if err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (e *evaluator) segment(seg ast.Segment, inputs []*document.Node) ([]*document.Node, error) {
	var out []*document.Node
	for _, in := range inputs {
		var err error
		switch seg.Kind {
		case ast.SegChild:
			for _, sel := range seg.Selectors {
				out, err = e.selector(sel, in, out)
				if err != nil {
					return nil, err
				}
			}
		case ast.SegDescendant:
			out, err = e.descendants(seg.Selectors, in, out)
			if err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

// descendants 按前序访问节点（包含自身），先应用全部 selector，再访问子节点。
func (e *evaluator) descendants(selectors []ast.Selector, in *document.Node, out []*document.Node) ([]*document.Node, error) {
	var err error
	for _, sel := range selectors {
		out, err = e.selector(sel, in, out)
		if err != nil {
			return nil, err
		}
	}
	switch in.Kind() {
	case document.KindArray:
		for _, child := range in.Array() {
			out, err = e.descendants(selectors, child, out)
			if err != nil {
				return nil, err
			}
		}
	case document.KindObject:
		for _, member := range in.Members() {
			out, err = e.descendants(selectors, member.Value, out)
			if err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

func (e *evaluator) selector(sel ast.Selector, in *document.Node, out []*document.Node) ([]*document.Node, error) {
	switch s := sel.(type) {
	case *ast.NameSelector:
		if child := in.Get(s.Name); child != nil {
			out = append(out, child)
		}
	case *ast.WildcardSelector:
		switch in.Kind() {
		case document.KindArray:
			out = append(out, in.Array()...)
		case document.KindObject:
			for _, member := range in.Members() {
				out = append(out, member.Value)
			}
		}
	case *ast.IndexSelector:
		if in.Kind() == document.KindArray {
			idx := s.Index
			length := int64(in.Len())
			if idx < 0 {
				idx += length
			}
			if idx >= 0 && idx < length {
				out = append(out, in.Array()[idx])
			}
		}
	case *ast.SliceSelector:
		out = e.sliceSelector(s, in, out)
	case *ast.FilterSelector:
		switch in.Kind() {
		case document.KindArray:
			for _, child := range in.Array() {
				ok, err := e.evalLogical(s.Expr, child)
				if err != nil {
					return nil, err
				}
				if ok {
					out = append(out, child)
				}
			}
		case document.KindObject:
			for _, member := range in.Members() {
				ok, err := e.evalLogical(s.Expr, member.Value)
				if err != nil {
					return nil, err
				}
				if ok {
					out = append(out, member.Value)
				}
			}
		}
	}
	return out, nil
}

func (e *evaluator) sliceSelector(s *ast.SliceSelector, in *document.Node, out []*document.Node) []*document.Node {
	if in.Kind() != document.KindArray {
		return out
	}
	step := int64(1)
	if s.Step != nil {
		step = *s.Step
	}
	if step == 0 {
		return out
	}

	length := int64(in.Len())
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
			out = append(out, in.Array()[i])
			// 在相加前判断是否到达边界，避免大步长导致整数溢出。
			if step >= end-i {
				break
			}
			i += step
		}
	} else {
		for i := start; i > end; {
			out = append(out, in.Array()[i])
			if step <= end-i {
				break
			}
			i += step
		}
	}
	return out
}
