// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/kottera/metais/internal/liquid/value"
)

// A template's values are the data model's (ADR-004 D4 item 1), with one
// distinction the data model does not make: a number is an integer or a
// float. A template number is an integer when its literal is written as
// an integer — no fraction, no exponent — and its value is an integer of
// magnitude at most 2^53−1; every other number is a float. A numeric
// literal of the template keeps the spelling it was written with, so 4 is
// an integer and 4.0 a float (ADR-004 D4 item 2). Data reaches a template
// through Data, which spells every number by its value.

// Data returns v as a template sees it (ADR-004 D4 item 2): every number
// whose value is an integer of magnitude at most 2^53−1 is an integer,
// and every other number is a float, however the supplier wrote it.
// Env.Globals holds values made by Data; internal/template makes them
// once, before any render.
func Data(v value.Value) value.Value {
	switch v := v.(type) {
	case value.Number:
		if i, ok := v.SafeInt(); ok {
			return intValue(i)
		}
		// NumberFromFloat writes every number this branch sees with a
		// fraction or an exponent: a non-integral value has a fraction in
		// its shortest digits, and an integral one is at least 2^53, which
		// 'g' writes with an exponent.
		n, err := value.NumberFromFloat(v.Float64())
		if err != nil {
			panic(err) // a Number is never infinite or NaN
		}
		return n
	case value.Array:
		elems := make([]value.Value, v.Len())
		for i, e := range v.All() {
			elems[i] = Data(e)
		}
		return mustArray(elems)
	case value.Object:
		members := make([]value.Member, 0, v.Len())
		for name, m := range v.All() {
			members = append(members, value.Member{Name: name, Value: Data(m)})
		}
		return mustObject(members)
	}
	return v
}

// intValue returns the template integer i, which is within ±(2^53−1).
func intValue(i int64) value.Number {
	n, err := value.ParseNumber(strconv.FormatInt(i, 10))
	if err != nil {
		panic(err) // FormatInt writes RFC 8259 integers
	}
	return n
}

// mustArray and mustObject build values from members that are already
// values of the data model, which their constructors accept.
func mustArray(elems []value.Value) value.Array {
	a, err := value.NewArray(elems...)
	if err != nil {
		panic(err)
	}
	return a
}

func mustObject(members []value.Member) value.Object {
	o, err := value.NewObject(members...)
	if err != nil {
		panic(err)
	}
	return o
}

func stringValue(s string) value.String {
	v, err := value.NewString(s)
	if err != nil {
		panic(err) // the engine builds strings only of Unicode text
	}
	return v
}

// templateInt returns n as a template integer, and whether it is one.
func templateInt(n value.Number) (int64, bool) {
	if !n.WrittenAsInteger() {
		return 0, false
	}
	return n.SafeInt()
}

// kindName names a value's kind as a template sees it, for an error's
// Args: nil, boolean, integer, float, string, array or object.
func kindName(v value.Value) string {
	switch v := v.(type) {
	case value.Null:
		return "nil"
	case value.Bool:
		return "boolean"
	case value.Number:
		if _, ok := templateInt(v); ok {
			return "integer"
		}
		return "float"
	case value.String:
		return "string"
	case value.Array:
		return "array"
	case value.Object:
		return "object"
	}
	return "unknown"
}

// truthy is Shopify's truthiness (ADR-004 D5 item 1): nil and false are
// falsy, and everything else — 0, "", [] and {} too — is truthy.
func truthy(v value.Value) bool {
	switch v := v.(type) {
	case value.Null:
		return false
	case value.Bool:
		return bool(v)
	}
	return true
}

// numberText is a number's text (ADR-004 D4 item 4): an integer's decimal
// digits, and a float as Ruby's Float#to_s writes it.
func numberText(n value.Number) string {
	if i, ok := templateInt(n); ok {
		return strconv.FormatInt(i, 10)
	}
	return floatText(n.Float64())
}

