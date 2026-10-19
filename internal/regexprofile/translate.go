// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package regexprofile

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// translate reads pattern as an ECMA-262 u-mode Pattern (ECMA-262
// §22.2.1, with the u flag and no other) and writes the RE2 pattern that
// matches the same strings: every literal as \x{…}, every class as an
// explicit set, and every group as a non-capturing one, since nothing a
// schema asks of a pattern depends on what a group captured.
func translate(pattern string) (string, error) {
	if !utf8.ValidString(pattern) {
		return "", &UnsupportedError{Construct: Syntax}
	}
	p := &parser{src: pattern, names: map[string]bool{}}
	if err := p.disjunction(); err != nil {
		return "", err
	}
	if p.pos < len(p.src) {
		// Only an unmatched ")" stops a top-level disjunction early.
		return "", p.fail(Syntax, p.pos)
	}
	return p.out.String(), nil
}

type parser struct {
	src   string
	pos   int
	out   strings.Builder
	names map[string]bool
}

func (p *parser) fail(c Construct, at int) error {
	return &UnsupportedError{Construct: c, Offset: at}
}

func (p *parser) eof() bool { return p.pos >= len(p.src) }

// peek returns the rune at the read position and its width, or -1 at the
// end.
func (p *parser) peek() (rune, int) {
	if p.eof() {
		return -1, 0
	}
	return utf8.DecodeRuneInString(p.src[p.pos:])
}

func (p *parser) next() rune {
	r, w := p.peek()
	p.pos += w
	return r
}

func (p *parser) lookingAt(s string) bool { return strings.HasPrefix(p.src[p.pos:], s) }

// disjunction reads Alternatives separated by "|" up to a ")" or the end,
// which it leaves unread.
func (p *parser) disjunction() error {
	for {
		if err := p.alternative(); err != nil {
			return err
		}
		if p.eof() || p.src[p.pos] != '|' {
			return nil
		}
		p.pos++
		p.out.WriteByte('|')
	}
}

func (p *parser) alternative() error {
	for !p.eof() && p.src[p.pos] != '|' && p.src[p.pos] != ')' {
		if err := p.term(); err != nil {
			return err
		}
	}
	return nil
}

// term reads an Assertion, or an Atom and its optional Quantifier.
func (p *parser) term() error {
	start := p.pos
	switch p.src[p.pos] {
	case '^', '$':
		p.out.WriteByte(p.src[p.pos])
		p.pos++
		return p.noQuantifier()
	case '\\':
		if p.lookingAt(`\b`) || p.lookingAt(`\B`) {
			p.out.WriteString(p.src[p.pos : p.pos+2])
			p.pos += 2
			return p.noQuantifier()
		}
		p.pos++
		if err := p.atomEscape(start); err != nil {
			return err
		}
	case '(':
		if err := p.group(); err != nil {
			return err
		}
	case '.':
		p.pos++
		p.out.WriteString(dotClass)
	case '[':
		if err := p.class(); err != nil {
			return err
		}
	case '*', '+', '?', '{', '}', ']':
		// Nothing to repeat, or a lone brace or bracket, which u mode
		// refuses.
		return p.fail(Syntax, start)
	default:
		writeLiteral(&p.out, p.next())
	}
	return p.quantifier()
}

// noQuantifier refuses a quantifier after an assertion, which u mode
// does not allow.
func (p *parser) noQuantifier() error {
	if p.eof() {
		return nil
	}
	switch p.src[p.pos] {
	case '*', '+', '?', '{':
		return p.fail(Syntax, p.pos)
	}
	return nil
}

