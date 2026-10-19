// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package jsonx

import (
	"fmt"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
)

// parser is one reading of one text: a recursive descent over RFC 8259's
// grammar whose depth value-depth bounds before it recurses.
type parser struct {
	src      []byte
	from     Source
	lim      limits.Set
	lines    *lineIndex
	i        int // the next byte to read
	depth    int // arrays and objects open
	findings []finding.Finding
	stopped  bool // a finding the reading cannot get past was reported
}

// path is where in the document a value is, as the frames of the arrays
// and objects above it. Its JSON Pointer is built only when a finding
// needs it, and once per frame, so reading costs nothing per value for the
// pointers no finding names.
type path struct {
	parent *path
	token  string
	ptr    string
	built  bool
}

func (p *path) pointer() string {
	if p == nil {
		return ""
	}
	if !p.built {
		p.ptr, p.built = p.parent.pointer()+finding.Pointer(p.token), true
	}
	return p.ptr
}

func (p *parser) document() (value.Value, *node) {
	if n := int64(len(p.src)); n > p.lim.Value(limits.InputSize) {
		p.limitExceeded(limits.InputSize, 0, "", false, n)
		return nil, nil
	}
	if !p.encoding() {
		return nil, nil
	}
	p.i = p.lines.starts[0]
	p.space()
	var root node
	v := p.value(nil, &root)
	if p.stopped {
		return nil, &root
	}
	p.space()
	if p.i < len(p.src) {
		p.report(CodeTrailingContent, p.at(p.i, "", false), nil)
		return nil, &root
	}
	return v, &root
}

// encoding refuses text that is not UTF-8 (ADR-003 D8 item 1), at the
// first byte that is not, and reports whether the text is UTF-8.
func (p *parser) encoding() bool {
	src := p.src
	if len(src) >= 2 && (src[0] == 0xFE && src[1] == 0xFF || src[0] == 0xFF && src[1] == 0xFE) {
		p.report(CodeEncodingInvalid, p.at(0, "", false), finding.Params{"reason": finding.String(reasonUTF16), "offset": finding.Int(0)})
		return false
	}
	if utf8.Valid(src) {
		return true
	}
	for i := 0; i < len(src); {
		r, n := utf8.DecodeRune(src[i:])
		if r == utf8.RuneError && n == 1 {
			p.report(CodeEncodingInvalid, p.at(i, "", false), finding.Params{"reason": finding.String(reasonInvalidUTF8), "offset": finding.Int(int64(i))})
			return false
		}
		i += n
	}
	return true
}

// space skips RFC 8259's whitespace: space, tab, line feed and carriage
// return.
func (p *parser) space() {
	for p.i < len(p.src) {
		switch p.src[p.i] {
		case ' ', '\t', '\n', '\r':
			p.i++
		default:
			return
		}
	}
}

// found names the class of the byte at the reading position, for a
// json.syntax finding.
func (p *parser) found() string {
	switch {
	case p.i >= len(p.src):
		return foundEnd
	case p.src[p.i] < 0x20 || p.src[p.i] == 0x7F:
		return foundControl
	}
	return foundCharacter
}

// value reads the value at the reading position into n, at pa.
func (p *parser) value(pa *path, n *node) value.Value {
	n.off = p.i
	if p.i >= len(p.src) {
		p.syntaxError(p.i, pa.pointer(), wantValue, foundEnd)
		return nil
	}
	switch c := p.src[p.i]; {
	case c == '{':
		return p.object(pa, n)
	case c == '[':
		return p.array(pa, n)
	case c == '"':
		s, ok := p.str(pa)
		if !ok || !p.clean() {
			return nil
		}
		v, err := value.NewString(s)
		if err != nil {
			panic(fmt.Sprintf("jsonx: a string read as scalar values is refused: %v", err))
		}
		return v
	case c == '-' || '0' <= c && c <= '9':
		return p.number(pa)
	case c == 't':
		return p.literal(pa, "true", value.Bool(true))
	case c == 'f':
		return p.literal(pa, "false", value.Bool(false))
	case c == 'n':
		return p.literal(pa, "null", value.Null{})
	}
	p.syntaxError(p.i, pa.pointer(), wantValue, p.found())
	return nil
}

// clean reports whether no finding has been reported, so a value read is
// still worth building.
func (p *parser) clean() bool { return len(p.findings) == 0 }

func (p *parser) literal(pa *path, word string, v value.Value) value.Value {
	for k := 0; k < len(word); k++ {
		if p.i+k >= len(p.src) || p.src[p.i+k] != word[k] {
			p.i += k
			p.syntaxError(p.i, pa.pointer(), wantValue, p.found())
			return nil
		}
	}
	p.i += len(word)
	return v
}

// open enters an array or object, refusing one value-depth does not allow
// before reading anything inside it (ADR-009 D10 item 2).
func (p *parser) open(pa *path) bool {
	p.depth++
	if int64(p.depth) > p.lim.Value(limits.ValueDepth) {
		p.limitExceeded(limits.ValueDepth, p.i, pa.pointer(), true, -1)
		p.stopped = true
		return false
	}
	p.i++
	p.space()
	return true
}

