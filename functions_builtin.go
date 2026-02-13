package jsonpath

import (
	"regexp"
	"unicode/utf8"
)

func registerLength() {
	functionRegistry["length"] = FunctionSignature{
		Name:       "length",
		ParamTypes: []FunctionValueType{FunctionValueTypeValue},
		ReturnType: FunctionValueTypeValue,
		Handler: func(args []interface{}) (interface{}, error) {
			val := args[0].(Result)

			switch {
			case val.IsString():
				count := utf8.RuneCountInString(val.Str)
				return Result{Type: JSONTypeNumber, Num: float64(count), Raw: val.Str}, nil
			case val.IsArray():
				arr := val.Array()
				return Result{Type: JSONTypeNumber, Num: float64(len(arr))}, nil
			case val.IsObject():
				obj := val.MapKVList()
				return Result{Type: JSONTypeNumber, Num: float64(len(obj))}, nil
			default:
				return FunctionValueNothing, nil
			}
		},
	}
}

func registerCount() {
	functionRegistry["count"] = FunctionSignature{
		Name:       "count",
		ParamTypes: []FunctionValueType{FunctionValueTypeNodes},
		ReturnType: FunctionValueTypeValue,
		Handler: func(args []interface{}) (interface{}, error) {
			nodes := args[0].([]Result)
			return Result{Type: JSONTypeNumber, Num: float64(len(nodes))}, nil
		},
	}
}

func registerMatch() {
	functionRegistry["match"] = FunctionSignature{
		Name:       "match",
		ParamTypes: []FunctionValueType{FunctionValueTypeValue, FunctionValueTypeValue},
		ReturnType: FunctionValueTypeLogical,
		Handler: func(args []interface{}) (interface{}, error) {
			strVal := args[0].(Result)
			patternVal := args[1].(Result)

			if !strVal.IsString() || !patternVal.IsString() {
				return false, nil
			}

			re, err := regexp.Compile("^" + patternVal.String() + "$")
			if err != nil {
				return false, nil
			}

			return re.MatchString(strVal.Str), nil
		},
	}
}

func registerSearch() {
	functionRegistry["search"] = FunctionSignature{
		Name:       "search",
		ParamTypes: []FunctionValueType{FunctionValueTypeValue, FunctionValueTypeValue},
		ReturnType: FunctionValueTypeLogical,
		Handler: func(args []interface{}) (interface{}, error) {
			strVal := args[0].(Result)
			patternVal := args[1].(Result)

			if !strVal.IsString() || !patternVal.IsString() {
				return false, nil
			}

			re, err := regexp.Compile(patternVal.Str)
			if err != nil {
				return false, nil
			}

			return re.MatchString(strVal.Str), nil
		},
	}
}

func registerValue() {
	functionRegistry["value"] = FunctionSignature{
		Name:       "value",
		ParamTypes: []FunctionValueType{FunctionValueTypeNodes},
		ReturnType: FunctionValueTypeValue,
		Handler: func(args []interface{}) (interface{}, error) {
			nodes := args[0].([]Result)

			switch len(nodes) {
			case 0:
				return FunctionValueNothing, nil
			case 1:
				return nodes[0], nil
			default:
				return FunctionValueNothing, nil // 多节点 → Nothing
			}
		},
	}
}
