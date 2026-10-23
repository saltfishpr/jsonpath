package jsonpath

import (
	"errors"
	"strconv"
	"unicode/utf8"

	"github.com/saltfishpr/jsonpath/internal/ast"
	"github.com/saltfishpr/jsonpath/internal/document"
	"github.com/saltfishpr/jsonpath/internal/iregexp"
)

// 字段的有效性由函数签名决定，无需在每个值中重复保存类型。
type functionValue struct {
	value   document.Value
	nothing bool // 区别于 JSON null。
	logical bool
	nodes   []document.Value
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
		switch v := args[0].value.(type) {
		case document.String:
			return numberResult(utf8.RuneCountInString(v.V)), nil
		case document.Array:
			return numberResult(v.Len()), nil
		case document.Object:
			return numberResult(v.MemberCount()), nil
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
	text, textOK := args[0].value.(document.String)
	pattern, patternOK := args[1].value.(document.String)
	if args[0].nothing || args[1].nothing || !textOK || !patternOK {
		return functionValue{}, nil
	}
	r, cached := e.regexes[pattern.V]
	if !cached {
		// Query 的缓存只读，动态模式不写入共享状态。
		var err error
		r, err = iregexp.Compile(pattern.V)
		if err != nil {
			if errors.Is(err, iregexp.ErrPatternTooLarge) {
				return functionValue{}, err
			}
		}
	}
	if r == nil {
		return logicalResult(false), nil
	}
	if search {
		return functionValue{logical: r.Search(text.V)}, nil
	}
	return functionValue{logical: r.Match(text.V)}, nil
}

func numberResult(n int) functionValue {
	return functionValue{value: document.ParseNumber([]byte(strconv.Itoa(n)))}
}

func logicalResult(b bool) functionValue {
	return functionValue{logical: b}
}
