// Package jsonpath 编译并执行 RFC 9535 JSONPath 查询。一次性查询可用
// QueryValues 和 QueryNodes；重复查询可复用 Compile 返回的 Query。
// 解析结果和查询结果可能与输入字节共享内存，调用方不应修改输入或节点。
package jsonpath

import (
	"encoding/json"
	"errors"

	"github.com/saltfishpr/jsonpath/internal/ast"
	"github.com/saltfishpr/jsonpath/internal/document"
	"github.com/saltfishpr/jsonpath/internal/iregexp"
	"github.com/saltfishpr/jsonpath/internal/parse"
)

// Query 是可重复执行的编译查询。零值不可用，必须由 Compile 创建。
// 同一 Query 可由多个 goroutine 并发使用，
// 前提是调用方不修改对应的 Document。
type Query struct {
	src      string
	query    *ast.Query
	literals map[*ast.Literal]*document.Node
	regexes  map[string]*iregexp.Regexp
}

// Compile 使用内置函数编译查询串。
// 查询的词法、语法和语义错误返回 *CompileError。
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
	return &Query{src: src, query: q, literals: c.literals, regexes: c.regexes}, nil
}

// Values 返回按选择顺序排列的原始 JSON 片段，保留重复项并与文档共享内存。
// 无匹配时返回空切片；求值失败时不返回部分结果。q 和 doc 必须是 Compile 和 ParseDocument 返回的非 nil 值。
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

// Nodes 返回按选择顺序排列的匹配节点及其 Normalized Path，并保留重复项。
// Value 与文档共享内存；无匹配时返回空切片，求值失败时不返回部分结果。
// q 和 doc 必须是 Compile 和 ParseDocument 返回的非 nil 值。
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

// Document 保存解析后的节点树和原始输入。零值不可用，必须由 ParseDocument 创建。
// 调用方应将其视为不可变。
type Document struct {
	root *document.Node
	src  []byte
}

// ParseDocument 将 UTF-8 编码的 JSON 字节解析为 Document，允许 JSON 空白字符包围输入，
// 但不接受 BOM、非法 UTF-8 或重复的对象成员名。解析错误返回 *DocumentError。
// Document 与 data 共享底层字节，调用方不应修改 data。
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

// QueryValues 返回 expr 在 data 中匹配的原始 JSON 片段，按选择顺序保留重复项。
// 结果与 data 共享内存；无匹配时返回空切片。
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

// Node 包含匹配节点的原始 JSON 片段及其 Normalized Path；Value 与输入共享内存。
type Node struct {
	Value json.RawMessage
	Path  string
}

// QueryNodes 返回 expr 在 data 中匹配的节点及其 Normalized Path，按选择顺序保留重复项。
// Value 与 data 共享内存；无匹配时返回空切片。
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
