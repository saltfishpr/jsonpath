package jsonpath

import (
	"errors"
	"fmt"

	"github.com/saltfishpr/jsonpath/internal/ast"
	"github.com/saltfishpr/jsonpath/internal/document"
	"github.com/saltfishpr/jsonpath/internal/iregexp"
)

type compiler struct {
	literals map[*ast.Literal]*document.Node
	regexes  map[string]*iregexp.Regexp
}

func (c *compiler) walkQuery(q *ast.Query) error {
	for _, seg := range q.Segments {
		for _, sel := range seg.Selectors {
			switch s := sel.(type) {
			case *ast.FilterSelector:
				if err := c.walkExpr(s.Expr); err != nil {
					return err
				}
				if s.Expr.Type() != ast.TypeLogical {
					return &CompileError{Offset: s.Pos.Offset, Message: "filter expression must be LogicalType"}
				}
			case *ast.SliceSelector:
				if s.Step != nil && *s.Step == 0 {
					// RFC 9535 §2.3.4：step=0 的 slice 语法合法，求值时返回空，此处无需报错。
				}
			}
		}
	}
	return nil
}

func (c *compiler) walkExpr(e ast.Expr) error {
	switch v := e.(type) {
	case *ast.OrExpr:
		for _, op := range v.Operands {
			if err := c.walkExpr(op); err != nil {
				return err
			}
		}
	case *ast.AndExpr:
		for _, op := range v.Operands {
			if err := c.walkExpr(op); err != nil {
				return err
			}
		}
	case *ast.NotExpr:
		return c.walkExpr(v.Operand)
	case *ast.ParenExpr:
		return c.walkExpr(v.Inner)
	case *ast.ComparisonExpr:
		if err := c.walkExpr(v.Left); err != nil {
			return err
		}
		if err := c.walkExpr(v.Right); err != nil {
			return err
		}
		// 函数返回类型要等到签名解析后才知道，所以比较操作数的类型检查放在子节点走完之后。
		if v.Left.Type() != ast.TypeValue {
			return &CompileError{Offset: v.Pos.Offset, Message: "left operand of comparison must have ValueType"}
		}
		if v.Right.Type() != ast.TypeValue {
			return &CompileError{Offset: v.Pos.Offset, Message: "right operand of comparison must have ValueType"}
		}
	case *ast.TestExpr:
		if err := c.walkExpr(v.Target); err != nil {
			return err
		}
		if v.Target.Type() == ast.TypeValue {
			return &CompileError{Offset: v.Pos.Offset, Message: "test expression requires LogicalType or NodesType, got ValueType"}
		}
	case *ast.FilterQuery:
		return c.walkQuery(v.Query)
	case *ast.SingularQuery:
	case *ast.Literal:
		if c.literals == nil {
			c.literals = make(map[*ast.Literal]*document.Node)
		}
		c.literals[v] = literalValue(v)
	case *ast.FunctionCall:
		return c.resolveFunction(v)
	}
	return nil
}

func (c *compiler) resolveFunction(f *ast.FunctionCall) error {
	def, ok := builtinFunctions[f.Name]
	if !ok {
		return &CompileError{Offset: f.Pos.Offset, Message: fmt.Sprintf("unknown function %q", f.Name)}
	}
	if len(f.Args) != len(def.params) {
		return &CompileError{Offset: f.Pos.Offset, Message: fmt.Sprintf("function %q: expected %d arguments, got %d", f.Name, len(def.params), len(f.Args))}
	}
	for i, arg := range f.Args {
		if err := c.walkExpr(arg); err != nil {
			return err
		}
		adapted, err := adaptArg(arg, def.params[i], f.Name, i)
		if err != nil {
			return err
		}
		f.Args[i] = adapted
	}
	f.Ret = def.result
	if f.Name == "match" || f.Name == "search" {
		if pattern, ok := f.Args[1].(*ast.Literal); ok && pattern.Value.Kind == ast.LitString {
			if c.regexes == nil {
				c.regexes = make(map[string]*iregexp.Regexp)
			}

			if _, cached := c.regexes[pattern.Value.Str]; !cached {
				r, err := iregexp.Compile(pattern.Value.Str)
				if err != nil {
					var syntaxErr *iregexp.SyntaxError
					if !errors.As(err, &syntaxErr) {
						return err
					}
					// RFC 9535 §2.4.6、§2.4.7 规定：模式不符合 I-Regexp 时，match/search
					// 的求值结果应为 LogicalFalse，因此此处不将无效模式作为查询编译错误返回。
				}
				c.regexes[pattern.Value.Str] = r
			}
		}
	}
	return nil
}

// adaptArg 按 RFC 9535 §2.4.3 将实参调整为形参期望的类型：singular query 可以以
// ValueType 传入，NodesType 按节点列表是否非空隐式转为 LogicalType（在求值时
// 完成）。转换不成立时返回 CompileError。
func adaptArg(arg ast.Expr, want ast.ExprType, fnName string, idx int) (ast.Expr, error) {
	switch want {
	case ast.TypeValue:
		switch v := arg.(type) {
		case *ast.Literal, *ast.SingularQuery:
			return arg, nil
		case *ast.FunctionCall:
			if v.Ret != ast.TypeValue {
				return nil, argTypeError(fnName, idx, want, v.Ret, v.Pos)
			}
			return arg, nil
		case *ast.FilterQuery:
			segs, ok := toSingularSegments(v.Query.Segments)
			if !ok {
				return nil, &CompileError{
					Offset:  v.Pos.Offset,
					Message: fmt.Sprintf("function %q arg %d: expected ValueType or singular query", fnName, idx+1),
				}
			}
			return &ast.SingularQuery{
				IsRoot:   v.Query.IsRoot,
				Segments: segs,
				Pos:      v.Pos,
			}, nil
		}
	case ast.TypeLogical:
		if _, ok := arg.(ast.LogicalExpr); ok {
			return arg, nil
		}
		switch v := arg.(type) {
		case *ast.FilterQuery:
			return arg, nil
		case *ast.FunctionCall:
			if v.Ret == ast.TypeLogical || v.Ret == ast.TypeNodes {
				return arg, nil
			}
		}
	case ast.TypeNodes:
		switch v := arg.(type) {
		case *ast.FilterQuery, *ast.SingularQuery:
			return arg, nil
		case *ast.FunctionCall:
			if v.Ret == ast.TypeNodes {
				return arg, nil
			}
		}
	}
	return nil, argTypeError(fnName, idx, want, arg.Type(), arg.Position())
}

func argTypeError(fnName string, idx int, want, got ast.ExprType, pos ast.Pos) error {
	return &CompileError{
		Offset:  pos.Offset,
		Message: fmt.Sprintf("function %q arg %d: expected %s, got %s", fnName, idx+1, want, got),
	}
}

func toSingularSegments(segs []ast.Segment) ([]ast.SingularSegment, bool) {
	out := make([]ast.SingularSegment, 0, len(segs))
	for _, seg := range segs {
		if seg.Kind != ast.SegChild || len(seg.Selectors) != 1 {
			return nil, false
		}
		switch sel := seg.Selectors[0].(type) {
		case *ast.NameSelector:
			name := sel.Name
			out = append(out, ast.SingularSegment{Name: &name, Pos: sel.Pos})
		case *ast.IndexSelector:
			idx := sel.Index
			out = append(out, ast.SingularSegment{Index: &idx, Pos: sel.Pos})
		default:
			return nil, false
		}
	}
	return out, true
}
