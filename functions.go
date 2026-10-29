package jsonpath

import (
	"errors"
	"strconv"

	"github.com/saltfishpr/jsonpath/internal/document"
	"github.com/saltfishpr/jsonpath/internal/iregexp"
)

// The compiled function signature determines which field is active. ValueType
// uses value and nothing; LogicalType uses logical; NodesType uses nodes.
type functionValue struct {
	value   *document.Node
	nothing bool // Nothing has no JSON representation; a null node is an ordinary value.
	logical bool
	nodes   []*document.Node
}

type functionDefinition struct {
	params []exprType
	result exprType
}

var builtinFunctions = map[string]functionDefinition{
	"length": {[]exprType{exprTypeValue}, exprTypeValue},
	"count":  {[]exprType{exprTypeNodes}, exprTypeValue},
	"value":  {[]exprType{exprTypeNodes}, exprTypeValue},
	"match":  {[]exprType{exprTypeValue, exprTypeValue}, exprTypeLogical},
	"search": {[]exprType{exprTypeValue, exprTypeValue}, exprTypeLogical},
}

func lengthFn(args []functionValue) (functionValue, error) {
	if !args[0].nothing {
		if length := args[0].value.Len(); length >= 0 {
			return numberResult(length), nil
		}
	}
	return functionValue{nothing: true}, nil
}

func countFn(args []functionValue) (functionValue, error) {
	return numberResult(len(args[0].nodes)), nil
}

func valueFn(args []functionValue) (functionValue, error) {
	if len(args[0].nodes) == 1 {
		return functionValue{value: args[0].nodes[0]}, nil
	}
	return functionValue{nothing: true}, nil
}

func (e *evaluator) matchRegexp(args []functionValue, search bool) (functionValue, error) {
	text, pattern := args[0].value, args[1].value
	if args[0].nothing || args[1].nothing || text == nil || pattern == nil ||
		text.Kind() != document.KindString || pattern.Kind() != document.KindString {
		return functionValue{}, nil
	}
	patternText := pattern.String()
	r, cached := e.regexes[patternText]
	if !cached {
		r, cached = e.dynamicRegexes[patternText]
	}
	if !cached {
		var err error
		r, err = iregexp.Compile(patternText)
		if err != nil {
			var syntaxErr *iregexp.SyntaxError
			if !errors.As(err, &syntaxErr) {
				return functionValue{}, err
			}
		}
		if e.dynamicRegexes == nil {
			e.dynamicRegexes = make(map[string]*iregexp.Regexp)
		}
		// A cached nil records invalid syntax, which must yield false on every use.
		e.dynamicRegexes[patternText] = r
	}
	if r == nil {
		return logicalResult(false), nil
	}
	if search {
		return functionValue{logical: r.Search(text.String())}, nil
	}
	return functionValue{logical: r.Match(text.String())}, nil
}

func numberResult(n int) functionValue {
	return functionValue{value: document.NewNumber(strconv.Itoa(n))}
}

func logicalResult(b bool) functionValue {
	return functionValue{logical: b}
}
