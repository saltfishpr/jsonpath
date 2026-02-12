package jsonpath

import (
	"regexp"
	"strconv"
	"sync"
)

// FuncResultType is the return type of a function
type FuncResultType int

const (
	ResultTypeValueType FuncResultType = iota
	ResultTypeLogicalType
	ResultTypeNodesType
)

// FuncParamType is the parameter type of a function
type FuncParamType int

const (
	ParamTypeValueType FuncParamType = iota
	ParamTypeLogicalType
	ParamTypeNodesType
)

// FuncSignature defines a function's signature
type FuncSignature struct {
	Name       string
	ParamTypes []FuncParamType
	ReturnType FuncResultType
}

// FuncContext is the context where a function is called
type FuncContext int

const (
	ContextComparable FuncContext = iota // As part of comparison expression
	ContextTest                          // As test expression
	ContextArgument                      // As function argument
)

// builtinSignatures contains signatures for built-in functions
var builtinSignatures = map[string]*FuncSignature{
	"length": {Name: "length", ParamTypes: []FuncParamType{ParamTypeValueType}, ReturnType: ResultTypeValueType},
	"count":  {Name: "count", ParamTypes: []FuncParamType{ParamTypeNodesType}, ReturnType: ResultTypeValueType},
	"match":  {Name: "match", ParamTypes: []FuncParamType{ParamTypeValueType, ParamTypeValueType}, ReturnType: ResultTypeLogicalType},
	"search": {Name: "search", ParamTypes: []FuncParamType{ParamTypeValueType, ParamTypeValueType}, ReturnType: ResultTypeLogicalType},
	"value":  {Name: "value", ParamTypes: []FuncParamType{ParamTypeNodesType}, ReturnType: ResultTypeValueType},
}

// Custom function registries
var (
	customSignatures = make(map[string]*FuncSignature)
	customHandlers   = make(map[string]FunctionHandler)
	registryMutex    sync.RWMutex
)

// FunctionHandler implements a custom function
// Parameters: evaluator, argument values, function signature
// Returns: result value, success flag
type FunctionHandler func(*Evaluator, []evalFuncResult, *FuncSignature) (Result, bool)

// RegisterFunction registers a custom function signature
func RegisterFunction(name string, paramTypes []FuncParamType, returnType FuncResultType) {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	customSignatures[name] = &FuncSignature{
		Name:       name,
		ParamTypes: paramTypes,
		ReturnType: returnType,
	}
}

// RegisterFunctionHandler registers a custom function handler
func RegisterFunctionHandler(name string, handler FunctionHandler) {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	customHandlers[name] = handler
}

// UnregisterFunction unregisters a custom function
func UnregisterFunction(name string) {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	delete(customSignatures, name)
	delete(customHandlers, name)
}

// logicalResult converts a bool to Result (LogicalTrue/LogicalFalse)
func (e *Evaluator) logicalResult(value bool) Result {
	if value {
		return Result{Type: JSONTypeTrue, Raw: "true"}
	}
	return Result{Type: JSONTypeFalse, Raw: "false"}
}

// evalFuncResult holds the result of evaluating a function argument
type evalFuncResult struct {
	value      Result   // ValueType result
	logical    bool     // LogicalType result
	nodes      []Result // NodesType result
	resultType FuncResultType
	isNothing  bool // Whether result is Nothing
}

// findFunctionSignature finds a function's signature (builtin or custom)
func (e *Evaluator) findFunctionSignature(name string) *FuncSignature {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	if sig, ok := builtinSignatures[name]; ok {
		return sig
	}

	if sig, ok := customSignatures[name]; ok {
		return sig
	}

	return nil
}

// checkFunctionWellTyped checks if function call matches signature
func (e *Evaluator) checkFunctionWellTyped(fn *FuncCall, sig *FuncSignature, context FuncContext) bool {
	if len(fn.Args) != len(sig.ParamTypes) {
		return false
	}

	switch context {
	case ContextTest:
		// As test-expr: return must be LogicalType or NodesType
		if sig.ReturnType != ResultTypeLogicalType && sig.ReturnType != ResultTypeNodesType {
			return false
		}
	case ContextComparable:
		// As comparable: return must be ValueType
		if sig.ReturnType != ResultTypeValueType {
			return false
		}
	case ContextArgument:
		// As argument: checked by caller
	}

	return true
}

