package jsonpath

import "github.com/saltfishpr/jsonpath/internal/iregexp"

type pathExpr struct {
	root     bool
	segments []segment
}
type segment struct {
	descendant bool
	selectors  []selector
}
type selector struct {
	kind             byte
	name             string
	index            int64
	start, end, step *int64
	filter           *expression
}
type expression struct {
	kind          byte
	offset        int
	literal       *jsonNode
	path          pathExpr
	name          string
	args          []*expression
	left, right   *expression
	regex         *iregexp.Pattern
	regexConstant bool
}
type exprType uint8

const (
	valueType exprType = iota
	nodesType
	logicalType
)

func (p pathExpr) singular() bool {
	for _, s := range p.segments {
		if s.descendant || len(s.selectors) != 1 {
			return false
		}
		k := s.selectors[0].kind
		if k != 'n' && k != 'i' {
			return false
		}
	}
	return true
}
