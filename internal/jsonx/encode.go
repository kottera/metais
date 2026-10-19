// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package jsonx

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/gowebpki/jcs"

	"github.com/kottera/metais/internal/liquid/value"
)

// MaxIndent is the largest indent the json filter takes (ADR-004 D10 item
// 1).
const MaxIndent = 8

// ErrIndent is returned for an indent outside 0 to MaxIndent.
var ErrIndent = errors.New("indent is not an integer from 1 to 8")

// Encode writes v as the json filter does (ADR-004 D10 item 1). With
// indent 0 it is RFC 8785's text: members in canonical order, strings
// escaped as RFC 8785 §3.2.2.2 says, numbers as ECMAScript's
// Number::toString writes them, no whitespace; that is byte-identical to
// Canonical. With an indent of 1 to MaxIndent it pretty-prints: a line
// feed and indent more spaces after the [ or { of a non-empty array or
// object and after each comma, ": " between a name and its value, the
// closing bracket on its own line at the enclosing indentation, [] and {}
// for empty ones, and no final line feed.
//
// Every byte is written here, not by a library (ADR-010 D3 item 2).
func Encode(v value.Value, indent int) ([]byte, error) {
	if indent < 0 || indent > MaxIndent {
		return nil, fmt.Errorf("%w: %d", ErrIndent, indent)
	}
	e := encoder{indent: indent, number: FormatNumber}
	e.value(v, 0)
	return []byte(e.b.String()), nil
}

// EncodeInput writes v as Encode does, with every number written by
// FormatInputNumber: text the JSON reader reads back as v. Encode writes
// an integral number of magnitude above 2^53−1 and below 1e21 as an
// integer literal, which ADR-003 D8 item 5 refuses as input; EncodeInput
// spells it with an exponent. It is the writer of a JSON input file, such
// as the sample generator's, not the json filter's.
func EncodeInput(v value.Value, indent int) ([]byte, error) {
	if indent < 0 || indent > MaxIndent {
		return nil, fmt.Errorf("%w: %d", ErrIndent, indent)
	}
	e := encoder{indent: indent, number: FormatInputNumber}
	e.value(v, 0)
	return []byte(e.b.String()), nil
}

type encoder struct {
	b      strings.Builder
	indent int
	number func(float64) string
}

func (e *encoder) newline(depth int) {
	if e.indent == 0 {
		return
	}
	e.b.WriteByte('\n')
	for range depth * e.indent {
		e.b.WriteByte(' ')
	}
}

func (e *encoder) value(v value.Value, depth int) {
	switch v := v.(type) {
	case value.Null:
		e.b.WriteString("null")
	case value.Bool:
		e.b.WriteString(strconv.FormatBool(bool(v)))
	case value.Number:
		e.b.WriteString(e.number(v.Float64()))
	case value.String:
		writeString(&e.b, v.Text())
	case value.Array:
		if v.Len() == 0 {
			e.b.WriteString("[]")
			return
		}
		e.b.WriteByte('[')
		for i, el := range v.All() {
			if i > 0 {
				e.b.WriteByte(',')
			}
			e.newline(depth + 1)
			e.value(el, depth+1)
		}
		e.newline(depth)
		e.b.WriteByte(']')
	case value.Object:
		if v.Len() == 0 {
			e.b.WriteString("{}")
			return
		}
		e.b.WriteByte('{')
		first := true
		for name, m := range v.All() {
			if !first {
				e.b.WriteByte(',')
			}
			first = false
			e.newline(depth + 1)
			writeString(&e.b, name)
			e.b.WriteByte(':')
			if e.indent > 0 {
				e.b.WriteByte(' ')
			}
			e.value(m, depth+1)
		}
		e.newline(depth)
		e.b.WriteByte('}')
	default:
		// value's interface is sealed to the six kinds above.
		panic(fmt.Sprintf("jsonx: %T is not a value of the data model", v))
	}
}

// writeString writes s as RFC 8785 §3.2.2.2 does: " and \ escaped with a
// backslash, U+0008, U+0009, U+000A, U+000C and U+000D as \b, \t, \n, \f
// and \r, every other code point below U+0020 as \u00xx in lowercase hex,
// and everything else as itself.
func writeString(b *strings.Builder, s string) {
	const hex = "0123456789abcdef"
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"' || c == '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		case c == '\b':
			b.WriteString(`\b`)
		case c == '\t':
			b.WriteString(`\t`)
		case c == '\n':
			b.WriteString(`\n`)
		case c == '\f':
			b.WriteString(`\f`)
		case c == '\r':
			b.WriteString(`\r`)
		case c < 0x20:
			b.WriteString(`\u00`)
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&0xF])
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
}