func (e *Evaluator) evalFuncArgs(currentNode Result, fn *FuncCall, sig *FuncSignature) ([]evalFuncResult, bool) {
	args := make([]evalFuncResult, len(fn.Args))

	for i, arg := range fn.Args {
		paramType := sig.ParamTypes[i]
		result, ok := e.evalFuncArg(currentNode, arg, paramType)
		if !ok {
			return nil, false
		}
		args[i] = result
	}
	return args, true
}

func (e *Evaluator) evalFuncArg(currentNode Result, arg *FuncArg, expectedType FuncParamType) (evalFuncResult, bool) {
	switch arg.Type {
	case FuncArgLiteral:
		return e.evalLiteralArg(arg.Literal, expectedType)
	case FuncArgFilterQuery:
		return e.evalFilterQueryArg(currentNode, arg.FilterQuery, expectedType)
	case FuncArgLogicalExpr:
		return e.evalLogicalExprArg(currentNode, arg.LogicalExpr, expectedType)
	case FuncArgFuncExpr:
		return e.evalFuncExprArg(currentNode, arg.FuncExpr, expectedType)
	}
	return evalFuncResult{}, false
}

func (e *Evaluator) evalLiteralArg(lit *LiteralValue, expectedType FuncParamType) (evalFuncResult, bool) {
	switch expectedType {
	case ParamTypeValueType:
		return evalFuncResult{
			value:      e.evalLiteral(lit),
			resultType: ResultTypeValueType,
			isNothing:  false,
		}, true
	case ParamTypeLogicalType:
		return evalFuncResult{}, false
	case ParamTypeNodesType:
		return evalFuncResult{}, false
	}
	return evalFuncResult{}, false
}

func (e *Evaluator) evalFilterQueryArg(currentNode Result, fq *FilterQuery, expectedType FuncParamType) (evalFuncResult, bool) {
	results := e.evalFilterQuery(currentNode, fq)

	switch expectedType {
	case ParamTypeValueType:
		if len(results) == 0 {
			return evalFuncResult{isNothing: true, resultType: ResultTypeValueType}, true
		}
		if len(results) == 1 {
			return evalFuncResult{
				value:      results[0],
				resultType: ResultTypeValueType,
				isNothing:  false,
			}, true
		}
		// Multiple nodes return Nothing (RFC 9535)
		return evalFuncResult{isNothing: true, resultType: ResultTypeValueType}, true
	case ParamTypeNodesType:
		return evalFuncResult{
			nodes:      results,
			resultType: ResultTypeNodesType,
			isNothing:  false,
		}, true
	case ParamTypeLogicalType:
		logical := len(results) > 0
		return evalFuncResult{
			logical:    logical,
			resultType: ResultTypeLogicalType,
			isNothing:  false,
		}, true
	}
	return evalFuncResult{}, false
}

func (e *Evaluator) evalLogicalExprArg(currentNode Result, expr *FilterExpr, expectedType FuncParamType) (evalFuncResult, bool) {
	if expectedType != ParamTypeLogicalType {
		return evalFuncResult{}, false
	}

	logical := e.evalFilterExpr(currentNode, expr)
	return evalFuncResult{
		logical:    logical,
		resultType: ResultTypeLogicalType,
		isNothing:  false,
	}, true
}

func (e *Evaluator) evalFuncExprArg(currentNode Result, fn *FuncCall, expectedType FuncParamType) (evalFuncResult, bool) {
	result, ok := e.evalFuncCall(currentNode, fn, ContextArgument)
	if !ok {
		return evalFuncResult{}, false
	}

	sig := e.findFunctionSignature(fn.Name)
	if sig == nil {
		return evalFuncResult{}, false
	}

	switch sig.ReturnType {
	case ResultTypeValueType:
		if expectedType == ParamTypeValueType {
			return evalFuncResult{
				value:      result,
				resultType: ResultTypeValueType,
				isNothing:  !result.Exists(),
			}, true
		}
	case ResultTypeLogicalType:
		if expectedType == ParamTypeLogicalType {
			logical := result.Exists() && result.Type != JSONTypeNull && result.Type != JSONTypeFalse
			return evalFuncResult{
				logical:    logical,
				resultType: ResultTypeLogicalType,
				isNothing:  false,
			}, true
		}
	case ResultTypeNodesType:
		if expectedType == ParamTypeNodesType {
			return evalFuncResult{}, false
		}
		if expectedType == ParamTypeLogicalType {
			logical := result.Exists()
			return evalFuncResult{
				logical:    logical,
				resultType: ResultTypeLogicalType,
				isNothing:  false,
			}, true
		}
	}

	return evalFuncResult{}, false
}

