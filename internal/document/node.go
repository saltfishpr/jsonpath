package document

import (
	"fmt"
	"unicode/utf16"
	"unicode/utf8"
)

// Error 表示 JSON 解析失败；Offset 是相对于输入起点的字节偏移。
type Error struct {
	Offset  int
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("jsonpath: parse json at offset %d: %s", e.Offset, e.Message)
}

// Node 保存 JSON 值、父子关系和路径段。
type Node struct {
	Value Value
	Step  NormalStep

	Parent   *Node
	Children []*Node
}

// Parse 解析 JSON 并返回根节点；节点与 data 共享底层字节，调用方不应修改它们。
func Parse(data []byte) (*Node, error) {
	d := &jsonDecoder{src: data}
	d.skipWS()
	root, err := d.parseNode(nil, NormalStep{})
	if err != nil {
		return nil, err
	}
	d.skipWS()
	if d.off != len(data) {
		return nil, d.errorf("unexpected trailing data")
	}
	return root, nil
}

type jsonDecoder struct {
	src         []byte
	off         int
	unusedNodes []Node
	blockSize   int
}

func (d *jsonDecoder) newNode(parent *Node, step NormalStep) *Node {
	if len(d.unusedNodes) == 0 {
		// 已发布的节点地址必须稳定；分配新块而不扩容或复用已有块。
		if d.blockSize == 0 {
			d.blockSize = 1
		} else if d.blockSize < 64 {
			d.blockSize *= 2
		}
		d.unusedNodes = make([]Node, d.blockSize)
	}
	node := &d.unusedNodes[0]
	d.unusedNodes = d.unusedNodes[1:]
	node.Parent = parent
	node.Step = step
	return node
}

func (d *jsonDecoder) errorf(format string, args ...interface{}) error {
	return &Error{Offset: d.off, Message: fmt.Sprintf(format, args...)}
}

func (d *jsonDecoder) skipWS() {
	for d.off < len(d.src) {
		switch d.src[d.off] {
		case ' ', '\t', '\n', '\r':
			d.off++
		default:
			return
		}
	}
}

