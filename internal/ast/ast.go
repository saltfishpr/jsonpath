// Package ast represents JSONPath syntax with source positions for diagnostics.
// Trees describe parsed structure; function names and types still require
// semantic validation before evaluation.
package ast

// Pos locates a token in the original expression.
type Pos struct {
	Offset int // Zero-based byte offset.
	Line   int // One-based line number.
	Col    int // One-based column counted in runes.
}

// Query applies [Query.Segments] in order from the root or the current filter node.
// An empty segment list selects the starting node itself.
type Query struct {
	IsRoot   bool // True starts at $; false starts at @.
	Segments []Segment
	Pos      Pos
}

// SegmentKind determines which nodes receive a segment's selectors.
type SegmentKind uint8

const (
	// SegChild applies selectors to each input node.
	SegChild SegmentKind = iota
	// SegDescendant applies selectors to each input node and all its descendants.
	SegDescendant
)

// Segment combines selector results in source order without deduplication.
type Segment struct {
	Kind      SegmentKind
	Selectors []Selector // Must contain at least one selector.
	Pos       Pos
}

// Selector is a selection rule implemented only by this package's node types.
type Selector interface {
	selectorNode()
	// Position returns the selector's starting source position.
	Position() Pos
}

// NameSelector selects an object member by its decoded name, without Unicode
// normalization. It selects nothing from non-object inputs.
type NameSelector struct {
	Name string // Decoded UTF-8 member name.
	Raw  string // Source spelling, including quotes when present.
	Pos  Pos
}

// WildcardSelector selects all immediate array elements or object member values.
// Scalars have no children.
type WildcardSelector struct {
	Pos Pos
}

// IndexSelector selects an array element by zero-based index. Negative indices
// count from the end; out-of-range indices select nothing.
type IndexSelector struct {
	Index int64
	Pos   Pos
}

// SliceSelector selects array elements with an exclusive end bound. Nil fields
// retain omitted arguments for direction-dependent defaults; a zero step
// selects nothing.
type SliceSelector struct {
	Start, End, Step *int64
	Pos              Pos
}

// FilterSelector tests [FilterSelector.Expr] with each immediate child as the current node.
// Only children for which the expression is true are selected.
type FilterSelector struct {
	Expr LogicalExpr
	Pos  Pos
}

func (*NameSelector) selectorNode()     {}
func (*WildcardSelector) selectorNode() {}
func (*IndexSelector) selectorNode()    {}
func (*SliceSelector) selectorNode()    {}
func (*FilterSelector) selectorNode()   {}

// Position returns the member selector's starting source position.
func (n *NameSelector) Position() Pos { return n.Pos }

// Position returns the wildcard's source position.
func (n *WildcardSelector) Position() Pos { return n.Pos }

// Position returns the index token's source position.
func (n *IndexSelector) Position() Pos { return n.Pos }

// Position returns the slice's starting source position.
func (n *SliceSelector) Position() Pos { return n.Pos }

// Position returns the filter's question-mark position.
func (n *FilterSelector) Position() Pos { return n.Pos }

// Expr is a filter expression implemented only by this package's node types.
type Expr interface {
	exprNode()
	// Position returns the source location used to diagnose this expression.
	Position() Pos
}

// LogicalExpr marks syntax that yields a logical result after semantic validation.
type LogicalExpr interface {
	Expr
	logicalExpr()
}

// Comparable marks syntax allowed as a comparison operand. Function calls
// satisfy this interface before their ValueType return requirement is checked.
type Comparable interface {
	Expr
	comparable()
}

// OrExpr is a disjunction whose operands retain source evaluation order.
type OrExpr struct {
	Operands []LogicalExpr // At least two operands.
	Pos      Pos
}

// AndExpr is a conjunction whose operands retain source evaluation order.
type AndExpr struct {
	Operands []LogicalExpr // At least two operands.
	Pos      Pos
}

// NotExpr negates a logical expression; [NotExpr.Pos] locates the exclamation mark.
type NotExpr struct {
	Operand LogicalExpr
	Pos     Pos
}

// ParenExpr preserves explicit grouping and the opening parenthesis position.
type ParenExpr struct {
	Inner LogicalExpr
	Pos   Pos
}

// ComparisonExpr compares two ValueType operands; [ComparisonExpr.Pos] locates the operator.
type ComparisonExpr struct {
	Op          CmpOp
	Left, Right Comparable
	Pos         Pos
}

// CmpOp identifies an RFC 9535 comparison, whose operands may include Nothing.
type CmpOp uint8

const (
	// CmpEq tests equality, including equality between two missing values.
	CmpEq CmpOp = iota
	// CmpNe negates equality.
	CmpNe
	// CmpLt orders numbers or strings strictly.
	CmpLt
	// CmpLe accepts either less-than ordering or equality.
	CmpLe
	// CmpGt reverses the operands of less-than ordering.
	CmpGt
	// CmpGe accepts either greater-than ordering or equality.
	CmpGe
)

