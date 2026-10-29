package document

import (
	"math"
	"math/big"
	"strconv"
	"strings"
)

// Number holds an exact decimal value, including exponents outside int64 range.
// Its zero value is numeric zero. Copies may be compared concurrently.
type Number struct {
	negative bool
	// Nonzero values use digits without leading or trailing zeros and a decimal
	// magnitude (exponent plus digit count). Empty digits represent zero.
	digits       string
	magnitude    int64
	bigMagnitude *big.Int // Non-nil only when the magnitude is outside int64.
}

// ParseNumber converts valid JSON number text without rounding. The caller must
// validate syntax first; empty input panics and other invalid input has no
// defined result. All spellings of zero, including negative zero, compare equally.
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
	adjustment := int64(len(s)) - int64(fraction)
	if expText == "" {
		n.magnitude = adjustment
		return n
	}
	exponent, err := strconv.ParseInt(expText, 10, 64)
	if err == nil {
		n.magnitude = exponent
	} else {
		n.bigMagnitude = new(big.Int)
		n.bigMagnitude.SetString(expText, 10)
	}
	if adjustment != 0 {
		if n.bigMagnitude == nil {
			if (adjustment > 0 && n.magnitude > math.MaxInt64-adjustment) ||
				(adjustment < 0 && n.magnitude < math.MinInt64-adjustment) {
				n.bigMagnitude = big.NewInt(n.magnitude)
			} else {
				n.magnitude += adjustment
			}
		}
		if n.bigMagnitude != nil {
			n.bigMagnitude.Add(n.bigMagnitude, big.NewInt(adjustment))
		}
	}
	if n.bigMagnitude != nil && n.bigMagnitude.IsInt64() {
		n.magnitude = n.bigMagnitude.Int64()
		n.bigMagnitude = nil
	}
	return n
}

// Cmp returns -1, 0, or 1 when n is less than, equal to, or greater than other.
// Comparison is exact and does not mutate either operand.
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
	if n.bigMagnitude == nil && other.bigMagnitude == nil {
		if n.magnitude < other.magnitude {
			return -1
		}
		if n.magnitude > other.magnitude {
			return 1
		}
	} else if n.bigMagnitude == nil {
		return -other.bigMagnitude.Sign()
	} else if other.bigMagnitude == nil {
		return n.bigMagnitude.Sign()
	} else if c := n.bigMagnitude.Cmp(other.bigMagnitude); c != 0 {
		return c
	}
	// Equal magnitudes align the first digits. With no trailing zeros, a longer
	// coefficient always exceeds its prefix even when that prefix is zero-padded.
	return strings.Compare(n.digits, other.digits)
}
