package jsonpath

import "fmt"

// CompileError 表示查询的词法、语法或语义错误，可通过 errors.As 提取。
type CompileError struct {
	// Offset 是从 0 开始的原始查询字符串字节偏移，EOF 位置等于 len(src)。
	Offset  int
	Message string
}

func (e *CompileError) Error() string {
	return fmt.Sprintf("jsonpath: compile error at offset %d: %s", e.Offset, e.Message)
}

// DocumentError 表示 JSON 文档解析错误，可通过 errors.As 提取。
type DocumentError struct {
	// Offset 是从 0 开始的输入字节偏移，EOF 位置等于 len(data)。
	Offset  int
	Message string
}

func (e *DocumentError) Error() string {
	return fmt.Sprintf("jsonpath: parse json at offset %d: %s", e.Offset, e.Message)
}
