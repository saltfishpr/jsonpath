package jsonpath

import (
	"strconv"
	"unicode/utf8"

	"github.com/saltfishpr/jsonpath/internal/iregexp"
)

func integerValue(n int) *jsonNode {
	return &jsonNode{kind: numberKind, number: parseDecimal(strconv.Itoa(n))}
}

func (c *evalContext) function(e *expression, n *jsonNode) expressionResult {
	r := expressionResult{typ: valueType}
	switch e.name {
	case "length":
		v := c.value(e.args[0], n)
		if v != nil {
			switch v.kind {
			case stringKind:
				r.value = integerValue(utf8.RuneCountInString(v.text))
			case arrayKind, objectKind:
				r.value = integerValue(len(v.children))
			}
		}
	case "count":
		r.value = integerValue(len(c.expression(e.args[0], n).nodes))
	case "value":
		nodes := c.expression(e.args[0], n).nodes
		if len(nodes) == 1 {
			r.value = nodes[0]
		}
	case "match", "search":
		r.typ = logicalType
		value, pattern := c.value(e.args[0], n), c.value(e.args[1], n)
		if value == nil || pattern == nil || value.kind != stringKind || pattern.kind != stringKind {
			return r
		}
		re := e.regex
		if !e.regexConstant {
			var ok bool
			re, ok = c.regex[pattern.text]
			if !ok {
				var err error
				re, err = iregexp.Compile(pattern.text)
				if limit, ok := err.(*iregexp.LimitError); ok {
					c.err = &LimitError{limit.Resource, limit.Limit}
					return r
				}
				if c.regex == nil {
					c.regex = make(map[string]*iregexp.Pattern)
				}
				c.regex[pattern.text] = re
			}
		}
		if re != nil {
			r.logical = re.Match(value.text, e.name == "search")
		}
	}
	return r
}
