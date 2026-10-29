// Package document parses JSON into trees with exact numbers and source fragments.
// Parsed nodes retain their location in the tree. Trees support concurrent reads
// provided callers do not modify input bytes or returned slices.
package document

import "unicode/utf8"

// Kind identifies the JSON value category and its applicable [Node] accessors.
type Kind uint8

const (
	// KindNull represents the JSON null value.
	KindNull Kind = iota
	// KindBool exposes its value through [Node.Bool].
	KindBool
	// KindNumber exposes an exact numeric value through [Node.Number].
	KindNumber
	// KindString exposes decoded text through [Node.String].
	KindString
	// KindArray exposes ordered elements through [Node.Array].
	KindArray
	// KindObject exposes named values through [Node.Members] and [Node.Get].
	KindObject
)

// String returns a lowercase category name, or "unknown" for an invalid kind.
func (k Kind) String() string {
	switch k {
	case KindNull:
		return "null"
	case KindBool:
		return "bool"
	case KindNumber:
		return "number"
	case KindString:
		return "string"
	case KindArray:
		return "array"
	case KindObject:
		return "object"
	}
	return "unknown"
}

// Member pairs a decoded object name with its value. Names are compared exactly,
// without Unicode normalization.
type Member struct {
	Name  string
	Value *Node
}

// Node is an immutable JSON value. Use [Parse] or the New constructors to create
// nodes with valid location metadata. Unless documented otherwise, methods
// require a non-nil receiver.
type Node struct {
	kind    Kind
	boolean bool
	raw     []byte
	number  Number
	str     string
	arr     []*Node
	members []Member // Preserve source order for traversal.

	memberIndex map[string]*Node

	parent *Node
	index  int64
	name   string
}

// NewNull returns a standalone null value with no raw source or parent.
func NewNull() *Node { return &Node{kind: KindNull, index: -1} }

// NewTrue returns a standalone true value with no raw source or parent.
func NewTrue() *Node { return &Node{kind: KindBool, boolean: true, index: -1} }

// NewFalse returns a standalone false value with no raw source or parent.
func NewFalse() *Node { return &Node{kind: KindBool, boolean: false, index: -1} }

// NewNumber returns an exact numeric value with no raw source or parent.
// The caller must supply valid JSON number syntax; this function does not validate it.
func NewNumber(number string) *Node {
	return &Node{kind: KindNumber, number: ParseNumber(number), index: -1}
}

// NewString wraps decoded text without JSON unescaping or UTF-8 validation.
// The returned node has no raw source or parent.
func NewString(s string) *Node {
	return &Node{kind: KindString, str: s, index: -1}
}

// Kind identifies which scalar accessor or container view applies to n.
func (n *Node) Kind() Kind { return n.kind }

// IsNull reports whether n is a JSON null value. A nil receiver returns false.
func (n *Node) IsNull() bool { return n != nil && n.kind == KindNull }

// Raw returns the node's JSON source fragment, excluding surrounding whitespace.
// It aliases the parser input and must not be modified. Nodes created with New
// constructors have no source and return nil.
func (n *Node) Raw() []byte { return n.raw }

// Bool returns the value of a [KindBool] node. Other kinds return false.
func (n *Node) Bool() bool { return n.boolean }

// Number returns the exact value of a [KindNumber] node. Other kinds return the
// zero [Number].
func (n *Node) Number() Number { return n.number }

// String returns decoded text for a [KindString] node. Other kinds return "".
func (n *Node) String() string { return n.str }

// Array returns array elements in source order, or nil for other kinds.
// The slice belongs to the node and must not be modified.
func (n *Node) Array() []*Node { return n.arr }

// Members returns object members in source order, or nil for other kinds.
// The slice and its [Member] entries belong to the node and must not be modified.
func (n *Node) Members() []Member { return n.members }

// Len counts array elements, object members, or Unicode code points in a string.
// It returns -1 for other kinds and for a nil receiver.
func (n *Node) Len() int {
	if n == nil {
		return -1
	}
	switch n.kind {
	case KindArray:
		return len(n.arr)
	case KindObject:
		return len(n.members)
	case KindString:
		return utf8.RuneCountInString(n.str)
	}
	return -1
}

// Get finds an object member by its exact decoded name. Missing names and
// non-object nodes return nil.
func (n *Node) Get(name string) *Node {
	return n.memberIndex[name]
}

// Parent returns the containing array or object, or nil for a root or standalone node.
func (n *Node) Parent() *Node { return n.parent }

// Index returns the zero-based position in the parent array, or -1 when the
// node is not an array element.
func (n *Node) Index() int64 { return n.index }

// Name returns the decoded member name in the parent object, or "" otherwise.
// Check [Node.Parent] to distinguish an empty member name from a node outside an object.
func (n *Node) Name() string { return n.name }
