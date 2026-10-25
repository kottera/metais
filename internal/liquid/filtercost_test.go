// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"metais.org/metais/internal/liquid/value"
)

// TestDecimalSum holds sum's accumulator to the sum it must equal: every
// term read by tnum.rat and added as a rational. The terms reach both
// ends of a float's exponents, its least subnormal, both signs, terms
// that cancel, integers at 2^53-1, and a mix of every exponent at once.
func TestDecimalSum(t *testing.T) {
	ints := func(is ...int64) []tnum {
		var out []tnum
		for _, i := range is {
			out = append(out, tnum{isInt: true, i: i})
		}
		return out
	}
	floats := func(fs ...float64) []tnum {
		var out []tnum
		for _, f := range fs {
			out = append(out, tnum{f: f})
		}
		return out
	}
	r := rand.New(rand.NewPCG(1, 2))
	var mixed []tnum
	for range 5_000 {
		switch r.IntN(3) {
		case 0:
			mixed = append(mixed, tnum{isInt: true, i: r.Int64N(1<<53) - 1<<52})
		case 1:
			mixed = append(mixed, tnum{f: math.Float64frombits(r.Uint64()&^(0x7ff<<52) | uint64(r.IntN(0x7ff))<<52)})
		default:
			mixed = append(mixed, tnum{f: (r.Float64() - 0.5) * math.Pow10(r.IntN(600)-300)})
		}
	}
	cases := map[string][]tnum{
		"none":        nil,
		"integers":    ints(1, 2, 3, -7, 0),
		"safe edge":   ints(9007199254740991, 1, -1),
		"tenths":      floats(0.1, 0.2),
		"extremes":    floats(math.MaxFloat64, math.SmallestNonzeroFloat64, -math.MaxFloat64),
		"subnormals":  floats(5e-324, 1e-323, -2.5e-320),
		"cancelling":  floats(1e300, 1e-300, -1e300),
		"zeros":       floats(0, math.Copysign(0, -1)),
		"mixed kinds": append(ints(3, -4), floats(2.5, 1e20, -1.25e-7)...),
		"mixed":       mixed,
	}
	for name, terms := range cases {
		t.Run(name, func(t *testing.T) {
			var s decimalSum
			want := new(big.Rat)
			for _, n := range terms {
				s.add(n)
				want.Add(want, n.rat())
			}
			if got := s.rat(); got.Cmp(want) != 0 {
				t.Fatalf("the sum is %s, not %s", got.RatString(), want.RatString())
			}
		})
	}
}

// TestDecimalSumAddsWithoutAllocating proves a term costs no allocation,
// so sum's work over its input is its charge's, not a rational's.
func TestDecimalSumAddsWithoutAllocating(t *testing.T) {
	var s decimalSum
	terms := []tnum{{isInt: true, i: 7}, {f: 0.5}, {f: -1.2345678901234567e-300}, {f: math.MaxFloat64}}
	if n := testing.AllocsPerRun(100, func() {
		for _, n := range terms {
			s.add(n)
		}
	}); n != 0 {
		t.Fatalf("add allocates %v times a run", n)
	}
}

// TestUniqKeyIsIdentity holds uniq's keys to values' identities: two
// values share a key exactly when they share an identity. The values are
// of every kind, an integer beside the float of its value, both zeros,
// strings that spell numbers, and arrays and objects that hold them.
func TestUniqKeyIsIdentity(t *testing.T) {
	var vs []value.Value
	for _, j := range []string{
		`null`, `true`, `false`, `0`, `-0`, `0.0`, `-0.0`, `1`, `1.0`, `1e0`, `2.5`, `9007199254740993`,
		`""`, `"1"`, `"i1;"`, `"a"`, `[]`, `[1]`, `[1.0]`, `["1"]`, `{}`, `{"a":1}`, `{"a":1.0}`, `[[]]`, `[{}]`,
	} {
		v := jv(t, j)
		vs = append(vs, v, v)
	}
	for _, a := range vs {
		for _, b := range vs {
			same := string(appendIdentity(nil, a)) == string(appendIdentity(nil, b))
			if (keyOf(a) == keyOf(b)) != same {
				t.Errorf("%s and %s: the keys are equal %v, the identities %v", show(a), show(b), keyOf(a) == keyOf(b), same)
			}
		}
	}
}
