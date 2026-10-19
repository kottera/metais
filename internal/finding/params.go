// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Params are a finding's parameters, by name (ADR-006 D3 item 6). They are
// encoded in the order the code's registry entry declares them.
type Params map[string]Value

// ParamType is the JSON type of a parameter.
type ParamType uint8

const (
	TypeNumber ParamType = iota + 1
	TypeString
	TypeBoolean
	TypeArray
)

var paramTypeNames = [...]string{TypeNumber: "number", TypeString: "string", TypeBoolean: "boolean", TypeArray: "array"}

func (t ParamType) String() string {
	if t >= TypeNumber && t <= TypeArray {
		return paramTypeNames[t]
	}
	return fmt.Sprintf("ParamType(%d)", uint8(t))
}

// maxExactInt is 2^53, the largest magnitude below which every integer is
// a double, so its decimal digits are its canonical JSON number form
// (ADR-002 D9 item 1).
const maxExactInt = 1 << 53

// Value is one parameter value: a number, a string, a boolean, or an array
// of those (ADR-006 D3 item 6).
type Value struct {
	typ  ParamType
	text string // a number's canonical form, or a string
	b    bool
	list []Value
}

// Int is the number n. It panics when n is beyond ±2^53, where an integer
// has no exact canonical JSON form.
func Int(n int64) Value {
	if n > maxExactInt || n < -maxExactInt {
		panic(fmt.Sprintf("finding: %d has no exact canonical JSON number form", n))
	}
	return Value{typ: TypeNumber, text: strconv.FormatInt(n, 10)}
}

var jsonNumber = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?$`)

// Num is the number a JSON number literal writes, in its canonical form
// (ADR-002 D9 item 1, RFC 8785 §3.2.2.3): the shortest text that reads back
// as the same double. It refuses text that is not a JSON number, and a
// literal beyond the doubles' range.
func Num(literal string) (Value, error) {
	if !jsonNumber.MatchString(literal) {
		return Value{}, fmt.Errorf("finding: %q is not a JSON number", literal)
	}
	f, err := strconv.ParseFloat(literal, 64)
	if err != nil {
		return Value{}, fmt.Errorf("finding: %s has no double", literal)
	}
	return Value{typ: TypeNumber, text: canonicalNumber(f)}, nil
}

// canonicalNumber is f as ECMAScript's Number.prototype.toString writes
// it, which RFC 8785 adopts.
func canonicalNumber(f float64) string {
	if f == 0 {
		return "0"
	}
	sign := ""
	if f < 0 {
		sign, f = "-", -f
	}
	// e is d.ddde±x, the shortest digits that read back as f.
	e := strconv.FormatFloat(f, 'e', -1, 64)
	mant, exp, _ := strings.Cut(e, "e")
	digits := strings.Replace(mant, ".", "", 1)
	x, _ := strconv.Atoi(exp)
	k, n := len(digits), x+1
	switch {
	case k <= n && n <= 21:
		return sign + digits + strings.Repeat("0", n-k)
	case 0 < n && n <= 21:
		return sign + digits[:n] + "." + digits[n:]
	case -6 < n && n <= 0:
		return sign + "0." + strings.Repeat("0", -n) + digits
	}
	out := sign + digits[:1]
	if k > 1 {
		out += "." + digits[1:]
	}
	if n-1 < 0 {
		return out + "e-" + strconv.Itoa(1-n)
	}
	return out + "e+" + strconv.Itoa(n-1)
}

// String is the string s. A string from a supplied input is never a
// parameter (ADR-006 D9 item 4).
func String(s string) Value { return Value{typ: TypeString, text: s} }

// Bool is the boolean b.
func Bool(b bool) Value { return Value{typ: TypeBoolean, b: b} }

// List is the array of vs. An element is never itself an array.
func List(vs ...Value) Value {
	for _, v := range vs {
		if v.typ == TypeArray {
			panic("finding: a parameter array holds an array")
		}
	}
	return Value{typ: TypeArray, list: append([]Value{}, vs...)}
}

// Type is v's JSON type.
func (v Value) Type() ParamType { return v.typ }

// Int returns a number's value; ok is false for any other value.
func (v Value) Int() (n int64, ok bool) {
	if v.typ != TypeNumber {
		return 0, false
	}
	n, err := strconv.ParseInt(v.text, 10, 64)
	return n, err == nil
}

// Str returns a string's value; ok is false for any other value.
func (v Value) Str() (s string, ok bool) { return v.text, v.typ == TypeString }

// Bool returns a boolean's value; ok is false for any other value.
func (v Value) Bool() (b, ok bool) { return v.b, v.typ == TypeBoolean }

// render is v as a message or hint writes it: a number in its canonical
// form, a string as it is, a boolean as true or false, an array as its
// elements separated by ", ".
func (v Value) render() string {
	switch v.typ {
	case TypeBoolean:
		return strconv.FormatBool(v.b)
	case TypeArray:
		parts := make([]string, len(v.list))
		for i, e := range v.list {
			parts[i] = e.render()
		}
		return strings.Join(parts, ", ")
	}
	return v.text
}

func (v Value) clean() Value {
	if v.typ == TypeString {
		v.text = validUTF8(v.text)
	}
	if v.list != nil {
		l := make([]Value, len(v.list))
		for i, e := range v.list {
			l[i] = e.clean()
		}
		v.list = l
	}
	return v
}

// appendJSON appends v to b: numbers in canonical form, strings escaped by
// esc.
func (v Value) appendJSON(b []byte, esc func([]byte, string) []byte) []byte {
	switch v.typ {
	case TypeNumber:
		return append(b, v.text...)
	case TypeString:
		return esc(b, v.text)
	case TypeBoolean:
		return strconv.AppendBool(b, v.b)
	}
	b = append(b, '[')
	for i, e := range v.list {
		if i > 0 {
			b = append(b, ',')
		}
		b = e.appendJSON(b, esc)
	}
	return append(b, ']')
}
