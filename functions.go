package jsonpath

import (
	"errors"
	"strconv"

	"github.com/saltfishpr/jsonpath/internal/ast"
	"github.com/saltfishpr/jsonpath/internal/document"
	"github.com/saltfishpr/jsonpath/internal/iregexp"
)

// 字段的有效性由函数签名决定，无需在每个值中重复保存类型。
type functionValue struct {
	value   *document.Node
	nothing bool // 区别于 JSON null。
	logical bool
	nodes   []*document.Node
}

type functionDefinition struct {
	params []ast.ExprType
	result ast.ExprType
}

var builtinFunctions = map[string]functionDefinition{
	"length": {[]ast.ExprType{ast.TypeValue}, ast.TypeValue},
	"count":  {[]ast.ExprType{ast.TypeNodes}, ast.TypeValue},
	"value":  {[]ast.ExprType{ast.TypeNodes}, ast.TypeValue},
	"match":  {[]ast.ExprType{ast.TypeValue, ast.TypeValue}, ast.TypeLogical},
	"search": {[]ast.ExprType{ast.TypeValue, ast.TypeValue}, ast.TypeLogical},
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
		// nil 也写入缓存，避免对无效模式重复编译。
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
