// Package iregexp provides RFC 9485 I-Regexp matching for JSONPath functions.
// Patterns are translated to Go regular expressions. Syntax errors are distinct
// from implementation limits so callers can treat invalid patterns as false
// without hiding failures to process valid patterns.
package iregexp

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Regexp is a compiled pattern safe for concurrent [Regexp.Match] and [Regexp.Search] calls.
// Create it with [Compile] or [MustCompile]; the zero value is not usable.
// Matching treats each invalid UTF-8 byte in the input as U+FFFD, as Go regexp does.
type Regexp struct {
	src       string
	literal   string
	isLiteral bool
	search    *regexp.Regexp
	match     *regexp.Regexp
}

// ErrPatternTooLarge means translating a pattern would exceed the output size
// limit. It does not imply invalid I-Regexp syntax.
var ErrPatternTooLarge = errors.New("iregexp: translated pattern exceeds safety limit")

// ErrNestingTooDeep identifies the implementation limit of 128 nested groups.
// [Compile] wraps this error with the source position; use [errors.Is] to detect it.
var ErrNestingTooDeep = errors.New("iregexp: pattern nesting exceeds safety limit")

// SyntaxError reports a violation of I-Regexp grammar. Use [errors.As] to
// distinguish it from resource limits and failures in the regexp backend.
type SyntaxError struct {
	// Offset is the zero-based byte position at which validation failed.
	Offset int
	// Message describes the failure without a location prefix.
	Message string
}

// Error returns the diagnostic with its byte offset in the pattern.
func (e *SyntaxError) Error() string {
	return fmt.Sprintf("iregexp: at offset %d: %s", e.Offset, e.Message)
}

// Compile prepares pattern for whole-string and substring matching.
// Grammar violations return *[SyntaxError]; size, nesting, numeric conversion,
// and backend failures return other errors. Any error returns a nil [Regexp].
func Compile(pattern string) (*Regexp, error) {
	t := &translator{src: pattern}
	translated, err := t.translate()
	if err != nil {
		return nil, err
	}
	search, err := regexp.Compile(translated)
	if err != nil {
		return nil, fmt.Errorf("iregexp: internal RE2 compile error: %w", err)
	}
	match, err := regexp.Compile(`\A(?:` + translated + `)\z`)
	if err != nil {
		return nil, fmt.Errorf("iregexp: internal RE2 compile error: %w", err)
	}
	literal, complete := search.LiteralPrefix()
	return &Regexp{src: pattern, literal: literal, isLiteral: complete, search: search, match: match}, nil
}

// MustCompile returns [Compile]'s result or panics with its error. It is intended
// for patterns whose validity is established before runtime.
func MustCompile(pattern string) *Regexp {
	r, err := Compile(pattern)
	if err != nil {
		panic(err)
	}
	return r
}

// Match reports whether the pattern matches all of s, including any newlines.
// The I-Regexp dot excludes carriage return and line feed.
func (r *Regexp) Match(s string) bool {
	// Byte equality is equivalent only for valid UTF-8; otherwise the backend
	// must interpret malformed bytes as replacement characters.
	if r.isLiteral && utf8.ValidString(s) {
		return s == r.literal
	}
	return r.match.MatchString(s)
}

// Search reports whether the pattern matches any substring of s, including an
// empty substring when the pattern permits one.
func (r *Regexp) Search(s string) bool {
	if r.isLiteral && utf8.ValidString(s) {
		return strings.Contains(s, r.literal)
	}
	return r.search.MatchString(s)
}

// String returns the pattern supplied to [Compile], without translation or normalization.
func (r *Regexp) String() string { return r.src }

const (
	// Go limits the product of nested finite repetition counts, not just each count.
	reWeightLimit = 1000
	// Expansion must remain bounded even when an I-Regexp repeat count is valid.
	maxTranslatedBytes = 8 << 20
	// Small translated output can still require excessive parser recursion.
	maxNestingDepth = 128
)

