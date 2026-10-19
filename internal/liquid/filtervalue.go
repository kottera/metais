// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/kottera/metais/internal/liquid/value"
)

// This file holds what the standard filters share: a template number's
// kind and arithmetic (ADR-004 D4 items 2 and 3), the text of a value
// (D4 item 4), Shopify's coercions of an operand to a number or an
// integer, the iteration of a filter's input, and an element's property.

// tnum is a template number: an integer or a float (ADR-004 D4 item 2).
// An integer's magnitude is at most value.MaxSafeInteger.
type tnum struct {
	isInt bool
	i     int64
	f     float64
}

// numberKind returns n as a template number. A number is an integer when
// it is written as one and its magnitude is at most 2^53−1, and a float
// otherwise: a literal takes its kind from its spelling (ADR-004 D4 item
// 2), and whatever binds a data number writes an integral one of that
// magnitude as an integer, so its kind is its value's.
func numberKind(n value.Number) tnum {
	if n.WrittenAsInteger() {
		if i, ok := n.SafeInt(); ok {
			return tnum{isInt: true, i: i}
		}
	}
	return tnum{f: n.Float64()}
}

// value returns the template number as a value.Number: an integer written
// as its digits, a float written as its text, which always has a point or
// an exponent. An integer of magnitude above 2^53−1 and a float beyond
// binary64 are CodeNumberRange.
func (n tnum) value(filter string) (value.Value, *Error) {
	if n.isInt {
		if n.i > value.MaxSafeInteger || n.i < -value.MaxSafeInteger {
			return nil, filterErr(filter, CodeNumberRange, nil, "an integer result of magnitude above 2^53-1")
		}
		v, err := value.ParseNumber(strconv.FormatInt(n.i, 10))
		if err != nil {
			return nil, filterErr(filter, CodeInternal, nil, err.Error())
		}
		return v, nil
	}
	if math.IsInf(n.f, 0) || math.IsNaN(n.f) {
		return nil, filterErr(filter, CodeNumberRange, nil, "a float result beyond binary64")
	}
	v, err := value.ParseNumber(floatText(n.f))
	if err != nil {
		return nil, filterErr(filter, CodeInternal, nil, err.Error())
	}
	return v, nil
}

// rat returns the number's exact value: an integer's, or a float's
// shortest round-trip decimal value, which is what ADR-004 D4 item 3
// computes on.
func (n tnum) rat() *big.Rat {
	if n.isInt {
		return new(big.Rat).SetInt64(n.i)
	}
	r, ok := new(big.Rat).SetString(strconv.FormatFloat(n.f, 'g', -1, 64))
	if !ok {
		// FormatFloat writes nothing SetString refuses for a finite value.
		panic("liquid: unreadable float text " + strconv.FormatFloat(n.f, 'g', -1, 64))
	}
	return r
}

// cmp compares two numbers by value.
func (n tnum) cmp(m tnum) int {
	if n.isInt && m.isInt {
		switch {
		case n.i < m.i:
			return -1
		case n.i > m.i:
			return 1
		}
		return 0
	}
	return n.rat().Cmp(m.rat())
}

// isZero reports whether the number's value is zero.
func (n tnum) isZero() bool {
	if n.isInt {
		return n.i == 0
	}
	return n.f == 0
}

// fromRat returns the float nearest r, ties to even.
func fromRat(r *big.Rat) tnum {
	f, _ := r.Float64()
	return tnum{f: f}
}

// floatText is a float's text, ADR-004 D4 item 4's: Ruby's Float#to_s,
// the shortest digits that round-trip, in fixed notation with at least one
// digit after the point when 1e-4 ≤ |x| < 1e15, and otherwise as
// d.ddde±XX with at least two exponent digits. Zero is 0.0, negative zero
// -0.0.
func floatText(f float64) string {
	sign := ""
	if math.Signbit(f) {
		sign, f = "-", -f
	}
	if f == 0 {
		return sign + "0.0"
	}
	if f >= 1e-4 && f < 1e15 {
		s := strconv.FormatFloat(f, 'f', -1, 64)
		if !strings.Contains(s, ".") {
			s += ".0"
		}
		return sign + s
	}
	// Go writes the exponent with a sign and at least two digits.
	mant, exp, _ := strings.Cut(strconv.FormatFloat(f, 'e', -1, 64), "e")
	if !strings.Contains(mant, ".") {
		mant += ".0"
	}
	return sign + mant + "e" + exp
}

// text returns the text of v (ADR-004 D4 item 4): nil is empty, a boolean
// true or false, an integer its digits, a float floatText's, a string
// itself and an array its elements' texts concatenated. An object has no
// text: CodeObjectOutput. The caller has charged the work; textWork says
// what it is.
func text(filter string, v value.Value) (string, *Error) {
	var b strings.Builder
	if err := appendText(filter, &b, v); err != nil {
		return "", err
	}
	return b.String(), nil
}

