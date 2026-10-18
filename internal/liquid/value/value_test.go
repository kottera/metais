// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package value

import (
	"errors"
	"testing"
)

// mustNumber parses lit, failing t if it is refused.
func mustNumber(t *testing.T, lit string) Number {
	t.Helper()
	n, err := ParseNumber(lit)
	if err != nil {
		t.Fatalf("ParseNumber(%q): %v", lit, err)
	}
	return n
}

// mustString makes s a String, failing t if it is refused.
func mustString(t *testing.T, s string) String {
	t.Helper()
	v, err := NewString(s)
	if err != nil {
		t.Fatalf("NewString(%q): %v", s, err)
	}
	return v
}

// mustArray makes an array, failing t if it is refused.
func mustArray(t *testing.T, elems ...Value) Array {
	t.Helper()
	a, err := NewArray(elems...)
	if err != nil {
		t.Fatalf("NewArray: %v", err)
	}
	return a
}

// mustObject makes an object, failing t if it is refused.
func mustObject(t *testing.T, members ...Member) Object {
	t.Helper()
	o, err := NewObject(members...)
	if err != nil {
		t.Fatalf("NewObject: %v", err)
	}
	return o
}

func TestKinds(t *testing.T) {
	for _, c := range []struct {
		v    Value
		kind Kind
		name string
	}{
		{Null{}, KindNull, "null"},
		{Bool(true), KindBool, "boolean"},
		{Number{}, KindNumber, "number"},
		{String{}, KindString, "string"},
		{Array{}, KindArray, "array"},
		{Object{}, KindObject, "object"},
	} {
		if got := c.v.Kind(); got != c.kind {
			t.Errorf("%T.Kind() = %v, want %v", c.v, got, c.kind)
		}
		if got := c.kind.String(); got != c.name {
			t.Errorf("Kind %d String() = %q, want %q", c.kind, got, c.name)
		}
	}
	if got := Kind(0).String(); got != "Kind(0)" {
		t.Errorf("Kind(0).String() = %q", got)
	}
}

// TestZeroValues pins that every type's zero value is a valid value of its
// kind, so no consumer meets a half-made value.
func TestZeroValues(t *testing.T) {
	var n Number
	if n.Literal() != "0" || n.Float64() != 0 || !Equal(n, mustNumber(t, "0")) {
		t.Errorf("zero Number = %q, %v; want the number 0", n.Literal(), n.Float64())
	}
	if i, ok := n.SafeInt(); !ok || i != 0 || !n.WrittenAsInteger() {
		t.Errorf("zero Number SafeInt = %d, %v; WrittenAsInteger = %v", i, ok, n.WrittenAsInteger())
	}
	if (String{}).Text() != "" || (Array{}).Len() != 0 || (Object{}).Len() != 0 {
		t.Error("zero String, Array or Object is not empty")
	}
	if !Equal(Array{}, mustArray(t)) || !Equal(Object{}, mustObject(t)) {
		t.Error("zero Array or Object differs from a constructed empty one")
	}
	mustArray(t, Null{}, Bool(false), Number{}, String{}, Array{}, Object{})
}

// smuggled embeds a value type and so inherits its methods, satisfying the
// Value interface from outside the six types.
type smuggled struct{ String }

// TestNothingElse pins that an array or object holds only the six value
// types (acceptance criterion 1).
func TestNothingElse(t *testing.T) {
	s := mustString(t, "x")
	for _, c := range []struct {
		name string
		v    Value
	}{
		{"nil", nil},
		{"pointer", &s},
		{"embedding type", smuggled{s}},
		{"pointer to null", &Null{}},
	} {
		if _, err := NewArray(Null{}, c.v); !errors.Is(err, ErrNotDataValue) {
			t.Errorf("NewArray with %s: err = %v, want ErrNotDataValue", c.name, err)
		}
		if _, err := NewObject(Member{Name: "a", Value: c.v}); !errors.Is(err, ErrNotDataValue) {
			t.Errorf("NewObject with %s: err = %v, want ErrNotDataValue", c.name, err)
		}
	}
}

func TestArray(t *testing.T) {
	elems := []Value{mustNumber(t, "1"), mustString(t, "b"), Null{}}
	a := mustArray(t, elems...)
	elems[0] = Bool(true)
	if a.Len() != 3 || !Equal(a.At(0), mustNumber(t, "1")) {
		t.Errorf("array changed when its argument slice did: At(0) = %v", a.At(0))
	}
	var kinds []Kind
	for i, e := range a.All() {
		if i != len(kinds) {
			t.Fatalf("All yielded index %d at position %d", i, len(kinds))
		}
		kinds = append(kinds, e.Kind())
	}
	if want := []Kind{KindNumber, KindString, KindNull}; len(kinds) != 3 || kinds[0] != want[0] || kinds[1] != want[1] || kinds[2] != want[2] {
		t.Errorf("All kinds = %v, want %v", kinds, want)
	}
	for range a.All() {
		break // All stops when the loop does
	}
}

func TestEqual(t *testing.T) {
	one, oneDotZero := mustNumber(t, "1"), mustNumber(t, "1.0")
	a, b := mustString(t, "a"), mustString(t, "b")
	for _, c := range []struct {
		name string
		x, y Value
		want bool
	}{
		{"null", Null{}, Null{}, true},
		{"true", Bool(true), Bool(true), true},
		{"true false", Bool(true), Bool(false), false},
		{"1 and 1.0", one, oneDotZero, true},
		{"1 and 1e0", one, mustNumber(t, "1e0"), true},
		{"1 and 10e-1", one, mustNumber(t, "10e-1"), true},
		{"0 and -0", mustNumber(t, "0"), mustNumber(t, "-0"), true},
		{"1 and 2", one, mustNumber(t, "2"), false},
		{"string", a, mustString(t, "a"), true},
		{"strings", a, b, false},
		{"number and string", one, mustString(t, "1"), false},
		{"null and false", Null{}, Bool(false), false},
		{"empty array and empty object", Array{}, Object{}, false},
		{"arrays", mustArray(t, one, a), mustArray(t, oneDotZero, a), true},
		{"array order", mustArray(t, one, a), mustArray(t, a, one), false},
		{"array lengths", mustArray(t, one), mustArray(t, one, one), false},
		{
			"object order",
			mustObject(t, Member{"x", one}, Member{"y", a}),
			mustObject(t, Member{"y", a}, Member{"x", oneDotZero}),
			true,
		},
		{"object values", mustObject(t, Member{"x", one}), mustObject(t, Member{"x", a}), false},
		{"object names", mustObject(t, Member{"x", one}), mustObject(t, Member{"y", one}), false},
		{"object sizes", mustObject(t, Member{"x", one}), mustObject(t, Member{"x", one}, Member{"y", one}), false},
		{"smuggled", a, smuggled{a}, false},
		{"smuggled first", smuggled{a}, a, false},
	} {
		if got := Equal(c.x, c.y); got != c.want {
			t.Errorf("%s: Equal = %v, want %v", c.name, got, c.want)
		}
		if got := Equal(c.y, c.x); got != c.want {
			t.Errorf("%s: Equal reversed = %v, want %v", c.name, got, c.want)
		}
	}
}
