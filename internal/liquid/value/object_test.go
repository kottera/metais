// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package value

import (
	"errors"
	"slices"
	"testing"
	"unicode/utf16"
)

// TestObjectCanonicalOrder pins ADR-004 D4 item 5's order: an object is
// iterated by the UTF-16 code units of its names, whatever order its
// members were given in. The names include one outside the Basic
// Multilingual Plane and one in U+E000 to U+FFFF, where UTF-16 order and
// UTF-8 byte order disagree. Names, All and At agree on it.
func TestObjectCanonicalOrder(t *testing.T) {
	want := []string{"", "1", "A", "B", "a", "aa", "ab", "b", "é", "\U0001F600", "\uE000", "\uFFFD"}
	for _, given := range [][]string{
		{"b", "\uFFFD", "a", "", "\U0001F600", "ab", "B", "\uE000", "1", "aa", "é", "A"},
		{"\uFFFD", "\uE000", "\U0001F600", "é", "b", "ab", "aa", "a", "B", "A", "1", ""},
		want,
	} {
		var members []Member
		for _, name := range given {
			members = append(members, Member{Name: name, Value: Null{}})
		}
		o := mustObject(t, members...)
		if got := o.Names(); !slices.Equal(got, want) {
			t.Errorf("given %q, Names() = %q, want %q", given, got, want)
		}
		var iterated []string
		for name := range o.All() {
			iterated = append(iterated, name)
		}
		if !slices.Equal(iterated, want) {
			t.Errorf("given %q, All() = %q, want %q", given, iterated, want)
		}
		var indexed []string
		for i := range o.Len() {
			indexed = append(indexed, o.At(i).Name)
		}
		if !slices.Equal(indexed, want) {
			t.Errorf("given %q, At(0…) = %q, want %q", given, indexed, want)
		}
	}
}

// TestCompareNames checks CompareNames against the definition, comparing
// UTF-16 encodings, over every pair drawn from names chosen at the
// boundaries where the encodings change.
func TestCompareNames(t *testing.T) {
	names := []string{
		"", "a", "ab", "b", "\u007F", "\u0080", "\u07FF", "\u0800", "\uD7FF",
		"\uE000", "\uFFFD", "\uFFFF", "\U00010000", "\U0001F600", "\U0010FFFF",
		"a\U00010000", "a\uFFFF", "\U00010000a", "\U00010000\U00010001",
	}
	for _, a := range names {
		for _, b := range names {
			want := slices.Compare(utf16.Encode([]rune(a)), utf16.Encode([]rune(b)))
			if got := CompareNames(a, b); got != want {
				t.Errorf("CompareNames(%q, %q) = %d, want %d", a, b, got, want)
			}
		}
	}
}

// TestObjectUniqueNames pins that member names are unique, compared
// exactly: names that differ only by normalization or case are different
// names.
func TestObjectUniqueNames(t *testing.T) {
	for _, names := range [][]string{
		{"a", "a"},
		{"a", "b", "c", "b"},
		{"", ""},
		{"\U0001F600", "x", "\U0001F600"},
	} {
		var members []Member
		for i, name := range names {
			n, err := NumberFromFloat(float64(i))
			if err != nil {
				t.Fatal(err)
			}
			members = append(members, Member{Name: name, Value: n})
		}
		if _, err := NewObject(members...); !errors.Is(err, ErrDuplicateName) {
			t.Errorf("NewObject(%q): err = %v, want ErrDuplicateName", names, err)
		}
	}
	o := mustObject(t,
		Member{"e\u0301", Null{}}, // e and a combining acute accent
		Member{"\u00E9", Null{}},  // é, its NFC form
		Member{"A", Null{}},
		Member{"a", Null{}},
	)
	if o.Len() != 4 {
		t.Errorf("Len() = %d, want 4", o.Len())
	}
}

func TestObjectGet(t *testing.T) {
	one := mustNumber(t, "1")
	members := []Member{
		{"z", one}, {"\uE000", mustString(t, "bmp")}, {"\U0001F600", Bool(true)}, {"a", Null{}},
	}
	o := mustObject(t, members...)
	members[0].Value = Bool(false)
	for _, m := range []Member{{"z", one}, {"\uE000", mustString(t, "bmp")}, {"\U0001F600", Bool(true)}, {"a", Null{}}} {
		got, ok := o.Get(m.Name)
		if !ok || !Equal(got, m.Value) {
			t.Errorf("Get(%q) = %v, %v; want %v", m.Name, got, ok, m.Value)
		}
	}
	for _, name := range []string{"", "b", "Z", "zz", "\uFFFF", "\U0001F601"} {
		if got, ok := o.Get(name); ok {
			t.Errorf("Get(%q) = %v, want no member", name, got)
		}
	}
	if _, ok := (Object{}).Get("a"); ok {
		t.Error("the empty object has a member")
	}
	names := o.Names()
	names[0] = "changed"
	if o.Names()[0] == "changed" {
		t.Error("Names() shares its slice with the object")
	}
	for range o.All() {
		break // All stops when the loop does
	}
}
