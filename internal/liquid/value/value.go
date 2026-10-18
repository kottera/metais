// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package value is the data model's Go representation: the I-JSON value of
// ADR-003 D8 item 3 and ADR-002 D4 item 1, which every reader of an input
// file or of the manifest produces and every later consumer reads.
//
// A Value is exactly one of six kinds — Null, Bool, Number, String, Array
// and Object — and nothing else can be put in one: the interface is sealed,
// and Array and Object refuse any member that is not one of those six
// types, so a type that embeds one of them to inherit its methods is
// refused too. Every value is immutable once made, and every type's zero
// value is a valid value of its kind (Number's is zero).
//
// The rules the representation carries are the data model's own:
//
//   - one number type, IEEE 754 binary64, holding the literal text it was
//     written as beside its value, never a bare float64 (ADR-003 D8 item 5,
//     ADR-002 D4 item 3, ADR-010 D5 item 4);
//   - strings of Unicode scalar values (ADR-003 D8 item 4);
//   - objects with unique member names, whose member order has no meaning
//     and which every consumer iterates in canonical order, ascending by
//     the UTF-16 code units of the name (ADR-003 D8 item 3, ADR-004 D4
//     item 5).
//
// Equal is the data model's equality. Where a value came from, and the
// findings a reader reports for text it refuses, are the readers' own: this
// package holds no positions, and its errors carry none.
//
// The package is part of internal/liquid, so it imports nothing outside
// the standard library and no other internal package (ADR-010 D2 item 2).
package value

import (
	"errors"
	"fmt"
	"iter"
)

// Kind names which of the data model's six kinds a value is.
type Kind uint8

// The kinds. The zero Kind names none of them.
const (
	KindNull Kind = iota + 1
	KindBool
	KindNumber
	KindString
	KindArray
	KindObject
)

// String returns the kind's name as JSON Schema's type keyword spells it.
func (k Kind) String() string {
	switch k {
	case KindNull:
		return "null"
	case KindBool:
		return "boolean"
	case KindNumber:
		return "number"
	case KindString:
		return "string"
	case KindArray:
		return "array"
	case KindObject:
		return "object"
	}
	return fmt.Sprintf("Kind(%d)", uint8(k))
}

// Value is a value of the data model. Its implementations are Null, Bool,
// Number, String, Array and Object, and no others.
type Value interface {
	// Kind returns which of the six kinds the value is.
	Kind() Kind
	// dataValue seals the interface to this package.
	dataValue()
}

// Null is JSON's null.
type Null struct{}

// Bool is JSON's true or false.
type Bool bool

func (Null) Kind() Kind   { return KindNull }
func (Bool) Kind() Kind   { return KindBool }
func (Number) Kind() Kind { return KindNumber }
func (String) Kind() Kind { return KindString }
func (Array) Kind() Kind  { return KindArray }
func (Object) Kind() Kind { return KindObject }

func (Null) dataValue()   {}
func (Bool) dataValue()   {}
func (Number) dataValue() {}
func (String) dataValue() {}
func (Array) dataValue()  {}
func (Object) dataValue() {}

// ErrNotDataValue is returned when an array element or an object member is
// not one of the six value types: a nil Value, a pointer to a value, or a
// type that embeds one.
var ErrNotDataValue = errors.New("not a value of the data model")

// checkMember reports whether v is exactly one of the six value types.
func checkMember(v Value) error {
	switch v.(type) {
	case Null, Bool, Number, String, Array, Object:
		return nil
	case nil:
		return fmt.Errorf("%w: nil", ErrNotDataValue)
	}
	return fmt.Errorf("%w: %T", ErrNotDataValue, v)
}

// Array is an ordered sequence of values. Its order is meaningful and kept.
type Array struct {
	elems []Value
}

// NewArray returns the array of elems, in order. It refuses an element that
// is not a value of the data model. elems is copied.
func NewArray(elems ...Value) (Array, error) {
	for i, e := range elems {
		if err := checkMember(e); err != nil {
			return Array{}, fmt.Errorf("array element %d: %w", i, err)
		}
	}
	return Array{elems: append([]Value(nil), elems...)}, nil
}

// Len returns the number of elements.
func (a Array) Len() int { return len(a.elems) }

// At returns the element at index i, which must be in [0, Len()).
func (a Array) At(i int) Value { return a.elems[i] }

// All yields each index and element, in order.
func (a Array) All() iter.Seq2[int, Value] {
	return func(yield func(int, Value) bool) {
		for i, e := range a.elems {
			if !yield(i, e) {
				return
			}
		}
	}
}

// Equal reports whether a and b are the same value of the data model:
// the same kind, with numbers equal by their binary64 value (so 1 and 1.0
// are equal, and so are 0 and -0), strings equal scalar for scalar, arrays
// equal element by element in order, and objects holding the same member
// names with equal values, whatever order they were written in. It is the
// equality JSON Schema's const, enum and uniqueItems use.
func Equal(a, b Value) bool {
	switch a := a.(type) {
	case Null:
		_, ok := b.(Null)
		return ok
	case Bool:
		b, ok := b.(Bool)
		return ok && a == b
	case Number:
		b, ok := b.(Number)
		return ok && a.f == b.f
	case String:
		b, ok := b.(String)
		return ok && a.s == b.s
	case Array:
		b, ok := b.(Array)
		if !ok || a.Len() != b.Len() {
			return false
		}
		for i := range a.elems {
			if !Equal(a.elems[i], b.elems[i]) {
				return false
			}
		}
		return true
	case Object:
		b, ok := b.(Object)
		if !ok || a.Len() != b.Len() {
			return false
		}
		// Both hold their members in canonical order, so equal objects
		// hold equal names at equal indices.
		for i := range a.members {
			if a.members[i].Name != b.members[i].Name || !Equal(a.members[i].Value, b.members[i].Value) {
				return false
			}
		}
		return true
	}
	return false
}
