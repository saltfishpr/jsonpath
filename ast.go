package jsonpath

// Query represents a complete JSONPath query: $ followed by segments
type Query struct {
	Segments []*Segment
}

// SegmentType distinguishes child vs descendant segments
type SegmentType int

const (
	ChildSegment SegmentType = iota
	DescendantSegment
)

// Segment is a child or descendant segment containing selectors
type Segment struct {
	Type      SegmentType
	Selectors []*Selector
}

// SelectorKind distinguishes different selector types
type SelectorKind int

const (
	NameSelector     SelectorKind = iota // 'name' or "name"
	WildcardSelector                     // *
	IndexSelector                        // integer index
	SliceSelector                        // start:end:step
	FilterSelector                       // ?<logical-expr>
)

// Selector represents a single selector within a segment
type Selector struct {
	Kind   SelectorKind
	Name   string       // for NameSelector
	Index  int          // for IndexSelector
	Slice  *SliceParams // for SliceSelector
	Filter *FilterExpr  // for FilterSelector
}

// SliceParams holds the start:end:step parameters for a slice selector
type SliceParams struct {
	Start *int // nil means default
	End   *int // nil means default
	Step  *int // nil means default (1)
}

// FilterExprKind identifies the kind of filter expression
type FilterExprKind int

const (
	FilterLogicalOr  FilterExprKind = iota // left || right
	FilterLogicalAnd                       // left && right
	FilterLogicalNot                       // !operand
	FilterParen                            // (operand)
	FilterComparison                       // comparison
	FilterTest                             // test expression
)

// FilterExpr represents a filter expression (logical expression)
type FilterExpr struct {
	Kind FilterExprKind
	// For LogicalOr/LogicalAnd
	Left  *FilterExpr
	Right *FilterExpr
	// For LogicalNot/Paren
	Operand *FilterExpr
	// For Comparison
	Comp *Comparison
	// For TestExpr (existence test or function test)
	Test *TestExpr
}

// CompOp is a comparison operator
type CompOp int

const (
	CompEq CompOp = iota // ==
	CompNe               // !=
	CompLt               // <
	CompLe               // <=
	CompGt               // >
	CompGe               // >=
)

// Comparison represents a comparison expression
type Comparison struct {
	Left  *Comparable
	Op    CompOp
	Right *Comparable
}

// ComparableKind identifies what a comparable holds
type ComparableKind int

const (
	ComparableLiteral ComparableKind = iota
	ComparableSingularQuery
	ComparableFuncExpr
)

// Comparable is one side of a comparison (literal, singular query, or function)
type Comparable struct {
	Kind ComparableKind
	// For literal
	Literal *LiteralValue
	// For singular query
	SingularQuery *SingularQuery
	// For function expression
	FuncExpr *FuncCall
}

type LiteralType int

const (
	LiteralTypeString LiteralType = iota
	LiteralTypeNumber
	LiteralTypeTrue
	LiteralTypeFalse
	LiteralTypeNull
)

// LiteralValue 字面量
type LiteralValue struct {
	Type  LiteralType
	Value string
}

// SingularQuery is a query that produces at most one node
type SingularQuery struct {
	Relative bool // true = starts with @, false = starts with $
	Segments []*SingularSegment
}

// SingularSegment is a name or index segment in a singular query
type SingularSegment struct {
	IsIndex bool
	Name    string
	Index   int
}

// TestExpr represents a test expression (existence or function)
type TestExpr struct {
	Negated bool
	// Either a filter query (existence test) or function call
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

// FuncArg represents a function argument
type FuncArg struct {
	Kind        FuncArgKind
	Literal     *LiteralValue
	FilterQuery *FilterQuery
	LogicalExpr *FilterExpr
	FuncExpr    *FuncCall
}

// FuncArgKind identifies the kind of function argument
type FuncArgKind int

const (
	FuncArgLiteral FuncArgKind = iota
	FuncArgFilterQuery
	FuncArgLogicalExpr
	FuncArgFuncExpr
)
