// Package jsonpath evaluates RFC 9535 JSONPath expressions against JSON documents.
// Use [QueryValues] or [QueryNodes] for a single evaluation, or reuse a [Query] and
// [Document] to avoid compiling expressions and parsing documents repeatedly.
//
// Documents and result values retain the input bytes. Callers must keep those
// bytes unchanged while using a document or its results, including through any
// returned [json.RawMessage].
package jsonpath

import (
	"encoding/json"
	"errors"

	"github.com/saltfishpr/jsonpath/internal/ast"
	"github.com/saltfishpr/jsonpath/internal/document"
	"github.com/saltfishpr/jsonpath/internal/iregexp"
	"github.com/saltfishpr/jsonpath/internal/parse"
)

// Query is a compiled expression that supports concurrent evaluation against
// one or more documents. Create it with [Compile]; its zero value is not usable.
type Query struct {
	src       string
	query     *ast.Query
	functions map[*ast.FunctionCall]compiledFunctionCall
	literals  map[*ast.Literal]*document.Node
	regexes   map[string]*iregexp.Regexp
}

// Compile parses src and checks function signatures and expression types.
// Invalid JSONPath expressions return a *[CompileError] and a nil query.
// Failures to compile regular expressions other than invalid pattern syntax
// are returned as underlying errors.
func Compile(src string) (*Query, error) {
	q, err := parse.Parse(src)
	if err != nil {
		var parseErr *parse.Error
		if errors.As(err, &parseErr) {
			return nil, &CompileError{Offset: parseErr.Pos.Offset, Message: parseErr.Msg}
		}
		return nil, err
	}
	c := &compiler{}
	if err := c.walkQuery(q); err != nil {
		return nil, err
	}
	return &Query{
		src:       src,
		query:     q,
		functions: c.functions,
		literals:  c.literals,
		regexes:   c.regexes,
	}, nil
}

// Values evaluates q against doc and returns matching JSON fragments in selection
// order, preserving duplicates. No matches produce a non-nil empty slice; an
// evaluation error produces a nil slice. Each fragment aliases the document's
// input bytes and must not be modified.
// Both q and doc must be non-nil values created by [Compile] and [ParseDocument].
func (q *Query) Values(doc *Document) ([]json.RawMessage, error) {
	nodes, err := q.run(doc)
	if err != nil {
		return nil, err
	}
	out := make([]json.RawMessage, len(nodes))
	for i, n := range nodes {
		out[i] = json.RawMessage(n.Raw())
	}
	return out, nil
}

// Nodes evaluates q against doc and returns matches with RFC 9535 Normalized
// Paths in selection order, preserving duplicates. No matches produce a non-nil
// empty slice; an evaluation error produces a nil slice. Result values alias
// the document's input bytes and must not be modified.
// Both q and doc must be non-nil values created by [Compile] and [ParseDocument].
func (q *Query) Nodes(doc *Document) ([]Node, error) {
	nodes, err := q.run(doc)
	if err != nil {
		return nil, err
	}
	out := make([]Node, len(nodes))
	for i, n := range nodes {
		out[i] = Node{Value: json.RawMessage(n.Raw()), Path: normalizedPath(n)}
	}
	return out, nil
}

// Document is a parsed JSON value that can be queried concurrently while its
// input bytes remain unchanged. Create it with [ParseDocument]; its zero value
// is not usable.
type Document struct {
	root *document.Node
	src  []byte
}

// ParseDocument accepts exactly one JSON value with optional surrounding JSON
// whitespace. It rejects a BOM, invalid UTF-8, unpaired surrogate escapes,
// duplicate member names, and nesting beyond the parser's depth limit.
// Invalid input returns a *[DocumentError] and a nil document.
// The document retains data; callers must not modify it while the document
// or any result values are in use.
func ParseDocument(data []byte) (*Document, error) {
	root, err := document.Parse(data)
	if err != nil {
		var documentErr *document.SyntaxError
		if errors.As(err, &documentErr) {
			return nil, &DocumentError{Offset: documentErr.Offset, Message: documentErr.Message}
		}
		return nil, err
	}
	return &Document{root: root, src: data}, nil
}

// QueryValues compiles expr, parses data, and returns matches as described by
// [Query.Values]. Compilation errors take precedence over document errors.
// On any error it returns a nil slice. Returned fragments alias data and must
// not be modified.
func QueryValues(expr string, data []byte) ([]json.RawMessage, error) {
	query, err := Compile(expr)
	if err != nil {
		return nil, err
	}
	doc, err := ParseDocument(data)
	if err != nil {
		return nil, err
	}
	return query.Values(doc)
}

// Node identifies one match. Repeated selections of the same location produce
// separate [Node] entries with the same path.
type Node struct {
	// Value is the original JSON fragment, including whitespace within containers.
	// It aliases the input bytes and must not be modified.
	Value json.RawMessage
	// Path is an RFC 9535 Normalized Path identifying the value from the root.
	Path string
}

// QueryNodes compiles expr, parses data, and returns matches as described by
// [Query.Nodes]. Compilation errors take precedence over document errors.
// On any error it returns a nil slice. Returned values alias data and must
// not be modified.
func QueryNodes(expr string, data []byte) ([]Node, error) {
	query, err := Compile(expr)
	if err != nil {
		return nil, err
	}
	doc, err := ParseDocument(data)
	if err != nil {
		return nil, err
	}
	return query.Nodes(doc)
}
