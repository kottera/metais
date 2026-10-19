// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sample

import (
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/liquid/value"
)

// member is the member name of v, an object, or nil.
func member(v value.Value, name string) value.Value {
	o, ok := v.(value.Object)
	if !ok {
		return nil
	}
	m, _ := o.Get(name)
	return m
}

// TestRepeatable is W3-5.1's last sentence and ADR-007 D9 item 3: the
// same schema, seed, mode and item count give the same value on every
// run, and another seed gives another.
func TestRepeatable(t *testing.T) {
	sch := `{"type":"object","properties":{
		"name":{"type":"string"},"n":{"type":"integer"},"x":{"type":"number"},
		"when":{"type":"string","format":"date-time"},"id":{"type":"string","format":"uuid"},
		"code":{"type":"string","pattern":"^[A-Z]{3}-[0-9]{2,4}$"},
		"items":{"type":"array","items":{"type":"object","properties":{"a":{"type":"string"},"b":{"type":["integer","null"]}}}},
		"pick":{"anyOf":[{"type":"string"},{"type":"integer"}]}}}`
	for _, mode := range []Mode{All, None, Mixed} {
		for _, seed := range []uint64{0, 1, 42, 1<<64 - 1} {
			opts := Options{Seed: seed, Optional: mode, Items: 4}
			a, b := mustOne(t, sch, opts), mustOne(t, sch, opts)
			if text(a) != text(b) {
				t.Errorf("mode %s, seed %d: %s, then %s", mode, seed, text(a), text(b))
			}
		}
	}
	if a, b := mustOne(t, sch, Options{Seed: 1, Items: 3}), mustOne(t, sch, Options{Seed: 2, Items: 3}); text(a) == text(b) {
		t.Errorf("seeds 1 and 2 give the same value %s", text(a))
	}
}

// TestValueDependsOnPlace is item 5: a value depends only on where it
// sits, so editing one field's schema leaves every other generated value
// unchanged.
func TestValueDependsOnPlace(t *testing.T) {
	before := mustOne(t, `{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"integer"},"c":{"type":"number"}}}`, Options{Items: 3})
	after := mustOne(t, `{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"integer","minimum":500},"c":{"type":"number"}}}`, Options{Items: 3})
	for _, name := range []string{"a", "c"} {
		if text(member(before, name)) != text(member(after, name)) {
			t.Errorf("%s changed from %s to %s", name, text(member(before, name)), text(member(after, name)))
		}
	}
	if text(member(before, "b")) == text(member(after, "b")) {
		t.Error("b did not change")
	}
}

// TestPreference is item 12: const, else an entry of examples, else
// default, else an enum member, else a generated value, each kept only if
// it validates.
func TestPreference(t *testing.T) {
	for _, tc := range []struct {
		name, schema, want string
	}{
		{"const", `{"type":"string","const":"c","examples":["e"],"default":"d","enum":["c","n"]}`, `"c"`},
		{"examples before default", `{"type":"string","examples":["e"],"default":"d","enum":["e","d"]}`, `"e"`},
		{"default before enum", `{"type":"string","default":"d","enum":["n","d"]}`, `"d"`},
		{"enum", `{"enum":[7,"n"]}`, `7`},
		{"an example that does not validate is skipped", `{"type":"string","maxLength":2,"examples":["long","ok"]}`, `"ok"`},
		{"so is a default", `{"type":"integer","maximum":3,"default":9,"enum":[9,2]}`, `2`},
		{"examples of a number", `{"type":"number","examples":[2.5]}`, `2.5`},
		{"an object with no declared properties is taken whole", `{"type":"object","examples":[{"k":1}]}`, `{"k":1}`},
		{"an array with no declared items is taken whole", `{"type":"array","default":[1,2]}`, `[1,2]`},
		{"a const object is taken whole", `{"const":{"a":[1]}}`, `{"a":[1]}`},
		{"examples through $ref", `{"$ref":"#/$defs/x","$defs":{"x":{"type":"string","examples":["r"]}}}`, `"r"`},
	} {
		if got := text(mustOne(t, tc.schema, Options{Items: 3})); got != tc.want {
			t.Errorf("%s: %s, want %s", tc.name, got, tc.want)
		}
	}
}

// TestNotTakenWhole is item 13: an object or an array is not taken whole
// from examples or default when it declares properties or items; the
// generator descends into them, whose own sources apply.
func TestNotTakenWhole(t *testing.T) {
	v := mustOne(t, `{"type":"object","properties":{"a":{"type":"string","examples":["own"]}},"examples":[{"a":"whole"}],"default":{"a":"whole"}}`, Options{Items: 3})
	if got := text(v); got != `{"a":"own"}` {
		t.Errorf("object: %s", got)
	}
	v = mustOne(t, `{"type":"array","items":{"type":"integer","examples":[5]},"default":[1,2,3,4]}`, Options{Items: 2})
	if got := text(v); got != `[5,5]` {
		t.Errorf("array: %s", got)
	}
}

