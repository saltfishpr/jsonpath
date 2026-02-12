package jsonpath

// Query represents a JSONPath query: $ followed by segments.
type Query struct {
	Segments []*Segment
}

// SegmentType distinguishes child vs descendant segments.
type SegmentType int

const (
	ChildSegment SegmentType = iota
	DescendantSegment
)

// Segment is a child or descendant segment containing selectors.
type Segment struct {
	Type      SegmentType
	Selectors []*Selector
}

// SelectorType distinguishes different selector types.
type SelectorType int

const (
	NameSelector     SelectorType = iota // 'name' or "name"
	WildcardSelector                     // *
	IndexSelector                        // integer index
	SliceSelector                        // start:end:step
	FilterSelector                       // ?<logical-expr>
)

// Selector represents a single selector within a segment.
type Selector struct {
	Type   SelectorType
	Name   string       // for NameSelector
	Index  int          // for IndexSelector
	Slice  *SliceParams // for SliceSelector
	Filter *FilterExpr  // for FilterSelector
}

// SliceParams holds start:end:step for a slice selector.
type SliceParams struct {
	Start *int
	End   *int
	Step  *int
}

// FilterExprType identifies the type of filter expression.
type FilterExprType int

const (
	FilterLogicalOr  FilterExprType = iota // left || right
	FilterLogicalAnd                       // left && right
	FilterLogicalNot                       // !operand
	FilterParen                            // (operand)
	FilterComparison                       // comparison
	FilterTest                             // test expression
)

// FilterExpr represents a filter expression.
type FilterExpr struct {
	Type    FilterExprType
	Left    *FilterExpr
	Right   *FilterExpr
	Operand *FilterExpr
	Comp    *Comparison
	Test    *TestExpr
}

// CompOp is a comparison operator.
type CompOp int

const (
	CompEq CompOp = iota // ==
	CompNe               // !=
	CompLt               // <
	CompLe               // <=
	CompGt               // >
	CompGe               // >=
)

// Comparison represents a comparison expression.
type Comparison struct {
	Left  *Comparable
	Op    CompOp
	Right *Comparable
}

// ComparableType identifies what a comparable holds.
type ComparableType int

const (
	ComparableLiteral ComparableType = iota
	ComparableSingularQuery
	ComparableFuncExpr
)

// Comparable is one side of a comparison.
type Comparable struct {
	Type          ComparableType
	Literal       *LiteralValue
	SingularQuery *SingularQuery
	FuncExpr      *FuncCall
}

type LiteralType int

const (
	LiteralString LiteralType = iota
	LiteralNumber
	LiteralTrue
	LiteralFalse
	LiteralNull
)

// LiteralValue represents a literal value in expressions
type LiteralValue struct {
	Type  LiteralType
	Value string
}

// SingularQuery is a query that produces at most one node
type SingularQuery struct {
	Relative bool // true = starts with @, false = starts with $
	Segments []*SingularSegment
}

type SingularSegmentType int

const (
	SingularNameSegment SingularSegmentType = iota
	SingularIndexSegment
)

// SingularSegment is a name or index segment in a singular query
type SingularSegment struct {
	Type  SingularSegmentType
	Name  string
	Index int
}

// TestExpr represents a test expression (existence or function)
type TestExpr struct {
	FilterQuery *FilterQuery
	FuncExpr    *FuncCall
}

// FilterQuery is a query used in a filter (relative or absolute)
type FilterQuery struct {
	Relative bool // true = starts with @, false = starts with $
	Segments []*Segment
}

// FuncCall represents a function call expression
type FuncCall struct {
	Name string
	Args []*FuncArg
}

// FuncArgType identifies the type of function argument
type FuncArgType int

const (
	FuncArgLiteral FuncArgType = iota
	FuncArgFilterQuery
	FuncArgLogicalExpr
	FuncArgFuncExpr
)

// FuncArg represents a function argument
type FuncArg struct {
	Type        FuncArgType
	Literal     *LiteralValue
	FilterQuery *FilterQuery
	LogicalExpr *FilterExpr
	FuncExpr    *FuncCall
}
