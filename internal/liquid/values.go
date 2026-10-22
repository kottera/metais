// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"strconv"

	"metais.org/metais/internal/liquid/value"
)

// A template's values are the data model's (ADR-004 D4 item 1), with one
// distinction the data model does not make: a number is an integer or a
// float (numberKind). A numeric literal of the template keeps the
// spelling it was written with, so 4 is an integer and 4.0 a float
// (ADR-004 D4 item 2). Data reaches a template through Data, which spells
// every number by its value.

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
		return newArray(elems)
	case value.Object:
		members := make([]value.Member, 0, v.Len())
		for name, m := range v.All() {
			members = append(members, value.Member{Name: name, Value: Data(m)})
		}
		return newObject(members)
	}
	return v
}

// intValue returns the template integer i.
func intValue(i int64) value.Number {
	n, err := value.ParseNumber(strconv.FormatInt(i, 10))
	if err != nil {
		panic(err) // FormatInt writes RFC 8259 integers
	}
	return n
}

// newObject returns the object of members, which are values of the data
// model with unique names.
func newObject(members []value.Member) value.Object {
	o, err := value.NewObject(members...)
	if err != nil {
		panic("liquid: the renderer built an object of non-values: " + err.Error())
	}
	return o
}

// templateInt returns n as a template integer, and whether it is one.
func templateInt(n value.Number) (int64, bool) {
	k := numberKind(n)
	return k.i, k.isInt
}

// numberText is a number's text (ADR-004 D4 item 4): an integer's decimal
// digits, and a float as Ruby's Float#to_s writes it.
func numberText(n value.Number) string {
	if k := numberKind(n); k.isInt {
		return strconv.FormatInt(k.i, 10)
	}
	return floatText(n.Float64())
}

// composite reports whether v is an array or an object.
func composite(v value.Value) bool {
	switch v.(type) {
	case value.Array, value.Object:
		return true
	}
	return false
}
