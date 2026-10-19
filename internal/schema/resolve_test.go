// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"errors"
	"slices"
	"testing"
)

// TestReference: $ref and $dynamicRef each resolve to their static target,
// by pointer, by anchor, by a dynamic anchor and across resources.
func TestReference(t *testing.T) {
	w := walker(t, `{
		"$defs": {
			"byPointer": {"$ref": "#/$defs/target"},
			"byAnchor": {"$ref": "#here"},
			"dynamic": {"$dynamicRef": "#node"},
			"target": {"type": "string"},
			"anchored": {"$anchor": "here"},
			"tree": {"$dynamicAnchor": "node", "type": "object"},
			"other": {"$id": "other.json", "$dynamicAnchor": "node", "properties": {"up": {"$ref": "s.json#/$defs/target"}}},
			"plain": {"type": "integer"},
			"broken": {"$ref": "#/$defs/missing"}
		}
	}`)
	cases := []struct {
		ptr, kw, want string
		ok            bool
	}{
		{"/$defs/byPointer", "$ref", "/$defs/target", true},
		{"/$defs/byAnchor", "$ref", "/$defs/anchored", true},
		{"/$defs/dynamic", "$dynamicRef", "/$defs/tree", true},
		{"/$defs/dynamic", "$ref", "", false},
		{"/$defs/other/properties/up", "$ref", "/$defs/target", true},
		{"/$defs/plain", "$ref", "", false},
	}
	for _, c := range cases {
		got, ok, err := w.Reference(c.ptr, c.kw)
		if err != nil || ok != c.ok || got != c.want {
			t.Errorf("Reference(%q, %q) = (%q, %v, %v), want (%q, %v, nil)", c.ptr, c.kw, got, ok, err, c.want, c.ok)
		}
	}
	if _, _, err := w.Reference("/$defs/broken", "$ref"); !errors.Is(err, ErrUnresolved) {
		t.Errorf("a reference to nothing: err = %v, want ErrUnresolved", err)
	}
	if _, _, err := w.Reference("/$defs/plain", "$anchor"); !errors.Is(err, ErrMalformed) {
		t.Errorf("a keyword that is no reference: err = %v, want ErrMalformed", err)
	}
	if got, want := w.DynamicAnchors("node"), []string{"/$defs/other", "/$defs/tree"}; !slices.Equal(got, want) {
		t.Errorf("DynamicAnchors(node) = %q, want %q", got, want)
	}
	if got := w.DynamicAnchors("here"); got != nil {
		t.Errorf("DynamicAnchors(here) = %q: a $anchor is no dynamic anchor", got)
	}
	want := []string{"/$defs/broken", "/$defs/byAnchor", "/$defs/byPointer", "/$defs/dynamic", "/$defs/other/properties/up"}
	if got := w.References(); !slices.Equal(got, want) {
		t.Errorf("References() = %q, want %q", got, want)
	}
	if v, ok := w.Lookup("/$defs/target/type"); !ok || canonical(v) != `"string"` {
		t.Errorf("Lookup(/$defs/target/type) = (%v, %v)", v, ok)
	}
	if _, ok := w.Lookup("/$defs/nothing"); ok {
		t.Errorf("Lookup of a missing pointer answered")
	}
}
