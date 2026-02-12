package jsonpath

import (
	"strconv"
)

// Evaluator JSONPath 表达式求值器
type Evaluator struct {
	json  string
	query *Query
}

// NewEvaluator 创建新的求值器
func NewEvaluator(json string, query *Query) *Evaluator {
	return &Evaluator{
		json:  json,
		query: query,
	}
}

// Evaluate 执行查询并返回结果列表
func (e *Evaluator) Evaluate() []Result {
	root := parseValue(e.json)
	if !root.Exists() {
		return nil
	}

	results := []Result{root}

	for _, segment := range e.query.Segments {
		results = e.evaluateSegment(results, segment)
		if len(results) == 0 {
			return nil
		}
	}

	return results
}

func (e *Evaluator) evaluateSegment(input []Result, segment *Segment) []Result {
	var output []Result

	if segment.Type == DescendantSegment {
		// 后代段: 递归查找所有后代节点
		for _, result := range input {
			descendants := e.evalDescendant(result, segment.Selectors)
			output = append(output, descendants...)
		}
	} else {
		// 子段: 只查找直接子节点
		for _, result := range input {
			for _, selector := range segment.Selectors {
				selected := e.evaluateSelector(result, selector)
				output = append(output, selected...)
			}
		}
	}

	return output
}

// evalDescendant 评估后代段，递归查找所有后代节点
func (e *Evaluator) evalDescendant(result Result, selectors []*Selector) []Result {
	var results []Result

	// 深度优先遍历
	e.collectDescendants(result, selectors, &results)

	return results
}

// collectDescendants 递归收集后代节点
func (e *Evaluator) collectDescendants(result Result, selectors []*Selector, results *[]Result) {
	// 先对当前节点应用选择器
	for _, selector := range selectors {
		selected := e.evaluateSelector(result, selector)
		*results = append(*results, selected...)
	}

	// 递归处理子节点
	if result.IsArray() {
		for _, elem := range result.Array() {
			e.collectDescendants(elem, selectors, results)
		}
	} else if result.IsObject() {
		for _, kv := range result.MapKVList() {
			e.collectDescendants(kv.Value, selectors, results)
		}
	}
}

// evaluateSelector 在单个结果上评估选择器
func (e *Evaluator) evaluateSelector(result Result, selector *Selector) []Result {
	switch selector.Type {
	case NameSelector:
		return e.evalNameSelector(result, selector.Name)
	case WildcardSelector:
		return e.evalWildcardSelector(result)
	case IndexSelector:
		return e.evalIndexSelector(result, selector.Index)
	case SliceSelector:
		return e.evalSliceSelector(result, selector.Slice)
	case FilterSelector:
		return e.evalFilterSelector(result, selector.Filter)
	default:
		return nil
	}
}

// evalNameSelector 评估名称选择器
func (e *Evaluator) evalNameSelector(result Result, name string) []Result {
	if !result.IsObject() {
		return nil
	}
	m := result.Map()
	if v, ok := m[name]; ok {
		return []Result{v}
	}
	return nil
}

// evalWildcardSelector 评估通配符选择器
func (e *Evaluator) evalWildcardSelector(result Result) []Result {
	if result.IsArray() {
		return result.Array()
	}
	if result.IsObject() {
		var results []Result
		for _, kv := range result.MapKVList() {
			results = append(results, kv.Value)
		}
		return results
	}
	return nil
}

// evalIndexSelector 评估索引选择器
func (e *Evaluator) evalIndexSelector(result Result, index int) []Result {
	if !result.IsArray() {
		return nil
	}
	arr := result.Array()
	length := len(arr)

	// 处理负索引
	if index < 0 {
		index = length + index
	}

	// 超出范围返回空（RFC 9535 规定）
	if index < 0 || index >= length {
		return nil
	}

	return []Result{arr[index]}
}