// evalFuncCall evaluates a function call
func (e *Evaluator) evalFuncCall(currentNode Result, fn *FuncCall, context FuncContext) (Result, bool) {
	sig := e.findFunctionSignature(fn.Name)
	if sig == nil {
		return Result{}, false
	}

	if !e.checkFunctionWellTyped(fn, sig, context) {
		return Result{}, false
	}

	args, ok := e.evalFuncArgs(currentNode, fn, sig)
	if !ok {
		return Result{}, false
	}

	return e.callFunction(fn.Name, args, sig)
}

// callFunction calls a function implementation
func (e *Evaluator) callFunction(name string, args []evalFuncResult, sig *FuncSignature) (Result, bool) {
	registryMutex.RLock()
	handler, hasCustom := customHandlers[name]
	registryMutex.RUnlock()

	if hasCustom {
		return handler(e, args, sig)
	}

	switch name {
	case "length":
		return e.builtinLength(args)
	case "count":
		return e.builtinCount(args)
	case "match":
		return e.builtinMatch(args)
	case "search":
		return e.builtinSearch(args)
	case "value":
		return e.builtinValue(args)
	}

	return Result{}, false
}

// builtinLength implements the length() function
func (e *Evaluator) builtinLength(args []evalFuncResult) (Result, bool) {
	if len(args) != 1 {
		return Result{}, false
	}
	arg := args[0]

	if arg.isNothing {
		return Result{}, true
	}

	if arg.resultType != ResultTypeValueType {
		return Result{}, false
	}

	v := arg.value
	var length int

	switch {
	case v.IsArray():
		length = len(v.Array())
	case v.IsObject():
		length = len(v.Map())
	case v.Type == JSONTypeString:
		length = len(v.Str)
	default:
		return Result{}, true
	}

	return Result{
		Type: JSONTypeNumber,
		Num:  float64(length),
		Raw:  strconv.Itoa(length),
	}, true
}

// builtinCount implements the count() function
func (e *Evaluator) builtinCount(args []evalFuncResult) (Result, bool) {
	if len(args) != 1 {
		return Result{}, false
	}
	arg := args[0]

	if arg.resultType != ResultTypeNodesType {
		return Result{}, false
	}

	count := len(arg.nodes)
	return Result{
		Type: JSONTypeNumber,
		Num:  float64(count),
		Raw:  strconv.Itoa(count),
	}, true
}

// builtinMatch implements the match() function
func (e *Evaluator) builtinMatch(args []evalFuncResult) (Result, bool) {
	input, pattern, ok := e.checkTwoStringArgs(args)
	if !ok {
		return e.logicalResult(false), true
	}

	re, err := regexp.Compile("^" + pattern + "$")
	if err != nil {
		return e.logicalResult(false), true
	}

	matched := re.MatchString(input)
	return e.logicalResult(matched), true
}

// checkTwoStringArgs verifies both args are strings
func (e *Evaluator) checkTwoStringArgs(args []evalFuncResult) (string, string, bool) {
	if len(args) != 2 {
		return "", "", false
	}

	input := args[0]
	if input.isNothing || input.resultType != ResultTypeValueType || input.value.Type != JSONTypeString {
		return "", "", false
	}

	pattern := args[1]
	if pattern.isNothing || pattern.resultType != ResultTypeValueType || pattern.value.Type != JSONTypeString {
		return "", "", false
	}

	return input.value.Str, pattern.value.Str, true
}

// builtinSearch implements the search() function
func (e *Evaluator) builtinSearch(args []evalFuncResult) (Result, bool) {
	input, pattern, ok := e.checkTwoStringArgs(args)
	if !ok {
		return e.logicalResult(false), true
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return e.logicalResult(false), true
	}

	matched := re.MatchString(input)
	return e.logicalResult(matched), true
}

// builtinValue implements the value() function
func (e *Evaluator) builtinValue(args []evalFuncResult) (Result, bool) {
	if len(args) != 1 {
		return Result{}, false
	}
	arg := args[0]

	if arg.resultType != ResultTypeNodesType {
		return Result{}, false
	}

	if len(arg.nodes) == 1 {
		return arg.nodes[0], true
	}
	return Result{}, true
}
