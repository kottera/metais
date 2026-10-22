// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sample

import (
	"math/big"

	"metais.org/metais/internal/jsonx"
	"metais.org/metais/internal/liquid/value"
)

var (
	ratOne      = big.NewRat(1, 1)
	ratHundred  = big.NewRat(100, 1)
	ratHundreth = big.NewRat(1, 100)
	ratMillion  = big.NewRat(1, 1_000_000)
	// safeMax is 2^53−1. The generator makes no integral number beyond
	// ±safeMax: a CSV field has no text an integer column reads back as
	// one, since written as an integer literal ADR-003 D8 item 5 refuses
	// it.
	safeMax = new(big.Rat).SetInt64(value.MaxSafeInteger)
)

// number makes a number or an integer from C (item 17). The range is C's
// bounds; unbounded on both sides it is [1, 100], and bounded on one side
// it extends 100 from that bound. It is held within ±(2^53−1). An
// integer is drawn uniformly from the integers in range that are
// multiples of C's multipleOf; a number without multipleOf from n/100 for
// the integers n in range, refining to n/10^6 when the range holds no
// hundredth; a number with multipleOf from its multiples in range. All of
// it is exact rationals; the value drawn is rounded to the nearest binary64
// only to be written, and its literal is the text the writers write for
// it.
func (s *Generator) number(loc *place, c *conj, integer bool) (value.Value, bool) {
	lo, hi := c.numBounds()
	switch {
	case lo == nil && hi == nil:
		lo, hi = &bound{v: ratOne}, &bound{v: ratHundred}
	case lo == nil:
		lo = &bound{v: new(big.Rat).Sub(hi.v, ratHundred)}
	case hi == nil:
		hi = &bound{v: new(big.Rat).Add(lo.v, ratHundred)}
	}
	if lo.v.Cmp(new(big.Rat).Neg(safeMax)) < 0 {
		lo = &bound{v: new(big.Rat).Neg(safeMax)}
	}
	if hi.v.Cmp(safeMax) > 0 {
		hi = &bound{v: safeMax}
	}
	m := c.multipleOf()
	var steps []*big.Rat
	switch {
	case integer && m == nil:
		steps = []*big.Rat{ratOne}
	case integer:
		// The integer multiples of p/q in lowest terms are the multiples
		// of p.
		steps = []*big.Rat{new(big.Rat).SetInt(m.Num())}
	case m == nil:
		steps = []*big.Rat{ratHundreth, ratMillion}
	default:
		steps = []*big.Rat{m}
	}
	for _, step := range steps {
		first, last := multiples(lo, hi, step)
		n := new(big.Int).Sub(last, first)
		n.Add(n, big.NewInt(1))
		if n.Sign() <= 0 {
			continue
		}
		t := s.stream(loc.ptr, loc.valuePurpose()).BelowBig(n)
		t.Add(t, first)
		v := new(big.Rat).Mul(new(big.Rat).SetInt(t), step)
		f, _ := v.Float64()
		num, err := value.ParseNumber(jsonx.FormatNumber(f))
		if err != nil {
			// FormatNumber writes an RFC 8259 number for every finite
			// double, and v is within ±(2^53−1).
			panic(err)
		}
		return num, true
	}
	reason, at := ReasonRange, "minimum"
	if m != nil {
		reason, at = ReasonMultipleOf, "multipleOf"
	}
	s.unsatisfiable(loc, s.holder(c, at), reason)
	return nil, false
}

// multiples returns the least and greatest integers t for which t·step is
// within lo and hi; first exceeds last when there is none.
func multiples(lo, hi *bound, step *big.Rat) (first, last *big.Int) {
	q := new(big.Rat).Quo(lo.v, step)
	first = ceil(q)
	if lo.exclusive && q.IsInt() {
		first.Add(first, big.NewInt(1))
	}
	q = new(big.Rat).Quo(hi.v, step)
	last = floor(q)
	if hi.exclusive && q.IsInt() {
		last.Sub(last, big.NewInt(1))
	}
	return first, last
}

// floor is the greatest integer at most r.
func floor(r *big.Rat) *big.Int {
	// Div is Euclidean division, which with a positive divisor is the
	// floor.
	return new(big.Int).Div(r.Num(), r.Denom())
}

// ceil is the least integer at least r.
func ceil(r *big.Rat) *big.Int {
	f := floor(r)
	if !r.IsInt() {
		f.Add(f, big.NewInt(1))
	}
	return f
}
