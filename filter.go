package jsonpath

import (
	"fmt"

	"github.com/saltfishpr/jsonpath/internal/ast"
	"github.com/saltfishpr/jsonpath/internal/document"
)

func (e *evaluator) evalLogical(expr ast.Expr, cur *document.Node) bool {
	switch v := expr.(type) {
	case *ast.OrExpr:
		for _, op := range v.Operands {
			if e.evalLogical(op, cur) {
				return true
			}
		}
		return false
	case *ast.AndExpr:
		for _, op := range v.Operands {
			if !e.evalLogical(op, cur) {
				return false
			}
		}
		return true
	case *ast.NotExpr:
		return !e.evalLogical(v.Operand, cur)
	case *ast.ParenExpr:
		return e.evalLogical(v.Inner, cur)
	case *ast.ComparisonExpr:
		return e.evalComparison(v, cur)
	case *ast.TestExpr:
		return e.evalTest(v, cur)
	case *ast.FilterQuery:
		// LogicalType 形参位置收到子查询时，按 RFC 9535 §2.4.3 转为存在性判定。
		return len(e.query(v.Query, cur)) > 0
	case *ast.FunctionCall:
		r := e.callFunction(v, cur)
		switch v.Ret {
		case ast.TypeLogical:
			return r.logical
		case ast.TypeNodes:
			return len(r.nodes) > 0
		}
	}
	return false
}

func (e *evaluator) evalTest(t *ast.TestExpr, cur *document.Node) bool {
	ok := e.evalLogical(t.Target, cur)
	if t.Negated {
		ok = !ok
	}
	return ok
}

// evalComparable 求值一个 Comparable。第二个返回值为 true 时表示 Nothing
// （查询无结果或函数返回 Nothing），用以与 JSON null 区分。
func (e *evaluator) evalComparable(c ast.Comparable, cur *document.Node) (document.Value, bool) {
	switch v := c.(type) {
	case *ast.Literal:
		return e.literals[v], false
	case *ast.SingularQuery:
		return e.evalSingularQuery(v, cur)
	case *ast.FunctionCall:
		r := e.callFunction(v, cur)
		if r.nothing {
			return nil, true
		}
		return r.value, false
	}
	return nil, true
}

func literalValue(l *ast.Literal) document.Value {
	switch l.Value.Kind {
	case ast.LitNull:
		return document.Null{}
	case ast.LitBool:
		return document.Bool{V: l.Value.Bool}
	case ast.LitInt, ast.LitFloat:
		return document.ParseNumber([]byte(l.Value.Raw))
	case ast.LitString:
		return document.String{V: l.Value.Str}
	}
	return document.Null{}
}

func (e *evaluator) evalSingularQuery(q *ast.SingularQuery, cur *document.Node) (document.Value, bool) {
	var v document.Value
	if q.IsRoot {
		v = e.root.Value
	} else {
		v = cur.Value
	}
	for _, s := range q.Segments {
		if v == nil {
			return nil, true
		}
		switch {
		case s.Name != nil:
			obj, ok := v.(document.Object)
			if !ok {
				return nil, true
			}
			val, ok := obj.Get(*s.Name)
			if !ok {
				return nil, true
			}
			v = val
		case s.Index != nil:
			arr, ok := v.(document.Array)
			if !ok {
				return nil, true
			}
			idx := *s.Index
			if idx < 0 {
				idx += int64(arr.Len())
			}
			if idx < 0 || idx >= int64(arr.Len()) {
				return nil, true
			}
			v = arr.Items[idx]
		}
	}
	return v, false
}

func (e *evaluator) evalNodes(expr ast.Expr, cur *document.Node) []document.Value {
	switch v := expr.(type) {
	case *ast.FilterQuery:
		nodes := e.query(v.Query, cur)
		out := make([]document.Value, len(nodes))
		for i, n := range nodes {
			out[i] = n.Value
		}
		return out
	case *ast.SingularQuery:
		if val, nothing := e.evalSingularQuery(v, cur); !nothing {
			return []document.Value{val}
		}
		return nil
	case *ast.FunctionCall:
		r := e.callFunction(v, cur)
		if v.Ret == ast.TypeNodes {
			return r.nodes
		}
	}
	return nil
}

