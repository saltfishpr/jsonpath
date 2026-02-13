// Package jsonpath implements JSONPath query syntax parser and evaluator with gjson-style API.
//
// The implementation follows RFC 9535: https://www.rfc-editor.org/rfc/rfc9535.html
package jsonpath

import (
	"strconv"
	"strings"
)

// JSONType represents the type of a JSON value
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

// String returns a string representation of the type
func (t JSONType) String() string {
	return []string{"Null", "False", "Number", "String", "True", "JSON"}[t]
}

// Result represents a JSON value returned from Get()
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

// Get executes a JSONPath query and returns the first result
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

// GetBytes executes a JSONPath query with []byte input
func GetBytes(json []byte, path string) Result {
	return Get(string(json), path)
}

// Get continues a query from the current result
func (r Result) Get(path string) Result {
	if !r.Exists() {
		return Result{}
	}
	return Get(r.Raw, path)
}

// GetMany executes a JSONPath query and returns all results
func GetMany(json, path string) []Result {
	query, err := Parse(path)
	if err != nil {
		return nil
	}
	eval := NewEvaluator(json, query)
	return eval.Evaluate()
}

// GetManyBytes executes a JSONPath query with []byte input
func GetManyBytes(json []byte, path string) []Result {
	return GetMany(string(json), path)
}

// GetMany continues a query from the current result
func (r Result) GetMany(path string) []Result {
	if !r.Exists() {
		return nil
	}
	return GetMany(r.Raw, path)
}

// Exists checks if the result exists
func (r Result) Exists() bool {
	return r.Type != JSONTypeNull || len(r.Raw) != 0
}

// IsObject checks if the result is a JSON object
func (r Result) IsObject() bool {
	return r.Type == JSONTypeJSON && len(r.Raw) > 0 && r.Raw[0] == '{'
}

// IsArray checks if the result is a JSON array
func (r Result) IsArray() bool {
	return r.Type == JSONTypeJSON && len(r.Raw) > 0 && r.Raw[0] == '['
}

// IsString checks if the result is a string
func (r Result) IsString() bool {
	return r.Type == JSONTypeString
}

// IsBool checks if the result is a boolean
func (r Result) IsBool() bool {
	return r.Type == JSONTypeTrue || r.Type == JSONTypeFalse
}

// String returns the string representation
func (r Result) String() string {
	switch r.Type {
	case JSONTypeNull:
		return ""
	case JSONTypeFalse:
		return "false"
	case JSONTypeTrue:
		return "true"
	case JSONTypeString:
		return r.Str
	case JSONTypeNumber:
		if r.Raw == "" {
			return strconv.FormatFloat(r.Num, 'f', -1, 64)
		}
		return r.Raw
	case JSONTypeJSON:
		return r.Raw
	default:
		return ""
	}
}

// Int returns the int64 representation
func (r Result) Int() int64 {
	switch r.Type {
	case JSONTypeTrue:
		return 1
	case JSONTypeNumber:
		return int64(r.Num)
	case JSONTypeString:
		n, _ := strconv.ParseInt(r.Str, 10, 64)
		return n
	}
	return 0
}

// Uint returns the uint64 representation
func (r Result) Uint() uint64 {
	switch r.Type {
	case JSONTypeTrue:
		return 1
	case JSONTypeNumber:
		return uint64(r.Num)
	case JSONTypeString:
		n, _ := strconv.ParseUint(r.Str, 10, 64)
		return n
	}
	return 0
}

// Float returns the float64 representation
func (r Result) Float() float64 {
	switch r.Type {
	case JSONTypeTrue:
		return 1
	case JSONTypeNumber:
		return r.Num
	case JSONTypeString:
		f, _ := strconv.ParseFloat(r.Str, 64)
		return f
	}
	return 0
}

// Bool returns the bool representation
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
	}
	return false
}

// Array returns the []Result representation
func (r Result) Array() []Result {
	if r.Type == JSONTypeNull {
		return []Result{}
	}
	if !r.IsArray() {
		return []Result{r}
	}

	var results []Result
	i := 1
	for i < len(r.Raw) {
		i = skipWhitespaceJSON(r.Raw, i)
		if i >= len(r.Raw) || r.Raw[i] == ']' {
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

// Map returns the map[string]Result representation
func (r Result) Map() map[string]Result {
	results := make(map[string]Result)
	for _, kv := range r.MapKVList() {
		results[kv.Key] = kv.Value
	}
	return results
}

type KV struct {
	Key   string
	Value Result
}

func (r Result) MapKVList() []KV {
	if r.Type == JSONTypeNull {
		return []KV{}
	}
	if !r.IsObject() {
		return []KV{}
	}

	var results []KV
	i := 1
	for i < len(r.Raw) {
		i = skipWhitespaceJSON(r.Raw, i)
		if i >= len(r.Raw) || r.Raw[i] == '}' {
			break
		}
		key, value, next := parseObjectMember(r.Raw, i)
		// Stop parsing on invalid JSON to prevent infinite loop
		if key == "" {
			break
		}
		results = append(results, KV{Key: key, Value: value})
		i = next

		i = skipWhitespaceJSON(r.Raw, i)
		if i < len(r.Raw) && r.Raw[i] == ',' {
			i++
		}
	}
	return results
}

// Value returns the Go native value representation
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
	}
	return nil
}
