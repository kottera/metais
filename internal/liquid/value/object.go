// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package value

import (
	"errors"
	"fmt"
	"iter"
	"slices"
	"sort"
	"unicode/utf8"
)

// Member is one name and value of an object.
type Member struct {
	Name  string
	Value Value
}

// Object is a set of members with unique names (ADR-003 D8 item 3, ADR-002
// D4 item 1). The data model gives member order no meaning, so an Object
// keeps none: it holds its members in canonical order, ascending by the
// UTF-16 code units of the name, the order RFC 8785 sorts by, and every
// consumer iterates them in that order. That is the order ADR-004 D4 item 5
// requires of every template iteration and every serializer, so the same
// object read from JSON, from YAML or from CSV is iterated identically, and
// no consumer ever iterates a Go map (ADR-010 D2 item 3, D4).
//
// The zero Object is the empty object.
type Object struct {
	members []Member
}

// ErrDuplicateName is returned when two members of an object have the same
// name. Names are the same when they are the same sequence of scalar
// values; nothing is normalized or folded.
var ErrDuplicateName = errors.New("duplicate member name")

// NewObject returns the object holding members. It refuses a member whose
// name is not a sequence of Unicode scalar values (ErrNotScalarValues),
// whose value is not a value of the data model (ErrNotDataValue), or whose
// name another member also has (ErrDuplicateName). The order of members
// does not matter; members is copied.
func NewObject(members ...Member) (Object, error) {
	for _, m := range members {
		if err := scalarValues(m.Name); err != nil {
			return Object{}, fmt.Errorf("member name %q: %w", m.Name, err)
		}
		if err := checkMember(m.Value); err != nil {
			return Object{}, fmt.Errorf("member %q: %w", m.Name, err)
		}
	}
	sorted := slices.Clone(members)
	slices.SortStableFunc(sorted, func(a, b Member) int { return CompareNames(a.Name, b.Name) })
	for i := 1; i < len(sorted); i++ {
		if sorted[i].Name == sorted[i-1].Name {
			return Object{}, fmt.Errorf("%w: %q", ErrDuplicateName, sorted[i].Name)
		}
	}
	return Object{members: sorted}, nil
}

// Len returns the number of members.
func (o Object) Len() int { return len(o.members) }

// Get returns the value of the member named name, and whether there is one.
func (o Object) Get(name string) (Value, bool) {
	i := sort.Search(len(o.members), func(i int) bool { return CompareNames(o.members[i].Name, name) >= 0 })
	if i < len(o.members) && o.members[i].Name == name {
		return o.members[i].Value, true
	}
	return nil, false
}

// All yields each member's name and value in canonical order.
func (o Object) All() iter.Seq2[string, Value] {
	return func(yield func(string, Value) bool) {
		for _, m := range o.members {
			if !yield(m.Name, m.Value) {
				return
			}
		}
	}
}

// Names returns the member names in canonical order.
func (o Object) Names() []string {
	names := make([]string, len(o.members))
	for i, m := range o.members {
		names[i] = m.Name
	}
	return names
}

// CompareNames compares two member names by their UTF-16 code units, the
// canonical order of RFC 8785 §3.2.3 and ADR-004 D4 item 5, returning -1,
// 0 or +1. It differs from comparing UTF-8 bytes, or code points, only
// where a character outside the Basic Multilingual Plane meets one in
// U+E000 to U+FFFF: in UTF-16 the first is a surrogate pair, whose leading
// unit, at most U+DBFF, sorts below U+E000. a and b are sequences of
// scalar values, as every member name is.
func CompareNames(a, b string) int {
	for a != "" && b != "" {
		ra, na := utf8.DecodeRuneInString(a)
		rb, nb := utf8.DecodeRuneInString(b)
		if ka, kb := utf16Key(ra), utf16Key(rb); ka != kb {
			if ka < kb {
				return -1
			}
			return 1
		}
		a, b = a[na:], b[nb:]
	}
	switch {
	case a == "" && b != "":
		return -1
	case a != "" && b == "":
		return 1
	}
	return 0
}

// utf16Key maps a scalar value to a key whose order is the order of the
// scalar values' UTF-16 encodings: U+E000 to U+FFFF move above every
// supplementary character, and nothing else moves.
func utf16Key(r rune) rune {
	if 0xE000 <= r && r <= 0xFFFF {
		return r + 0x200000
	}
	return r
}