// Parsing appends to a shared builder and returns the largest nested finite
// repeat product. Tracking that weight allows valid I-Regexp repetitions to be
// expanded when Go's regexp parser cannot accept their counted form.
type translator struct {
	src   string
	pos   int
	depth int
	out   strings.Builder
}

func (t *translator) write(text string) error {
	if len(text) > maxTranslatedBytes-t.out.Len() {
		return ErrPatternTooLarge
	}
	t.out.WriteString(text)
	return nil
}

func (t *translator) errf(format string, args ...any) error {
	return &SyntaxError{Offset: t.pos, Message: fmt.Sprintf(format, args...)}
}

func (t *translator) peek() (rune, int, bool) {
	if t.pos >= len(t.src) {
		return 0, 0, false
	}
	r, w := utf8.DecodeRuneInString(t.src[t.pos:])
	return r, w, true
}

func (t *translator) peekAt(off int) (rune, bool) {
	p := t.pos + off
	if p >= len(t.src) {
		return 0, false
	}
	r, _ := utf8.DecodeRuneInString(t.src[p:])
	return r, true
}

func (t *translator) advance() (rune, bool) {
	r, w, ok := t.peek()
	if !ok {
		return 0, false
	}
	t.pos += w
	return r, true
}

func (t *translator) translate() (string, error) {
	_, err := t.regexp()
	if err != nil {
		return "", err
	}
	if t.pos != len(t.src) {
		return "", t.errf("unexpected %q", t.src[t.pos:])
	}
	return t.out.String(), nil
}

func (t *translator) regexp() (int, error) {
	weight, err := t.branch()
	if err != nil {
		return 0, err
	}
	for {
		r, _, ok := t.peek()
		if !ok || r != '|' {
			break
		}
		t.advance()
		if err := t.write("|"); err != nil {
			return 0, err
		}
		bw, err := t.branch()
		if err != nil {
			return 0, err
		}
		if bw > weight {
			weight = bw
		}
	}
	return weight, nil
}

func (t *translator) branch() (int, error) {
	weight := 1
	for {
		r, _, ok := t.peek()
		if !ok || r == '|' || r == ')' {
			return weight, nil
		}
		pw, err := t.piece()
		if err != nil {
			return 0, err
		}
		if pw > weight {
			weight = pw
		}
	}
}

func (t *translator) piece() (int, error) {
	atomStart := t.out.Len()
	atomWeight, err := t.atom()
	if err != nil {
		return 0, err
	}
	r, _, ok := t.peek()
	if !ok {
		return atomWeight, nil
	}
	switch r {
	case '?', '*', '+':
		t.advance()
		// Go's repeat-product check passes through these operators unchanged.
		return atomWeight, t.write(string(r))
	case '{':
		return t.braceQuantifier(atomStart, atomWeight)
	}
	return atomWeight, nil
}

