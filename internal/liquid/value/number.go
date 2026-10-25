// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package value

import (
	"errors"
	"fmt"
	"math"
	"strconv"
)

// MaxSafeInteger is 2^53−1, the largest magnitude at which every integer
// is a distinct binary64 (RFC 7493 §2.2).
const MaxSafeInteger = 1<<53 - 1

// Number is the data model's one number type, IEEE 754 binary64 (ADR-003
// D8 item 5, ADR-002 D4 item 3). It holds its value and the literal text it
// was written as. The value is what the number means: two numbers are
// Equal when their values are, so 1 and 1.0 are the same number, and -0 is
// zero. The literal is what was written, kept so that a consumer passes on
// the source's own text rather than a re-formatted double (ADR-010 D5 item
// 4) and so that a reader can tell how a number was spelled.
//
// The zero Number is zero, written "0".
type Number struct {
	lit string
	f   float64
}

// Number errors.
var (
	// ErrNumberSyntax is returned for text that is not an RFC 8259 number.
	ErrNumberSyntax = errors.New("not a JSON number")
	// ErrNumberRange is returned for a number that is infinite or NaN, or
	// whose literal rounds to an infinity: binary64 holds neither, and the
	// data model has no such number.
	ErrNumberRange = errors.New("number out of the range of binary64")
)

// ParseNumber returns the number written as lit, which must match RFC 8259
// §6's grammar exactly: an optional minus, an integer part without leading
// zeros, an optional fraction and an optional exponent. Its value is lit
// rounded to the nearest binary64, ties to even; a literal that rounds to
// an infinity is refused with ErrNumberRange. A literal that underflows
// rounds to zero or a subnormal, as the rounding rule says.
//
// ParseNumber does not refuse a large integer: whether one is refused, and
// what counts as one, is the reader's rule (ADR-003 D8 item 5, ADR-002 D4
// item 3), written against Integral, WrittenAsInteger and SafeInt.
func ParseNumber(lit string) (Number, error) {
	if !numberGrammar(lit) {
		return Number{}, fmt.Errorf("%w: %q", ErrNumberSyntax, lit)
	}
	f, err := strconv.ParseFloat(lit, 64)
	if math.IsInf(f, 0) {
		return Number{}, fmt.Errorf("%w: %q", ErrNumberRange, lit)
	}
	if err != nil {
		// The grammar admits nothing else ParseFloat refuses.
		return Number{}, fmt.Errorf("%w: %q: %v", ErrNumberSyntax, lit, err)
	}
	return Number{lit: lit, f: f}, nil
}

// NumberFromFloat returns the number whose value is f, for a value made
// rather than read. Its literal is the shortest text that parses back to f.
// An infinity or NaN is refused with ErrNumberRange.
func NumberFromFloat(f float64) (Number, error) {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return Number{}, fmt.Errorf("%w: %v", ErrNumberRange, f)
	}
	// 'g' writes an RFC 8259 number for every finite double: digits, an
	// optional fraction, and an exponent of the form e+NN or e-NN.
	return Number{lit: strconv.FormatFloat(f, 'g', -1, 64), f: f}, nil
}

// NumberFromInt returns the number whose literal is i in decimal, as
// ParseNumber returns it for that text: Go's conversion of an integer to
// binary64 rounds to the nearest, ties to even, as ParseFloat does.
func NumberFromInt(i int64) Number {
	return Number{lit: strconv.FormatInt(i, 10), f: float64(i)}
}

// Literal returns the number's text as it was written.
func (n Number) Literal() string {
	if n.lit == "" {
		return "0"
	}
	return n.lit
}

// Float64 returns the number's binary64 value. It is never an infinity or
// NaN.
func (n Number) Float64() float64 { return n.f }

// Integral reports whether the number's value has no fractional part:
// true for 1, 1.0, 1e3 and 1e300, false for 1.5.
func (n Number) Integral() bool { return n.f == math.Trunc(n.f) }

// SafeInt returns the number's value as an int64 when it is integral and
// its magnitude is at most MaxSafeInteger, which is when ADR-004 D4 item 2
// presents it to a template as an integer; ok is false for every other
// number. -0 is 0.
func (n Number) SafeInt() (i int64, ok bool) {
	if !n.Integral() || math.Abs(n.f) > MaxSafeInteger {
		return 0, false
	}
	return int64(n.f), true
}

// WrittenAsInteger reports whether the literal has neither a fraction nor
// an exponent: true for 12 and -0, false for 12.0 and 12e0. ADR-003 D8
// item 5's refusal of an integer beyond 2^53−1 applies to these only.
func (n Number) WrittenAsInteger() bool {
	for _, c := range []byte(n.Literal()) {
		if c == '.' || c == 'e' || c == 'E' {
			return false
		}
	}
	return true
}

// numberGrammar reports whether s matches RFC 8259 §6:
//
//	number = [ minus ] int [ frac ] [ exp ]
//	int    = zero / ( digit1-9 *DIGIT )
//	frac   = decimal-point 1*DIGIT
//	exp    = e [ minus / plus ] 1*DIGIT
func numberGrammar(s string) bool {
	i := 0
	if i < len(s) && s[i] == '-' {
		i++
	}
	switch {
	case i < len(s) && s[i] == '0':
		i++
	case i < len(s) && '1' <= s[i] && s[i] <= '9':
		i = digits(s, i)
	default:
		return false
	}
	if i < len(s) && s[i] == '.' {
		j := digits(s, i+1)
		if j == i+1 {
			return false
		}
		i = j
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		i++
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			i++
		}
		j := digits(s, i)
		if j == i {
			return false
		}
		i = j
	}
	return i == len(s)
}

// digits returns the index of the first byte at or after i that is not an
// ASCII digit.
func digits(s string, i int) int {
	for i < len(s) && '0' <= s[i] && s[i] <= '9' {
		i++
	}
	return i
}
