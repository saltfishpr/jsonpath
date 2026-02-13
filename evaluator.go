package jsonpath

import (
	"strconv"
)

// Evaluator evaluates JSONPath expressions against JSON data
type Evaluator struct {
	json  string
	query *Query
}

// NewEvaluator creates a new evaluator for the given JSON and query
func NewEvaluator(json string, query *Query) *Evaluator {
	return &Evaluator{
		json:  json,
		query: query,
	}
}

// Evaluate executes the query and returns all matching results
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
		for _, result := range input {
			descendants := e.evalDescendant(result, segment.Selectors)
			output = append(output, descendants...)
		}
	} else {
		for _, result := range input {
			for _, selector := range segment.Selectors {
				selected := e.evaluateSelector(result, selector)
				output = append(output, selected...)
			}
		}
	}

	return output
}

// evalDescendant evaluates descendant segments (recursive)
func (e *Evaluator) evalDescendant(result Result, selectors []*Selector) []Result {
	var results []Result
	e.collectDescendants(result, selectors, &results)

	return results
}

// collectDescendants recursively collects descendant nodes
func (e *Evaluator) collectDescendants(result Result, selectors []*Selector, results *[]Result) {
	for _, selector := range selectors {
		selected := e.evaluateSelector(result, selector)
		*results = append(*results, selected...)
	}

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

func (e *Evaluator) evalIndexSelector(result Result, index int) []Result {
	if !result.IsArray() {
		return nil
	}
	arr := result.Array()
	length := len(arr)

	// Handle negative indices
	if index < 0 {
		index = length + index
	}

	// Out of bounds returns empty (RFC 9535)
	if index < 0 || index >= length {
		return nil
	}

	return []Result{arr[index]}
}

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
		return nil // RFC 9535: step=0 returns empty
	}

	start, end, endIsDefault := e.normalizeSliceBounds(slice.Start, slice.End, step, arrLen)

	var results []Result
	if step > 0 {
		for i := start; i < end; i += step {
			if i >= 0 && i < arrLen {
				results = append(results, arr[i])
			}
		}
	} else {
		if endIsDefault {
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

// normalizeSliceBounds normalizes slice bounds
// Returns (start, end, endIsDefault)
func (e *Evaluator) normalizeSliceBounds(start, end *int, step, arrLen int) (int, int, bool) {
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
		en = -1 // Mark as default end, needs special handling
		endIsDefault = true
	}

	s = clamp(s, 0, arrLen-1)
	if !endIsDefault {
		en = clamp(en, 0, arrLen)
	}

	return s, en, endIsDefault
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

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

func (e *Evaluator) evalComparison(currentNode Result, comp *Comparison) bool {
	left := e.evalComparable(currentNode, comp.Left)
	right := e.evalComparable(currentNode, comp.Right)

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

func (e *Evaluator) evalComparable(currentNode Result, c *Comparable) Result {
	switch c.Type {
	case ComparableLiteral:
		return e.evalLiteral(c.Literal)
	case ComparableSingularQuery:
		return e.evalSingularQuery(currentNode, c.SingularQuery)
	case ComparableFuncExpr:
		result, err := e.evalFuncCall(currentNode, c.FuncExpr, FunctionValueTypeValue)
		if err != nil {
			return Result{}
		}
		return result.(Result)
	}
	return Result{}
}

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

func (e *Evaluator) evalSingularQuery(currentNode Result, query *SingularQuery) Result {
	results := e.evalQuerySegments(currentNode, query.Relative, query.Segments)
	if len(results) == 0 {
		return Result{}
	}
	return results[0]
}

// evalQuerySegments evaluates segments (shared logic for singular/filter queries)
func (e *Evaluator) evalQuerySegments(currentNode Result, relative bool, segments []*SingularSegment) []Result {
	var results []Result
	if relative {
		results = []Result{currentNode}
	} else {
		results = []Result{parseValue(e.json)}
	}

	for _, seg := range segments {
		var newResults []Result
		for _, r := range results {
			switch seg.Type {
			case SingularNameSegment:
				selected := e.evalNameSelector(r, seg.Name)
				newResults = append(newResults, selected...)
			case SingularIndexSegment:
				selected := e.evalIndexSelector(r, seg.Index)
				newResults = append(newResults, selected...)
			}
		}
		results = newResults
		if len(results) == 0 {
			return nil
		}
	}
	return results
}

func (e *Evaluator) evalTestExpr(currentNode Result, test *TestExpr) bool {
	if test.FilterQuery != nil {
		return e.evalFilterQueryTest(currentNode, test.FilterQuery)
	}
	if test.FuncExpr != nil {
		result, err := e.evalFuncCall(currentNode, test.FuncExpr, FunctionValueTypeLogical)
		if err != nil {
			return false
		}
		if logical, ok := result.(bool); ok {
			return logical
		}
		if nodes, ok := result.([]Result); ok {
			return len(nodes) > 0
		}
		return false
	}
	return false
}

func (e *Evaluator) evalFilterQueryTest(currentNode Result, fq *FilterQuery) bool {
	results := e.evalFilterQuery(currentNode, fq)
	return len(results) > 0
}

func (e *Evaluator) evalFilterQuery(currentNode Result, fq *FilterQuery) []Result {
	var results []Result
	if fq.Relative {
		results = []Result{currentNode}
	} else {
		results = []Result{parseValue(e.json)}
	}

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
			return nil
		}
	}
	return results
}

func (e *Evaluator) compareEqual(a, b Result) bool {
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
		return a.Raw == b.Raw
	}
	return false
}

func (e *Evaluator) compareLess(a, b Result) bool {
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