// FormatNumber writes f as ECMAScript's Number::toString does (ECMA-262
// §6.1.6.1.20), which RFC 8785 §3.2.2.3 adopts: the shortest digits that
// round-trip, in fixed notation when the decimal exponent is from -6 to
// 20 and in exponent notation, e+ or e-, otherwise; zero, negative zero
// included, is 0. f is finite, as every number of the data model is.
func FormatNumber(f float64) string {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		panic(fmt.Sprintf("jsonx: %v is not a number of the data model", f))
	}
	if f == 0 {
		return "0"
	}
	var b strings.Builder
	if f < 0 {
		b.WriteByte('-')
		f = -f
	}
	// 'e' with -1 writes the shortest round-trip digits as d.ddde±xx.
	sci := strconv.FormatFloat(f, 'e', -1, 64)
	mant, exp, _ := strings.Cut(sci, "e")
	digits := strings.Replace(mant, ".", "", 1)
	x, err := strconv.Atoi(exp)
	if err != nil {
		panic(fmt.Sprintf("jsonx: strconv wrote exponent %q", exp))
	}
	k, n := len(digits), x+1 // ECMA-262's k and n: digits, and the point's place
	switch {
	case k <= n && n <= 21:
		b.WriteString(digits)
		b.WriteString(strings.Repeat("0", n-k))
	case 0 < n && n <= 21:
		b.WriteString(digits[:n])
		b.WriteByte('.')
		b.WriteString(digits[n:])
	case -6 < n && n <= 0:
		b.WriteString("0.")
		b.WriteString(strings.Repeat("0", -n))
		b.WriteString(digits)
	default:
		b.WriteByte(digits[0])
		if k > 1 {
			b.WriteByte('.')
			b.WriteString(digits[1:])
		}
		b.WriteByte('e')
		if n-1 >= 0 {
			b.WriteByte('+')
		}
		b.WriteString(strconv.Itoa(n - 1))
	}
	return b.String()
}

// FormatInputNumber is f as FormatNumber writes it, except for an
// integral f of magnitude above 2^53−1 and below 1e21. ECMAScript writes
// that in fixed notation, as an integer literal, which ADR-003 D8 item 5
// refuses for an input; so it is written in the exponent notation
// ECMAScript uses from 1e21, the shortest digits that round-trip, a point
// after the first, and e+ with the exponent, 15 to 20. Every reader of
// an input reads the result back as f.
func FormatInputNumber(f float64) string {
	s := FormatNumber(f)
	if math.Abs(f) <= value.MaxSafeInteger || strings.ContainsAny(s, ".e") {
		return s
	}
	return strconv.FormatFloat(f, 'e', -1, 64)
}

// Canonical returns v's canonical form (ADR-002 D9): the RFC 8785
// serialization of the data model, with no byte order mark and no final
// line feed. METAIS writes v as JSON text, each number as its literal, and
// gowebpki/jcs's Transform makes it canonical; RFC 8785 fixes every byte
// of the result, so a library may write it (ADR-010 D3 item 2, D7 item
// 2).
func Canonical(v value.Value) []byte {
	var b strings.Builder
	literal(&b, v)
	out, err := jcs.Transform([]byte(b.String()))
	if err != nil {
		// Every value of the data model is I-JSON, which RFC 8785
		// canonicalizes: a refusal is a defect here or in the library.
		panic(fmt.Sprintf("jsonx: jcs refused a value of the data model: %v", err))
	}
	return out
}

// literal writes v as compact JSON text with each number as its literal
// text and members in canonical order, the text Canonical hands to jcs.
func literal(b *strings.Builder, v value.Value) {
	switch v := v.(type) {
	case value.Null:
		b.WriteString("null")
	case value.Bool:
		b.WriteString(strconv.FormatBool(bool(v)))
	case value.Number:
		b.WriteString(v.Literal())
	case value.String:
		writeString(b, v.Text())
	case value.Array:
		b.WriteByte('[')
		for i, el := range v.All() {
			if i > 0 {
				b.WriteByte(',')
			}
			literal(b, el)
		}
		b.WriteByte(']')
	case value.Object:
		b.WriteByte('{')
		first := true
		for name, m := range v.All() {
			if !first {
				b.WriteByte(',')
			}
			first = false
			writeString(b, name)
			b.WriteByte(':')
			literal(b, m)
		}
		b.WriteByte('}')
	default:
		panic(fmt.Sprintf("jsonx: %T is not a value of the data model", v))
	}
}

// EncodedLen is the byte length of Encode(v, indent), counted without
// writing it, so a caller can charge the text before it is built (ADR-009
// D11 item 2). indent is from 0 to MaxIndent, as Encode takes it.
func EncodedLen(v value.Value, indent int) int64 {
	return encodedLen(v, int64(indent), 0)
}

func encodedLen(v value.Value, indent, depth int64) int64 {
	// newline is what Encode writes before a line at depth d.
	newline := func(d int64) int64 {
		if indent == 0 {
			return 0
		}
		return 1 + d*indent
	}
	switch v := v.(type) {
	case value.Null:
		return 4
	case value.Bool:
		if v {
			return 4
		}
		return 5
	case value.Number:
		return int64(len(FormatNumber(v.Float64())))
	case value.String:
		return stringLen(v.Text())
	case value.Array:
		if v.Len() == 0 {
			return 2
		}
		n := int64(2) + int64(v.Len()-1) + newline(depth)
		for _, el := range v.All() {
			n += newline(depth+1) + encodedLen(el, indent, depth+1)
		}
		return n
	case value.Object:
		if v.Len() == 0 {
			return 2
		}
		n := int64(2) + int64(v.Len()-1) + newline(depth)
		for name, m := range v.All() {
			n += newline(depth+1) + stringLen(name) + 1 + encodedLen(m, indent, depth+1)
			if indent > 0 {
				n++
			}
		}
		return n
	}
	panic(fmt.Sprintf("jsonx: %T is not a value of the data model", v))
}

// stringLen is the byte length writeString writes for s.
func stringLen(s string) int64 {
	n := int64(2)
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '"', c == '\\', c == '\b', c == '\t', c == '\n', c == '\f', c == '\r':
			n += 2
		case c < 0x20:
			n += 6
		default:
			n++
		}
	}
	return n
}
