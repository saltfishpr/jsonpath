// Package document 提供只读 JSON 节点树及保留原始文本的解析器。
package document

import (
	"unicode/utf8"
)

type Kind uint8

const (
	KindNull Kind = iota
	KindBool
	KindNumber
	KindString
	KindArray
	KindObject
)

// String 返回种类名，便于调试与错误信息。
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

// Member 是对象的一条键值对；Name 已解码，Value 带有用于重建路径的父节点信息。
type Member struct {
	Name  string
	Value *Node
}

// Node 是只读 JSON 值；解析完成后可安全地并发复用。
//
// parent、index 和 name 保存节点位置，用于重建 RFC 9535 Normalized Path：根节点没有
// parent；对象成员保存父对象和键名；数组元素保存父数组和下标。
//
// 每个节点都需要独立的 parent 链，因此 null、true 和 false 节点也不共享单例。
type Node struct {
	kind Kind

	raw     []byte   // 保留原始输入中的片段，供查询结果直接引用
	boolean bool     // KindBool
	number  Number   // KindNumber: 构造时解析，之后只读
	str     string   // KindString: 已解码的 UTF-8
	arr     []*Node  // KindArray
	members []Member // KindObject，按出现顺序存储

	parent *Node  // 所属容器；根节点为 nil
	index  int64  // 在父数组中的下标；非数组元素为 -1
	name   string // 在父对象中的键名；非对象成员为空字符串
}

func NewNull() *Node  { return &Node{kind: KindNull, index: -1} }
func NewTrue() *Node  { return &Node{kind: KindBool, boolean: true, index: -1} }
func NewFalse() *Node { return &Node{kind: KindBool, boolean: false, index: -1} }

func NewNumber(number string) *Node {
	return &Node{kind: KindNumber, number: ParseNumber(number), index: -1}
}

func NewString(s string) *Node {
	return &Node{kind: KindString, str: s, index: -1}
}

func (n *Node) Kind() Kind { return n.kind }

func (n *Node) IsNull() bool { return n != nil && n.kind == KindNull }

// Raw 返回与输入共享内存的 JSON 原文，调用方不得修改；手工构造的节点返回 nil。
func (n *Node) Raw() []byte { return n.raw }

// Bool 返回布尔值；调用方应先检查 KindBool。
func (n *Node) Bool() bool { return n.boolean }

// Number 返回数字原文；调用方应先检查 KindNumber。
func (n *Node) Number() Number { return n.number }

// String 返回解码后的字符串；调用方应先检查 KindString。
func (n *Node) String() string { return n.str }

// Array 返回只读的数组元素切片；其余种类返回 nil，调用方不得修改切片。
func (n *Node) Array() []*Node { return n.arr }

// Members 返回按输入顺序排列的只读对象成员切片；其余种类返回 nil。
// 调用方不得修改切片或其中的 Member。
func (n *Node) Members() []Member { return n.members }

// Len 返回数组元素数、对象成员数或字符串的 Unicode 码点数；其余种类返回 -1。
// 字符串按码点计数，与 RFC 9535 的 length() 语义一致。
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

// Get 返回指定名称的对象成员值；非对象节点或成员不存在时返回 nil。
func (n *Node) Get(name string) *Node {
	for _, member := range n.members {
		if member.Name == name {
			return member.Value
		}
	}
	return nil
}

// Parent 返回所属容器；根节点返回 nil。
func (n *Node) Parent() *Node { return n.parent }

// Index 返回在父数组中的下标；非数组元素返回 -1。
func (n *Node) Index() int64 { return n.index }

// Name 返回父对象中的键名；其他节点返回空字符串，空键名需结合 Parent 判断。
func (n *Node) Name() string { return n.name }
