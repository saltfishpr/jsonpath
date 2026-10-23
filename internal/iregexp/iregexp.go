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

// Regexp is a compiled I-Regexp.
type Regexp struct {
	src    string
	search *regexp.Regexp // unanchored (for Search)
	match  *regexp.Regexp // anchored with \A ... \z (for Match)
}

// ErrPatternTooLarge is returned by Compile when the translated RE2
// pattern would exceed the implementation's safety limit. I-Regexp
// itself places no bound on repetition counts; this limit protects
// against adversarial inputs rather than silently returning "no match"
// as earlier revisions did. The actual limit is intentionally large
// (several megabytes of expanded pattern text).
var ErrPatternTooLarge = errors.New("iregexp: translated pattern exceeds safety limit")

// Compile parses and compiles an I-Regexp.
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

// MustCompile is like Compile but panics on error.
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

// String returns the original I-Regexp source.
func (r *Regexp) String() string { return r.src }

// ---------------------------------------------------------------------------
// Translator / validator
// ---------------------------------------------------------------------------
//
// Each parsing function returns:
//   text   — the equivalent RE2 fragment
//   weight — the largest "finite max count" product that would be
//            observed by Go's regexp/syntax.repeatIsValid when walked
//            from this fragment's root. Keeping weight ≤ reWeightLimit
//            guarantees Go accepts any sub-tree we emit; when applying
//            a brace quantifier that would push the product above the
//            limit we physically unroll it instead of writing a {n,m}
//            that RE2 would reject.

const (
	// Go's regexp/syntax enforces a per-expansion limit of 1000.
	reWeightLimit = 1000
	// Hard cap on the translated pattern size, to bound unrolling of
	// pathologically large I-Regexp inputs. ~8 MiB is plenty for any
	// realistic use yet still bounded.
	maxTranslatedBytes = 8 << 20
)

type translator struct {
	src string
	pos int
}

func (t *translator) errf(format string, args ...any) error {
	return fmt.Errorf("iregexp: at offset %d: "+format,
		append([]any{t.pos}, args...)...)
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
	text, _, err := t.regexp()
	if err != nil {
		return "", err
	}
	if t.pos != len(t.src) {
		return "", t.errf("unexpected %q", t.src[t.pos:])
	}
	return text, nil
}

// regexp = branch *( "|" branch )
func (t *translator) regexp() (string, int, error) {
	text, weight, err := t.branch()
	if err != nil {
		return "", 0, err
	}
	r, _, ok := t.peek()
	if !ok || r != '|' {
		return text, weight, nil
	}
	// Accumulate alternations through a Builder to avoid the O(n²)
	// cost of repeated string concatenation when the pattern has many
	// '|' branches.
	var sb strings.Builder
	sb.WriteString(text)
	for {
		r, _, ok := t.peek()
		if !ok || r != '|' {
			break
		}
		t.advance()
		bt, bw, err := t.branch()
		if err != nil {
			return "", 0, err
		}
		sb.WriteByte('|')
		sb.WriteString(bt)
		if bw > weight {
			weight = bw
		}
		if sb.Len() > maxTranslatedBytes {
			return "", 0, ErrPatternTooLarge
		}
	}
	return sb.String(), weight, nil
}

// branch = *piece
func (t *translator) branch() (string, int, error) {
	var sb strings.Builder
	weight := 1
	for {
		r, _, ok := t.peek()
		if !ok || r == '|' || r == ')' {
			return sb.String(), weight, nil
		}
		pt, pw, err := t.piece()
		if err != nil {
			return "", 0, err
		}
		sb.WriteString(pt)
		if pw > weight {
			weight = pw
		}
		if sb.Len() > maxTranslatedBytes {
			return "", 0, ErrPatternTooLarge
		}
	}
}

