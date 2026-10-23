package jsonpath

import (
	"strconv"
	"strings"

	"github.com/saltfishpr/jsonpath/internal/document"
)

func normalizedPath(n *document.Node) string {
	var ancestors [16]*document.Node
	chain := ancestors[:0]
	for node := n; node != nil && node.Parent != nil; node = node.Parent {
		if len(chain) == cap(chain) {
			next := make([]*document.Node, len(chain), cap(chain)*2)
			copy(next, chain)
			chain = next
		}
		chain = append(chain, node)
	}

	var sb strings.Builder
	sb.WriteByte('$')
	for i := len(chain) - 1; i >= 0; i-- {
		step := chain[i].Step
		sb.WriteByte('[')
		if step.IsIndex() {
			sb.WriteString(strconv.FormatInt(step.Index(), 10))
		} else {
			sb.WriteByte('\'')
			name := step.Name()
			// RFC 9535 第 2.7 节只要求转义这些 ASCII 字节，其余字符原样保留。
			for j := 0; j < len(name); j++ {
				c := name[j]
				switch c {
				case '\'', '\\':
					sb.WriteByte('\\')
					sb.WriteByte(c)
				case '\b':
					sb.WriteString(`\b`)
				case '\f':
					sb.WriteString(`\f`)
				case '\n':
					sb.WriteString(`\n`)
				case '\r':
					sb.WriteString(`\r`)
				case '\t':
					sb.WriteString(`\t`)
				default:
					if c < 0x20 {
						const hex = "0123456789abcdef"
						sb.WriteString(`\u00`)
						sb.WriteByte(hex[c>>4])
						sb.WriteByte(hex[c&0x0f])
					} else {
						sb.WriteByte(c)
					}
				}
			}
			sb.WriteByte('\'')
		}
		sb.WriteByte(']')
	}
	return sb.String()
}
