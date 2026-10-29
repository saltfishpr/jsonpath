package document

import (
	"fmt"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// DefaultMaxDepth limits simultaneously nested arrays and objects when no
// positive depth limit is supplied. Scalars do not add a nesting level.
const DefaultMaxDepth = 10000

// SyntaxError reports invalid JSON or a violation of the parser's Unicode,
// member uniqueness, or nesting requirements.
type SyntaxError struct {
	// Offset is the zero-based byte position where the error was detected.
	// A position equal to the input length denotes EOF.
	Offset int
	// Message describes the failure without a location prefix.
	Message string
}

// Error returns the diagnostic with its byte offset in the input.
func (e *SyntaxError) Error() string {
	return fmt.Sprintf("document: json syntax error at offset %d: %s", e.Offset, e.Message)
}

// Parse reads one JSON value with optional surrounding JSON whitespace, using
// [DefaultMaxDepth]. It rejects a BOM, invalid UTF-8, unpaired surrogate escapes,
// and duplicate decoded object names. Errors are *[SyntaxError] with a nil root.
// Nodes' [Node.Raw] slices alias data, which must remain unchanged while the tree is
// in use; decoded strings and numeric values are stored independently.
func Parse(data []byte) (*Node, error) {
	return ParseWithDepth(data, DefaultMaxDepth)
}

// ParseString parses s with the validation and depth limit of [Parse].
// Returned [Node.Raw] slices must be treated as immutable.
func ParseString(s string) (*Node, error) {
	return Parse([]byte(s))
}

// ParseWithDepth behaves like [Parse] with a caller-specified container nesting
// limit. A root array or object has depth one; maxDepth <= 0 selects
// [DefaultMaxDepth]. Exceeding the limit returns a *[SyntaxError] and a nil root.
func ParseWithDepth(data []byte, maxDepth int) (*Node, error) {
	if maxDepth <= 0 {
		maxDepth = DefaultMaxDepth
	}
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return nil, &SyntaxError{Offset: 0, Message: "unexpected byte order mark (U+FEFF)"}
	}
	p := &parser{src: data, maxDepth: maxDepth, nodeBlockSize: 1}
	p.skipWS()
	n, err := p.parseValue(0)
	if err != nil {
		return nil, err
	}
	p.skipWS()
	if p.pos != len(p.src) {
		return nil, p.errf("unexpected trailing data")
	}
	return n, nil
}

type parser struct {
	src      []byte
	pos      int
	maxDepth int

	nodes         []Node
	nodeBlockSize int

	scratch []byte
	text    strings.Builder
}

func (p *parser) newNode(kind Kind) *Node {
	if len(p.nodes) == 0 {
		// With the 64-bit Node layout, 23 nodes plus allocation metadata fit
		// the allocator's 4 KiB size class.
		const maxBlockSize = 23
		size := p.nodeBlockSize
		p.nodeBlockSize *= 2
		if p.nodeBlockSize > maxBlockSize {
			p.nodeBlockSize = maxBlockSize
		}
		if remaining := len(p.src) - p.pos + 1; size > remaining {
			size = remaining
		}
		// Previously issued pointers must keep their addresses and contents.
		p.nodes = make([]Node, size)
	}
	n := &p.nodes[0]
	p.nodes = p.nodes[1:]
	n.kind = kind
	n.index = -1
	return n
}

func (p *parser) storeString(raw []byte) string {
	// Tiny strings and values ending at EOF cannot benefit from a shared block.
	if len(raw) < 2 || p.pos == len(p.src) {
		return string(raw)
	}
	if len(raw) > p.text.Cap()-p.text.Len() {
		const maxBlockSize = 4 << 10
		capacity := p.text.Cap()
		if capacity > maxBlockSize/2 {
			capacity = maxBlockSize
		} else {
			capacity *= 2
		}
		if capacity < len(raw) {
			capacity = len(raw)
		}
		if remaining := len(p.src) - p.pos + len(raw); capacity > remaining {
			capacity = remaining
		}
		// Existing strings retain the old buffer. Starting a new block avoids
		// copying bytes that future strings will never use.
		p.text.Reset()
		p.text.Grow(capacity)
	}
	start := p.text.Len()
	p.text.Write(raw)
	return p.text.String()[start:]
}

func (p *parser) errf(format string, args ...interface{}) *SyntaxError {
	return &SyntaxError{Offset: p.pos, Message: fmt.Sprintf(format, args...)}
}

