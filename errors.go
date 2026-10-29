package jsonpath

import "fmt"

// CompileError reports an invalid JSONPath expression. Use [errors.As] to inspect
// its location and diagnostic.
type CompileError struct {
	// Offset is a zero-based byte position in the expression; len(src) denotes EOF.
	Offset int
	// Message describes the failure without a location prefix.
	Message string
}

// Error returns the diagnostic with its byte offset in the expression.
func (e *CompileError) Error() string {
	return fmt.Sprintf("jsonpath: compile error at offset %d: %s", e.Offset, e.Message)
}

// DocumentError reports rejected JSON input, including violations of the
// parser's Unicode, member uniqueness, or depth requirements.
type DocumentError struct {
	// Offset is a zero-based byte position in the input; len(data) denotes EOF.
	Offset int
	// Message describes why the input was rejected without a location prefix.
	Message string
}

// Error returns the diagnostic with its byte offset in the JSON input.
func (e *DocumentError) Error() string {
	return fmt.Sprintf("jsonpath: parse json at offset %d: %s", e.Offset, e.Message)
}