// String returns the JSONPath operator spelling, or "?" for an unknown value.
func (op CmpOp) String() string {
	switch op {
	case CmpEq:
		return "=="
	case CmpNe:
		return "!="
	case CmpLt:
		return "<"
	case CmpLe:
		return "<="
	case CmpGt:
		return ">"
	case CmpGe:
		return ">="
	}
	return "?"
}

// TestExpr uses a query's non-emptiness or a function's logical result as a
// condition, optionally negated. ValueType functions are invalid targets.
type TestExpr struct {
	Target  Expr // *FilterQuery or *FunctionCall returning LogicalType or NodesType.
	Negated bool
	Pos     Pos
}

func (*OrExpr) exprNode()         {}
func (*AndExpr) exprNode()        {}
func (*NotExpr) exprNode()        {}
func (*ParenExpr) exprNode()      {}
func (*ComparisonExpr) exprNode() {}
func (*TestExpr) exprNode()       {}

// Position returns the first operand's source position.
func (e *OrExpr) Position() Pos { return e.Pos }

// Position returns the first operand's source position.
func (e *AndExpr) Position() Pos { return e.Pos }

// Position returns the negation operator's source position.
func (e *NotExpr) Position() Pos { return e.Pos }

// Position returns the opening parenthesis position.
func (e *ParenExpr) Position() Pos { return e.Pos }

// Position returns the comparison operator's source position.
func (e *ComparisonExpr) Position() Pos { return e.Pos }

// Position returns the test's starting source position.
func (e *TestExpr) Position() Pos { return e.Pos }

func (*OrExpr) logicalExpr()         {}
func (*AndExpr) logicalExpr()        {}
func (*NotExpr) logicalExpr()        {}
func (*ParenExpr) logicalExpr()      {}
func (*ComparisonExpr) logicalExpr() {}
func (*TestExpr) logicalExpr()       {}

// LiteralKind selects the relevant decoded field or numeric source text in a
// [LiteralValue].
type LiteralKind uint8

const (
	// LitString uses the decoded [LiteralValue.Str] field.
	LitString LiteralKind = iota
	// LitInt uses [LiteralValue.Raw] for a number without a fraction or exponent.
	LitInt
	// LitFloat uses [LiteralValue.Raw] for a number with a fraction or exponent; it does not
	// imply conversion to floating point.
	LitFloat
	// LitBool uses the [LiteralValue.Bool] field.
	LitBool
	// LitNull represents JSON null and needs no decoded payload.
	LitNull
)

// LiteralValue retains source text in [LiteralValue.Raw], including string quotes.
// [LiteralValue.Kind] determines whether [LiteralValue.Str] or [LiteralValue.Bool]
// is meaningful; numbers remain exact in [LiteralValue.Raw].
type LiteralValue struct {
	Kind LiteralKind
	Str  string
	Bool bool
	Raw  string
}

// Literal supplies a JSON scalar value independently of the current node.
type Literal struct {
	Value LiteralValue
	Pos   Pos
}

// SingularQuery contains only single name or index selections, so it can yield
// at most one node. [SingularQuery.IsRoot] chooses $ over @; no segments select that node itself.
type SingularQuery struct {
	IsRoot   bool
	Segments []SingularSegment
	Pos      Pos
}

// SingularSegment selects one object member or array element. Exactly one of
// [SingularSegment.Name] and [SingularSegment.Index] must be non-nil; empty names and index zero are valid selections.
type SingularSegment struct {
	Name  *string
	Index *int64
	Pos   Pos
}

// FunctionCall retains arguments in source order. The compiler must resolve
// [FunctionCall.Name] and validate argument and result types before evaluation.
type FunctionCall struct {
	Name string
	Args []Expr
	Pos  Pos
}

// FilterQuery yields a node list inside a filter. Context determines whether
// that list is used directly, tested for existence, or converted to a value.
type FilterQuery struct {
	Query *Query
	Pos   Pos
}

func (*Literal) exprNode()       {}
func (*SingularQuery) exprNode() {}
func (*FunctionCall) exprNode()  {}
func (*FilterQuery) exprNode()   {}

// Position returns the literal token's source position.
func (n *Literal) Position() Pos { return n.Pos }

// Position returns the query's $ or @ position.
func (n *SingularQuery) Position() Pos { return n.Pos }

// Position returns the function name's source position.
func (n *FunctionCall) Position() Pos { return n.Pos }

// Position returns the query's $ or @ position.
func (n *FilterQuery) Position() Pos { return n.Pos }

func (*Literal) comparable()       {}
func (*SingularQuery) comparable() {}
func (*FunctionCall) comparable()  {}
