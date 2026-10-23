package jsonpath

import "fmt"

type CompileError struct {
	Offset  int
	Message string
}

func (e *CompileError) Error() string {
	return fmt.Sprintf("jsonpath: compile error at offset %d: %s", e.Offset, e.Message)
}