func (d *jsonDecoder) parseNode(parent *Node, step NormalStep) (*Node, error) {
	if d.off >= len(d.src) {
		return nil, d.errorf("unexpected end of input")
	}
	node := d.newNode(parent, step)
	var err error
	switch c := d.src[d.off]; {
	case c == '{':
		node.Value, err = d.parseObject(node)
	case c == '[':
		node.Value, err = d.parseArray(node)
	case c == '"':
		node.Value, err = d.parseString()
	case c == 't' || c == 'f':
		node.Value, err = d.parseBool()
	case c == 'n':
		node.Value, err = d.parseNull()
	case c == '-' || (c >= '0' && c <= '9'):
		node.Value, err = d.parseNumber()
	default:
		return nil, d.errorf("unexpected character %q", c)
	}
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (d *jsonDecoder) parseObject(node *Node) (Value, error) {
	start := d.off
	d.off++
	o := Object{index: map[string]int{}}
	d.skipWS()
	if d.off < len(d.src) && d.src[d.off] == '}' {
		d.off++
		o.raw = d.src[start:d.off]
		return o, nil
	}
	for {
		d.skipWS()
		if d.off >= len(d.src) || d.src[d.off] != '"' {
			return nil, d.errorf("expected string key")
		}
		k, err := d.parseString()
		if err != nil {
			return nil, err
		}
		key := k.(String).V
		d.skipWS()
		if d.off >= len(d.src) || d.src[d.off] != ':' {
			return nil, d.errorf("expected ':' after key")
		}
		d.off++
		d.skipWS()
		child, err := d.parseNode(node, NameStep(key))
		if err != nil {
			return nil, err
		}
		o.Keys = append(o.Keys, key)
		o.Values = append(o.Values, child.Value)
		o.index[key] = len(o.Keys) - 1
		node.Children = append(node.Children, child)
		d.skipWS()
		if d.off >= len(d.src) {
			return nil, d.errorf("unterminated object")
		}
		switch d.src[d.off] {
		case ',':
			d.off++
			continue
		case '}':
			d.off++
			o.raw = d.src[start:d.off]
			return o, nil
		}
		return nil, d.errorf("expected ',' or '}' in object")
	}
}

func (d *jsonDecoder) parseArray(node *Node) (Value, error) {
	start := d.off
	d.off++
	a := Array{}
	d.skipWS()
	if d.off < len(d.src) && d.src[d.off] == ']' {
		d.off++
		a.raw = d.src[start:d.off]
		return a, nil
	}
	for {
		d.skipWS()
		child, err := d.parseNode(node, IndexStep(int64(len(a.Items))))
		if err != nil {
			return nil, err
		}
		a.Items = append(a.Items, child.Value)
		node.Children = append(node.Children, child)
		d.skipWS()
		if d.off >= len(d.src) {
			return nil, d.errorf("unterminated array")
		}
		switch d.src[d.off] {
		case ',':
			d.off++
			continue
		case ']':
			d.off++
			a.raw = d.src[start:d.off]
			return a, nil
		}
		return nil, d.errorf("expected ',' or ']' in array")
	}
}

// parseString 读取一个 JSON 字符串字面量；调用方需保证当前位置是起始的 '"'。
func (d *jsonDecoder) parseString() (Value, error) {
	start := d.off
	d.off++
	// 不含转义时，直接把原文切片作为解码结果，省去逐字符复制。
	plain := true
	runStart := d.off
	for d.off < len(d.src) {
		b := d.src[d.off]
		if b == '"' {
			if plain {
				raw := d.src[start : d.off+1]
				val := string(d.src[runStart:d.off])
				d.off++
				return String{V: val, raw: raw}, nil
			}
			break
		}
		if b == '\\' {
			plain = false
			break
		}
		if b < 0x20 {
			return nil, d.errorf("unescaped control character U+%04X in string", b)
		}
		d.off++
	}
	if plain {
		return nil, d.errorf("unterminated string")
	}
	var buf []byte
	buf = append(buf, d.src[runStart:d.off]...)
	for d.off < len(d.src) {
		b := d.src[d.off]
		if b == '"' {
			raw := d.src[start : d.off+1]
			d.off++
			return String{V: string(buf), raw: raw}, nil
		}
		if b == '\\' {
			d.off++
			if d.off >= len(d.src) {
				return nil, d.errorf("unterminated escape")
			}
			esc := d.src[d.off]
			d.off++
			switch esc {
			case '"':
				buf = append(buf, '"')
			case '\\':
				buf = append(buf, '\\')
			case '/':
				buf = append(buf, '/')
			case 'b':
				buf = append(buf, 0x08)
			case 'f':
				buf = append(buf, 0x0C)
			case 'n':
				buf = append(buf, '\n')
			case 'r':
				buf = append(buf, '\r')
			case 't':
				buf = append(buf, '\t')
			case 'u':
				r, err := d.readUnicodeEscape()
				if err != nil {
					return nil, err
				}
				buf = utf8.AppendRune(buf, r)
			default:
				return nil, d.errorf("invalid escape \\%c", esc)
			}
			continue
		}
		if b < 0x20 {
			return nil, d.errorf("unescaped control character U+%04X in string", b)
		}
		buf = append(buf, b)
		d.off++
	}
	return nil, d.errorf("unterminated string")
}

func (d *jsonDecoder) readUnicodeEscape() (rune, error) {
	hi, err := d.readHex4()
	if err != nil {
		return 0, err
	}
	if utf16.IsSurrogate(hi) {
		// 高位代理必须紧接着低位代理的 \u 转义。
		if d.off+6 > len(d.src) || d.src[d.off] != '\\' || d.src[d.off+1] != 'u' {
			return 0, d.errorf("unpaired surrogate in \\u escape")
		}
		d.off += 2
		lo, err := d.readHex4()
		if err != nil {
			return 0, err
		}
		r := utf16.DecodeRune(hi, lo)
		if r == utf8.RuneError {
			return 0, d.errorf("invalid surrogate pair")
		}
		return r, nil
	}
	return hi, nil
}

func (d *jsonDecoder) readHex4() (rune, error) {
	if d.off+4 > len(d.src) {
		return 0, d.errorf("truncated \\u escape")
	}
	var v rune
	for i := 0; i < 4; i++ {
		c := d.src[d.off+i]
		var dgt rune
		switch {
		case c >= '0' && c <= '9':
			dgt = rune(c - '0')
		case c >= 'a' && c <= 'f':
			dgt = rune(c-'a') + 10
		case c >= 'A' && c <= 'F':
			dgt = rune(c-'A') + 10
		default:
			return 0, d.errorf("invalid hex digit %q in \\u escape", c)
		}
		v = v<<4 | dgt
	}
	d.off += 4
	return v, nil
}

func (d *jsonDecoder) parseBool() (Value, error) {
	start := d.off
	switch {
	case d.off+4 <= len(d.src) && string(d.src[d.off:d.off+4]) == "true":
		d.off += 4
		return Bool{V: true, raw: d.src[start:d.off]}, nil
	case d.off+5 <= len(d.src) && string(d.src[d.off:d.off+5]) == "false":
		d.off += 5
		return Bool{V: false, raw: d.src[start:d.off]}, nil
	}
	return nil, d.errorf("invalid literal")
}

func (d *jsonDecoder) parseNull() (Value, error) {
	start := d.off
	if d.off+4 > len(d.src) || string(d.src[d.off:d.off+4]) != "null" {
		return nil, d.errorf("invalid literal")
	}
	d.off += 4
	return Null{raw: d.src[start:d.off]}, nil
}

func (d *jsonDecoder) parseNumber() (Value, error) {
	start := d.off
	if d.src[d.off] == '-' {
		d.off++
	}
	if d.off >= len(d.src) {
		return nil, d.errorf("truncated number")
	}
	switch c := d.src[d.off]; {
	case c == '0':
		d.off++
	case c >= '1' && c <= '9':
		for d.off < len(d.src) && d.src[d.off] >= '0' && d.src[d.off] <= '9' {
			d.off++
		}
	default:
		return nil, d.errorf("invalid number")
	}
	if d.off < len(d.src) && d.src[d.off] == '.' {
		d.off++
		cnt := 0
		for d.off < len(d.src) && d.src[d.off] >= '0' && d.src[d.off] <= '9' {
			d.off++
			cnt++
		}
		if cnt == 0 {
			return nil, d.errorf("fractional part requires at least one digit")
		}
	}
	if d.off < len(d.src) && (d.src[d.off] == 'e' || d.src[d.off] == 'E') {
		d.off++
		if d.off < len(d.src) && (d.src[d.off] == '+' || d.src[d.off] == '-') {
			d.off++
		}
		cnt := 0
		for d.off < len(d.src) && d.src[d.off] >= '0' && d.src[d.off] <= '9' {
			d.off++
			cnt++
		}
		if cnt == 0 {
			return nil, d.errorf("exponent requires at least one digit")
		}
	}
	return ParseNumber(d.src[start:d.off]), nil
}
