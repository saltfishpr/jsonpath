package jsonpath

import (
	"regexp"
	"strconv"
	"sync"
)

// FuncResultType 函数返回类型
type FuncResultType int

const (
	ResultTypeValueType FuncResultType = iota
	ResultTypeLogicalType
	ResultTypeNodesType
)

// FuncParamType 函数参数类型
type FuncParamType int

const (
	ParamTypeValueType FuncParamType = iota
	ParamTypeLogicalType
	ParamTypeNodesType
)

// FuncSignature 函数签名
type FuncSignature struct {
	Name       string
	ParamTypes []FuncParamType
	ReturnType FuncResultType
}

// FuncContext 函数调用上下文
type FuncContext int

const (
	ContextComparable FuncContext = iota // 作为比较表达式的一部分
	ContextTest                           // 作为测试表达式
	ContextArgument                       // 作为函数参数
)

// 内置函数签名表
var builtinSignatures = map[string]*FuncSignature{
	"length": {Name: "length", ParamTypes: []FuncParamType{ParamTypeValueType}, ReturnType: ResultTypeValueType},
	"count":  {Name: "count", ParamTypes: []FuncParamType{ParamTypeNodesType}, ReturnType: ResultTypeValueType},
	"match":  {Name: "match", ParamTypes: []FuncParamType{ParamTypeValueType, ParamTypeValueType}, ReturnType: ResultTypeLogicalType},
	"search": {Name: "search", ParamTypes: []FuncParamType{ParamTypeValueType, ParamTypeValueType}, ReturnType: ResultTypeLogicalType},
	"value":  {Name: "value", ParamTypes: []FuncParamType{ParamTypeNodesType}, ReturnType: ResultTypeValueType},
}

// 自定义函数注册表
var (
	customSignatures = make(map[string]*FuncSignature)
	customHandlers   = make(map[string]FunctionHandler)
	registryMutex    sync.RWMutex
)

// FunctionHandler 函数处理器类型
// 参数: evaluator, 参数值列表, 函数签名
// 返回: 结果值, 是否成功
type FunctionHandler func(*Evaluator, []evalFuncResult, *FuncSignature) (Result, bool)

// RegisterFunction 注册自定义函数签名
func RegisterFunction(name string, paramTypes []FuncParamType, returnType FuncResultType) {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	customSignatures[name] = &FuncSignature{
		Name:       name,
		ParamTypes: paramTypes,
		ReturnType: returnType,
	}
}

// RegisterFunctionHandler 注册自定义函数处理器
func RegisterFunctionHandler(name string, handler FunctionHandler) {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	customHandlers[name] = handler
}

// UnregisterFunction 注销自定义函数
func UnregisterFunction(name string) {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	delete(customSignatures, name)
	delete(customHandlers, name)
}

// logicalResult 将逻辑值转换为 Result（用于表示 LogicalTrue/LogicalFalse）
func (e *Evaluator) logicalResult(value bool) Result {
	if value {
		return Result{Type: JSONTypeTrue, Raw: "true"}
	}
	return Result{Type: JSONTypeFalse, Raw: "false"}
}

// evalFuncResult 函数求值结果
type evalFuncResult struct {
	value      Result   // ValueType 结果
	logical    bool     // LogicalType 结果
	nodes      []Result // NodesType 结果
	resultType FuncResultType
	isNothing  bool // 标记是否为 Nothing
}

// findFunctionSignature 查找函数签名
func (e *Evaluator) findFunctionSignature(name string) *FuncSignature {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	// 先查内置函数
	if sig, ok := builtinSignatures[name]; ok {
		return sig
	}

	// 再查自定义函数
	if sig, ok := customSignatures[name]; ok {
		return sig
	}

	return nil
}