func (e *evaluator) evalComparison(c *ast.ComparisonExpr, cur *document.Node) bool {
	lv, lNothing := e.evalComparable(c.Left, cur)
	rv, rNothing := e.evalComparable(c.Right, cur)
	switch c.Op {
	case ast.CmpEq:
		return compareEq(lv, lNothing, rv, rNothing)
	case ast.CmpNe:
		return !compareEq(lv, lNothing, rv, rNothing)
	case ast.CmpLt:
		return compareLt(lv, lNothing, rv, rNothing)
	case ast.CmpLe:
		return compareLt(lv, lNothing, rv, rNothing) || compareEq(lv, lNothing, rv, rNothing)
	case ast.CmpGt:
		return compareLt(rv, rNothing, lv, lNothing)
	case ast.CmpGe:
		return compareLt(rv, rNothing, lv, lNothing) || compareEq(lv, lNothing, rv, rNothing)
	}
	return false
}

// compareEq 实现 RFC 9535 §2.3.5.2.2 的 == 判定。
func compareEq(a document.Value, aNothing bool, b document.Value, bNothing bool) bool {
	if aNothing || bNothing {
		return aNothing && bNothing
	}
	if a == nil && b == nil {
		return true
	}
	if a.Kind() != b.Kind() {
		return false
	}
	switch av := a.(type) {
	case document.Null:
		return true
	case document.Bool:
		return av.V == b.(document.Bool).V
	case document.String:
		return av.V == b.(document.String).V
	case document.Number:
		return av.Cmp(b.(document.Number)) == 0
	case document.Array:
		bv := b.(document.Array)
		if av.Len() != bv.Len() {
			return false
		}
		for i := range av.Items {
			if !compareEq(av.Items[i], false, bv.Items[i], false) {
				return false
			}
		}
		return true
	case document.Object:
		bv := b.(document.Object)
		// 按 RFC 9535 §2.3.5.2.2，对象比较是成员集合的集合比较；重复键不影响集合大小。
		// Get 对重复键返回最后一次出现的值，此处用 index 映射保持一致语义。
		if av.MemberCount() != bv.MemberCount() {
			return false
		}
		for _, k := range av.Keys {
			i, _ := av.MemberIndex(k)
			j, ok := bv.MemberIndex(k)
			if !ok {
				return false
			}
			if !compareEq(av.Values[i], false, bv.Values[j], false) {
				return false
			}
		}
		return true
	}
	return false
}

// compareLt 实现 RFC 9535 §2.3.5.2.3 的 < 判定。
func compareLt(a document.Value, aNothing bool, b document.Value, bNothing bool) bool {
	if aNothing || bNothing {
		return false
	}
	if a == nil || b == nil {
		return false
	}
	if a.Kind() != b.Kind() {
		return false
	}
	switch av := a.(type) {
	case document.Number:
		return av.Cmp(b.(document.Number)) < 0
	case document.String:
		return av.V < b.(document.String).V
	}
	return false
}

func (e *evaluator) callFunction(f *ast.FunctionCall, cur *document.Node) functionValue {
	def := builtinFunctions[f.Name]
	args := make([]functionValue, len(f.Args))
	for i, arg := range f.Args {
		args[i] = e.prepareArg(arg, def.params[i], cur)
	}
	return def.call(e, args)
}

func (e *evaluator) prepareArg(expr ast.Expr, want ast.ExprType, cur *document.Node) functionValue {
	switch want {
	case ast.TypeValue:
		// 编译阶段的 adaptArg 已检查类型并转换 singular query，保证此处实现了 Comparable。
		v, nothing := e.evalComparable(expr.(ast.Comparable), cur)
		return functionValue{value: v, nothing: nothing}
	case ast.TypeLogical:
		return functionValue{logical: e.evalLogical(expr, cur)}
	case ast.TypeNodes:
		return functionValue{nodes: e.evalNodes(expr, cur)}
	default:
		// 内置函数签名只允许上述三种类型；其他类型表示内部定义违反约束。
		panic(fmt.Sprintf("jsonpath: invalid function parameter type %v", want))
	}
}
