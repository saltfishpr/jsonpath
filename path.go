package jsonpath

import (
	"strconv"
	"strings"
)

func normalizedPath(n *jsonNode) string {
	var ancestors []*jsonNode
	for n.parent != nil {
		ancestors = append(ancestors, n)
		n = n.parent
	}
	var b strings.Builder
	b.WriteByte('$')
	for i := len(ancestors) - 1; i >= 0; i-- {
		n = ancestors[i]
		b.WriteByte('[')
		if n.parent.kind == arrayKind {
			b.WriteString(strconv.Itoa(n.index))
		} else {
			b.WriteByte('\'')
			for _, r := range n.name {
				switch r {
				case '\'', '\\':
					b.WriteByte('\\')
					b.WriteRune(r)
				case '\b':
					b.WriteString(`\b`)
				case '\f':
					b.WriteString(`\f`)
				case '\n':
					b.WriteString(`\n`)
				case '\r':
					b.WriteString(`\r`)
				case '\t':
					b.WriteString(`\t`)
				default:
					if r < 32 {
						const digits = "0123456789abcdef"
						b.WriteString(`\u00`)
						b.WriteByte(digits[r>>4])
						b.WriteByte(digits[r&15])
					} else {
						b.WriteRune(r)
					}
				}
			}
			b.WriteByte('\'')
		}
		b.WriteByte(']')
	}
	return b.String()
}
