// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"errors"
	"testing"
)

// TestChain: a static subschema's chain is the schema itself, then each
// `$ref` target in turn, each with the type set declared from it onward.
func TestChain(t *testing.T) {
	w := walker(t, `{
		"properties": {
			"a": {"$ref": "#/$defs/b", "default": 1},
			"plain": {"type": "string"},
			"loop": {"$ref": "#/$defs/loop"},
			"meta": {"$ref": "https://json-schema.org/draft/2020-12/schema"}
		},
		"$defs": {
			"b": {"$ref": "#/$defs/c", "title": "B"},
			"c": {"type": "integer", "default": 2},
			"loop": {"$ref": "#/$defs/loop"}
		}
	}`)
	at := func(name string) Static {
		s, ok, err := w.Walk([]Step{Member(name)})
		if err != nil || !ok {
			t.Fatalf("Walk(%s) = %v, %v", name, ok, err)
		}
		return s
	}
	chain, err := w.Chain(at("a"))
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		ptr   string
		types TypeSet
	}{{"/properties/a", TypeInteger}, {"/$defs/b", TypeInteger}, {"/$defs/c", TypeInteger}}
	if len(chain) != len(want) {
		t.Fatalf("chain %v", chain)
	}
	for i, s := range chain {
		if s.Pointer != want[i].ptr || s.Types != want[i].types {
			t.Errorf("chain[%d] = %q %s, want %q %s", i, s.Pointer, s.Types, want[i].ptr, want[i].types)
		}
	}
	if chain, err := w.Chain(at("plain")); err != nil || len(chain) != 1 || chain[0].Types != TypeString {
		t.Errorf("a schema with no $ref is its own chain: %v, %v", chain, err)
	}
	loop, _ := lookup(w.doc, "/properties/loop")
	if _, err := w.Chain(Static{Schema: loop, Pointer: "/properties/loop"}); !errors.Is(err, ErrRefCycle) {
		t.Errorf("a $ref cycle: %v", err)
	}
	meta, _ := lookup(w.doc, "/properties/meta")
	if _, err := w.Chain(Static{Schema: meta, Pointer: "/properties/meta"}); !errors.Is(err, ErrUnresolved) {
		t.Errorf("a $ref to a meta-schema: %v", err)
	}
}