func appendText(filter string, b *strings.Builder, v value.Value) *Error {
	switch v := v.(type) {
	case value.Null:
	case value.Bool:
		b.WriteString(strconv.FormatBool(bool(v)))
	case value.Number:
		n := numberKind(v)
		if n.isInt {
			b.WriteString(strconv.FormatInt(n.i, 10))
		} else {
			b.WriteString(floatText(n.f))
		}
	case value.String:
		b.WriteString(v.Text())
	case value.Array:
		for _, e := range v.All() {
			if err := appendText(filter, b, e); err != nil {
				return err
			}
		}
	case value.Object:
		return filterErr(filter, CodeObjectOutput, map[string]string{"use": "json"}, "an object has no text; use json")
	default:
		return filterErr(filter, CodeInternal, nil, "not a value of the data model")
	}
	return nil
}

// textWork is the work of taking v's text beyond its shallow size: the
// deep size of an array, which text visits whole.
func textWork(v value.Value) int64 {
	if a, ok := v.(value.Array); ok {
		return deepSize(a)
	}
	return 0
}

// shallow is ADR-009 D11 item 2's shallow size: a string's byte length,
// an array's element count, an object's key count, 1 for anything else.
func shallow(v value.Value) int64 {
	switch v := v.(type) {
	case value.String:
		return int64(len(v.Text()))
	case value.Array:
		return int64(v.Len())
	case value.Object:
		return int64(v.Len())
	}
	return 1
}

// deepSize is the deep size of v: the shallow sizes of v and of every
// value beneath it.
func deepSize(v value.Value) int64 {
	n := shallow(v)
	switch v := v.(type) {
	case value.Array:
		for _, e := range v.All() {
			n += deepSize(e)
		}
	case value.Object:
		for _, e := range v.All() {
			n += deepSize(e)
		}
	}
	return n
}

// rubySpace reports whether c is whitespace as Ruby's strip removes it:
// U+0000, tab, line feed, vertical tab, form feed, carriage return and
// space.
func rubySpace(c byte) bool {
	return c == 0 || c == ' ' || ('\t' <= c && c <= '\r')
}

// asciiSpace reports whether c is whitespace as Ruby's \s and its awk-style
// split see it: tab, line feed, vertical tab, form feed, carriage return
// and space.
func asciiSpace(c byte) bool {
	return c == ' ' || ('\t' <= c && c <= '\r')
}

func rubyStrip(s string) string {
	i, j := 0, len(s)
	for i < j && rubySpace(s[i]) {
		i++
	}
	for j > i && rubySpace(s[j-1]) {
		j--
	}
	return s[i:j]
}

// toNumber is Shopify's coercion of an operand to a number: a number is
// itself; a string that, stripped, is an optional minus, digits, a point
// and digits is the float it spells; any other string is the integer its
// leading digits spell, as Ruby's String#to_i reads them, and 0 when it
// has none; anything else is 0. An integer beyond 2^53−1 is
// CodeNumberRange, and so is a float beyond binary64.
func toNumber(filter string, v value.Value) (tnum, *Error) {
	switch v := v.(type) {
	case value.Number:
		return numberKind(v), nil
	case value.String:
		s := v.Text()
		if t := rubyStrip(s); decimalText(t) {
			f, err := strconv.ParseFloat(t, 64)
			if err != nil || math.IsInf(f, 0) {
				return tnum{}, filterErr(filter, CodeNumberRange, nil, "a numeric string beyond binary64")
			}
			return tnum{f: f}, nil
		}
		return leadingInteger(filter, s)
	}
	return tnum{isInt: true}, nil
}

// decimalText reports whether s is an optional minus, one or more ASCII
// digits, a point and one or more ASCII digits.
func decimalText(s string) bool {
	s = strings.TrimPrefix(s, "-")
	whole, frac, ok := strings.Cut(s, ".")
	return ok && allDigits(whole) && allDigits(frac)
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// leadingInteger reads s as Ruby's String#to_i does: leading whitespace,
// an optional sign, then ASCII digits, each underscore between two digits
// skipped; the rest ignored. No digits is 0.
func leadingInteger(filter string, s string) (tnum, *Error) {
	i := 0
	for i < len(s) && asciiSpace(s[i]) {
		i++
	}
	neg := false
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		neg = s[i] == '-'
		i++
	}
	var n int64
	for ; i < len(s); i++ {
		c := s[i]
		if c == '_' && i > 0 && isDigit(s[i-1]) && i+1 < len(s) && isDigit(s[i+1]) {
			continue
		}
		if !isDigit(c) {
			break
		}
		n = n*10 + int64(c-'0')
		if n > value.MaxSafeInteger {
			return tnum{}, filterErr(filter, CodeNumberRange, nil, "a numeric string of magnitude above 2^53-1")
		}
	}
	if neg {
		n = -n
	}
	return tnum{isInt: true, i: n}, nil
}