// next reads what follows a member or an element: a comma, reporting true,
// or the closing bracket, reporting false. Anything else stops the
// reading.
func (p *parser) next(pa *path, closing byte) bool {
	p.space()
	if p.i < len(p.src) {
		switch p.src[p.i] {
		case ',':
			p.i++
			p.space()
			return true
		case closing:
			p.i++
			return false
		}
	}
	p.syntaxError(p.i, pa.pointer(), fmt.Sprintf(wantCommaOrClose, closing), p.found())
	return false
}

func (p *parser) array(pa *path, n *node) value.Value {
	if !p.open(pa) {
		return nil
	}
	defer func() { p.depth-- }()
	var elems []value.Value
	n.elems = []node{}
	if p.i < len(p.src) && p.src[p.i] == ']' {
		p.i++
		return value.Array{}
	}
	for {
		n.elems = append(n.elems, node{})
		el := &path{parent: pa, token: strconv.Itoa(len(n.elems) - 1)}
		v := p.value(el, &n.elems[len(n.elems)-1])
		if p.stopped {
			return nil
		}
		if p.clean() {
			elems = append(elems, v)
		}
		if !p.next(pa, ']') {
			break
		}
	}
	if p.stopped || !p.clean() {
		return nil
	}
	a, err := value.NewArray(elems...)
	if err != nil {
		panic(fmt.Sprintf("jsonx: an array of read values is refused: %v", err))
	}
	return a
}

func (p *parser) object(pa *path, n *node) value.Value {
	if !p.open(pa) {
		return nil
	}
	defer func() { p.depth-- }()
	defer n.sortMembers()
	var members []value.Member
	n.members = []member{}
	first := map[string]int{} // each name's first key offset
	if p.i < len(p.src) && p.src[p.i] == '}' {
		p.i++
		return value.Object{}
	}
	for {
		if p.i >= len(p.src) || p.src[p.i] != '"' {
			p.syntaxError(p.i, pa.pointer(), wantName, p.found())
			return nil
		}
		keyOff := p.i
		name, ok := p.str(pa)
		if p.stopped {
			return nil
		}
		mp := &path{parent: pa, token: name}
		p.space()
		if p.i >= len(p.src) || p.src[p.i] != ':' {
			p.syntaxError(p.i, mp.pointer(), wantColon, p.found())
			return nil
		}
		p.i++
		p.space()
		var vn node
		v := p.value(mp, &vn)
		if p.stopped {
			return nil
		}
		if at, dup := first[name]; dup && ok {
			first := p.at(at, mp.pointer(), true)
			p.report(CodeDuplicateName, p.at(keyOff, mp.pointer(), true), finding.Params{"name": finding.String(name)},
				finding.WithRelated(finding.Related{Location: *first, Note: noteFirst}))
		} else if ok {
			first[name] = keyOff
			n.members = append(n.members, member{name: name, keyOff: keyOff, node: vn})
			if p.clean() {
				members = append(members, value.Member{Name: name, Value: v})
			}
		}
		if !p.next(pa, '}') {
			break
		}
	}
	if p.stopped || !p.clean() {
		return nil
	}
	o, err := value.NewObject(members...)
	if err != nil {
		panic(fmt.Sprintf("jsonx: an object of read members is refused: %v", err))
	}
	return o
}

// number reads one number, keeping its literal text (ADR-003 D8 item 5).
func (p *parser) number(pa *path) value.Value {
	start := p.i
	if p.src[p.i] == '-' {
		p.i++
	}
	switch {
	case p.i < len(p.src) && p.src[p.i] == '0':
		p.i++
		if p.i < len(p.src) && '0' <= p.src[p.i] && p.src[p.i] <= '9' {
			p.syntaxError(start, pa.pointer(), wantValue, foundLeadZero)
			return nil
		}
	case p.i < len(p.src) && '1' <= p.src[p.i] && p.src[p.i] <= '9':
		p.digits()
	default:
		p.syntaxError(p.i, pa.pointer(), wantDigit, p.found())
		return nil
	}
	if p.i < len(p.src) && p.src[p.i] == '.' {
		p.i++
		if !p.digits() {
			p.syntaxError(p.i, pa.pointer(), wantDigit, p.found())
			return nil
		}
	}
	if p.i < len(p.src) && (p.src[p.i] == 'e' || p.src[p.i] == 'E') {
		p.i++
		if p.i < len(p.src) && (p.src[p.i] == '+' || p.src[p.i] == '-') {
			p.i++
		}
		if !p.digits() {
			p.syntaxError(p.i, pa.pointer(), wantDigit, p.found())
			return nil
		}
	}
	lit := string(p.src[start:p.i])
	if n := int64(len(lit)); n > p.lim.Value(limits.NumberLength) {
		p.limitExceeded(limits.NumberLength, start, pa.pointer(), true, n)
		return nil
	}
	num, err := value.ParseNumber(lit)
	switch {
	case err != nil:
		// The grammar is RFC 8259's, so a refusal is an overflow.
		p.report(CodeNumberOutOfRange, p.at(start, pa.pointer(), true), finding.Params{"reason": finding.String(reasonOverflow)})
		return nil
	case num.WrittenAsInteger():
		if _, safe := num.SafeInt(); !safe {
			p.report(CodeNumberOutOfRange, p.at(start, pa.pointer(), true), finding.Params{"reason": finding.String(reasonUnsafeInteger)})
			return nil
		}
	}
	return num
}