// The atom is already in out. Preserve it while choosing a counted suffix or
// explicit copies that keep nested repetition within the backend's limit.
func (t *translator) braceQuantifier(atomStart, atomWeight int) (int, error) {
	t.advance()
	nStr, err := t.readDigits()
	if err != nil {
		return 0, err
	}
	if nStr == "" {
		return 0, t.errf("expected digits in quantifier")
	}
	nStr = normalizeCount(nStr)

	r, _, ok := t.peek()
	if !ok {
		return 0, t.errf("unterminated quantifier")
	}

	mStr := nStr
	unbounded := false
	if r == ',' {
		t.advance()
		mStr, err = t.readDigits()
		if err != nil {
			return 0, err
		}
		if mStr == "" {
			unbounded = true
		} else {
			mStr = normalizeCount(mStr)
		}
		r, _, ok = t.peek()
		if !ok {
			return 0, t.errf("unterminated quantifier")
		}
	}
	if r != '}' {
		return 0, t.errf("expected '}' in quantifier, got %q", r)
	}
	t.advance()

	// Diagnose reversed bounds as syntax errors even when either count is too
	// large for int. Valid but unrepresentable counts are implementation failures.
	if !unbounded && (len(nStr) > len(mStr) || (len(nStr) == len(mStr) && nStr > mStr)) {
		return 0, t.errf("quantifier range out of order: {%s,%s}", nStr, mStr)
	}
	n, err := strconv.Atoi(nStr)
	if err != nil {
		return 0, fmt.Errorf("iregexp: quantifier minimum exceeds implementation limit at offset %d: %w", t.pos, err)
	}
	var m int
	if !unbounded {
		m, err = strconv.Atoi(mStr)
		if err != nil {
			return 0, fmt.Errorf("iregexp: quantifier maximum exceeds implementation limit at offset %d: %w", t.pos, err)
		}
	}

	// For {n,}, Go checks the minimum count; for bounded repeats it checks
	// the maximum. A zero count leaves the inner weight unchanged.
	maxCount := m
	if unbounded {
		maxCount = n
	}
	effective := maxCount
	if effective < 1 {
		effective = 1
	}

	if effective <= reWeightLimit/atomWeight && n <= reWeightLimit &&
		(unbounded || m <= reWeightLimit) {
		suffix := "{" + nStr
		if unbounded {
			suffix += ","
		} else if m != n {
			suffix += "," + mStr
		}
		return effective * atomWeight, t.write(suffix + "}")
	}

	// Size estimates saturate so huge counts cannot wrap below the output
	// limit and reach Grow with an invalid allocation size.
	atomText := t.out.String()[atomStart:]
	atomLen := uint64(len(atomText))
	remaining := n - 1
	optional := m - n
	var extra uint64
	if n == 0 {
		// The emitted atom accounts for one optional copy when the minimum is
		// zero. The unbounded {0,} case already used the counted form above.
		remaining = 0
		optional--
		extra = 1 // Account for the existing atom's '?' suffix.
	}
	extra = addSat(extra, mulSat(uint64(remaining), atomLen))
	if unbounded {
		extra = addSat(extra, addSat(atomLen, 5))
	} else {
		extra = addSat(extra, mulSat(uint64(optional), addSat(atomLen, 5)))
	}
	if extra > uint64(maxTranslatedBytes-t.out.Len()) {
		return 0, ErrPatternTooLarge
	}

	t.out.Grow(int(extra))
	if n == 0 {
		t.out.WriteByte('?')
	}
	for i := 0; i < remaining; i++ {
		t.out.WriteString(atomText)
	}
	if unbounded {
		t.out.WriteString("(?:")
		t.out.WriteString(atomText)
		t.out.WriteString(")*")
	} else {
		for i := 0; i < optional; i++ {
			t.out.WriteString("(?:")
			t.out.WriteString(atomText)
			t.out.WriteString(")?")
		}
	}
	// Concatenation and optional/star wrappers add no finite repeat weight.
	return atomWeight, nil
}

// Saturation preserves an over-budget estimate when multiplication overflows.
func mulSat(a, b uint64) uint64 {
	if a == 0 || b == 0 {
		return 0
	}
	if a > ^uint64(0)/b {
		return ^uint64(0)
	}
	return a * b
}

// Saturation preserves an over-budget estimate when addition overflows.
func addSat(a, b uint64) uint64 {
	s := a + b
	if s < a {
		return ^uint64(0)
	}
	return s
}

func (t *translator) readDigits() (string, error) {
	start := t.pos
	for {
		r, _, ok := t.peek()
		if !ok || r < '0' || r > '9' {
			break
		}
		t.advance()
	}
	return t.src[start:t.pos], nil
}

func normalizeCount(s string) string {
	// Strip redundant zeros so length and lexical order compare arbitrary counts.
	i := 0
	for i < len(s)-1 && s[i] == '0' {
		i++
	}
	return s[i:]
}

