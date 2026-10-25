// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package value

import (
	"errors"
	"math"
	"strconv"
	"testing"
)

// TestNumberGrammar pins RFC 8259 §6's grammar, and that ParseNumber
// accepts nothing else: none of strconv's own spellings, no leading zero,
// no bare point, no sign but a leading minus.
func TestNumberGrammar(t *testing.T) {
	for _, lit := range []string{
		"0", "-0", "1", "-1", "10", "123456789", "0.5", "-0.5", "1.25",
		"1e2", "1E2", "1e+2", "1e-2", "1.5e10", "0e0", "-0.0e-0", "1e007",
	} {
		n, err := ParseNumber(lit)
		if err != nil {
			t.Errorf("ParseNumber(%q): %v", lit, err)
			continue
		}
		if n.Literal() != lit {
			t.Errorf("ParseNumber(%q).Literal() = %q", lit, n.Literal())
		}
	}
	for _, lit := range []string{
		"", "-", "+1", "01", "-01", "00", ".5", "5.", "1.e2", "1e", "1e+",
		"1E-", " 1", "1 ", "0x10", "0o7", "0b1", "1_000", "Inf", "-Inf",
		"Infinity", "NaN", "nan", "1.5.2", "1e2e3", "--1", "١", "1,5",
	} {
		if _, err := ParseNumber(lit); !errors.Is(err, ErrNumberSyntax) {
			t.Errorf("ParseNumber(%q): err = %v, want ErrNumberSyntax", lit, err)
		}
	}
}

// TestNumberOneType pins ADR-003 D8 item 5's one number type: spellings of
// one value are one number, and the literal text is kept beside it.
func TestNumberOneType(t *testing.T) {
	spellings := []string{"1", "1.0", "1.000", "1e0", "10e-1", "0.1e1", "1E+0"}
	first := mustNumber(t, spellings[0])
	for _, lit := range spellings {
		n := mustNumber(t, lit)
		if !Equal(n, first) {
			t.Errorf("%q is not the same number as %q", lit, spellings[0])
		}
		if n.Literal() != lit {
			t.Errorf("%q kept literal %q", lit, n.Literal())
		}
		if i, ok := n.SafeInt(); !ok || i != 1 {
			t.Errorf("%q SafeInt = %d, %v; want 1, true", lit, i, ok)
		}
	}
	z, nz := mustNumber(t, "0"), mustNumber(t, "-0")
	if !Equal(z, nz) || nz.Literal() != "-0" {
		t.Errorf("-0 is not the number zero, or lost its literal: %q", nz.Literal())
	}
}

// TestNumberRounding pins that a literal is rounded to the nearest
// binary64, ties to even.
func TestNumberRounding(t *testing.T) {
	for _, c := range []struct {
		lit  string
		want float64
	}{
		{"0.1", 0.1},
		{"9007199254740993", 9007199254740992}, // 2^53+1: a tie, to the even 2^53
		{"9007199254740995", 9007199254740996}, // 2^53+3: a tie, to the even 2^53+4
		{"1e-400", 0},                          // underflow rounds to zero
		{"4e-324", 5e-324},                     // nearest subnormal
		{"1.7976931348623157e308", math.MaxFloat64},
	} {
		if got := mustNumber(t, c.lit).Float64(); got != c.want {
			t.Errorf("%q = %v, want %v", c.lit, got, c.want)
		}
	}
}

// TestNumberRange pins that the data model holds no infinity: a literal
// that rounds to one is refused.
func TestNumberRange(t *testing.T) {
	for _, lit := range []string{"1e309", "-1e309", "1e99999999999999999999", "179769313486231580793728971405303415079934132710037826936173778980444968292764750946649017977587207096330286416692887910946555547851940402630657488671505820681908902000708383676273854845817711531764475730270069855571366959622842914819860834936475292719074168444365510704342711559699508093042880177904174497792"} {
		if _, err := ParseNumber(lit); !errors.Is(err, ErrNumberRange) {
			t.Errorf("ParseNumber(%.20q): err = %v, want ErrNumberRange", lit, err)
		}
	}
	for _, f := range []float64{math.Inf(1), math.Inf(-1), math.NaN()} {
		if _, err := NumberFromFloat(f); !errors.Is(err, ErrNumberRange) {
			t.Errorf("NumberFromFloat(%v): err = %v, want ErrNumberRange", f, err)
		}
	}
}