// quantifier reads an optional Quantifier after an atom.
func (p *parser) quantifier() error {
	if p.eof() {
		return nil
	}
	start := p.pos
	switch c := p.src[p.pos]; c {
	case '*', '+', '?':
		p.pos++
		p.out.WriteByte(c)
	case '{':
		p.pos++
		lo, ok := p.decimal()
		if !ok {
			return p.fail(Syntax, start)
		}
		hi, unbounded := lo, false
		if !p.eof() && p.src[p.pos] == ',' {
			p.pos++
			if !p.eof() && p.src[p.pos] == '}' {
				unbounded = true
			} else if hi, ok = p.decimal(); !ok {
				return p.fail(Syntax, start)
			}
		}
		if p.eof() || p.src[p.pos] != '}' {
			return p.fail(Syntax, start)
		}
		p.pos++
		if !unbounded && hi < lo {
			return p.fail(Syntax, start)
		}
		if lo > MaxRepeat || hi > MaxRepeat {
			return p.fail(RepeatBound, start)
		}
		switch {
		case unbounded:
			fmt.Fprintf(&p.out, "{%d,}", lo)
		case strings.Contains(p.src[start:p.pos], ","):
			fmt.Fprintf(&p.out, "{%d,%d}", lo, hi)
		default:
			fmt.Fprintf(&p.out, "{%d}", lo)
		}
	default:
		return nil
	}
	if !p.eof() && p.src[p.pos] == '?' {
		p.pos++
		p.out.WriteByte('?')
	}
	return nil
}

// decimal reads DecimalDigits, saturating above MaxRepeat so that a bound
// of any length is read without overflow.
func (p *parser) decimal() (int, bool) {
	start, n := p.pos, 0
	for !p.eof() && p.src[p.pos] >= '0' && p.src[p.pos] <= '9' {
		if n <= MaxRepeat {
			n = n*10 + int(p.src[p.pos]-'0')
		}
		p.pos++
	}
	return n, p.pos > start
}

// group reads a parenthesised group; the read position is at "(".
func (p *parser) group() error {
	start := p.pos
	p.pos++
	if p.lookingAt("?") {
		switch {
		case p.lookingAt("?:"):
			p.pos += 2
		case p.lookingAt("?=") || p.lookingAt("?!"):
			return p.fail(Lookahead, start)
		case p.lookingAt("?<=") || p.lookingAt("?<!"):
			return p.fail(Lookbehind, start)
		case p.lookingAt("?<"):
			p.pos += 2
			name, err := p.groupName(start)
			if err != nil {
				return err
			}
			if p.names[name] {
				return p.fail(Syntax, start)
			}
			p.names[name] = true
		case p.lookingAt("?P"):
			return p.fail(RE2Syntax, start)
		default:
			if r, _ := utf8.DecodeRuneInString(p.src[p.pos+1:]); isFlagStart(r) {
				return p.fail(RE2Syntax, start)
			}
			return p.fail(Syntax, start)
		}
	}
	p.out.WriteString("(?:")
	if err := p.disjunction(); err != nil {
		return err
	}
	if p.eof() {
		return p.fail(Syntax, start)
	}
	p.pos++
	p.out.WriteByte(')')
	return nil
}

// isFlagStart reports whether r begins RE2's inline flags, (?i), (?-s:…)
// and the like, or ECMA-262's modifiers, which RE2 spells the same way and
// the profile refuses with them.
func isFlagStart(r rune) bool {
	return r == '-' || r == 'i' || r == 'm' || r == 's' || r == 'U'
}

// groupName reads a RegExpIdentifierName and its closing ">".
func (p *parser) groupName(start int) (string, error) {
	var b strings.Builder
	for first := true; ; first = false {
		if p.eof() {
			return "", p.fail(Syntax, start)
		}
		r := p.next()
		if r == '>' {
			if first {
				return "", p.fail(Syntax, start)
			}
			return b.String(), nil
		}
		if r == '\\' {
			if p.next() != 'u' {
				return "", p.fail(Syntax, start)
			}
			var err error
			if r, err = p.unicodeEscape(start); err != nil {
				return "", err
			}
		}
		if !isIdentifierRune(r, first) {
			return "", p.fail(Syntax, start)
		}
		b.WriteRune(r)
	}
}

// isIdentifierRune approximates ECMA-262's IdentifierStartChar and
// IdentifierPartChar by Unicode categories: letters and letter numbers
// start a name, and marks, digits and connector punctuation continue one.
func isIdentifierRune(r rune, first bool) bool {
	if r == '$' || r == '_' || unicode.IsLetter(r) || unicode.Is(unicode.Nl, r) {
		return true
	}
	if first {
		return false
	}
	return r == '\u200C' || r == '\u200D' || unicode.In(r, unicode.Mn, unicode.Mc, unicode.Nd, unicode.Pc)
}