// digits skips a run of ASCII digits and reports whether there was one.
func (p *parser) digits() bool {
	start := p.i
	for p.i < len(p.src) && '0' <= p.src[p.i] && p.src[p.i] <= '9' {
		p.i++
	}
	return p.i > start
}

// str reads one string, a value or a member name, at pa. ok is false when
// it held an escaped unpaired surrogate or a noncharacter, which are
// reported and do not stop the reading (ADR-003 D8 item 4, RFC 7493 §2.1);
// the reading stops on a syntax error.
func (p *parser) str(pa *path) (s string, ok bool) {
	start := p.i
	p.i++ // the opening quote
	var b []byte
	ok = true
	bad := func(code finding.Code) {
		if ok {
			p.report(code, p.at(start, pa.pointer(), true), nil)
		}
		ok = false
	}
	for {
		if p.i >= len(p.src) {
			p.syntaxError(p.i, pa.pointer(), wantStringEnd, foundEnd)
			return "", false
		}
		c := p.src[p.i]
		switch {
		case c == '"':
			p.i++
			return string(b), ok
		case c < 0x20:
			p.syntaxError(p.i, pa.pointer(), wantStringEnd, foundControl)
			return "", false
		case c == '\\':
			r, okEsc := p.escape(pa)
			if !okEsc {
				return "", false
			}
			switch {
			case utf16.IsSurrogate(r):
				bad(CodeLoneSurrogate)
			case noncharacter(r):
				bad(CodeNoncharacter)
			default:
				b = utf8.AppendRune(b, r)
			}
		case c < utf8.RuneSelf:
			b = append(b, c)
			p.i++
		default:
			// The text is valid UTF-8, checked before reading.
			r, n := utf8.DecodeRune(p.src[p.i:])
			if noncharacter(r) {
				bad(CodeNoncharacter)
			}
			b = append(b, p.src[p.i:p.i+n]...)
			p.i += n
		}
	}
}

// escape reads one escape sequence at the backslash, returning the code
// point it writes: a surrogate pair is one code point, and an unpaired
// surrogate is returned as itself for the caller to refuse.
func (p *parser) escape(pa *path) (rune, bool) {
	at := p.i
	p.i++
	if p.i >= len(p.src) {
		p.syntaxError(p.i, pa.pointer(), wantStringEnd, foundEnd)
		return 0, false
	}
	c := p.src[p.i]
	p.i++
	switch c {
	case '"', '\\', '/':
		return rune(c), true
	case 'b':
		return '\b', true
	case 'f':
		return '\f', true
	case 'n':
		return '\n', true
	case 'r':
		return '\r', true
	case 't':
		return '\t', true
	case 'u':
		r, ok := p.hex4(pa)
		if !ok {
			return 0, false
		}
		if utf16.IsSurrogate(r) && r < 0xDC00 && p.i+1 < len(p.src) && p.src[p.i] == '\\' && p.src[p.i+1] == 'u' {
			save := p.i
			p.i += 2
			lo, ok := p.hex4(pa)
			if !ok {
				return 0, false
			}
			if 0xDC00 <= lo && lo <= 0xDFFF {
				return utf16.DecodeRune(r, lo), true
			}
			// Not a low surrogate: the high one is unpaired, and the
			// escape after it is read on its own.
			p.i = save
		}
		return r, true
	}
	p.syntaxError(at, pa.pointer(), wantStringEnd, foundEscape)
	return 0, false
}

// hex4 reads the four hexadecimal digits of a \u escape.
func (p *parser) hex4(pa *path) (rune, bool) {
	var r rune
	for k := 0; k < 4; k++ {
		if p.i >= len(p.src) {
			p.syntaxError(p.i, pa.pointer(), wantHex, foundEnd)
			return 0, false
		}
		c := p.src[p.i]
		var d byte
		switch {
		case '0' <= c && c <= '9':
			d = c - '0'
		case 'a' <= c && c <= 'f':
			d = c - 'a' + 10
		case 'A' <= c && c <= 'F':
			d = c - 'A' + 10
		default:
			p.syntaxError(p.i, pa.pointer(), wantHex, foundCharacter)
			return 0, false
		}
		r = r<<4 | rune(d)
		p.i++
	}
	return r, true
}

// noncharacter reports whether r is one of Unicode's 66 noncharacters:
// U+FDD0 to U+FDEF, and the last two code points of every plane.
func noncharacter(r rune) bool {
	return 0xFDD0 <= r && r <= 0xFDEF || r&0xFFFE == 0xFFFE
}
