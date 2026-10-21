package jsonpath

import "fmt"

const maxDepth = 1024

// CompileError describes an invalid JSONPath expression. Offset is a zero-based
// byte offset in the expression.
type CompileError struct {
	// Offset is the zero-based byte position where compilation failed.
	Offset int
	// Message describes the invalid syntax or type usage.
	Message string
}

// Error describes the expression error and its byte offset.
func (e *CompileError) Error() string {
	return fmt.Sprintf("jsonpath: expression at byte %d: %s", e.Offset, e.Message)
}

// DocumentError describes invalid JSON input. Offset is a zero-based byte offset.
type DocumentError struct {
	// Offset is the zero-based byte position where parsing failed.
	Offset int
	// Message describes the invalid JSON input.
	Message string
}

// Error describes the document error and its byte offset.
func (e *DocumentError) Error() string {
	return fmt.Sprintf("jsonpath: document at byte %d: %s", e.Offset, e.Message)
}

// LimitError reports an implementation resource limit, not invalid syntax.
type LimitError struct {
	// Resource identifies the operation that exceeded its bound.
	Resource string
	// Limit is the maximum supported nesting depth.
	Limit int
}

// Error describes the resource and its limit.
func (e *LimitError) Error() string {
	return fmt.Sprintf("jsonpath: %s exceeds limit %d", e.Resource, e.Limit)
}
