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
	e := encoder{indent: indent}
	e.value(v, 0)
	return []byte(e.b.String()), nil
}

type encoder struct {
	b      strings.Builder
	indent int
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
		e.b.WriteString(FormatNumber(v.Float64()))
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