// atomEscape reads what follows "\" outside a class; start is the
// position of the "\".
func (p *parser) atomEscape(start int) error {
	if p.eof() {
		return p.fail(Syntax, start)
	}
	r := p.next()
	switch {
	case r >= '1' && r <= '9', r == 'k':
		return p.fail(Backreference, start)
	case r == 'd' || r == 'D' || r == 'w' || r == 'W' || r == 's' || r == 'S':
		p.out.WriteString(classOf(escapeSet(r), false))
		return nil
	case r == 'p' || r == 'P':
		item, err := p.property(start, r == 'P')
		if err != nil {
			return err
		}
		p.out.WriteString("[" + item + "]")
		return nil
	}
	c, err := p.characterEscape(start, r)
	if err != nil {
		return err
	}
	writeLiteral(&p.out, c)
	return nil
}

// characterEscape reads a CharacterEscape whose first character, r, has
// been read; start is the position of the "\".
func (p *parser) characterEscape(start int, r rune) (rune, error) {
	switch r {
	case 'f':
		return '\f', nil
	case 'n':
		return '\n', nil
	case 'r':
		return '\r', nil
	case 't':
		return '\t', nil
	case 'v':
		return '\v', nil
	case 'c':
		c, _ := p.peek()
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
			p.pos++
			return c % 32, nil
		}
		return 0, p.fail(Syntax, start)
	case '0':
		if c, _ := p.peek(); c >= '0' && c <= '9' {
			return 0, p.fail(Syntax, start)
		}
		return 0, nil
	case 'x':
		v, ok := p.hex(2)
		if !ok {
			return 0, p.fail(Syntax, start)
		}
		return v, nil
	case 'u':
		return p.unicodeEscape(start)
	case '^', '$', '\\', '.', '*', '+', '?', '(', ')', '[', ']', '{', '}', '|', '/':
		return r, nil
	case 'A', 'z', 'Z', 'Q', 'E', 'C', 'a':
		// \A, \z, \Q…\E, \C and \a are RE2's; u mode gives them no
		// meaning.
		return 0, p.fail(RE2Syntax, start)
	}
	return 0, p.fail(Syntax, start)
}

// hex reads exactly n hexadecimal digits.
func (p *parser) hex(n int) (rune, bool) {
	if len(p.src)-p.pos < n {
		return 0, false
	}
	var v rune
	for i := 0; i < n; i++ {
		d, ok := hexDigit(p.src[p.pos+i])
		if !ok {
			return 0, false
		}
		v = v<<4 | d
	}
	p.pos += n
	return v, true
}

func hexDigit(c byte) (rune, bool) {
	switch {
	case c >= '0' && c <= '9':
		return rune(c - '0'), true
	case c >= 'a' && c <= 'f':
		return rune(c-'a') + 10, true
	case c >= 'A' && c <= 'F':
		return rune(c-'A') + 10, true
	}
	return 0, false
}

func isSurrogate(r rune) bool { return r >= 0xD800 && r <= 0xDFFF }

// unicodeEscape reads what follows "\u": \u{…}, a surrogate pair written
// as two \uXXXX, or one \uXXXX. A surrogate that is not half of a pair is
// refused.
func (p *parser) unicodeEscape(start int) (rune, error) {
	if !p.eof() && p.src[p.pos] == '{' {
		p.pos++
		var v rune
		digits := 0
		for !p.eof() && p.src[p.pos] != '}' {
			d, ok := hexDigit(p.src[p.pos])
			if !ok {
				return 0, p.fail(Syntax, start)
			}
			v = v<<4 | d
			if v > unicode.MaxRune {
				return 0, p.fail(Syntax, start)
			}
			digits++
			p.pos++
		}
		if p.eof() || digits == 0 {
			return 0, p.fail(Syntax, start)
		}
		p.pos++
		if isSurrogate(v) {
			return 0, p.fail(LoneSurrogate, start)
		}
		return v, nil
	}
	v, ok := p.hex(4)
	if !ok {
		return 0, p.fail(Syntax, start)
	}
	if v >= 0xD800 && v <= 0xDBFF && p.lookingAt(`\u`) {
		save := p.pos
		p.pos += 2
		if lo, ok := p.hex(4); ok && lo >= 0xDC00 && lo <= 0xDFFF {
			return (v-0xD800)<<10 + (lo - 0xDC00) + 0x10000, nil
		}
		p.pos = save
	}
	if isSurrogate(v) {
		return 0, p.fail(LoneSurrogate, start)
	}
	return v, nil
}

