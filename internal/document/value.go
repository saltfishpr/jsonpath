// Package document 提供 JSON 值、节点树及保留原始文本的解析器。
package document

import (
	"math"
	"math/big"
	"strings"
)

// Kind 表示 JSON 值的类型。
type Kind uint8

const (
	// KindNull 对应 JSON null。
	KindNull Kind = iota
	// KindBool 对应 JSON true / false。
	KindBool
	// KindNumber 对应 JSON 数值。
	KindNumber
	// KindString 对应 JSON 字符串。
	KindString
	// KindArray 对应 JSON 数组。
	KindArray
	// KindObject 对应 JSON 对象。
	KindObject
)

// String 返回 Kind 的类型名；未知值返回 "?"。
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
	return "?"
}

// Value 是 JSON 值节点的接口，由 Null、Bool、Number、String、Array 和 Object 实现。
type Value interface {
	// Kind 返回当前节点的 JSON 值类型。
	Kind() Kind
	// Raw 返回节点在原始输入中的字节切片。直接构造的值可能返回 nil。
	Raw() []byte

	isValue()
}

// Null 表示 JSON null。
type Null struct {
	raw []byte
}

func (Null) Kind() Kind    { return KindNull }
func (n Null) Raw() []byte { return n.raw }
func (Null) isValue()      {}

// Bool 表示 JSON 布尔值。
type Bool struct {
	V   bool
	raw []byte
}

func (Bool) Kind() Kind    { return KindBool }
func (b Bool) Raw() []byte { return b.raw }
func (Bool) isValue()      {}

// Number 表示任意精度的十进制数字，并保留原始 JSON 文本。
// 零值表示零；解析后的 Number 不可变，可并发读取。
type Number struct {
	raw      []byte
	digits   string
	exponent big.Int
	negative bool
}

func (Number) Kind() Kind    { return KindNumber }
func (n Number) Raw() []byte { return n.raw }
func (Number) isValue()      {}

// ParseNumber 只接收已经由 JSON decoder 或 JSONPath lexer 校验的数字文本。
// exponent 在构造后只读，避免 Number 值复制后修改共享的 big.Int 存储。
func ParseNumber(raw []byte) Number {
	n := Number{raw: raw}
	s := string(raw)
	if s[0] == '-' {
		n.negative = true
		s = s[1:]
	}
	expText := ""
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		expText = s[i+1:]
		s = s[:i]
	}
	fraction := 0
	if i := strings.IndexByte(s, '.'); i >= 0 {
		fraction = len(s) - i - 1
		s = s[:i] + s[i+1:]
	}
	s = strings.TrimLeft(s, "0")
	if s == "" {
		n.negative = false
		return n
	}
	n.digits = strings.TrimRight(s, "0")
	adjustment := int64(len(s)-len(n.digits)) - int64(fraction)
	if expText != "" {
		n.exponent.SetString(expText, 10)
	}
	if adjustment != 0 {
		n.exponent.Add(&n.exponent, big.NewInt(adjustment))
	}
	return n
}

// Cmp 精确比较两个十进制数，小于、等于或大于 other 时分别返回 -1、0 或 1。
func (n Number) Cmp(other Number) int {
	if n.negative != other.negative {
		if n.negative {
			return -1
		}
		return 1
	}
	c := n.compareAbs(other)
	if n.negative {
		return -c
	}
	return c
}

func (n Number) compareAbs(other Number) int {
	if n.digits == "" {
		if other.digits == "" {
			return 0
		}
		return -1
	}
	if other.digits == "" {
		return 1
	}
	alen, blen := int64(len(n.digits)), int64(len(other.digits))
	a, b := n.exponent.Int64(), other.exponent.Int64()
	// 常见指数直接比较数量级，避免 big.Int 临时分配；加数均为正数。
	if n.exponent.IsInt64() && other.exponent.IsInt64() &&
		a <= math.MaxInt64-alen && b <= math.MaxInt64-blen {
		a, b = a+alen, b+blen
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
	} else {
		var x, y big.Int
		x.Add(&n.exponent, big.NewInt(alen))
		y.Add(&other.exponent, big.NewInt(blen))
		if c := x.Cmp(&y); c != 0 {
			return c
		}
	}
	// 数量级相同时，缺少的末尾数字视为零，无需展开指数或分配补零字符串。
	length := len(n.digits)
	if len(other.digits) > length {
		length = len(other.digits)
	}
	for i := 0; i < length; i++ {
		a, b := byte('0'), byte('0')
		if i < len(n.digits) {
			a = n.digits[i]
		}
		if i < len(other.digits) {
			b = other.digits[i]
		}
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
	}
	return 0
}

// String 表示 JSON 字符串，V 为解码后的 UTF-8 文本，raw 为含双引号的原文。
type String struct {
	V   string
	raw []byte
}

func (String) Kind() Kind    { return KindString }
func (s String) Raw() []byte { return s.raw }
func (String) isValue()      {}

// Array 表示 JSON 数组，按输入顺序保留元素。
type Array struct {
	Items []Value
	raw   []byte
}

func (Array) Kind() Kind    { return KindArray }
func (a Array) Raw() []byte { return a.raw }
func (Array) isValue()      {}

// Len 返回数组元素个数。
func (a Array) Len() int { return len(a.Items) }

// Index 返回第 i 个元素；越界时返回 (nil, false)。
func (a Array) Index(i int) (Value, bool) {
	if i < 0 || i >= len(a.Items) {
		return nil, false
	}
	return a.Items[i], true
}

// Object 表示 JSON 对象，按输入顺序保留全部键值对，包括重复键。
type Object struct {
	Keys   []string
	Values []Value
	raw    []byte

	// index 为成员名到 Keys/Values 中最后一次出现位置的映射，服务 Get。
	index map[string]int
}

func (Object) Kind() Kind    { return KindObject }
func (o Object) Raw() []byte { return o.raw }
func (Object) isValue()      {}

// Len 返回键值对个数，包括重复键。
func (o Object) Len() int { return len(o.Keys) }

// Get 按成员名取值，重复键返回最后一次出现的值。
func (o Object) Get(name string) (Value, bool) {
	i, ok := o.index[name]
	if !ok {
		return nil, false
	}
	return o.Values[i], true
}

// MemberIndex 返回成员最后一次出现的位置；不存在时返回 false。
func (o Object) MemberIndex(name string) (int, bool) { i, ok := o.index[name]; return i, ok }

// MemberCount 返回不重复的成员名个数。
func (o Object) MemberCount() int { return len(o.index) }
