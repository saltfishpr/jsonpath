// Package jsonpath 实现了 JSONPath 语法的解析和查询功能，提供 gjson 风格的 API。
// 实现基于 RFC 9535 规范：https://www.rfc-editor.org/rfc/rfc9535.html
package jsonpath

import (
	"strconv"
	"strings"
)

// JSONType is Result type
type JSONType int

const (
	// JSONTypeNull is a null json value
	JSONTypeNull JSONType = iota
	// JSONTypeFalse is a json false boolean
	JSONTypeFalse
	// JSONTypeNumber is json number
	JSONTypeNumber
	// JSONTypeString is json string
	JSONTypeString
	// JSONTypeTrue is a json true boolean
	JSONTypeTrue
	// JSONTypeJSON is a raw block of JSON
	JSONTypeJSON
)

// String returns a string representation of the type.
func (t JSONType) String() string {
	switch t {
	default:
		return ""
	case JSONTypeNull:
		return "Null"
	case JSONTypeFalse:
		return "False"
	case JSONTypeNumber:
		return "Number"
	case JSONTypeString:
		return "String"
	case JSONTypeTrue:
		return "True"
	case JSONTypeJSON:
		return "JSON"
	}
}

// Result represents a json value that is returned from Get().
type Result struct {
	// Type is the json type
	Type JSONType
	// Raw is the raw json
	Raw string
	// Str is the json string
	Str string
	// Num is the json number
	Num float64
	// Index of raw value in original json, zero means index unknown
	Index int
}

// Get 执行 JSONPath 查询，返回第一个匹配结果
func Get(json, path string) Result {
	query, err := Parse(path)
	if err != nil {
		return Result{}
	}
	eval := NewEvaluator(json, query)
	results := eval.Evaluate()
	if len(results) == 0 {
		return Result{}
	}
	return results[0]
}

// GetBytes 执行 JSONPath 查询，支持 []byte 输入
func GetBytes(json []byte, path string) Result {
	return Get(string(json), path)
}

// Get 在当前结果上继续查询
func (r Result) Get(path string) Result {
	if !r.Exists() {
		return Result{}
	}
	return Get(r.Raw, path)
}

// GetMany 执行 JSONPath 查询，返回所有匹配结果
func GetMany(json, path string) []Result {
	query, err := Parse(path)
	if err != nil {
		return nil
	}
	eval := NewEvaluator(json, query)
	return eval.Evaluate()
}

// GetManyBytes 执行 JSONPath 查询，支持 []byte 输入
func GetManyBytes(json []byte, path string) []Result {
	return GetMany(string(json), path)
}

// GetMany 在当前结果上继续查询
func (r Result) GetMany(path string) []Result {
	if !r.Exists() {
		return nil
	}
	return GetMany(r.Raw, path)
}

// Exists 检查结果是否存在
func (r Result) Exists() bool {
	return r.Type != JSONTypeNull || len(r.Raw) != 0
}

// IsObject 检查结果是否为 JSON 对象
func (r Result) IsObject() bool {
	return r.Type == JSONTypeJSON && len(r.Raw) > 0 && r.Raw[0] == '{'
}

// IsArray 检查结果是否为 JSON 数组
func (r Result) IsArray() bool {
	return r.Type == JSONTypeJSON && len(r.Raw) > 0 && r.Raw[0] == '['
}

// IsBool 检查结果是否为布尔值
func (r Result) IsBool() bool {
	return r.Type == JSONTypeTrue || r.Type == JSONTypeFalse
}

// String 返回字符串表示
func (r Result) String() string {
	switch r.Type {
	default:
		return ""
	case JSONTypeNull:
		return ""
	case JSONTypeFalse:
		return "false"
	case JSONTypeNumber:
		if len(r.Raw) == 0 {
			return strconv.FormatFloat(r.Num, 'f', -1, 64)
		}
		return r.Raw
	case JSONTypeString:
		return r.Str
	case JSONTypeTrue:
		return "true"
	case JSONTypeJSON:
		return r.Raw
	}
}

// Int 返回整数表示
func (r Result) Int() int64 {
	panic("TODO")
}

// Uint 返回无符号整数表示
func (r Result) Uint() uint64 {
	panic("TODO")
}

// Float 返回浮点数表示
func (r Result) Float() float64 {
	switch r.Type {
	case JSONTypeTrue:
		return 1
	case JSONTypeNumber:
		return r.Num
	case JSONTypeString:
		f, _ := strconv.ParseFloat(r.Str, 64)
		return f
	default:
		return 0
	}
}

// Bool 返回布尔值表示
func (r Result) Bool() bool {
	switch r.Type {
	case JSONTypeTrue:
		return true
	case JSONTypeFalse:
		return false
	case JSONTypeString:
		b, _ := strconv.ParseBool(strings.ToLower(r.Str))
		return b
	case JSONTypeNumber:
		return r.Num != 0
	default:
		return false
	}
}

// Array 返回数组表示
func (r Result) Array() []Result {
	if r.Type == JSONTypeNull {
		return []Result{}
	}
	if !r.IsArray() {
		return []Result{r}
	}

	var results []Result
	i := 1 // 跳过 '['
	for i < len(r.Raw) {
		i = skipWhitespaceJSON(r.Raw, i)
		if i >= len(r.Raw) {
			break
		}
		if r.Raw[i] == ']' {
			break
		}

		elem, next := parseArrayElement(r.Raw, i)
		results = append(results, elem)
		i = next

		i = skipWhitespaceJSON(r.Raw, i)
		if i < len(r.Raw) && r.Raw[i] == ',' {
			i++
		}
	}
	return results
}

// Map 返回对象表示
func (r Result) Map() map[string]Result {
	if r.Type == JSONTypeNull {
		return map[string]Result{}
	}
	if !r.IsObject() {
		return map[string]Result{}
	}

	results := make(map[string]Result)
	i := 1 // 跳过 '{'
	for i < len(r.Raw) {
		i = skipWhitespaceJSON(r.Raw, i)
		if i >= len(r.Raw) {
			break
		}
		if r.Raw[i] == '}' {
			break
		}

		key, value, next := parseObjectMember(r.Raw, i)
		results[key] = value
		i = next

		i = skipWhitespaceJSON(r.Raw, i)
		if i < len(r.Raw) && r.Raw[i] == ',' {
			i++
		}
	}
	return results
}

// Value 返回 Go 原生值表示
func (r Result) Value() interface{} {
	switch r.Type {
	case JSONTypeNull:
		return nil
	case JSONTypeTrue:
		return true
	case JSONTypeFalse:
		return false
	case JSONTypeNumber:
		return r.Num
	case JSONTypeString:
		return r.Str
	case JSONTypeJSON:
		if r.IsArray() {
			return r.Array()
		}
		if r.IsObject() {
			return r.Map()
		}
		return nil
	default:
		return nil
	}
}
