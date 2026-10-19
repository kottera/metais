// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"math"
	"math/big"

	"github.com/kottera/metais/internal/liquid/value"
)

// The arithmetic filters (ADR-004 D4 item 3). Each coerces its input and
// its operand as Shopify's Utils.to_number does (toNumber). Integer with
// integer gives an integer; with a float operand the result is the exact
// real result on the operands' shortest decimal values, rounded to the
// nearest double, ties to even. An integer result of magnitude above
// 2^53−1 is CodeNumberRange; division or modulo by zero is
// CodeDivisionByZero.

func mathFilters() []*stdFilter {
	unary := signature{}
	binary := signature{min: 1, max: 1}
	divisor := signature{min: 1, max: 1, literal: zeroDivisor}
	return []*stdFilter{
		{name: "abs", sig: unary, fn: absFilter},
		{name: "ceil", sig: unary, fn: func(c *call, in value.Value) (value.Value, *Error) {
			return integerPart(c, in, math.Ceil)
		}},
		{name: "floor", sig: unary, fn: func(c *call, in value.Value) (value.Value, *Error) {
			return integerPart(c, in, math.Floor)
		}},
		{name: "round", sig: signature{max: 1}, fn: roundFilter},
		{name: "at_least", sig: binary, fn: func(c *call, in value.Value) (value.Value, *Error) {
			return bound(c, in, 1)
		}},
		{name: "at_most", sig: binary, fn: func(c *call, in value.Value) (value.Value, *Error) {
			return bound(c, in, -1)
		}},
		{name: "plus", sig: binary, fn: arithmetic(plusOp)},
		{name: "minus", sig: binary, fn: arithmetic(minusOp)},
		{name: "times", sig: binary, fn: arithmetic(timesOp)},
		{name: "divided_by", sig: divisor, fn: arithmetic(divideOp)},
		{name: "modulo", sig: divisor, fn: arithmetic(moduloOp)},
		{name: "sum", sig: signature{max: 1}, fn: sumFilter},
	}
}

// zeroDivisor refuses, at parse, a literal divisor that coerces to zero:
// division by it is refused whatever the data (ADR-004 D16 item 1).
func zeroDivisor(filter string, _ int, _ string, v value.Value) *Error {
	n, err := toNumber(filter, v)
	if err != nil {
		return err
	}
	if n.isZero() {
		return filterErr(filter, CodeDivisionByZero, nil, "the divisor is zero")
	}
	return nil
}

func absFilter(c *call, in value.Value) (value.Value, *Error) {
	n, err := c.number(in)
	if err != nil {
		return nil, err
	}
	if n.isInt {
		if n.i < 0 {
			n.i = -n.i
		}
	} else {
		n.f = math.Abs(n.f)
	}
	return n.value(c.name)
}

// integerPart is ceil and floor: the integer, never a float.
func integerPart(c *call, in value.Value, f func(float64) float64) (value.Value, *Error) {
	n, err := c.number(in)
	if err != nil {
		return nil, err
	}
	if n.isInt {
		return n.value(c.name)
	}
	r := f(n.f)
	if math.Abs(r) > value.MaxSafeInteger {
		return nil, filterErr(c.name, CodeNumberRange, nil, "an integer result of magnitude above 2^53-1")
	}
	return tnum{isInt: true, i: int64(r)}.value(c.name)
}

// bound is at_least (sign 1) and at_most (sign −1): the greater, or the
// lesser, of the input and the argument, both coerced; the input when
// they are equal.
func bound(c *call, in value.Value, sign int) (value.Value, *Error) {
	n, err := c.number(in)
	if err != nil {
		return nil, err
	}
	a, _ := c.arg(0)
	m, err := c.number(a)
	if err != nil {
		return nil, err
	}
	if m.cmp(n)*sign > 0 {
		n = m
	}
	return n.value(c.name)
}

// roundFilter rounds half away from zero to the argument's number of
// decimal places, 0 when it is omitted. The argument is coerced as an
// operand is, and a float argument's fraction is dropped. An integer
// input stays an integer; a float input gives an integer when the places
// are zero or fewer, and a float otherwise (ADR-004 D4 item 3).
func roundFilter(c *call, in value.Value) (value.Value, *Error) {
	n, err := c.number(in)
	if err != nil {
		return nil, err
	}
	var places int64
	if a, ok := c.arg(0); ok {
		p, err := c.number(a)
		if err != nil {
			return nil, err
		}
		if p.isInt {
			places = p.i
		} else {
			places = int64(math.Trunc(p.f))
		}
	}
	// A binary64 has no digit beyond the 1075th place after the point or
	// the 309th before it, so a wider rounding changes nothing more.
	places = max(-400, min(places, 1100))
	if n.isInt && places >= 0 {
		return n.value(c.name)
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(abs64(places)), nil)
	x := n.rat()
	if places >= 0 {
		x.Mul(x, new(big.Rat).SetInt(scale))
	} else {
		x.Quo(x, new(big.Rat).SetInt(scale))
	}
	r := roundHalfAway(x)
	if places > 0 {
		return fromRat(new(big.Rat).SetFrac(r, scale)).value(c.name)
	}
	if places < 0 {
		r.Mul(r, scale)
	}
	if !r.IsInt64() || abs64(r.Int64()) > value.MaxSafeInteger {
		return nil, filterErr(c.name, CodeNumberRange, nil, "an integer result of magnitude above 2^53-1")
	}
	return tnum{isInt: true, i: r.Int64()}.value(c.name)
}

