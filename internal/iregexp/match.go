package iregexp

import (
	"math/big"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

func categoryContains(name string, r rune) bool {
	if name == "Cn" || name == "C" {
		assigned := false
		for _, category := range []string{"L", "M", "N", "P", "S", "Z", "Cc", "Cf", "Co"} {
			if unicode.Is(unicode.Categories[category], r) {
				assigned = true
				break
			}
		}
		if !assigned {
			return true
		}
		if name == "Cn" {
			return false
		}
	}
	return unicode.Is(unicode.Categories[name], r)
}

func (c characterClass) contains(r rune) bool {
	matched := false
	for _, part := range c.parts {
		yes := false
		if part.category != "" {
			yes = categoryContains(part.category, r)
			if part.negative {
				yes = !yes
			}
		} else {
			yes = r >= part.lo && r <= part.hi
		}
		if yes {
			matched = true
			break
		}
	}
	if c.negative {
		return !matched
	}
	return matched
}

type memoKey struct {
	node  *node
	start int
}
type matcher struct {
	text []rune
	memo map[memoKey][]int
}

// Match tests the whole string unless search is true, in which case any
// substring may match. It uses position sets and memoization, not backtracking.
func (p *Pattern) Match(s string, search bool) bool {
	if p == nil || p.root == nil {
		return false
	}
	// Invalid UTF-8 must retain the rune matcher's replacement-character semantics.
	if p.isLiteral && utf8.ValidString(s) {
		if search {
			return strings.Contains(s, p.literal)
		}
		return s == p.literal
	}
	m := matcher{text: []rune(s), memo: make(map[memoKey][]int)}
	end := 0
	if search {
		end = len(m.text)
	}
	for start := 0; start <= end; start++ {
		positions := m.eval(p.root, start)
		if search && len(positions) > 0 {
			return true
		}
		for _, pos := range positions {
			if pos == len(m.text) {
				return true
			}
		}
	}
	return false
}

func unique(values []int) []int {
	if len(values) < 2 {
		return values
	}
	sort.Ints(values)
	out := values[:1]
	for _, v := range values[1:] {
		if v != out[len(out)-1] {
			out = append(out, v)
		}
	}
	return out
}

func (m *matcher) apply(n *node, positions []int) []int {
	var next []int
	for _, pos := range positions {
		next = append(next, m.eval(n, pos)...)
	}
	return unique(next)
}

func same(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func bounded(n *big.Int, cap int) int {
	if n == nil || n.Cmp(big.NewInt(int64(cap))) > 0 {
		return cap
	}
	return int(n.Int64())
}

func (m *matcher) eval(n *node, start int) []int {
	key := memoKey{n, start}
	if cached, ok := m.memo[key]; ok {
		return cached
	}
	var out []int
	switch n.kind {
	case 'c':
		if start < len(m.text) && n.class.contains(m.text[start]) {
			out = []int{start + 1}
		}
	case '|':
		for _, child := range n.children {
			out = append(out, m.eval(child, start)...)
		}
		out = unique(out)
	case 's':
		out = []int{start}
		for _, child := range n.children {
			out = m.apply(child, out)
			if len(out) == 0 {
				break
			}
		}
	case 'r':
		child := n.children[0]
		cap := len(m.text) - start + 1
		if !child.nullable && n.min.Cmp(big.NewInt(int64(cap))) > 0 {
			break
		}
		lower, upper := bounded(n.min, cap), bounded(n.max, cap)
		positions := []int{start}
		for count := 0; count <= upper; count++ {
			if count >= lower {
				out = append(out, positions...)
			}
			if count == upper || len(positions) == 0 {
				break
			}
			next := m.apply(child, positions)
			// A nullable repeated atom produces monotonically growing position sets.
			// Once stable, padding with empty matches satisfies any remaining count.
			if child.nullable && same(next, positions) {
				out = append(out, positions...)
				break
			}
			positions = next
		}
		out = unique(out)
	}
	m.memo[key] = out
	return out
}
