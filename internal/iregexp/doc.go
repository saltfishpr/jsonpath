// Package iregexp implements the checking I-Regexp dialect from RFC 9485.
// Unicode properties follow the Go toolchain's unicode.Version. It matches
// Unicode scalar values and does not use the regexp package's repeat expansion.
package iregexp
