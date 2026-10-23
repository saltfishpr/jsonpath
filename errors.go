package jsonpath

import "fmt"

// CompileError 表示查询的词法、语法或语义错误，可通过 errors.As 提取。
type CompileError struct {
	// Offset 是从 0 开始的原始查询字符串字节偏移，EOF 位置等于 len(src)。
	Offset int
	// Message 描述错误原因。
	Message string
}

// Error 返回包含字节偏移和错误原因的描述。
func (e *CompileError) Error() string {
	return fmt.Sprintf("jsonpath: compile error at offset %d: %s", e.Offset, e.Message)
}
