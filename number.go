package jsonpath

import (
	"math/big"
	"strings"
)

type decimal struct {
	negative bool
	digits   string
	exponent big.Int
}

func parseDecimal(s string) decimal {
	var d decimal
	if s[0] == '-' {
		d.negative = true
		s = s[1:]
	}
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		d.exponent.SetString(s[i+1:], 10)
		s = s[:i]
	}
	if i := strings.IndexByte(s, '.'); i >= 0 {
		d.exponent.Sub(&d.exponent, big.NewInt(int64(len(s)-i-1)))
		s = s[:i] + s[i+1:]
	}
	s = strings.TrimLeft(s, "0")
	if s == "" {
		d.negative = false
		d.digits = "0"
		d.exponent.SetInt64(0)
		return d
	}
	end := len(s)
	s = strings.TrimRight(s, "0")
	d.exponent.Add(&d.exponent, big.NewInt(int64(end-len(s))))
	d.digits = s
	return d
}
func compareNumber(a, b decimal) int {
	if a.digits == "0" && b.digits == "0" {
		return 0
	}
	if a.negative != b.negative {
		if a.negative {
			return -1
		}
		return 1
	}
	sign := 1
	if a.negative {
		sign = -1
	}
	if a.digits == "0" {
		return -sign
	}
	if b.digits == "0" {
		return sign
	}
	var x, y big.Int
	x.Add(&a.exponent, big.NewInt(int64(len(a.digits))))
	y.Add(&b.exponent, big.NewInt(int64(len(b.digits))))
	if c := x.Cmp(&y); c != 0 {
		return sign * c
	}
	n := len(a.digits)
	if len(b.digits) > n {
		n = len(b.digits)
	}
	for i := 0; i < n; i++ {
		u, v := byte('0'), byte('0')
		if i < len(a.digits) {
			u = a.digits[i]
		}
		if i < len(b.digits) {
			v = b.digits[i]
		}
		if u < v {
			return -sign
		}
		if u > v {
			return sign
		}
	}
	return 0
}

func scanNumber(s string, pos *int) bool {
	start := *pos
	if *pos < len(s) && s[*pos] == '-' {
		*pos++
	}
	if *pos >= len(s) {
		return false
	}
	if s[*pos] == '0' {
		*pos++
	} else {
		if s[*pos] < '1' || s[*pos] > '9' {
			return false
		}
		for *pos < len(s) && digit(s[*pos]) {
			*pos++
		}
	}
	if *pos < len(s) && s[*pos] == '.' {
		*pos++
		p := *pos
		for *pos < len(s) && digit(s[*pos]) {
			*pos++
		}
		if p == *pos {
			return false
		}
	}
	if *pos < len(s) && (s[*pos] == 'e' || s[*pos] == 'E') {
		*pos++
		if *pos < len(s) && (s[*pos] == '+' || s[*pos] == '-') {
			*pos++
		}
		p := *pos
		for *pos < len(s) && digit(s[*pos]) {
			*pos++
		}
		if p == *pos {
			return false
		}
	}
	return *pos > start
}
