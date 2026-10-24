package jsonpath

import (
	"fmt"

	"github.com/saltfishpr/jsonpath/internal/ast"
	"github.com/saltfishpr/jsonpath/internal/document"
)

func (e *evaluator) evalLogical(expr ast.Expr, cur *document.Node) (bool, error) {
	switch v := expr.(type) {
	case *ast.OrExpr:
		for _, op := range v.Operands {
			ok, err := e.evalLogical(op, cur)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	case *ast.AndExpr:
		for _, op := range v.Operands {
			ok, err := e.evalLogical(op, cur)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}
		return true, nil
	case *ast.NotExpr:
		ok, err := e.evalLogical(v.Operand, cur)
		if err != nil {
			return false, err
		}
		return !ok, nil
	case *ast.ParenExpr:
		return e.evalLogical(v.Inner, cur)
	case *ast.ComparisonExpr:
		return e.evalComparison(v, cur)
	case *ast.TestExpr:
		return e.evalTest(v, cur)
	case *ast.FilterQuery:
		// LogicalType 形参位置收到子查询时，按 RFC 9535 §2.4.3 转为存在性判定。
		nodes, err := e.query(v.Query, cur)
		if err != nil {
			return false, err
		}
		return len(nodes) > 0, nil
	case *ast.FunctionCall:
		r, err := e.callFunction(v, cur)
		if err != nil {
			return false, err
		}
		switch v.Ret {
		case ast.TypeLogical:
			return r.logical, nil
		case ast.TypeNodes:
			return len(r.nodes) > 0, nil
		}
	}
	return false, nil
}

func (e *evaluator) evalTest(t *ast.TestExpr, cur *document.Node) (bool, error) {
	ok, err := e.evalLogical(t.Target, cur)
	if err != nil {
		return false, err
	}
	if t.Negated {
		ok = !ok
	}
	return ok, nil
}

// 第二个返回值为 true 时表示 Nothing，用以与 JSON null 区分。
func (e *evaluator) evalComparable(c ast.Comparable, cur *document.Node) (*document.Node, bool, error) {
	switch v := c.(type) {
	case *ast.Literal:
		return e.literals[v], false, nil
	case *ast.SingularQuery:
		val, nothing := e.evalSingularQuery(v, cur)
		return val, nothing, nil
	case *ast.FunctionCall:
		r, err := e.callFunction(v, cur)
		if err != nil {
			return nil, false, err
		}
		if r.nothing {
			return nil, true, nil
		}
		return r.value, false, nil
	}
	return nil, true, nil
}

func literalValue(l *ast.Literal) *document.Node {
	switch l.Value.Kind {
	case ast.LitNull:
		return document.NewNull()
	case ast.LitBool:
		if l.Value.Bool {
			return document.NewTrue()
		}
		return document.NewFalse()
	case ast.LitInt, ast.LitFloat:
		return document.NewNumber(l.Value.Raw)
	case ast.LitString:
		return document.NewString(l.Value.Str)
	}
	return document.NewNull()
}

func (e *evaluator) evalSingularQuery(q *ast.SingularQuery, cur *document.Node) (*document.Node, bool) {
	v := cur
	if q.IsRoot {
		v = e.root
	}
	for _, s := range q.Segments {
		if v == nil {
			return nil, true
		}
		switch {
		case s.Name != nil:
			v = v.Get(*s.Name)
		case s.Index != nil:
			if v.Kind() != document.KindArray {
				return nil, true
			}
			idx := *s.Index
			if idx < 0 {
				idx += int64(v.Len())
			}
			if idx < 0 || idx >= int64(v.Len()) {
				return nil, true
			}
			v = v.Array()[idx]
		}
	}
	return v, v == nil
}

func (e *evaluator) evalNodes(expr ast.Expr, cur *document.Node) ([]*document.Node, error) {
	switch v := expr.(type) {
	case *ast.FilterQuery:
		return e.query(v.Query, cur)
	case *ast.SingularQuery:
		if val, nothing := e.evalSingularQuery(v, cur); !nothing {
			return []*document.Node{val}, nil
		}
		return nil, nil
	case *ast.FunctionCall:
		r, err := e.callFunction(v, cur)
		if err != nil {
			return nil, err
		}
		if v.Ret == ast.TypeNodes {
			return r.nodes, nil
		}
	}
	return nil, nil
}

func (e *evaluator) evalComparison(c *ast.ComparisonExpr, cur *document.Node) (bool, error) {
	lv, lNothing, err := e.evalComparable(c.Left, cur)
	if err != nil {
		return false, err
	}
	rv, rNothing, err := e.evalComparable(c.Right, cur)
	if err != nil {
		return false, err
	}
	switch c.Op {
	case ast.CmpEq:
		return compareEq(lv, lNothing, rv, rNothing), nil
	case ast.CmpNe:
		return !compareEq(lv, lNothing, rv, rNothing), nil
	case ast.CmpLt:
		return compareLt(lv, lNothing, rv, rNothing), nil
	case ast.CmpLe:
		return compareLt(lv, lNothing, rv, rNothing) || compareEq(lv, lNothing, rv, rNothing), nil
	case ast.CmpGt:
		return compareLt(rv, rNothing, lv, lNothing), nil
	case ast.CmpGe:
		return compareLt(rv, rNothing, lv, lNothing) || compareEq(lv, lNothing, rv, rNothing), nil
	}
	return false, nil
}

func compareEq(a *document.Node, aNothing bool, b *document.Node, bNothing bool) bool {
	if aNothing || bNothing {
		return aNothing && bNothing
	}
	if a == nil || b == nil {
		return a == b
	}
	if a.Kind() != b.Kind() {
		return false
	}
	switch a.Kind() {
	case document.KindNull:
		return true
	case document.KindBool:
		return a.Bool() == b.Bool()
	case document.KindString:
		return a.String() == b.String()
	case document.KindNumber:
		return a.Number().Cmp(b.Number()) == 0
	case document.KindArray:
		if a.Len() != b.Len() {
			return false
		}
		for i, child := range a.Array() {
			if !compareEq(child, false, b.Array()[i], false) {
				return false
			}
		}
		return true
	case document.KindObject:
		if a.Len() != b.Len() {
			return false
		}
		for _, member := range a.Members() {
			right := b.Get(member.Name)
			if right == nil || !compareEq(member.Value, false, right, false) {
				return false
			}
		}
		return true
	}
	return false
}

func compareLt(a *document.Node, aNothing bool, b *document.Node, bNothing bool) bool {
	if aNothing || bNothing {
		return false
	}
	if a == nil || b == nil {
		return false
	}
	if a.Kind() != b.Kind() {
		return false
	}
	switch a.Kind() {
	case document.KindNumber:
		return a.Number().Cmp(b.Number()) < 0
	case document.KindString:
		return a.String() < b.String()
	}
	return false
}

func (e *evaluator) callFunction(f *ast.FunctionCall, cur *document.Node) (functionValue, error) {
	def := builtinFunctions[f.Name]
	args := make([]functionValue, len(f.Args))
	for i, arg := range f.Args {
		var err error
		args[i], err = e.prepareArg(arg, def.params[i], cur)
		if err != nil {
			return functionValue{}, err
		}
	}
	return def.call(e, args)
}

func (e *evaluator) prepareArg(expr ast.Expr, want ast.ExprType, cur *document.Node) (functionValue, error) {
	switch want {
	case ast.TypeValue:
		// 编译阶段的 adaptArg 已检查类型并转换 singular query，保证此处实现了 Comparable。
		v, nothing, err := e.evalComparable(expr.(ast.Comparable), cur)
		if err != nil {
			return functionValue{}, err
		}
		return functionValue{value: v, nothing: nothing}, nil
	case ast.TypeLogical:
		logical, err := e.evalLogical(expr, cur)
		if err != nil {
			return functionValue{}, err
		}
		return functionValue{logical: logical}, nil
	case ast.TypeNodes:
		nodes, err := e.evalNodes(expr, cur)
		if err != nil {
			return functionValue{}, err
		}
		return functionValue{nodes: nodes}, nil
	default:
		// 内置函数签名只允许上述三种类型；其他类型表示内部定义违反约束。
		panic(fmt.Sprintf("jsonpath: invalid function parameter type %v", want))
	}
}
