package jsonpath

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
		for _, value := range result.Map() {
			e.collectDescendants(value, selectors, results)
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