// property reads the {…} of a \p or \P whose letter has been read and
// returns the RE2 class item that matches it. ECMA-262 names a general
// category alone or as General_Category=, and a script as Script=; RE2
// names both alone, a script by its long name only. Anything else is a
// form RE2 cannot express.
func (p *parser) property(start int, negated bool) (string, error) {
	if p.eof() || p.src[p.pos] != '{' {
		// \pL is RE2's one-letter form; u mode requires the braces.
		return "", p.fail(RE2Syntax, start)
	}
	end := strings.IndexByte(p.src[p.pos:], '}')
	if end < 0 {
		return "", p.fail(Syntax, start)
	}
	body := p.src[p.pos+1 : p.pos+end]
	p.pos += end + 1
	for _, c := range []byte(body) {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '=') {
			return "", p.fail(Syntax, start)
		}
	}
	key, name, hasKey := strings.Cut(body, "=")
	if !hasKey {
		key, name = "General_Category", body
	}
	var ok bool
	switch key {
	case "General_Category", "gc":
		_, ok = unicode.Categories[name]
	case "Script", "sc":
		_, ok = unicode.Scripts[name]
	}
	if !ok || name == "" {
		return "", p.fail(UnicodeProperty, start)
	}
	if negated {
		return `\P{` + name + `}`, nil
	}
	return `\p{` + name + `}`, nil
}

// class reads a CharacterClass; the read position is at "[".
func (p *parser) class() error {
	start := p.pos
	p.pos++
	negated := false
	if !p.eof() && p.src[p.pos] == '^' {
		negated = true
		p.pos++
	}
	var set []rune // ranges, as lo, hi pairs
	var props []string
	for {
		if p.eof() {
			return p.fail(Syntax, start)
		}
		if p.src[p.pos] == ']' {
			p.pos++
			break
		}
		lo, loSet, loProp, err := p.classAtom()
		if err != nil {
			return err
		}
		if p.lookingAt("-") && !p.lookingAt("-]") {
			dash := p.pos
			p.pos++
			hi, hiSet, hiProp, err := p.classAtom()
			if err != nil {
				return err
			}
			if loSet != nil || hiSet != nil || loProp != "" || hiProp != "" || hi < lo {
				return p.fail(Syntax, dash)
			}
			set = append(set, lo, hi)
			continue
		}
		switch {
		case loProp != "":
			props = append(props, loProp)
		case loSet != nil:
			set = append(set, loSet...)
		default:
			set = append(set, lo, lo)
		}
	}
	set = normalize(set)
	if len(props) == 0 && len(set) == 0 {
		// [] matches nothing and [^] any code point.
		p.out.WriteString(classOf(nil, negated))
		return nil
	}
	p.out.WriteString(classItems(set, props, negated))
	return nil
}

// classAtom reads one ClassAtom: a code point, a class escape's set, or a
// property item.
func (p *parser) classAtom() (rune, []rune, string, error) {
	start := p.pos
	r := p.next()
	if r != '\\' {
		return r, nil, "", nil
	}
	if p.eof() {
		return 0, nil, "", p.fail(Syntax, start)
	}
	r = p.next()
	switch {
	case r == 'b':
		return '\b', nil, "", nil
	case r == '-':
		return '-', nil, "", nil
	case r == 'd' || r == 'D' || r == 'w' || r == 'W' || r == 's' || r == 'S':
		return 0, escapeSet(r), "", nil
	case r == 'p' || r == 'P':
		item, err := p.property(start, r == 'P')
		return 0, nil, item, err
	case r >= '1' && r <= '9', r == 'k', r == 'B':
		return 0, nil, "", p.fail(Syntax, start)
	}
	c, err := p.characterEscape(start, r)
	return c, nil, "", err
}