// roundHalfAway rounds x to the nearest integer, halves away from zero.
func roundHalfAway(x *big.Rat) *big.Int {
	num := new(big.Int).Abs(x.Num())
	q, r := new(big.Int).QuoRem(num, x.Denom(), new(big.Int))
	if r.Lsh(r, 1).Cmp(x.Denom()) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	if x.Sign() < 0 {
		q.Neg(q)
	}
	return q
}

func abs64(i int64) int64 {
	if i < 0 {
		return -i
	}
	return i
}

// op is one arithmetic operation: on two integers, and on two exact
// values. Either refuses a zero divisor.
type op struct {
	ints func(a, b int64) (int64, bool)
	rats func(a, b *big.Rat) *big.Rat
	div  bool
}

func arithmetic(o op) func(c *call, in value.Value) (value.Value, *Error) {
	return func(c *call, in value.Value) (value.Value, *Error) {
		a, err := c.number(in)
		if err != nil {
			return nil, err
		}
		arg, _ := c.arg(0)
		b, err := c.number(arg)
		if err != nil {
			return nil, err
		}
		if o.div && b.isZero() {
			return nil, filterErr(c.name, CodeDivisionByZero, nil, "the divisor is zero")
		}
		if a.isInt && b.isInt {
			r, ok := o.ints(a.i, b.i)
			if !ok {
				return nil, filterErr(c.name, CodeNumberRange, nil, "an integer result of magnitude above 2^53-1")
			}
			return tnum{isInt: true, i: r}.value(c.name)
		}
		return fromRat(o.rats(a.rat(), b.rat())).value(c.name)
	}
}

// safe reports whether i is within ±(2^53−1).
func safe(i int64) bool { return i <= value.MaxSafeInteger && i >= -value.MaxSafeInteger }

var (
	plusOp = op{
		ints: func(a, b int64) (int64, bool) { r := a + b; return r, safe(r) },
		rats: func(a, b *big.Rat) *big.Rat { return a.Add(a, b) },
	}
	minusOp = op{
		ints: func(a, b int64) (int64, bool) { r := a - b; return r, safe(r) },
		rats: func(a, b *big.Rat) *big.Rat { return a.Sub(a, b) },
	}
	timesOp = op{
		ints: func(a, b int64) (int64, bool) {
			if a != 0 && abs64(b) > value.MaxSafeInteger/abs64(a) {
				return 0, false
			}
			return a * b, true
		},
		rats: func(a, b *big.Rat) *big.Rat { return a.Mul(a, b) },
	}
	// divideOp floors an integer quotient, as Ruby's / does.
	divideOp = op{
		div: true,
		ints: func(a, b int64) (int64, bool) {
			q := a / b
			if a%b != 0 && (a < 0) != (b < 0) {
				q--
			}
			return q, safe(q)
		},
		rats: func(a, b *big.Rat) *big.Rat { return a.Quo(a, b) },
	}
	// moduloOp is a − b·⌊a/b⌋, whose sign is the divisor's, as Ruby's %
	// gives.
	moduloOp = op{
		div: true,
		ints: func(a, b int64) (int64, bool) {
			r := a % b
			if r != 0 && (r < 0) != (b < 0) {
				r += b
			}
			return r, true
		},
		rats: func(a, b *big.Rat) *big.Rat {
			q := new(big.Rat).Quo(a, b)
			// Rat's denominator is positive, so Div, which is Euclidean,
			// floors.
			fl := new(big.Int).Div(q.Num(), q.Denom())
			return a.Sub(a, new(big.Rat).Mul(b, new(big.Rat).SetInt(fl)))
		},
	}
)

// sumFilter adds the input's elements, or with a property the elements'
// values of it, each coerced as an operand is. An element that cannot be
// indexed adds 0; one that refuses the property is CodeFilterInput. The
// sum is an integer when every term is one.
func sumFilter(c *call, in value.Value) (value.Value, *Error) {
	items, err := c.iterate(in)
	if err != nil {
		return nil, err
	}
	prop, hasProp := c.arg(0)
	if hasProp {
		if _, isNull := prop.(value.Null); isNull {
			hasProp = false
		}
	}
	total := new(big.Rat)
	allInts := true
	for _, it := range items {
		term := it
		if hasProp {
			v, how := property(it, prop)
			switch how {
			case noIndex:
				continue
			case badIndex:
				return nil, inputErr(c.name, it, "an element cannot be indexed by the property")
			}
			term = v
		}
		n, err := c.number(term)
		if err != nil {
			return nil, err
		}
		allInts = allInts && n.isInt
		total.Add(total, n.rat())
	}
	if allInts {
		if !total.Num().IsInt64() || !safe(total.Num().Int64()) {
			return nil, filterErr(c.name, CodeNumberRange, nil, "an integer result of magnitude above 2^53-1")
		}
		return tnum{isInt: true, i: total.Num().Int64()}.value(c.name)
	}
	return fromRat(total).value(c.name)
}
