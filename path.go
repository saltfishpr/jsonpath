package jsonpath

import (
	"strconv"
	"strings"
)

func normalizedPath(n *jsonNode) string {
	if n.parent == nil {
		return "$"
	}
	var ancestors []*jsonNode
	size := 1
	for n.parent != nil {
		ancestors = append(ancestors, n)
		if n.parent.kind == arrayKind {
			size += 3 // Brackets and at least one digit.
			for index := n.index; index >= 10; index /= 10 {
				size++
			}
		} else {
			size += 4 + len(n.name)
			for i := 0; i < len(n.name); i++ {
				switch n.name[i] {
				case '\'', '\\', '\b', '\f', '\n', '\r', '\t':
					size++
				default:
					if n.name[i] < 32 {
						size += 5
					}
				}
			}
		}
		n = n.parent
	}
	var b strings.Builder
	b.Grow(size)
	b.WriteByte('$')
	var index [20]byte
	for i := len(ancestors) - 1; i >= 0; i-- {
		n = ancestors[i]
		b.WriteByte('[')
		if n.parent.kind == arrayKind {
			b.Write(strconv.AppendInt(index[:0], int64(n.index), 10))
		} else {
			b.WriteByte('\'')
			writePathName(&b, n.name)
			b.WriteByte('\'')
		}
		b.WriteByte(']')
	}
	return b.String()
}

func writePathName(b *strings.Builder, name string) {
	// Names are already valid UTF-8; only ASCII bytes need escaping.
	start := 0
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c >= 32 && c != '\'' && c != '\\' {
			continue
		}
		b.WriteString(name[start:i])
		switch c {
		case '\'', '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
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
			const digits = "0123456789abcdef"
			b.WriteString(`\u00`)
			b.WriteByte(digits[c>>4])
			b.WriteByte(digits[c&15])
		}
		start = i + 1
	}
	b.WriteString(name[start:])
}
