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
	call   func(*evaluator, []functionValue) (functionValue, error)
}

var builtinFunctions = map[string]functionDefinition{
	"length": {[]ast.ExprType{ast.TypeValue}, ast.TypeValue, lengthFn},
	"count":  {[]ast.ExprType{ast.TypeNodes}, ast.TypeValue, countFn},
	"value":  {[]ast.ExprType{ast.TypeNodes}, ast.TypeValue, valueFn},
	"match":  {[]ast.ExprType{ast.TypeValue, ast.TypeValue}, ast.TypeLogical, matchFn},
	"search": {[]ast.ExprType{ast.TypeValue, ast.TypeValue}, ast.TypeLogical, searchFn},
}

func lengthFn(_ *evaluator, args []functionValue) (functionValue, error) {
	if !args[0].nothing {
		if length := args[0].value.Len(); length >= 0 {
			return numberResult(length), nil
		}
	}
	return functionValue{nothing: true}, nil
}

func countFn(_ *evaluator, args []functionValue) (functionValue, error) {
	return numberResult(len(args[0].nodes)), nil
}

func valueFn(_ *evaluator, args []functionValue) (functionValue, error) {
	if len(args[0].nodes) == 1 {
		return functionValue{value: args[0].nodes[0]}, nil
	}
	return functionValue{nothing: true}, nil
}

func matchFn(e *evaluator, args []functionValue) (functionValue, error) {
	return e.matchRegexp(args, false)
}

func searchFn(e *evaluator, args []functionValue) (functionValue, error) {
	return e.matchRegexp(args, true)
}

func (e *evaluator) matchRegexp(args []functionValue, search bool) (functionValue, error) {
	text, pattern := args[0].value, args[1].value
	if args[0].nothing || args[1].nothing || text == nil || pattern == nil ||
		text.Kind() != document.KindString || pattern.Kind() != document.KindString {
		return functionValue{}, nil
	}
	r, cached := e.regexes[pattern.String()]
	if !cached {
		// Query 的缓存只读，动态模式不写入共享状态。
		var err error
		r, err = iregexp.Compile(pattern.String())
		if err != nil {
			var syntaxErr *iregexp.SyntaxError
			if !errors.As(err, &syntaxErr) {
				return functionValue{}, err
			}
		}
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