// floatText writes f as Ruby's Float#to_s does (ADR-004 D4 item 4): the
// shortest digits that round-trip, in fixed notation with at least one
// digit after the point when 1e-4 ≤ |f| < 1e15, and otherwise as
// d.ddde±XX with at least two exponent digits. Zero is 0.0, and negative
// zero -0.0.
func floatText(f float64) string {
	sign := ""
	if math.Signbit(f) {
		sign = "-"
	}
	a := math.Abs(f)
	if a == 0 {
		return sign + "0.0"
	}
	// 'e' with the shortest precision gives d[.ddd]e±XX.
	e := strconv.FormatFloat(a, 'e', -1, 64)
	mant, expText, _ := strings.Cut(e, "e")
	digits := strings.Replace(mant, ".", "", 1)
	exp, err := strconv.Atoi(expText)
	if err != nil {
		panic(err) // FormatFloat writes a decimal exponent
	}
	if 1e-4 <= a && a < 1e15 {
		var b strings.Builder
		b.WriteString(sign)
		if exp < 0 {
			b.WriteString("0.")
			b.WriteString(strings.Repeat("0", -exp-1))
			b.WriteString(digits)
			return b.String()
		}
		if len(digits) <= exp+1 {
			b.WriteString(digits)
			b.WriteString(strings.Repeat("0", exp+1-len(digits)))
			b.WriteString(".0")
			return b.String()
		}
		b.WriteString(digits[:exp+1])
		b.WriteByte('.')
		b.WriteString(digits[exp+1:])
		return b.String()
	}
	frac := digits[1:]
	if frac == "" {
		frac = "0"
	}
	expSign := "+"
	if exp < 0 {
		expSign, exp = "-", -exp
	}
	expDigits := strconv.Itoa(exp)
	if len(expDigits) < 2 {
		expDigits = "0" + expDigits
	}
	return sign + digits[:1] + "." + frac + "e" + expSign + expDigits
}

// shallowSize is a value's shallow size as ADR-009 D11 item 2 defines it:
// a string's byte length, an array's element count, an object's member
// count, and 1 for anything else.
func shallowSize(v value.Value) int64 {
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

// deepSize is a value's shallow size added to the deep size of every
// element or member value within it: what comparing two arrays or objects
// for equality costs (ADR-009 D11 item 2).
func deepSize(v value.Value) int64 {
	n := shallowSize(v)
	switch v := v.(type) {
	case value.Array:
		for _, e := range v.All() {
			n = saturatingAdd(n, deepSize(e))
		}
	case value.Object:
		for _, m := range v.All() {
			n = saturatingAdd(n, deepSize(m))
		}
	}
	return n
}

// composite reports whether v is an array or an object.
func composite(v value.Value) bool {
	switch v.(type) {
	case value.Array, value.Object:
		return true
	}
	return false
}

// rubyToI is Ruby's String#to_i over s: leading whitespace, an optional
// sign, then the longest run of decimal digits, single underscores
// allowed between them; 0 when there are none. ok is false for a value
// beyond ±(2^53−1).
func rubyToI(s string) (int64, bool) {
	i := 0
	for i < len(s) && isSpace(s[i]) {
		i++
	}
	neg := false
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		neg = s[i] == '-'
		i++
	}
	var n int64
	for i < len(s) {
		c := s[i]
		if c == '_' && i > 0 && isDigit(s[i-1]) && i+1 < len(s) && isDigit(s[i+1]) {
			i++
			continue
		}
		if !isDigit(c) {
			break
		}
		n = n*10 + int64(c-'0')
		if n > value.MaxSafeInteger {
			return 0, false
		}
		i++
	}
	if neg {
		n = -n
	}
	return n, true
}

// rubyInteger is Ruby's Integer() over a string, as Shopify's
// Utils.to_integer applies it: optional surrounding whitespace, an
// optional sign and decimal digits, single underscores allowed between
// them, and nothing else. ok is false for any other text, and for a value
// beyond ±(2^53−1).
func rubyInteger(s string) (int64, bool) {
	s = strings.TrimFunc(s, func(r rune) bool { return r < utf8.RuneSelf && isSpace(byte(r)) })
	body := strings.TrimLeft(s, "+-")
	if len(s)-len(body) > 1 || body == "" || !isDigit(body[0]) || !isDigit(body[len(body)-1]) {
		return 0, false
	}
	for i := 0; i < len(body); i++ {
		if !isDigit(body[i]) && (body[i] != '_' || !isDigit(body[i+1])) {
			return 0, false
		}
	}
	return rubyToI(s)
}