// evalSliceSelector 评估切片选择器
func (e *Evaluator) evalSliceSelector(result Result, slice *SliceParams) []Result {
	if !result.IsArray() {
		return nil
	}

	arr := result.Array()
	arrLen := len(arr)

	step := 1
	if slice.Step != nil {
		step = *slice.Step
	}

	if step == 0 {
		return nil // RFC 规定 step=0 返回空
	}

	start, end, endIsDefault := e.normalizeSliceBounds(slice.Start, slice.End, step, arrLen)

	var results []Result
	if step > 0 {
		// 正步长：从 start 到 end（不包含）
		for i := start; i < end; i += step {
			if i >= 0 && i < arrLen {
				results = append(results, arr[i])
			}
		}
	} else {
		// 负步长：从 start 向下到 end（不包含）
		// endIsDefault 表示 end 是否为默认值（对于默认的 -1，包含最后一个元素）
		if endIsDefault {
			// 默认 end，从最后一个元素开始
			for i := start; i >= 0; i += step {
				results = append(results, arr[i])
			}
		} else {
			for i := start; i > end; i += step {
				if i >= 0 && i < arrLen {
					results = append(results, arr[i])
				}
			}
		}
	}

	return results
}

// normalizeSliceBounds 计算切片边界的规范化值
// 返回 (start, end, endIsDefault)
func (e *Evaluator) normalizeSliceBounds(start, end *int, step, arrLen int) (int, int, bool) {
	// 默认值（RFC 9535 表 8）
	s := 0
	if start != nil {
		s = *start
		if s < 0 {
			s = arrLen + s
		}
	} else if step < 0 {
		s = arrLen - 1
	}

	en := arrLen
	endIsDefault := false
	if end != nil {
		en = *end
		if en < 0 {
			en = arrLen + en
		}
	} else if step < 0 {
		en = -1 // 标记为默认 end，需要特殊处理
		endIsDefault = true
	}

	// bounds 限制
	s = clamp(s, 0, arrLen-1)
	if !endIsDefault {
		en = clamp(en, 0, arrLen)
	}

	return s, en, endIsDefault
}

// clamp 将值限制在 [min, max] 范围内
func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// evalFilterSelector 评估过滤器选择器
func (e *Evaluator) evalFilterSelector(result Result, filter *FilterExpr) []Result {
	var results []Result

	if result.IsArray() {
		for _, elem := range result.Array() {
			if e.evalFilterExpr(elem, filter) {
				results = append(results, elem)
			}
		}
	} else if result.IsObject() {
		for _, kv := range result.MapKVList() {
			if e.evalFilterExpr(kv.Value, filter) {
				results = append(results, kv.Value)
			}
		}
	}

	return results
}

// evalFilterExpr 评估过滤表达式
func (e *Evaluator) evalFilterExpr(currentNode Result, expr *FilterExpr) bool {
	switch expr.Type {
	case FilterLogicalOr:
		return e.evalFilterExpr(currentNode, expr.Left) || e.evalFilterExpr(currentNode, expr.Right)
	case FilterLogicalAnd:
		return e.evalFilterExpr(currentNode, expr.Left) && e.evalFilterExpr(currentNode, expr.Right)
	case FilterLogicalNot:
		return !e.evalFilterExpr(currentNode, expr.Operand)
	case FilterParen:
		return e.evalFilterExpr(currentNode, expr.Operand)
	case FilterComparison:
		return e.evalComparison(currentNode, expr.Comp)
	case FilterTest:
		return e.evalTestExpr(currentNode, expr.Test)
	}
	return false
}

// evalComparison 评估比较表达式
func (e *Evaluator) evalComparison(currentNode Result, comp *Comparison) bool {
	left := e.evalComparable(currentNode, comp.Left)
	right := e.evalComparable(currentNode, comp.Right)

	// 处理空 nodelist（RFC 9535 规则）
	leftEmpty := !left.Exists()
	rightEmpty := !right.Exists()

	if leftEmpty || rightEmpty {
		switch comp.Op {
		case CompEq:
			return leftEmpty && rightEmpty
		case CompNe:
			return !leftEmpty || !rightEmpty
		default:
			return false
		}
	}

	// 实际比较
	switch comp.Op {
	case CompEq:
		return e.compareEqual(left, right)
	case CompNe:
		return !e.compareEqual(left, right)
	case CompLt:
		return e.compareLess(left, right)
	case CompLe:
		return e.compareLess(left, right) || e.compareEqual(left, right)
	case CompGt:
		return !e.compareLess(left, right) && !e.compareEqual(left, right)
	case CompGe:
		return !e.compareLess(left, right)
	}
	return false
}