// The sets ADR-009 D10 item 6 translates, as sorted lo, hi pairs.
var (
	digitSet = []rune{'0', '9'}
	wordSet  = []rune{'0', '9', 'A', 'Z', '_', '_', 'a', 'z'}
	// spaceSet is ECMA-262's WhiteSpace and LineTerminator: tab, line
	// feed, vertical tab, form feed, carriage return, the Zs category of
	// Unicode 15.0.0, the line and paragraph separators, and U+FEFF.
	spaceSet = []rune{
		'\t', '\r', ' ', ' ', 0xA0, 0xA0, 0x1680, 0x1680, 0x2000, 0x200A,
		0x2028, 0x2029, 0x202F, 0x202F, 0x205F, 0x205F, 0x3000, 0x3000, 0xFEFF, 0xFEFF,
	}
	// lineTerminators are what "." does not match.
	lineTerminators = []rune{'\n', '\n', '\r', '\r', 0x2028, 0x2029}
)

// dotClass is ".": any code point but a line terminator.
var dotClass = classOf(complement(lineTerminators), false)

func escapeSet(r rune) []rune {
	switch r {
	case 'd':
		return digitSet
	case 'D':
		return complement(digitSet)
	case 'w':
		return wordSet
	case 'W':
		return complement(wordSet)
	case 's':
		return spaceSet
	default:
		return complement(spaceSet)
	}
}

// complement returns the code points of 0 to U+10FFFF not in the sorted,
// disjoint ranges set.
func complement(set []rune) []rune {
	var out []rune
	next := rune(0)
	for i := 0; i < len(set); i += 2 {
		if set[i] > next {
			out = append(out, next, set[i]-1)
		}
		next = set[i+1] + 1
	}
	if next <= unicode.MaxRune {
		out = append(out, next, unicode.MaxRune)
	}
	return out
}

// normalize sorts ranges and merges those that overlap or touch.
func normalize(set []rune) []rune {
	type rng struct{ lo, hi rune }
	rs := make([]rng, 0, len(set)/2)
	for i := 0; i < len(set); i += 2 {
		rs = append(rs, rng{set[i], set[i+1]})
	}
	slices.SortFunc(rs, func(a, b rng) int { return int(a.lo - b.lo) })
	var out []rune
	for _, r := range rs {
		if n := len(out); n > 0 && r.lo <= out[n-1]+1 {
			out[n-1] = max(out[n-1], r.hi)
			continue
		}
		out = append(out, r.lo, r.hi)
	}
	return out
}

// classOf writes the ranges of set as an RE2 class. An empty set is the
// class that matches nothing, or, negated, every code point.
func classOf(set []rune, negated bool) string {
	if len(set) == 0 {
		set = []rune{0, unicode.MaxRune}
		negated = !negated
	}
	return classItems(set, nil, negated)
}

func classItems(set []rune, props []string, negated bool) string {
	var b strings.Builder
	b.WriteByte('[')
	if negated {
		b.WriteByte('^')
	}
	for i := 0; i < len(set); i += 2 {
		writeLiteral(&b, set[i])
		if set[i+1] != set[i] {
			b.WriteByte('-')
			writeLiteral(&b, set[i+1])
		}
	}
	for _, pr := range props {
		b.WriteString(pr)
	}
	b.WriteByte(']')
	return b.String()
}

// writeLiteral writes r so that RE2 reads it as itself: an ASCII letter or
// digit as it is, anything else as \x{…}.
func writeLiteral(b *strings.Builder, r rune) {
	if r < utf8.RuneSelf && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
		b.WriteRune(r)
		return
	}
	fmt.Fprintf(b, `\x{%X}`, r)
}