func (t *translator) atom() (int, error) {
	r, _, ok := t.peek()
	if !ok {
		return 0, t.errf("empty atom")
	}
	switch r {
	case '(':
		if t.depth == maxNestingDepth {
			return 0, fmt.Errorf("%w: limit %d at offset %d", ErrNestingTooDeep, maxNestingDepth, t.pos)
		}
		t.advance()
		if err := t.write("(?:"); err != nil {
			return 0, err
		}
		t.depth++
		w, err := t.regexp()
		t.depth--
		if err != nil {
			return 0, err
		}
		r2, _, ok := t.peek()
		if !ok || r2 != ')' {
			return 0, t.errf("expected ')'")
		}
		t.advance()
		return w, t.write(")")
	case '.':
		t.advance()
		// Go's default dot excludes LF only; I-Regexp also excludes CR.
		return 1, t.write(`[^\n\r]`)
	case '\\':
		s, err := t.escape()
		if err != nil {
			return 0, err
		}
		return 1, t.write(s)
	case '[':
		return 1, t.charClassExpr()
	case ')', ']', '|', '?', '*', '+', '{', '}':
		return 0, t.errf("unexpected %q", r)
	default:
		t.advance()
		return 1, t.write(literalString(r))
	}
}

// I-Regexp treats ^ and $ as ordinary characters outside character classes.
func literalString(r rune) string {
	switch r {
	case '\\', '.', '+', '*', '?', '(', ')', '|', '[', ']', '{', '}',
		'^', '$':
		return "\\" + string(r)
	}
	return string(r)
}

func (t *translator) escape() (string, error) {
	t.advance()
	r, _, ok := t.peek()
	if !ok {
		return "", t.errf("dangling backslash")
	}
	switch r {
	case 'n', 'r', 't', '\\', '|', '.', '-', '^', '?', '*', '+',
		'(', ')', '{', '}', '[', ']':
		t.advance()
		return "\\" + string(r), nil
	case 'p', 'P':
		return t.categoryEsc(r)
	default:
		return "", t.errf("invalid escape \\%c", r)
	}
}

func (t *translator) categoryEsc(kind rune) (string, error) {
	t.advance()
	r, _, ok := t.peek()
	if !ok || r != '{' {
		return "", t.errf("expected '{' after \\%c", kind)
	}
	t.advance()
	propStart := t.pos
	for {
		r2, _, ok := t.peek()
		if !ok {
			return "", t.errf("unterminated \\%c{...}", kind)
		}
		if r2 == '}' {
			break
		}
		t.advance()
	}
	propName := t.src[propStart:t.pos]
	if !isValidIsCategory(propName) {
		return "", t.errf("invalid IsCategory %q", propName)
	}
	t.advance()
	return "\\" + string(kind) + "{" + propName + "}", nil
}

// A bare hyphen is literal only at a class boundary; internal hyphens must
// belong to a range. This context cannot be delegated to Go's class parser.
func (t *translator) charClassExpr() error {
	t.advance()
	if err := t.write("["); err != nil {
		return err
	}

	if r, _, ok := t.peek(); ok && r == '^' {
		t.advance()
		if err := t.write("^"); err != nil {
			return err
		}
	}

	first := true
	for {
		r, _, ok := t.peek()
		if !ok {
			return t.errf("unterminated character class")
		}
		if r == ']' {
			if first {
				return t.errf("empty character class")
			}
			t.advance()
			return t.write("]")
		}
		if r == '-' {
			if first {
				t.advance()
				if err := t.write(`\-`); err != nil {
					return err
				}
				first = false
				continue
			}
			if next, ok := t.peekAt(1); ok && next == ']' {
				t.advance()
				if err := t.write(`\-`); err != nil {
					return err
				}
				continue
			}
			return t.errf("unexpected '-' in character class")
		}
		first = false
		cce, err := t.cce1()
		if err != nil {
			return err
		}
		if err := t.write(cce); err != nil {
			return err
		}
	}
}

