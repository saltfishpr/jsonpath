package document

import (
	"math"
	"math/big"
	"strconv"
	"strings"
)

// Number 表示精确的 JSON 数值，避免浮点数的溢出与精度损失。
type Number struct {
	digits      string
	exponent    int64
	bigExponent *big.Int
	negative    bool
}

// ParseNumber 解析已校验的 JSON 数字文本。
func ParseNumber(s string) Number {
	var n Number
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
		exponent, err := strconv.ParseInt(expText, 10, 64)
		if err == nil {
			n.exponent = exponent
		} else {
			n.bigExponent = new(big.Int)
			n.bigExponent.SetString(expText, 10)
		}
	}
	if adjustment != 0 {
		if n.bigExponent == nil {
			if (adjustment > 0 && n.exponent > math.MaxInt64-adjustment) ||
				(adjustment < 0 && n.exponent < math.MinInt64-adjustment) {
				n.bigExponent = big.NewInt(n.exponent)
			} else {
				n.exponent += adjustment
			}
		}
		if n.bigExponent != nil {
			n.bigExponent.Add(n.bigExponent, big.NewInt(adjustment))
		}
	}
	if n.bigExponent != nil && n.bigExponent.IsInt64() {
		n.exponent = n.bigExponent.Int64()
		n.bigExponent = nil
	}
	return n
}

// Cmp 比较两个数值，分别在小于、等于或大于 other 时返回 -1、0 或 1。
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
	a, b := n.exponent, other.exponent
	// 常见指数直接比较数量级，避免 big.Int 临时分配；加数均为正数。
	if n.bigExponent == nil && other.bigExponent == nil &&
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
		if n.bigExponent != nil {
			x.Set(n.bigExponent)
		} else {
			x.SetInt64(a)
		}
		if other.bigExponent != nil {
			y.Set(other.bigExponent)
		} else {
			y.SetInt64(b)
		}
		x.Add(&x, big.NewInt(alen))
		y.Add(&y, big.NewInt(blen))
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
