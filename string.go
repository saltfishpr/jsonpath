package jsonpath

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func space(c byte) bool { return c == ' ' || c == '\t' || c == '\r' || c == '\n' }
func digit(c byte) bool { return c >= '0' && c <= '9' }
func hex(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

// readString checks scalar-value validity before decoding, unlike encoding/json's
// replacement of malformed Unicode. The quote determines the allowed escape.
func readString(s string, pos *int) (string, error) {
	quote := s[*pos]
	*pos++
	var b strings.Builder
	for *pos < len(s) {
		c := s[*pos]
		*pos++
		if c == quote {
			return b.String(), nil
		}
		if c < 0x20 {
			return "", fmt.Errorf("unescaped control character")
		}
		if c == '\\' {
			if *pos == len(s) {
				break
			}
			c = s[*pos]
			*pos++
			switch c {
			case quote, '\\', '/':
				b.WriteByte(c)
			case 'b':
				b.WriteByte('\b')
			case 'f':
				b.WriteByte('\f')
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case 'u':
				readHex := func() (rune, error) {
					var r rune
					for j := 0; j < 4; j++ {
						if *pos >= len(s) || hex(s[*pos]) < 0 {
							return 0, fmt.Errorf("invalid Unicode escape")
						}
						r = r*16 + rune(hex(s[*pos]))
						*pos++
					}
					return r, nil
				}
				r, err := readHex()
				if err != nil {
					return "", err
				}
				if r >= 0xD800 && r <= 0xDBFF {
					if *pos+2 > len(s) || s[*pos:*pos+2] != "\\u" {
						return "", fmt.Errorf("unpaired high surrogate")
					}
					*pos += 2
					low, err := readHex()
					if err != nil {
						return "", err
					}
					if low < 0xDC00 || low > 0xDFFF {
						return "", fmt.Errorf("invalid surrogate pair")
					}
					r = 0x10000 + (r-0xD800)*0x400 + low - 0xDC00
				} else if r >= 0xDC00 && r <= 0xDFFF {
					return "", fmt.Errorf("unpaired low surrogate")
				}
				b.WriteRune(r)
			default:
				return "", fmt.Errorf("invalid escape")
			}
		} else if c < utf8.RuneSelf {
			b.WriteByte(c)
		} else {
			*pos--
			r, n := utf8.DecodeRuneInString(s[*pos:])
			if r == utf8.RuneError && n == 1 {
				return "", fmt.Errorf("invalid UTF-8")
			}
			b.WriteRune(r)
			*pos += n
		}
	}
	return "", fmt.Errorf("unterminated string")
}