// checkFunctionWellTyped 检查函数是否类型正确
func (e *Evaluator) checkFunctionWellTyped(fn *FuncCall, sig *FuncSignature, context FuncContext) bool {
	// 检查参数数量
	if len(fn.Args) != len(sig.ParamTypes) {
		return false
	}

	// 根据上下文检查返回类型
	switch context {
	case ContextTest:
		// 作为 test-expr: 返回类型必须是 LogicalType 或 NodesType
		if sig.ReturnType != ResultTypeLogicalType && sig.ReturnType != ResultTypeNodesType {
			return false
		}
	case ContextComparable:
		// 作为 comparable: 返回类型必须是 ValueType
		if sig.ReturnType != ResultTypeValueType {
			return false
		}
	case ContextArgument:
		// 作为参数: 由调用方检查，这里不做限制
	}

	return true
}

// evalFuncArgs 评估函数参数
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

// evalFuncArg 评估单个函数参数
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

// evalLiteralArg 评估字面量参数
func (e *Evaluator) evalLiteralArg(lit *LiteralValue, expectedType FuncParamType) (evalFuncResult, bool) {
	switch expectedType {
	case ParamTypeValueType:
		return evalFuncResult{
			value:      e.evalLiteral(lit),
			resultType: ResultTypeValueType,
			isNothing:  false,
		}, true
	case ParamTypeLogicalType:
		// 字面量不能作为 LogicalType 参数
		return evalFuncResult{}, false
	case ParamTypeNodesType:
		// 字面量不能作为 NodesType 参数
		return evalFuncResult{}, false
	}
	return evalFuncResult{}, false
}

// evalFilterQueryArg 评估查询参数
func (e *Evaluator) evalFilterQueryArg(currentNode Result, fq *FilterQuery, expectedType FuncParamType) (evalFuncResult, bool) {
	var results []Result

	if fq.Relative {
		results = []Result{currentNode}
	} else {
		results = []Result{parseValue(e.json)}
	}

	// 应用每个段
	for _, seg := range fq.Segments {
		var newResults []Result
		for _, r := range results {
			for _, selector := range seg.Selectors {
				selected := e.evaluateSelector(r, selector)
				newResults = append(newResults, selected...)
			}
		}
		results = newResults
		if len(results) == 0 {
			break
		}
	}

	switch expectedType {
	case ParamTypeValueType:
		// ValueType: 单值查询结果
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
		// 多个节点，返回 Nothing（RFC 9535 规定）
		return evalFuncResult{isNothing: true, resultType: ResultTypeValueType}, true
	case ParamTypeNodesType:
		return evalFuncResult{
			nodes:      results,
			resultType: ResultTypeNodesType,
			isNothing:  false,
		}, true
	case ParamTypeLogicalType:
		// NodesType 可以隐式转换为 LogicalType
		logical := len(results) > 0
		return evalFuncResult{
			logical:    logical,
			resultType: ResultTypeLogicalType,
			isNothing:  false,
		}, true
	}
	return evalFuncResult{}, false
}

