package jsonpath

import (
	"fmt"
)

// FunctionValueType 表示函数参数/返回值的类型
type FunctionValueType int

const (
	FunctionValueTypeValue   FunctionValueType = iota // Result / Noting
	FunctionValueTypeLogical                          // true/false
	FunctionValueTypeNodes                            // []Result
)

// String 返回类型的字符串表示
func (t FunctionValueType) String() string {
	switch t {
	case FunctionValueTypeValue:
		return "ValueType"
	case FunctionValueTypeLogical:
		return "LogicalType"
	case FunctionValueTypeNodes:
		return "NodesType"
	default:
		return "Unknown"
	}
}

var FunctionValueNothing = Result{}

// FunctionSignature 定义函数签名
type FunctionSignature struct {
	Name       string
	ParamTypes []FunctionValueType
	ReturnType FunctionValueType
	Handler    func(args []interface{}) (interface{}, error)
}

// 函数注册表
var functionRegistry = map[string]FunctionSignature{}

// init 注册所有标准函数
func init() {
	registerLength()
	registerCount()
	registerMatch()
	registerSearch()
	registerValue()
}

func (e *Evaluator) evalFuncCall(currentNode Result, fn *FuncCall, expectedType FunctionValueType) (interface{}, error) {
	sig, exists := functionRegistry[fn.Name]
	if !exists {
		return nil, fmt.Errorf("unknown function: %s", fn.Name)
	}

	if len(fn.Args) != len(sig.ParamTypes) {
		return nil, fmt.Errorf("%s() expects %d arguments, got %d",
			fn.Name, len(sig.ParamTypes), len(fn.Args))
	}

	args := make([]interface{}, len(fn.Args))
	for i, arg := range fn.Args {
		val, err := e.evalFuncArg(currentNode, arg, sig.ParamTypes[i])
		if err != nil {
			return nil, fmt.Errorf("argument %d of %s(): %w", i+1, fn.Name, err)
		}
		args[i] = val
	}

	return sig.Handler(args)
}

func (e *Evaluator) evalFuncArg(currentNode Result, arg *FuncArg, expectedType FunctionValueType) (interface{}, error) {
	switch arg.Type {
	case FuncArgLiteral:
		// 字面量只能是 ValueType
		if expectedType != FunctionValueTypeValue {
			return nil, fmt.Errorf("literal cannot be converted to %s", expectedType)
		}
		return e.evalLiteral(arg.Literal), nil

	case FuncArgFilterQuery:
		nodes := e.evalFilterQuery(currentNode, arg.FilterQuery)
		switch expectedType {
		case FunctionValueTypeValue:
			if len(nodes) == 1 {
				return nodes[0], nil
			}
			return FunctionValueNothing, nil
		case FunctionValueTypeLogical:
			return len(nodes) > 0, nil
		case FunctionValueTypeNodes:
			return nodes, nil
		default:
			return nil, fmt.Errorf("cannot convert nodes to %s", expectedType)
		}

	case FuncArgLogicalExpr:
		if expectedType != FunctionValueTypeLogical {
			return nil, fmt.Errorf("logical expression cannot be converted to %s", expectedType)
		}
		return e.evalFilterExpr(currentNode, arg.LogicalExpr), nil

	case FuncArgFuncExpr:
		fn := arg.FuncExpr
		sig, exists := functionRegistry[fn.Name]
		if !exists {
			return nil, fmt.Errorf("unknown function: %s", fn.Name)
		}
		// 嵌套函数调用：期望类型必须与嵌套函数的返回类型匹配
		resultAny, err := e.evalFuncCall(currentNode, fn, sig.ReturnType)
		if err != nil {
			return nil, err
		}
		// 类型兼容性检查
		if expectedType != sig.ReturnType {
			return nil, fmt.Errorf("type mismatch: %s() returns %s but %s is expected", fn.Name, sig.ReturnType, expectedType)
		}
		return resultAny, nil

	default:
		return nil, fmt.Errorf("unknown function argument type")
	}
}
