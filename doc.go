// Package jsonpath implements RFC 9535 JSONPath over raw JSON documents.
// Compile queries and Parse documents once to reuse them concurrently. Results
// are independent copies of the selected JSON text. Object traversal follows
// source order; array and selector order and duplicate selections are preserved.
package jsonpath
