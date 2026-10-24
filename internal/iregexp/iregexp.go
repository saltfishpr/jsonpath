// Package iregexp implements the I-Regexp dialect defined in RFC 9485
// (and referenced by RFC 9535 §2.3.3 for JSONPath regular expression
// matching). It exposes two matching semantics:
//
//   - (*Regexp).Match  — RFC 9485 §4 full-string match.
//   - (*Regexp).Search — any-substring match.
//
// Internally an I-Regexp pattern is validated against the ABNF in
// RFC 9485 §2 and translated to an equivalent Go `regexp` (RE2)
// pattern, then compiled.
package iregexp

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Regexp struct {
	src    string
	search *regexp.Regexp // unanchored (for Search)
	match  *regexp.Regexp // anchored with \A ... \z (for Match)
}

// ErrPatternTooLarge is returned when translation exceeds the safety limit.
// I-Regexp permits unbounded repetition, so the limit bounds pattern expansion.
var ErrPatternTooLarge = errors.New("iregexp: translated pattern exceeds safety limit")

// ErrNestingTooDeep is returned by Compile when a pattern nests more than
// 128 groups. This implementation limit bounds the recursive parser's stack.
var ErrNestingTooDeep = errors.New("iregexp: pattern nesting exceeds safety limit")

// SyntaxError reports an invalid I-Regexp pattern. Callers can use errors.As
// to distinguish invalid patterns from implementation limits and backend errors.
type SyntaxError struct {
	// Offset is the zero-based byte offset within the pattern.
	Offset int
	Message string
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("iregexp: at offset %d: %s", e.Offset, e.Message)
}

// Compile parses and compiles an I-Regexp.
// Invalid patterns return *SyntaxError; implementation limits and backend
// failures return other errors and must not be interpreted as a failed match.
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
	return &Regexp{src: pattern, search: search, match: match}, nil
}

func MustCompile(pattern string) *Regexp {
	r, err := Compile(pattern)
	if err != nil {
		panic(err)
	}
	return r
}

// Match reports whether s matches the regexp according to the
// RFC 9485 §4 semantics (boolean full-string match).
func (r *Regexp) Match(s string) bool { return r.match.MatchString(s) }

// Search reports whether any substring of s matches the regexp.
func (r *Regexp) Search(s string) bool { return r.search.MatchString(s) }

func (r *Regexp) String() string { return r.src }

// Translator / validator
//
// Parsing functions write their RE2 fragments to one Builder so nested
// groups do not repeatedly copy the same text. They return the largest
// finite repeat-count product Go's regexp/syntax.repeatIsValid would observe
// from the fragment's root. Keeping this weight ≤ reWeightLimit guarantees
// Go accepts every repeat we emit; larger repeats are unrolled instead.

const (
	// Go's regexp/syntax enforces a per-expansion limit of 1000.
	reWeightLimit = 1000
	// Hard cap on the translated pattern size, to bound unrolling of
	// pathologically large I-Regexp inputs. ~8 MiB is plenty for any
	// realistic use yet still bounded.
	maxTranslatedBytes = 8 << 20
	// Bound recursive parsing independently of the translated output size.
	maxNestingDepth = 128
)

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

// regexp = branch *( "|" branch )
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

// branch = *piece
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

// piece = atom [ quantifier ]
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
		// These quantifiers do not increase Go's finite repeat weight.
		return atomWeight, t.write(string(r))
	case '{':
		return t.braceQuantifier(atomStart, atomWeight)
	}
	return atomWeight, nil
}