func (p *parser) skipWS() {
	for p.pos < len(p.src) {
		switch p.src[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}

// depth counts enclosing containers, so entering an array or object adds one.
func (p *parser) parseValue(depth int) (node *Node, err error) {
	start := p.pos
	defer func() {
		if err == nil {
			node.raw = p.src[start:p.pos]
		}
	}()
	if p.pos >= len(p.src) {
		return nil, p.errf("unexpected end of input")
	}
	c := p.src[p.pos]
	switch {
	case c == '{':
		return p.parseObject(depth + 1)
	case c == '[':
		return p.parseArray(depth + 1)
	case c == '"':
		s, err := p.parseString()
		if err != nil {
			return nil, err
		}
		n := p.newNode(KindString)
		n.str = s
		return n, nil
	case c == 't':
		return p.parseLiteral("true", KindBool, true)
	case c == 'f':
		return p.parseLiteral("false", KindBool, false)
	case c == 'n':
		return p.parseLiteral("null", KindNull, false)
	case c == '-' || (c >= '0' && c <= '9'):
		return p.parseNumber()
	}
	return nil, p.errf("unexpected character %q", p.src[p.pos])
}

// Equal scalar values still need distinct nodes for parent links and [Node.Raw] slices.
func (p *parser) parseLiteral(lit string, kind Kind, b bool) (*Node, error) {
	if p.pos+len(lit) > len(p.src) || string(p.src[p.pos:p.pos+len(lit)]) != lit {
		return nil, p.errf("invalid literal, expected %q", lit)
	}
	p.pos += len(lit)
	n := p.newNode(kind)
	n.boolean = b
	return n, nil
}

func (p *parser) parseObject(depth int) (*Node, error) {
	if depth > p.maxDepth {
		return nil, p.errf("exceeded max nesting depth %d", p.maxDepth)
	}
	p.pos++
	p.skipWS()
	if p.pos < len(p.src) && p.src[p.pos] == '}' {
		p.pos++
		return p.newNode(KindObject), nil
	}
	var members []Member
	memberIndex := make(map[string]*Node)
	for {
		p.skipWS()
		if p.pos >= len(p.src) || p.src[p.pos] != '"' {
			return nil, p.errf("expected string key in object")
		}
		keyPos := p.pos
		key, err := p.parseString()
		if err != nil {
			return nil, err
		}
		// Compare decoded names so escaped spellings cannot create ambiguous members.
		if _, dup := memberIndex[key]; dup {
			return nil, &SyntaxError{Offset: keyPos, Message: fmt.Sprintf("duplicate object member %q", key)}
		}
		p.skipWS()
		if p.pos >= len(p.src) || p.src[p.pos] != ':' {
			return nil, p.errf("expected ':' after object key")
		}
		p.pos++
		p.skipWS()
		val, err := p.parseValue(depth)
		if err != nil {
			return nil, err
		}
		memberIndex[key] = val
		members = append(members, Member{Name: key, Value: val})
		p.skipWS()
		if p.pos >= len(p.src) {
			return nil, p.errf("unexpected end of input in object")
		}
		switch p.src[p.pos] {
		case ',':
			p.pos++
			continue
		case '}':
			p.pos++
			obj := p.newNode(KindObject)
			obj.members = members
			obj.memberIndex = memberIndex
			for i := range members {
				members[i].Value.parent = obj
				members[i].Value.name = members[i].Name
			}
			return obj, nil
		default:
			return nil, p.errf("expected ',' or '}' in object")
		}
	}
}

func (p *parser) parseArray(depth int) (*Node, error) {
	if depth > p.maxDepth {
		return nil, p.errf("exceeded max nesting depth %d", p.maxDepth)
	}
	p.pos++
	p.skipWS()
	if p.pos < len(p.src) && p.src[p.pos] == ']' {
		p.pos++
		return p.newNode(KindArray), nil
	}
	var arr []*Node
	for {
		p.skipWS()
		val, err := p.parseValue(depth)
		if err != nil {
			return nil, err
		}
		arr = append(arr, val)
		p.skipWS()
		if p.pos >= len(p.src) {
			return nil, p.errf("unexpected end of input in array")
		}
		switch p.src[p.pos] {
		case ',':
			p.pos++
			continue
		case ']':
			p.pos++
			arrNode := p.newNode(KindArray)
			arrNode.arr = arr
			for i, el := range arr {
				el.parent = arrNode
				el.index = int64(i)
			}
			return arrNode, nil
		default:
			return nil, p.errf("expected ',' or ']' in array")
		}
	}
}

// Scan before allocating decoded storage so strings without escapes need only
// one copy. Both paths must produce valid Unicode scalar sequences.
func (p *parser) parseString() (string, error) {
	if p.pos >= len(p.src) || p.src[p.pos] != '"' {
		return "", p.errf("expected '\"'")
	}
	start := p.pos + 1
	i := start
	hasEscape := false
	for i < len(p.src) {
		c := p.src[i]
		if c == '"' {
			raw := p.src[start:i]
			p.pos = i + 1
			if !hasEscape {
				return p.storeString(raw), nil
			}
			return p.unescape(raw, start)
		}
		if c == '\\' {
			hasEscape = true
			// An escaped quote cannot terminate the string. Escape validity is
			// checked by unescape after the closing quote is found.
			if i+1 >= len(p.src) {
				p.pos = i
				return "", p.errf("unterminated escape at end of input")
			}
			i += 2
			continue
		}
		if c < 0x20 {
			p.pos = i
			return "", p.errf("control character in string literal")
		}
		if c < 0x80 {
			i++
			continue
		}
		r, size := utf8.DecodeRune(p.src[i:])
		if r == utf8.RuneError && size <= 1 {
			p.pos = i
			return "", p.errf("invalid UTF-8 in string literal")
		}
		i += size
	}
	p.pos = i
	return "", p.errf("unterminated string literal")
}

// start maps offsets in the unquoted raw fragment back to the original input.
func (p *parser) unescape(raw []byte, start int) (string, error) {
	out := p.scratch[:0]
	if cap(out) < len(raw) {
		out = make([]byte, 0, len(raw))
	}
	i := 0
	for i < len(raw) {
		c := raw[i]
		if c == '\\' {
			i++
			if i >= len(raw) {
				p.pos = start + i
				return "", p.errf("truncated escape sequence")
			}
			switch raw[i] {
			case '"':
				out = append(out, '"')
				i++
			case '\\':
				out = append(out, '\\')
				i++
			case '/':
				out = append(out, '/')
				i++
			case 'b':
				out = append(out, '\b')
				i++
			case 'f':
				out = append(out, '\f')
				i++
			case 'n':
				out = append(out, '\n')
				i++
			case 'r':
				out = append(out, '\r')
				i++
			case 't':
				out = append(out, '\t')
				i++
			case 'u':
				if i+5 > len(raw) {
					p.pos = start + i
					return "", p.errf("short \\u escape")
				}
				r, ok := hex4(raw[i+1 : i+5])
				if !ok {
					p.pos = start + i
					return "", p.errf("invalid \\u escape")
				}
				i += 5
				if utf16.IsSurrogate(r) {
					// Surrogates are valid only as a high/low pair; never replace
					// malformed input with a replacement character.
					if r >= 0xDC00 ||
						i+6 > len(raw) || raw[i] != '\\' || raw[i+1] != 'u' {
						p.pos = start + i
						return "", p.errf("invalid surrogate pair")
					}
					r2, ok := hex4(raw[i+2 : i+6])
					if !ok {
						p.pos = start + i
						return "", p.errf("invalid \\u escape in surrogate pair")
					}
					dec := utf16.DecodeRune(r, r2)
					if dec == utf8.RuneError {
						p.pos = start + i
						return "", p.errf("invalid surrogate pair")
					}
					r = dec
					i += 6
				}
				var buf [utf8.UTFMax]byte
				n := utf8.EncodeRune(buf[:], r)
				out = append(out, buf[:n]...)
			default:
				p.pos = start + i
				return "", p.errf("invalid escape \\%c", raw[i])
			}
			continue
		}
		if c < 0x20 {
			p.pos = start + i
			return "", p.errf("control character in string literal")
		}
		if c < 0x80 {
			out = append(out, c)
			i++
			continue
		}
		r, size := utf8.DecodeRune(raw[i:])
		if r == utf8.RuneError && size <= 1 {
			p.pos = start + i
			return "", p.errf("invalid UTF-8 in string literal")
		}
		out = append(out, raw[i:i+size]...)
		i += size
	}
	// storeString copies the decoded bytes before the next string reuses scratch.
	p.scratch = out[:0]
	return p.storeString(out), nil
}

func hex4(b []byte) (rune, bool) {
	var r rune
	for _, c := range b {
		r <<= 4
		switch {
		case c >= '0' && c <= '9':
			r |= rune(c - '0')
		case c >= 'a' && c <= 'f':
			r |= rune(c-'a') + 10
		case c >= 'A' && c <= 'F':
			r |= rune(c-'A') + 10
		default:
			return 0, false
		}
	}
	return r, true
}

// Validate the numeric prefix before calling [ParseNumber], which assumes valid
// syntax. The enclosing container or EOF check rejects any trailing characters.
func (p *parser) parseNumber() (*Node, error) {
	start := p.pos
	i := p.pos
	if p.src[i] == '-' {
		i++
	}
	if i >= len(p.src) {
		p.pos = i
		return nil, p.errf("invalid number")
	}
	switch {
	case p.src[i] == '0':
		i++
	case p.src[i] >= '1' && p.src[i] <= '9':
		i++
		for i < len(p.src) && p.src[i] >= '0' && p.src[i] <= '9' {
			i++
		}
	default:
		p.pos = i
		return nil, p.errf("invalid number")
	}
	if i < len(p.src) && p.src[i] == '.' {
		i++
		if i >= len(p.src) || p.src[i] < '0' || p.src[i] > '9' {
			p.pos = i
			return nil, p.errf("number missing fraction digits")
		}
		for i < len(p.src) && p.src[i] >= '0' && p.src[i] <= '9' {
			i++
		}
	}
	if i < len(p.src) && (p.src[i] == 'e' || p.src[i] == 'E') {
		i++
		if i < len(p.src) && (p.src[i] == '+' || p.src[i] == '-') {
			i++
		}
		if i >= len(p.src) || p.src[i] < '0' || p.src[i] > '9' {
			p.pos = i
			return nil, p.errf("number missing exponent digits")
		}
		for i < len(p.src) && p.src[i] >= '0' && p.src[i] <= '9' {
			i++
		}
	}
	p.pos = i
	n := p.newNode(KindNumber)
	n.number = ParseNumber(p.storeString(p.src[start:i]))
	return n, nil
}