// evalLogicalExprArg 评估逻辑表达式参数
func (e *Evaluator) evalLogicalExprArg(currentNode Result, expr *FilterExpr, expectedType FuncParamType) (evalFuncResult, bool) {
	// 逻辑表达式只能作为 LogicalType 参数
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

// evalFuncExprArg 评估嵌套函数表达式参数
func (e *Evaluator) evalFuncExprArg(currentNode Result, fn *FuncCall, expectedType FuncParamType) (evalFuncResult, bool) {
	result, ok := e.evalFuncCall(currentNode, fn, ContextArgument)
	if !ok {
		return evalFuncResult{}, false
	}

	// 根据嵌套函数的返回类型和期望类型进行转换
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
			// NodesType 结果需要特殊处理
			// 但这里我们只有一个 Result，无法表示 nodelist
			// 这种情况应该由调用方处理
			return evalFuncResult{}, false
		}
		// NodesType 可以隐式转换为 LogicalType
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

// evalFuncCall 评估函数调用
func (e *Evaluator) evalFuncCall(currentNode Result, fn *FuncCall, context FuncContext) (Result, bool) {
	// 1. 查找函数签名
	sig := e.findFunctionSignature(fn.Name)
	if sig == nil {
		return Result{}, false
	}

	// 2. 类型检查
	if !e.checkFunctionWellTyped(fn, sig, context) {
		return Result{}, false
	}

	// 3. 评估参数
	args, ok := e.evalFuncArgs(currentNode, fn, sig)
	if !ok {
		return Result{}, false
	}

	// 4. 调用函数实现
	return e.callFunction(fn.Name, args, sig)
}

// callFunction 调用函数实现
func (e *Evaluator) callFunction(name string, args []evalFuncResult, sig *FuncSignature) (Result, bool) {
	registryMutex.RLock()
	handler, hasCustom := customHandlers[name]
	registryMutex.RUnlock()

	// 优先使用自定义处理器
	if hasCustom {
		return handler(e, args, sig)
	}

	// 使用内置函数实现
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

// builtinLength 实现 length() 函数
func (e *Evaluator) builtinLength(args []evalFuncResult) (Result, bool) {
	if len(args) != 1 {
		return Result{}, false
	}
	arg := args[0]

	// 如果是 Nothing，返回 Nothing
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
		// 其他类型返回 Nothing
		return Result{}, true
	}

	return Result{
		Type: JSONTypeNumber,
		Num:  float64(length),
		Raw:  strconv.Itoa(length),
	}, true
}

// builtinCount 实现 count() 函数
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

// builtinMatch 实现 match() 函数
func (e *Evaluator) builtinMatch(args []evalFuncResult) (Result, bool) {
	if len(args) != 2 {
		return Result{}, false
	}

	// 第一个参数: 字符串
	input := args[0]
	if input.isNothing || input.resultType != ResultTypeValueType || input.value.Type != JSONTypeString {
		return e.logicalResult(false), true // LogicalFalse
	}

	// 第二个参数: 正则表达式
	pattern := args[1]
	if pattern.isNothing || pattern.resultType != ResultTypeValueType || pattern.value.Type != JSONTypeString {
		return e.logicalResult(false), true // LogicalFalse
	}

	// 编译正则表达式
	re, err := regexp.Compile("^" + pattern.value.Str + "$")
	if err != nil {
		// 无效的正则表达式，返回 LogicalFalse
		return e.logicalResult(false), true
	}

	matched := re.MatchString(input.value.Str)
	return e.logicalResult(matched), true
}

// builtinSearch 实现 search() 函数
func (e *Evaluator) builtinSearch(args []evalFuncResult) (Result, bool) {
	if len(args) != 2 {
		return Result{}, false
	}

	// 第一个参数: 字符串
	input := args[0]
	if input.isNothing || input.resultType != ResultTypeValueType || input.value.Type != JSONTypeString {
		return e.logicalResult(false), true // LogicalFalse
	}

	// 第二个参数: 正则表达式
	pattern := args[1]
	if pattern.isNothing || pattern.resultType != ResultTypeValueType || pattern.value.Type != JSONTypeString {
		return e.logicalResult(false), true // LogicalFalse
	}

	// 编译正则表达式
	re, err := regexp.Compile(pattern.value.Str)
	if err != nil {
		// 无效的正则表达式，返回 LogicalFalse
		return e.logicalResult(false), true
	}

	matched := re.MatchString(input.value.Str)
	return e.logicalResult(matched), true
}

// builtinValue 实现 value() 函数
func (e *Evaluator) builtinValue(args []evalFuncResult) (Result, bool) {
	if len(args) != 1 {
		return Result{}, false
	}
	arg := args[0]

	if arg.resultType != ResultTypeNodesType {
		return Result{}, false
	}

	// 单节点返回值，空/多节点返回 Nothing
	if len(arg.nodes) == 1 {
		return arg.nodes[0], true
	}
	return Result{}, true // Nothing
}