// braceQuantifier handles { QuantExact [ "," [ QuantExact ] ] "}".
// It normalises leading zeros in the digit groups and transparently
// unrolls the repetition if emitting {n,m} would push the weight above
// Go RE2's internal limit.
func (t *translator) braceQuantifier(atomStart, atomWeight int) (int, error) {
	t.advance() // '{'
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

	// Validate the range before converting counts so an invalid pattern remains
	// distinguishable from a valid count that exceeds the machine's integer range.
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

	// Determine the finite "max count" that Go's repeatIsValid would use
	// at this node. For unbounded {n,} it uses Min (= n); for OpStar
	// (which we never emit here) it uses 0 (transparent).
	maxCount := m
	if unbounded {
		maxCount = n
	}
	effective := maxCount
	if effective < 1 {
		effective = 1 // transparent, as per Go's rule.
	}

	// Fast path: current limits already satisfy RE2.
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

	// Slow path: unroll. Writing (atomText){k} as atomText·k·times
	// concatenations keeps every surviving {…} ≤ limit (because atom
	// was translated under the same rule, so atomWeight ≤ limit) and
	// avoids any new {n} on this node.
	//
	// Compute the output size with saturating uint64 arithmetic so a
	// user-supplied value like {9223372036854775807} cannot overflow
	// an `int` and sneak past the size guard — which would then make
	// strings.Builder.Grow panic with "negative count".
	// Builder.String keeps this atom valid even if later appends grow the buffer.
	atomText := t.out.String()[atomStart:]
	atomLen := uint64(len(atomText))
	remaining := n - 1
	optional := m - n
	var extra uint64
	if n == 0 {
		// {0,} always takes the fast path; here the first atom is optional.
		remaining = 0
		optional--
		extra = 1 // '?' makes the existing atom the first optional copy.
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

	// The whole expansion, including the existing prefix, has been checked.
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
	// After unrolling we have a plain concat/star — no new finite
	// repeats surround the atom, so the overall weight collapses back
	// to the atom's own weight.
	return atomWeight, nil
}

// mulSat returns a*b, saturated to math.MaxUint64 on overflow.
func mulSat(a, b uint64) uint64 {
	if a == 0 || b == 0 {
		return 0
	}
	if a > ^uint64(0)/b {
		return ^uint64(0)
	}
	return a * b
}

// addSat returns a+b, saturated to math.MaxUint64 on overflow.
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
	// Canonical digits allow range comparison without expanding large integers.
	i := 0
	for i < len(s)-1 && s[i] == '0' {
		i++
	}
	return s[i:]
}

// atom = NormalChar / charClass / ( "(" regexp ")" )
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
		// RFC 9485 §4: '.' matches any character except CR / LF.
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

func literalString(r rune) string {
	switch r {
	case '\\', '.', '+', '*', '?', '(', ')', '|', '[', ']', '{', '}',
		'^', '$':
		return "\\" + string(r)
	}
	return string(r)
}

func (t *translator) escape() (string, error) {
	t.advance() // '\'
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
	t.advance() // 'p' or 'P'
	r, _, ok := t.peek()
	if !ok || r != '{' {
		return "", t.errf("expected '{' after \\%c", kind)
	}
	t.advance() // '{'
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
	t.advance() // '}'
	return "\\" + string(kind) + "{" + propName + "}", nil
}

// charClassExpr = "[" [ "^" ] ( "-" / CCE1 ) *CCE1 [ "-" ] "]"
func (t *translator) charClassExpr() error {
	t.advance() // '['
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

// CCE1 = ( CCchar [ "-" CCchar ] ) / charClassEsc
//
// Per RFC 9485 §3, CCchar excludes '\', ']', '-' AND '['.
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
			t.advance() // '\'
			s, err := t.categoryEsc(next)
			if err != nil {
				return "", err
			}
			// \p/\P cannot form a range endpoint; a following '-X'
			// is therefore illegal.
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
	t.advance() // '\'
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
		return "", nil // trailing literal '-' handled by caller
	}
	t.advance() // '-'
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
	// Range validity belongs to I-Regexp validation, not backend error handling.
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

// ---------------------------------------------------------------------------
// IsCategory validation (RFC 9485 §2 ABNF)
// ---------------------------------------------------------------------------
//
// IsCategory  = Letters / Marks / Numbers / Punctuation / Separators /
//               Symbols / Others
// Letters     = "L" [ ( "l" / "m" / "o" / "t" / "u" ) ]
// Marks       = "M" [ ( "c" / "e" / "n" ) ]
// Numbers     = "N" [ ( "d" / "l" / "o" ) ]
// Punctuation = "P" [ ( "c" / "d" / "e" / "f" / "i" / "o" / "s" ) ]
// Separators  = "Z" [ ( "l" / "p" / "s" ) ]
// Symbols     = "S" [ ( "c" / "k" / "m" / "o" ) ]
// Others      = "C" [ ( "c" / "f" / "n" / "o" ) ]

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