// TestNumberPredicates pins what Integral, SafeInt and WrittenAsInteger
// say, and that the readers' two different refusals of a large integer can
// be written with them: ADR-003 D8 item 5 refuses one written without
// fraction or exponent, ADR-002 D4 item 3 any whose value is an integer.
func TestNumberPredicates(t *testing.T) {
	for _, c := range []struct {
		lit                     string
		integral, safe, written bool
		safeInt                 int64
	}{
		{"0", true, true, true, 0},
		{"-0", true, true, true, 0},
		{"42", true, true, true, 42},
		{"-42", true, true, true, -42},
		{"42.0", true, true, false, 42},
		{"4.2e1", true, true, false, 42},
		{"1.5", false, false, false, 0},
		{"-0.5", false, false, false, 0},
		{"9007199254740991", true, true, true, 9007199254740991},
		{"-9007199254740991", true, true, true, -9007199254740991},
		{"9007199254740992", true, false, true, 0},
		{"-9007199254740992", true, false, true, 0},
		{"12345678901234567890", true, false, true, 0},
		{"9007199254740992.0", true, false, false, 0},
		{"1e300", true, false, false, 0},
		{"1E300", true, false, false, 0},
		{"42E0", true, true, false, 42},
		{"1e-300", false, false, false, 0},
	} {
		n := mustNumber(t, c.lit)
		i, ok := n.SafeInt()
		if n.Integral() != c.integral || ok != c.safe || n.WrittenAsInteger() != c.written || i != c.safeInt {
			t.Errorf("%q: Integral %v, SafeInt %d %v, WrittenAsInteger %v; want %v, %d %v, %v",
				c.lit, n.Integral(), i, ok, n.WrittenAsInteger(), c.integral, c.safeInt, c.safe, c.written)
		}
		adr003 := n.WrittenAsInteger() && !ok
		adr002 := n.Integral() && !ok
		if want := c.written && c.integral && !c.safe; adr003 != want {
			t.Errorf("%q: ADR-003 D8 item 5 refusal = %v, want %v", c.lit, adr003, want)
		}
		if want := c.integral && !c.safe; adr002 != want {
			t.Errorf("%q: ADR-002 D4 item 3 refusal = %v, want %v", c.lit, adr002, want)
		}
	}
}

// TestNumberFromFloat pins that a made number's literal is RFC 8259 text
// that parses back to the same value.
func TestNumberFromFloat(t *testing.T) {
	for _, f := range []float64{
		0, math.Copysign(0, -1), 1, -1, 0.1, 1.5, 100, 1e20, 1e21, 1e-7,
		123456789012345678, math.MaxFloat64, math.SmallestNonzeroFloat64,
		-math.MaxFloat64, MaxSafeInteger, 1 << 53,
	} {
		n, err := NumberFromFloat(f)
		if err != nil {
			t.Fatalf("NumberFromFloat(%v): %v", f, err)
		}
		back, err := ParseNumber(n.Literal())
		if err != nil {
			t.Errorf("NumberFromFloat(%v) literal %q does not parse: %v", f, n.Literal(), err)
			continue
		}
		if back.Float64() != f || math.Signbit(back.Float64()) != math.Signbit(f) || n.Float64() != f {
			t.Errorf("NumberFromFloat(%v) literal %q parses to %v", f, n.Literal(), back.Float64())
		}
	}
}

// TestNumberFromInt holds NumberFromInt to ParseNumber of the integer's
// decimal text, at zero, both signs, 2^53 and its neighbours, where the
// rounding to binary64 starts, and the ends of int64.
func TestNumberFromInt(t *testing.T) {
	for _, i := range []int64{
		0, 1, -1, 42, MaxSafeInteger, -MaxSafeInteger, 1 << 53, 1<<53 + 1, 1<<53 + 3,
		-(1<<53 + 1), 1<<62 + 513, math.MaxInt64, math.MinInt64,
	} {
		n := NumberFromInt(i)
		want, err := ParseNumber(strconv.FormatInt(i, 10))
		if err != nil {
			t.Fatal(err)
		}
		if n != want {
			t.Errorf("NumberFromInt(%d) is %q %v, not %q %v", i, n.Literal(), n.Float64(), want.Literal(), want.Float64())
		}
	}
}