// A class item is either a category escape or a single character with an
// optional range endpoint. Escaped punctuation counts as a single character.
func (t *translator) cce1() (string, error) {
	r, _, ok := t.peek()
	if !ok {
		return "", t.errf("unterminated character class")
	}

	if r == '\\' {
		next, ok := t.peekAt(1)
		if !ok {
			return "", t.errf("dangling backslash")
		}
		if next == 'p' || next == 'P' {
			t.advance()
			s, err := t.categoryEsc(next)
			if err != nil {
				return "", err
			}
			// A category denotes a set, so it cannot start a range. A trailing
			// literal hyphen remains valid.
			if r2, _, ok := t.peek(); ok && r2 == '-' {
				if nx, ok := t.peekAt(1); ok && nx != ']' {
					return "", t.errf(`\p/\P cannot be a range endpoint`)
				}
			}
			return s, nil
		}
		r, err := t.classSingleCharEsc()
		if err != nil {
			return "", err
		}
		tail, err := t.maybeRangeTail(r)
		if err != nil {
			return "", err
		}
		return classLiteral(r) + tail, nil
	}

	if r == ']' || r == '-' || r == '[' {
		return "", t.errf("unexpected %q in character class (must be escaped)", r)
	}
	t.advance()
	tail, err := t.maybeRangeTail(r)
	if err != nil {
		return "", err
	}
	return classLiteral(r) + tail, nil
}

func (t *translator) classSingleCharEsc() (rune, error) {
	t.advance()
	r, _, ok := t.peek()
	if !ok {
		return 0, t.errf("dangling backslash")
	}
	switch r {
	case 'n', 'r', 't', '\\', '|', '.', '-', '^', '?', '*', '+',
		'(', ')', '{', '}', '[', ']':
		t.advance()
		switch r {
		case 'n':
			r = '\n'
		case 'r':
			r = '\r'
		case 't':
			r = '\t'
		}
		return r, nil
	default:
		return 0, t.errf("invalid escape \\%c in character class", r)
	}
}

func (t *translator) maybeRangeTail(lower rune) (string, error) {
	r, _, ok := t.peek()
	if !ok || r != '-' {
		return "", nil
	}
	if next, ok := t.peekAt(1); ok && next == ']' {
		return "", nil // Leave a boundary hyphen for charClassExpr to emit literally.
	}
	t.advance()
	r2, _, ok := t.peek()
	if !ok {
		return "", t.errf("unterminated character class")
	}
	if r2 == '\\' {
		next, ok := t.peekAt(1)
		if !ok {
			return "", t.errf("dangling backslash")
		}
		if next == 'p' || next == 'P' {
			return "", t.errf(`\p/\P cannot be a range endpoint`)
		}
		var err error
		r2, err = t.classSingleCharEsc()
		if err != nil {
			return "", err
		}
	} else {
		if r2 == ']' || r2 == '-' || r2 == '[' {
			return "", t.errf("invalid range endpoint %q", r2)
		}
		t.advance()
	}
	// Report reversed endpoints as [SyntaxError] so JSONPath can yield false
	// rather than propagating a backend compilation failure.
	if lower > r2 {
		return "", t.errf("character range out of order: %q-%q", lower, r2)
	}
	return "-" + classLiteral(r2), nil
}

func classLiteral(r rune) string {
	switch r {
	case '\n':
		return `\n`
	case '\r':
		return `\r`
	case '\t':
		return `\t`
	case '\\', '[', ']', '^', '-':
		return "\\" + string(r)
	}
	return string(r)
}

// RFC 9485 permits only these general categories and subcategories, even when
// the backend recognizes additional Unicode property names.
var isCategorySubs = map[byte]string{
	'L': "lmotu",
	'M': "cen",
	'N': "dlo",
	'P': "cdefios",
	'Z': "lps",
	'S': "ckmo",
	'C': "cfno",
}

func isValidIsCategory(s string) bool {
	if len(s) == 0 || len(s) > 2 {
		return false
	}
	subs, ok := isCategorySubs[s[0]]
	if !ok {
		return false
	}
	if len(s) == 1 {
		return true
	}
	for i := 0; i < len(subs); i++ {
		if subs[i] == s[1] {
			return true
		}
	}
	return false
}