// evalComparable 评估可比较值
func (e *Evaluator) evalComparable(currentNode Result, c *Comparable) Result {
	switch c.Type {
	case ComparableLiteral:
		return e.evalLiteral(c.Literal)
	case ComparableSingularQuery:
		return e.evalSingularQuery(currentNode, c.SingularQuery)
	case ComparableFuncExpr:
		result, _ := e.evalFuncCall(currentNode, c.FuncExpr, ContextComparable)
		return result
	}
	return Result{}
}

// evalLiteral 评估字面量
func (e *Evaluator) evalLiteral(lit *LiteralValue) Result {
	switch lit.Type {
	case LiteralString:
		return Result{Type: JSONTypeString, Str: lit.Value}
	case LiteralNumber:
		num, _ := strconv.ParseFloat(lit.Value, 64)
		return Result{Type: JSONTypeNumber, Num: num, Raw: lit.Value}
	case LiteralTrue:
		return Result{Type: JSONTypeTrue}
	case LiteralFalse:
		return Result{Type: JSONTypeFalse}
	case LiteralNull:
		return Result{Type: JSONTypeNull}
	}
	return Result{}
}

// evalSingularQuery 评估单值查询
func (e *Evaluator) evalSingularQuery(currentNode Result, query *SingularQuery) Result {
	var results []Result

	if query.Relative {
		// 相对查询，从当前节点开始
		results = []Result{currentNode}
	} else {
		// 绝对查询，从根节点开始
		results = []Result{parseValue(e.json)}
	}

	// 应用每个段
	for _, seg := range query.Segments {
		var newResults []Result
		for _, r := range results {
			if seg.Type == SingularNameSegment {
				selected := e.evalNameSelector(r, seg.Name)
				newResults = append(newResults, selected...)
			} else if seg.Type == SingularIndexSegment {
				selected := e.evalIndexSelector(r, seg.Index)
				newResults = append(newResults, selected...)
			}
		}
		results = newResults
		if len(results) == 0 {
			return Result{}
		}
	}

	if len(results) == 0 {
		return Result{}
	}
	return results[0]
}

// evalTestExpr 评估测试表达式
func (e *Evaluator) evalTestExpr(currentNode Result, test *TestExpr) bool {
	if test.FilterQuery != nil {
		return e.evalFilterQueryTest(currentNode, test.FilterQuery)
	}
	if test.FuncExpr != nil {
		result, ok := e.evalFuncCall(currentNode, test.FuncExpr, ContextTest)
		if !ok {
			return false
		}
		// LogicalType 或 NodesType 结果转换为逻辑值
		// LogicalType: 存在即为 true
		// NodesType: 非空即为 true
		return e.funcResultToLogical(result)
	}
	return false
}

// evalFilterQueryTest 评估过滤查询测试（存在性测试）
func (e *Evaluator) evalFilterQueryTest(currentNode Result, fq *FilterQuery) bool {
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
			return false
		}
	}

	return len(results) > 0
}

// compareEqual 比较两个值是否相等
func (e *Evaluator) compareEqual(a, b Result) bool {
	// 类型不同则不相等
	if a.Type != b.Type {
		return false
	}

	switch a.Type {
	case JSONTypeNull:
		return true
	case JSONTypeTrue, JSONTypeFalse:
		return a.Type == b.Type
	case JSONTypeNumber:
		return a.Num == b.Num
	case JSONTypeString:
		return a.Str == b.Str
	case JSONTypeJSON:
		// 对象或数组的深度比较
		return a.Raw == b.Raw
	}
	return false
}

// compareLess 比较两个值的大小
func (e *Evaluator) compareLess(a, b Result) bool {
	// 只有数字和字符串可以比较大小
	if a.Type != b.Type {
		return false
	}

	switch a.Type {
	case JSONTypeNumber:
		return a.Num < b.Num
	case JSONTypeString:
		return a.Str < b.Str
	}
	return false
}

// funcResultToLogical 将函数结果转换为逻辑值
// 用于测试表达式中的函数调用结果
func (e *Evaluator) funcResultToLogical(result Result) bool {
	// 如果是 Nothing（不存在），返回 false
	if !result.Exists() {
		return false
	}
	// 对于 LogicalType 结果（JSONTypeTrue 或 JSONTypeFalse）
	// true 表示 LogicalTrue，false 表示 LogicalFalse
	if result.Type == JSONTypeTrue {
		return true
	}
	if result.Type == JSONTypeFalse {
		return false
	}
	// 对于其他类型（NodesType），存在即为 LogicalTrue
	return true
}
