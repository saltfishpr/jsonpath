package jsonpath

import (
	"errors"
	"fmt"

	"github.com/saltfishpr/jsonpath/internal/ast"
	"github.com/saltfishpr/jsonpath/internal/document"
	"github.com/saltfishpr/jsonpath/internal/iregexp"
)

// exprType describes function argument and result types, not JSON value kinds.
type exprType uint8

const (
	exprTypeInvalid exprType = iota
	exprTypeValue
	exprTypeLogical
	exprTypeNodes
)

// String returns the RFC 9535 type name, or "invalid" for an unknown type.
func (t exprType) String() string {
	switch t {
	case exprTypeValue:
		return "ValueType"
	case exprTypeLogical:
		return "LogicalType"
	case exprTypeNodes:
		return "NodesType"
	}
	return "invalid"
}

// Semantic annotations live outside the AST so validation can fail without
// leaving a partially rewritten syntax tree.
type compiler struct {
	functions map[*ast.FunctionCall]compiledFunctionCall
	literals  map[*ast.Literal]*document.Node
	regexes   map[string]*iregexp.Regexp
}

type compiledFunctionCall struct {
	definition functionDefinition
	arguments  []compiledArgument
}

// The parameter type selects the active field: value for ValueType, expr for
// LogicalType or NodesType. A derived singular query belongs here so the
// original argument retains its parsed form.
type compiledArgument struct {
	value ast.Comparable
	expr  ast.Expr
}

func (c *compiler) walkQuery(q *ast.Query) error {
	for _, seg := range q.Segments {
		for _, sel := range seg.Selectors {
			switch s := sel.(type) {
			case *ast.FilterSelector:
				if err := c.walkExpr(s.Expr); err != nil {
					return err
				}
				if c.expressionType(s.Expr) != exprTypeLogical {
					return &CompileError{Offset: s.Pos.Offset, Message: "filter expression must be LogicalType"}
				}
			case *ast.SliceSelector:
				if s.Step != nil && *s.Step == 0 {
					// A zero step is valid under RFC 9535 section 2.3.4 and selects no nodes.
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
		// Resolve operands first so function diagnostics precede comparison type errors.
		if c.expressionType(v.Left) != exprTypeValue {
			return &CompileError{Offset: v.Pos.Offset, Message: "left operand of comparison must have ValueType"}
		}
		if c.expressionType(v.Right) != exprTypeValue {
			return &CompileError{Offset: v.Pos.Offset, Message: "right operand of comparison must have ValueType"}
		}
	case *ast.TestExpr:
		if err := c.walkExpr(v.Target); err != nil {
			return err
		}
		if c.expressionType(v.Target) == exprTypeValue {
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
	function := compiledFunctionCall{definition: def, arguments: make([]compiledArgument, len(f.Args))}
	for i, arg := range f.Args {
		if err := c.walkExpr(arg); err != nil {
			return err
		}
		var err error
		function.arguments[i], err = c.compileArg(arg, def.params[i], f.Name, i)
		if err != nil {
			return err
		}
	}
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
					// Invalid I-Regexp syntax makes match/search false under RFC 9535.
					// Cache nil so evaluation preserves that result without recompiling.
				}
				c.regexes[pattern.Value.Str] = r
			}
		}
	}
	if c.functions == nil {
		c.functions = make(map[*ast.FunctionCall]compiledFunctionCall)
	}
	c.functions[f] = function
	return nil
}

// A query can supply ValueType only when its syntax guarantees at most one
// node. NodesType can supply LogicalType through an existence test; JSON
// values themselves have no implicit truth conversion.
func (c *compiler) compileArg(arg ast.Expr, want exprType, fnName string, idx int) (compiledArgument, error) {
	switch want {
	case exprTypeValue:
		switch v := arg.(type) {
		case *ast.Literal:
			return compiledArgument{value: v}, nil
		case *ast.SingularQuery:
			return compiledArgument{value: v}, nil
		case *ast.FunctionCall:
			if got := c.expressionType(v); got != exprTypeValue {
				return compiledArgument{}, argTypeError(fnName, idx, want, got, v.Pos)
			}
			return compiledArgument{value: v}, nil
		case *ast.FilterQuery:
			segs, ok := toSingularSegments(v.Query.Segments)
			if !ok {
				return compiledArgument{}, &CompileError{
					Offset:  v.Pos.Offset,
					Message: fmt.Sprintf("function %q arg %d: expected ValueType or singular query", fnName, idx+1),
				}
			}
			return compiledArgument{
				value: &ast.SingularQuery{
					IsRoot:   v.Query.IsRoot,
					Segments: segs,
					Pos:      v.Pos,
				},
			}, nil
		}
	case exprTypeLogical:
		if _, ok := arg.(ast.LogicalExpr); ok {
			return compiledArgument{expr: arg}, nil
		}
		switch v := arg.(type) {
		case *ast.FilterQuery:
			return compiledArgument{expr: arg}, nil
		case *ast.FunctionCall:
			if got := c.expressionType(v); got == exprTypeLogical || got == exprTypeNodes {
				return compiledArgument{expr: arg}, nil
			}
		}
	case exprTypeNodes:
		switch v := arg.(type) {
		case *ast.FilterQuery, *ast.SingularQuery:
			return compiledArgument{expr: arg}, nil
		case *ast.FunctionCall:
			if c.expressionType(v) == exprTypeNodes {
				return compiledArgument{expr: arg}, nil
			}
		}
	}
	return compiledArgument{}, argTypeError(fnName, idx, want, c.expressionType(arg), arg.Position())
}

// Function calls must already have been resolved by walkExpr before their
// result types can be used by the enclosing expression.
func (c *compiler) expressionType(expr ast.Expr) exprType {
	switch v := expr.(type) {
	case ast.LogicalExpr:
		return exprTypeLogical
	case *ast.Literal, *ast.SingularQuery:
		return exprTypeValue
	case *ast.FilterQuery:
		return exprTypeNodes
	case *ast.FunctionCall:
		return c.functions[v].definition.result
	default:
		panic(fmt.Sprintf("jsonpath: unexpected expression %T", expr))
	}
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

func argTypeError(fnName string, idx int, want, got exprType, pos ast.Pos) error {
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
