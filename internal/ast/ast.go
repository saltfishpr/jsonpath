// Package ast 定义 RFC 9535 JSONPath 的抽象语法树。
// 查询和表达式节点保留源码位置，供编译器报告错误。
package ast

type Pos struct {
	Offset int // 从 0 开始的字节偏移
	Line   int // 从 1 开始的行号
	Col    int // 从 1 开始的列号，按 rune 计数
}

type Query struct {
	IsRoot   bool // true 表示根查询（$），false 表示当前节点查询（@）
	Segments []Segment
	Pos      Pos
}

type SegmentKind uint8

const (
	SegChild SegmentKind = iota
	SegDescendant
)

type Segment struct {
	Kind      SegmentKind
	Selectors []Selector // 至少一个
	Pos       Pos
}

type Selector interface {
	selectorNode()
	Position() Pos
}

type NameSelector struct {
	Name string // 解码后的 UTF-8 字符串
	Raw  string // 原文（含引号或点号后的 member-name-shorthand）
	Pos  Pos
}

type WildcardSelector struct {
	Pos Pos
}

type IndexSelector struct {
	Index int64
	Pos   Pos
}

type SliceSelector struct {
	Start, End, Step *int64
	Pos              Pos
}

type FilterSelector struct {
	Expr LogicalExpr
	Pos  Pos
}

func (*NameSelector) selectorNode()     {}
func (*WildcardSelector) selectorNode() {}
func (*IndexSelector) selectorNode()    {}
func (*SliceSelector) selectorNode()    {}
func (*FilterSelector) selectorNode()   {}

func (n *NameSelector) Position() Pos     { return n.Pos }
func (n *WildcardSelector) Position() Pos { return n.Pos }
func (n *IndexSelector) Position() Pos    { return n.Pos }
func (n *SliceSelector) Position() Pos    { return n.Pos }
func (n *FilterSelector) Position() Pos   { return n.Pos }

// ExprType 对应 RFC 9535 §2.4.1 的三种类型。
type ExprType uint8

const (
	// TypeValue represents ValueType: a JSON value or Nothing.
	TypeValue ExprType = iota
	// TypeLogical represents LogicalType: LogicalTrue or LogicalFalse.
	TypeLogical
	// TypeNodes represents NodesType: a node list.
	TypeNodes
)

// String returns the RFC 9535 name of the expression type.
func (t ExprType) String() string {
	switch t {
	case TypeValue:
		return "ValueType"
	case TypeLogical:
		return "LogicalType"
	case TypeNodes:
		return "NodesType"
	}
	return "unknown"
}

type Expr interface {
	exprNode()
	Type() ExprType
	Position() Pos
}

// LogicalExpr 是逻辑上下文中的表达式（&&、||、!、比较、test）。
type LogicalExpr interface {
	Expr
	logicalExpr()
}

// Comparable 是 ComparisonExpr 两侧允许出现的表达式。
// RFC 9535 §2.3.5.2 规定两侧必须是：literal / singular-query / 返回 ValueType 的函数调用。
type Comparable interface {
	Expr
	comparable()
}

type OrExpr struct {
	Operands []LogicalExpr // 长度 >= 2
	Pos      Pos
}

type AndExpr struct {
	Operands []LogicalExpr // 长度 >= 2
	Pos      Pos
}

type NotExpr struct {
	Operand LogicalExpr
	Pos     Pos
}

// ParenExpr 是 (expr)，保留括号便于格式化输出。
type ParenExpr struct {
	Inner LogicalExpr
	Pos   Pos
}

type ComparisonExpr struct {
	Op          CmpOp
	Left, Right Comparable
	Pos         Pos
}

type CmpOp uint8

const (
	CmpEq CmpOp = iota // ==
	CmpNe              // !=
	CmpLt              // <
	CmpLe              // <=
	CmpGt              // >
	CmpGe              // >=
)

// String returns the comparison operator's source spelling.
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

// TestExpr 表示 filter 中用于存在性测试的查询或函数调用。
type TestExpr struct {
	Target  Expr // *FilterQuery 或 *FunctionCall（Logical/Nodes 返回类型）
	Negated bool
	Pos     Pos
}

func (*OrExpr) exprNode()         {}
func (*AndExpr) exprNode()        {}
func (*NotExpr) exprNode()        {}
func (*ParenExpr) exprNode()      {}
func (*ComparisonExpr) exprNode() {}
func (*TestExpr) exprNode()       {}

func (*OrExpr) Type() ExprType         { return TypeLogical }
func (*AndExpr) Type() ExprType        { return TypeLogical }
func (*NotExpr) Type() ExprType        { return TypeLogical }
func (*ParenExpr) Type() ExprType      { return TypeLogical }
func (*ComparisonExpr) Type() ExprType { return TypeLogical }
func (*TestExpr) Type() ExprType       { return TypeLogical }

func (e *OrExpr) Position() Pos         { return e.Pos }
func (e *AndExpr) Position() Pos        { return e.Pos }
func (e *NotExpr) Position() Pos        { return e.Pos }
func (e *ParenExpr) Position() Pos      { return e.Pos }
func (e *ComparisonExpr) Position() Pos { return e.Pos }
func (e *TestExpr) Position() Pos       { return e.Pos }

func (*OrExpr) logicalExpr()         {}
func (*AndExpr) logicalExpr()        {}
func (*NotExpr) logicalExpr()        {}
func (*ParenExpr) logicalExpr()      {}
func (*ComparisonExpr) logicalExpr() {}
func (*TestExpr) logicalExpr()       {}

type LiteralKind uint8

const (
	LitString LiteralKind = iota
	LitInt
	LitFloat
	LitBool
	LitNull
)

type LiteralValue struct {
	Kind LiteralKind
	Str  string
	Bool bool
	Raw  string
}

type Literal struct {
	Value LiteralValue
	Pos   Pos
}

// SingularQuery 是只含 name/index selector 的查询，静态可判定最多一个节点。
type SingularQuery struct {
	IsRoot   bool
	Segments []SingularSegment
	Pos      Pos
}

// SingularSegment 表示一个 name 或 index selector。
// Name 和 Index 必须恰有一个非 nil；指针用于区分未设置与合法的空成员名、索引零。
type SingularSegment struct {
	Name  *string
	Index *int64
	Pos   Pos
}

type FunctionCall struct {
	Name string
	Args []Expr
	Ret  ExprType
	Pos  Pos
}

type FilterQuery struct {
	Query *Query
	Pos   Pos
}

func (*Literal) exprNode()       {}
func (*SingularQuery) exprNode() {}
func (*FunctionCall) exprNode()  {}
func (*FilterQuery) exprNode()   {}

func (*Literal) Type() ExprType       { return TypeValue }
func (*SingularQuery) Type() ExprType { return TypeValue } // 用作 Comparable 时按 ValueType 对待
func (f *FunctionCall) Type() ExprType {
	// parse 阶段 Ret 为零值（TypeValue），compile 阶段覆写。
	return f.Ret
}
func (*FilterQuery) Type() ExprType { return TypeNodes }

func (n *Literal) Position() Pos       { return n.Pos }
func (n *SingularQuery) Position() Pos { return n.Pos }
func (n *FunctionCall) Position() Pos  { return n.Pos }
func (n *FilterQuery) Position() Pos   { return n.Pos }

func (*Literal) comparable()       {}
func (*SingularQuery) comparable() {}

// 函数返回类型由编译阶段检查，此处先允许函数调用作为比较操作数。
func (*FunctionCall) comparable() {}