// piece = atom [ quantifier ]
func (t *translator) piece() (string, int, error) {
	atomText, atomWeight, err := t.atom()
	if err != nil {
		return "", 0, err
	}
	r, _, ok := t.peek()
	if !ok {
		return atomText, atomWeight, nil
	}
	switch r {
	case '?':
		t.advance()
		// OpQuest: Max=1, weight contribution = 1 * atomWeight.
		return atomText + "?", atomWeight, nil
	case '*':
		t.advance()
		// OpStar: Max=-1, Min=0 → repeatIsValid uses m=0 (transparent).
		return atomText + "*", atomWeight, nil
	case '+':
		t.advance()
		// OpPlus: Max=-1, Min=1 → m=1 (transparent).
		return atomText + "+", atomWeight, nil
	case '{':
		return t.braceQuantifier(atomText, atomWeight)
	}
	return atomText, atomWeight, nil
}

// braceQuantifier handles { QuantExact [ "," [ QuantExact ] ] "}".
// It normalises leading zeros in the digit groups and transparently
// unrolls the repetition if emitting {n,m} would push the weight above
// Go RE2's internal limit.
func (t *translator) braceQuantifier(atomText string, atomWeight int) (string, int, error) {
	t.advance() // '{'
	nStr, err := t.readDigits()
	if err != nil {
		return "", 0, err
	}
	if nStr == "" {
		return "", 0, t.errf("expected digits in quantifier")
	}
	n, err := parseCount(nStr)
	if err != nil {
		return "", 0, t.errf("quantifier count too large: %s", nStr)
	}

	r, _, ok := t.peek()
	if !ok {
		return "", 0, t.errf("unterminated quantifier")
	}

	var (
		m         int
		unbounded bool
		hasMax    bool
	)
	if r == ',' {
		t.advance()
		mStr, err := t.readDigits()
		if err != nil {
			return "", 0, err
		}
		if mStr == "" {
			unbounded = true
		} else {
			m, err = parseCount(mStr)
			if err != nil {
				return "", 0, t.errf("quantifier count too large: %s", mStr)
			}
			hasMax = true
		}
		r, _, ok = t.peek()
		if !ok {
			return "", 0, t.errf("unterminated quantifier")
		}
	} else {
		m = n
		hasMax = true
	}
	if r != '}' {
		return "", 0, t.errf("expected '}' in quantifier, got %q", r)
	}
	t.advance()

	if hasMax && n > m {
		return "", 0, t.errf("quantifier range out of order: {%d,%d}", n, m)
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

	newWeight := effective * atomWeight

	// Fast path: current limits already satisfy RE2.
	if newWeight <= reWeightLimit && n <= reWeightLimit &&
		(unbounded || m <= reWeightLimit) {
		var sb strings.Builder
		sb.WriteString(atomText)
		sb.WriteByte('{')
		sb.WriteString(strconv.Itoa(n)) // normalised, no leading zeros
		if unbounded {
			sb.WriteByte(',')
		} else if m != n {
			sb.WriteByte(',')
			sb.WriteString(strconv.Itoa(m))
		}
		sb.WriteByte('}')
		return sb.String(), newWeight, nil
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
	atomLen := uint64(len(atomText))
	estimated := mulSat(uint64(n), atomLen)
	if unbounded {
		estimated = addSat(estimated, addSat(atomLen, 5))
	} else {
		delta := mulSat(uint64(m-n), addSat(atomLen, 5))
		estimated = addSat(estimated, delta)
	}
	if estimated > uint64(maxTranslatedBytes) {
		return "", 0, ErrPatternTooLarge
	}

	var sb strings.Builder
	sb.Grow(int(estimated))
	for i := 0; i < n; i++ {
		sb.WriteString(atomText)
	}
	if unbounded {
		sb.WriteString("(?:")
		sb.WriteString(atomText)
		sb.WriteString(")*")
	} else {
		for i := 0; i < m-n; i++ {
			sb.WriteString("(?:")
			sb.WriteString(atomText)
			sb.WriteString(")?")
		}
	}
	// After unrolling we have a plain concat/star — no new finite
	// repeats surround the atom, so the overall weight collapses back
	// to the atom's own weight.
	return sb.String(), atomWeight, nil
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

func parseCount(s string) (int, error) {
	// Strip leading zeros for robust parsing. "01" → 1, "0000" → 0.
	i := 0
	for i < len(s)-1 && s[i] == '0' {
		i++
	}
	return strconv.Atoi(s[i:])
}

// atom = NormalChar / charClass / ( "(" regexp ")" )
func (t *translator) atom() (string, int, error) {
	r, _, ok := t.peek()
	if !ok {
		return "", 0, t.errf("empty atom")
	}
	switch r {
	case '(':
		t.advance()
		inner, w, err := t.regexp()
		if err != nil {
			return "", 0, err
		}
		r2, _, ok := t.peek()
		if !ok || r2 != ')' {
			return "", 0, t.errf("expected ')'")
		}
		t.advance()
		return "(?:" + inner + ")", w, nil
	case '.':
		t.advance()
		// RFC 9485 §4: '.' matches any character except CR / LF.
		return `[^\n\r]`, 1, nil
	case '\\':
		s, err := t.escape()
		if err != nil {
			return "", 0, err
		}
		return s, 1, nil
	case '[':
		s, err := t.charClassExpr()
		if err != nil {
			return "", 0, err
		}
		return s, 1, nil
	case ')', ']', '|', '?', '*', '+', '{', '}':
		return "", 0, t.errf("unexpected %q", r)
	default:
		t.advance()
		return literalString(r), 1, nil
	}
}

// literalString escapes a rune for safe embedding in RE2.
func literalString(r rune) string {
	switch r {
	case '\\', '.', '+', '*', '?', '(', ')', '|', '[', ']', '{', '}',
		'^', '$':
		return "\\" + string(r)
	}
	return string(r)
}

// escape handles atom-level escapes.
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

// categoryEsc parses \p{IsCategory} / \P{IsCategory}.
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
func (t *translator) charClassExpr() (string, error) {
	t.advance() // '['
	var sb strings.Builder
	sb.WriteByte('[')

	if r, _, ok := t.peek(); ok && r == '^' {
		t.advance()
		sb.WriteByte('^')
	}

	first := true
	for {
		r, _, ok := t.peek()
		if !ok {
			return "", t.errf("unterminated character class")
		}
		if r == ']' {
			if first {
				return "", t.errf("empty character class")
			}
			t.advance()
			sb.WriteByte(']')
			return sb.String(), nil
		}
		if r == '-' {
			if first {
				t.advance()
				sb.WriteString(`\-`)
				first = false
				continue
			}
			if next, ok := t.peekAt(1); ok && next == ']' {
				t.advance()
				sb.WriteString(`\-`)
				continue
			}
			return "", t.errf("unexpected '-' in character class")
		}
		first = false
		cce, err := t.cce1()
		if err != nil {
			return "", err
		}
		sb.WriteString(cce)
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
		s, err := t.classSingleCharEsc()
		if err != nil {
			return "", err
		}
		tail, err := t.maybeRangeTail()
		if err != nil {
			return "", err
		}
		return s + tail, nil
	}

	if r == ']' || r == '-' || r == '[' {
		return "", t.errf("unexpected %q in character class (must be escaped)", r)
	}
	t.advance()
	tail, err := t.maybeRangeTail()
	if err != nil {
		return "", err
	}
	return classLiteral(r) + tail, nil
}

// classSingleCharEsc consumes a SingleCharEsc at the current position.
func (t *translator) classSingleCharEsc() (string, error) {
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
	default:
		return "", t.errf("invalid escape \\%c in character class", r)
	}
}

// maybeRangeTail consumes an optional "-X" range tail.
func (t *translator) maybeRangeTail() (string, error) {
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
		s, err := t.classSingleCharEsc()
		if err != nil {
			return "", err
		}
		return "-" + s, nil
	}
	if r2 == ']' || r2 == '-' || r2 == '[' {
		return "", t.errf("invalid range endpoint %q", r2)
	}
	t.advance()
	return "-" + classLiteral(r2), nil
}

func classLiteral(r rune) string {
	switch r {
	case '\\', ']', '^', '-':
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