// TestExampleIndex is item 12: among several examples or enum members,
// the one at index (seed + k) mod count, k the item's index in the
// innermost enclosing array, so a single value takes the author's first
// under the default seed and array items cycle through them; uniqueItems
// skips one equal to an earlier item.
func TestExampleIndex(t *testing.T) {
	single := `{"type":"string","examples":["a","b","c"]}`
	for seed, want := range []string{`"a"`, `"b"`, `"c"`, `"a"`} {
		if got := text(mustOne(t, single, Options{Seed: uint64(seed), Items: 3})); got != want {
			t.Errorf("seed %d: %s, want %s", seed, got, want)
		}
	}
	list := `{"type":"array","items":{"enum":["a","b","c"]}}`
	if got := text(mustOne(t, list, Options{Items: 5})); got != `["a","b","c","a","b"]` {
		t.Errorf("items cycle: %s", got)
	}
	if got := text(mustOne(t, list, Options{Seed: 1, Items: 4})); got != `["b","c","a","b"]` {
		t.Errorf("items cycle from the seed: %s", got)
	}
	unique := `{"type":"array","uniqueItems":true,"items":{"examples":["a","a","b"],"type":"string"}}`
	if got := text(mustOne(t, unique, Options{Items: 2})); got != `["a","b"]` {
		t.Errorf("uniqueItems: %s", got)
	}
}

// TestOptionalModes is items 14 and 15: a required property is always
// generated; an optional one under all, never under none, and under mixed
// when its include stream's first draw is even. With null in the type set,
// all never chooses it, none always does, mixed decides by the type
// stream. An omitted property with a default is not given it here.
func TestOptionalModes(t *testing.T) {
	sch := `{"type":"object","required":["r"],"properties":{
		"r":{"type":"string"},"o1":{"type":"string"},"o2":{"type":"string"},"o3":{"type":"string"},
		"o4":{"type":"string"},"o5":{"type":"string"},"o6":{"type":"string"},"d":{"type":"string","default":"x"},
		"n":{"type":["integer","null"]}}}`
	all := mustOne(t, sch, Options{Optional: All}).(value.Object)
	if got := all.Names(); !slices.Equal(got, []string{"d", "n", "o1", "o2", "o3", "o4", "o5", "o6", "r"}) {
		t.Errorf("all: %v", got)
	}
	if _, isNull := member(all, "n").(value.Null); isNull {
		t.Error("all chose null")
	}
	none := mustOne(t, sch, Options{Optional: None}).(value.Object)
	if got := none.Names(); !slices.Equal(got, []string{"r"}) {
		t.Errorf("none: %v", got)
	}
	nullable := mustOne(t, `{"type":"object","required":["n"],"properties":{"n":{"type":["integer","null"]}}}`, Options{Optional: None})
	if got := text(nullable); got != `{"n":null}` {
		t.Errorf("none with null: %s", got)
	}
	mixed := mustOne(t, sch, Options{Optional: Mixed}).(value.Object)
	for _, name := range []string{"o1", "o2", "o3", "o4", "o5", "o6", "d", "n"} {
		ptr := "/" + name
		_, present := mixed.Get(name)
		if want := Stream(0, "data", ptr, "include").Next()%2 == 0; present != want {
			t.Errorf("mixed: %s present %v, its include stream says %v", name, present, want)
		}
	}
	if _, ok := mixed.Get("r"); !ok {
		t.Error("mixed omitted the required property")
	}
	if n, ok := mixed.Get("n"); ok {
		_, isNull := n.(value.Null)
		if want := Stream(0, "data", "/n", "type").Next()%2 == 1; isNull != want {
			t.Errorf("mixed: n null %v, its type stream says %v", isNull, want)
		}
	}
	names := mixed.Names()
	if len(names) == len(all.Names()) || len(names) == 1 {
		t.Logf("mixed kept %v", names)
	}
}

// TestOptionalInput is ADR-003 D1's required, which the modes apply to:
// an optional input is omitted under none, and a required one never.
func TestOptionalInput(t *testing.T) {
	p := compile(t, map[string]string{entry: `{"type":"string"}`})
	for _, tc := range []struct {
		mode     Mode
		required bool
		want     bool
	}{{All, false, true}, {None, false, false}, {None, true, true}, {Mixed, true, true}} {
		_, present, fs, err := p.gen(Options{Optional: tc.mode}).Generate(Input{Name: "data", Schema: entry, Forms: []Form{FormJSON}, Required: tc.required})
		if err != nil || fs != nil || present != tc.want {
			t.Errorf("%s, required %v: present %v, want %v (%v %s)", tc.mode, tc.required, present, tc.want, err, describe(fs))
		}
	}
	_, present, _, _ := p.gen(Options{Optional: Mixed}).Generate(Input{Name: "data", Schema: entry, Forms: []Form{FormJSON}})
	if want := Stream(0, "data", "", "include").Next()%2 == 0; present != want {
		t.Errorf("mixed: present %v, its include stream says %v", present, want)
	}
}