// toInteger is the coercion of an argument that counts or indexes, which
// Shopify makes with Ruby's Integer(): an integer is itself, and a string
// of optional whitespace, an optional sign and decimal digits, without a
// leading zero, is the integer it spells. Ruby also reads a radix prefix,
// an underscore and a leading zero as octal; the dialect refuses those
// rather than read them differently. Anything else, a float included, is
// CodeFilterArgument naming argument arg.
func toInteger(filter string, arg string, v value.Value) (int64, *Error) {
	refuse := func(kind string) *Error {
		return filterErr(filter, CodeFilterArgument, map[string]string{"argument": arg, "type": kind},
			"the argument is not an integer")
	}
	switch v := v.(type) {
	case value.Number:
		if n := numberKind(v); n.isInt {
			return n.i, nil
		}
		return 0, refuse("float")
	case value.String:
		s := strings.TrimFunc(v.Text(), func(r rune) bool { return r < utf8.RuneSelf && asciiSpace(byte(r)) })
		digits := strings.TrimLeft(s, "+-")
		if len(s)-len(digits) > 1 || !allDigits(digits) || (len(digits) > 1 && digits[0] == '0') {
			return 0, refuse("string")
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil || n > value.MaxSafeInteger || n < -value.MaxSafeInteger {
			return 0, filterErr(filter, CodeNumberRange, map[string]string{"argument": arg}, "an integer of magnitude above 2^53-1")
		}
		return n, nil
	}
	return 0, refuse(kindName(v))
}

// truthy is ADR-004 D5 item 1's truthiness: only nil and false are falsy.
func truthy(v value.Value) bool {
	switch v := v.(type) {
	case value.Null:
		return false
	case value.Bool:
		return bool(v)
	}
	return true
}

// kindName names v's kind as a finding does: integer and float apart.
func kindName(v value.Value) string {
	switch v := v.(type) {
	case value.Number:
		if numberKind(v).isInt {
			return "integer"
		}
		return "float"
	case nil:
		return "nothing"
	}
	return v.Kind().String()
}

// filterErr returns an error of a filter, its Args carrying the filter's
// name beside args.
func filterErr(filter string, code Code, args map[string]string, detail string) *Error {
	a := map[string]string{"filter": filter}
	for k, v := range args {
		a[k] = v
	}
	return &Error{Code: code, Args: a, Detail: detail}
}

// inputErr is CodeFilterInput for an input of v's kind.
func inputErr(filter string, v value.Value, detail string) *Error {
	return filterErr(filter, CodeFilterInput, map[string]string{"type": kindName(v)}, detail)
}

// iterWork is the work of iterate beyond the input's shallow size: the
// elements of every nested array it flattens.
func iterWork(v value.Value) int64 {
	a, ok := v.(value.Array)
	if !ok {
		return 0
	}
	var n int64
	for _, e := range a.All() {
		if inner, ok := e.(value.Array); ok {
			n += int64(inner.Len()) + iterWork(inner)
		}
	}
	return n
}

// iterate is Shopify's iteration of a filter's input: an array's elements
// with every nested array flattened into it, in order; an object as the
// one element; nil as none; and anything else as the one element.
func iterate(v value.Value) []value.Value {
	switch v := v.(type) {
	case value.Array:
		out := make([]value.Value, 0, v.Len())
		return flattenInto(out, v)
	case value.Null:
		return nil
	}
	return []value.Value{v}
}

func flattenInto(out []value.Value, a value.Array) []value.Value {
	for _, e := range a.All() {
		if inner, ok := e.(value.Array); ok {
			out = flattenInto(out, inner)
			continue
		}
		out = append(out, e)
	}
	return out
}

// lookup is what indexing an element by a property gives in a filter
// that reads one (map, where, sort and the rest), as Shopify's do.
type lookup uint8

const (
	// found: the element has members or text, and val is the property's
	// value, nil when it has none.
	found lookup = iota
	// noIndex: the element is nil or a boolean, which cannot be indexed.
	// Shopify's filters then give nil, or skip the element.
	noIndex
	// badIndex: the element refuses the property: a number, or a string
	// indexed by anything but a string. CodeFilterInput.
	badIndex
)

// property returns item's property prop: an object's member, nil when it
// has none or prop is not a string; for a string, prop itself when it is
// a substring, and nil otherwise, as Ruby's String#[] gives.
func property(item, prop value.Value) (value.Value, lookup) {
	switch item := item.(type) {
	case value.Object:
		if p, ok := prop.(value.String); ok {
			if m, ok := item.Get(p.Text()); ok {
				return m, found
			}
		}
		return value.Null{}, found
	case value.String:
		p, ok := prop.(value.String)
		if !ok {
			return nil, badIndex
		}
		if strings.Contains(item.Text(), p.Text()) {
			return p, found
		}
		return value.Null{}, found
	case value.Null, value.Bool:
		return nil, noIndex
	}
	return nil, badIndex
}

// newString returns s as a value.String. Every filter builds its strings
// from the scalar values of strings, so s is valid UTF-8.
func newString(s string) value.String {
	v, err := value.NewString(s)
	if err != nil {
		panic("liquid: a filter built text that is not UTF-8")
	}
	return v
}

// newArray returns the array of elems, which are values of the data model.
func newArray(elems []value.Value) value.Array {
	a, err := value.NewArray(elems...)
	if err != nil {
		panic("liquid: a filter built an array of non-values: " + err.Error())
	}
	return a
}
