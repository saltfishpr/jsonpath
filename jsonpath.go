// Package jsonpath 编译并执行 RFC 9535 JSONPath 查询。一次性查询可用
// QueryValues 和 QueryNodes；重复查询可复用 Compile 返回的 Query。
// 解析结果和查询结果可能与输入字节共享内存，调用方不应修改输入或节点。
package jsonpath

import (
	"encoding/json"

	"github.com/saltfishpr/jsonpath/internal/ast"
	"github.com/saltfishpr/jsonpath/internal/document"
	"github.com/saltfishpr/jsonpath/internal/iregexp"
	"github.com/saltfishpr/jsonpath/internal/parse"
)

// Query 是可重复执行的编译查询。同一 Query 可由多个 goroutine 并发使用，
// 前提是调用方不修改对应的 Document。
type Query struct {
	src      string
	query    *ast.Query
	literals map[*ast.Literal]document.Value
	regexes  map[string]*iregexp.Regexp
}

// Compile 使用内置函数编译查询串。
func Compile(src string) (*Query, error) {
	q, err := parse.Parse(src)
	if err != nil {
		return nil, err
	}
	c := &compiler{}
	if err := c.walkQuery(q); err != nil {
		return nil, err
	}
	return &Query{src: src, query: q, literals: c.literals, regexes: c.regexes}, nil
}

// Values 返回匹配节点的原始 JSON 片段，按查询选择顺序保留重复选择。
// 片段与文档的原始输入共享内存；没有匹配时返回空切片。
func (q *Query) Values(doc *Document) ([]json.RawMessage, error) {
	nodes := q.run(doc)
	out := make([]json.RawMessage, len(nodes))
	for i, n := range nodes {
		out[i] = json.RawMessage(n.Value.Raw())
	}
	return out, nil
}

// Nodes 返回匹配节点及其 Normalized Path，按查询选择顺序保留重复选择。
// 节点的 Value 与 doc 共享内存；没有匹配时返回空切片。
func (q *Query) Nodes(doc *Document) ([]Node, error) {
	nodes := q.run(doc)
	out := make([]Node, len(nodes))
	for i, n := range nodes {
		out[i] = Node{Value: json.RawMessage(n.Value.Raw()), Path: normalizedPath(n)}
	}
	return out, nil
}

// Document 保存解析后的节点树和原始输入，应通过 Parse 创建。调用方应将其视为不可变。
type Document struct {
	root *document.Node
	src  []byte
}

// Parse 将 JSON 字节解析为 Document，允许 JSON 空白字符包围输入，但不接受 BOM。
// Document 与 data 共享底层字节，调用方不应修改 data。
func Parse(data []byte) (*Document, error) {
	root, err := document.Parse(data)
	if err != nil {
		return nil, err
	}
	return &Document{root: root, src: data}, nil
}

// QueryValues 编译 expr、解析 data，并返回匹配节点的原始 JSON 片段。
// 结果按查询选择顺序保留重复选择，与 data 共享底层字节；无匹配时返回空切片。
func QueryValues(expr string, data []byte) ([]json.RawMessage, error) {
	query, err := Compile(expr)
	if err != nil {
		return nil, err
	}
	doc, err := Parse(data)
	if err != nil {
		return nil, err
	}
	return query.Values(doc)
}

// Node 包含匹配节点的原始 JSON 片段及其 Normalized Path。Value 与输入共享底层字节。
type Node struct {
	Value json.RawMessage
	Path  string
}

// QueryNodes 编译 expr、解析 data，并返回匹配节点及其 Normalized Path。
// 节点的 Value 与 data 共享底层字节；无匹配时返回空切片。
func QueryNodes(expr string, data []byte) ([]Node, error) {
	query, err := Compile(expr)
	if err != nil {
		return nil, err
	}
	doc, err := Parse(data)
	if err != nil {
		return nil, err
	}
	return query.Nodes(doc)
}