// TestDeclarationOrder is item 15: properties are generated in
// declaration order, the order of first appearance in the schema files,
// walking C through allOf members and $ref targets; minProperties is
// reached with the omitted optional properties first, in that order, and
// maxProperties drops optional properties from the end.
func TestDeclarationOrder(t *testing.T) {
	sch := `{"type":"object","minProperties":3,"properties":{"zeta":{"type":"string"},"alpha":{"type":"string"},"mid":{"type":"string"},"beta":{"type":"string"}}}`
	got := mustOne(t, sch, Options{Optional: None}).(value.Object).Names()
	if !slices.Equal(got, []string{"alpha", "mid", "zeta"}) {
		t.Errorf("minProperties under none: %v, want zeta, alpha and mid", got)
	}
	sch = `{"type":"object","maxProperties":2,"required":["beta"],"properties":{"zeta":{"type":"string"},"alpha":{"type":"string"},"beta":{"type":"string"}}}`
	got = mustOne(t, sch, Options{}).(value.Object).Names()
	if !slices.Equal(got, []string{"beta", "zeta"}) {
		t.Errorf("maxProperties: %v, want beta and zeta", got)
	}
	p := compile(t, map[string]string{entry: `{"type":"object","properties":{"z":{"type":"integer"},"a":{"type":"integer"}},"allOf":[{"properties":{"y":{"type":"integer"}}}],"required":["q"]}`})
	g := p.gen(Options{})
	g.input = Input{Name: "data", Schema: entry}
	c := g.effective(&place{refs: map[node]int{}}, []node{{entry, ""}}, nil)
	if got := g.declared(c); !slices.Equal(got, []string{"z", "a", "y", "q"}) {
		t.Errorf("declared: %v, want z, a, y, q", got)
	}
}

// TestPropertyNames is item 15: beyond the declared properties, names
// reach minProperties from a patternProperties pattern, then key1, key2,
// … where additionalProperties is a schema, each satisfying
// propertyNames.
func TestPropertyNames(t *testing.T) {
	v := mustOne(t, `{"type":"object","minProperties":2,"additionalProperties":{"type":"integer"},"propertyNames":{"not":{"const":"key1"}}}`, Options{})
	if got := v.(value.Object).Names(); !slices.Equal(got, []string{"key2", "key3"}) {
		t.Errorf("additionalProperties: %v", got)
	}
	v = mustOne(t, `{"type":"object","minProperties":2,"patternProperties":{"^x-[a-z]{3}$":{"type":"boolean"}},"additionalProperties":false}`, Options{})
	names := v.(value.Object).Names()
	if len(names) != 2 || !strings.HasPrefix(names[0], "x-") || !strings.HasPrefix(names[1], "x-") {
		t.Errorf("patternProperties: %v", names)
	}
}

// TestSupplied is item 2: a supplied value is taken as given, and the
// generator fills only what is absent around a supplied object.
func TestSupplied(t *testing.T) {
	p := compile(t, map[string]string{entry: `{"type":"object","properties":{"stage":{"enum":["draft","final"]},"title":{"type":"string"},"n":{"type":"integer"},"nested":{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"integer"}}}},
		"if":{"properties":{"stage":{"const":"final"}}},"then":{"required":["approved"],"properties":{"approved":{"const":true}}}}`})
	supplied := parse(t, `{"stage":"final","n":99,"nested":{"a":"given"},"extra":[1]}`)
	v, _, fs, err := p.gen(Options{}).Generate(Input{Name: "data", Schema: entry, Forms: []Form{FormJSON}, Supplied: supplied})
	if err != nil || fs != nil {
		t.Fatal(err, describe(fs))
	}
	for _, tc := range []struct{ name, want string }{{"stage", `"final"`}, {"n", "99"}, {"extra", "[1]"}, {"approved", "true"}} {
		if got := text(member(v, tc.name)); got != tc.want {
			t.Errorf("%s: %s, want %s", tc.name, got, tc.want)
		}
	}
	if got := text(member(member(v, "nested"), "a")); got != `"given"` {
		t.Errorf("nested/a: %s", got)
	}
	if member(member(v, "nested"), "b") == nil || member(v, "title") == nil {
		t.Errorf("absent members were not generated: %s", text(v))
	}
}

// TestSuppliedOpen is item 2 under a schema that declares no type: a
// supplied object is filled around, not replaced.
func TestSuppliedOpen(t *testing.T) {
	p := compile(t, map[string]string{entry: `{"properties":{"a":{"type":"string"},"b":{"const":2}}}`})
	v, _, fs, err := p.gen(Options{}).Generate(Input{Name: "data", Schema: entry, Forms: []Form{FormJSON}, Supplied: parse(t, `{"a":"given"}`)})
	if err != nil || fs != nil || text(v) != `{"a":"given","b":2}` {
		t.Errorf("%s %v%s", text(v), err, describe(fs))
	}
}
