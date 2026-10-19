// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"errors"
	"testing"

	"github.com/kottera/metais/internal/liquid/value"
)

// walker indexes the JSON schema text src as the file inputs/s.json.
func walker(t *testing.T, src string) *Walker {
	t.Helper()
	w, err := NewWalker(parse(t, src), RetrievalURI("inputs/s.json"))
	if err != nil {
		t.Fatalf("NewWalker: %v", err)
	}
	return w
}

// walkCase is one walk and its answer: ok false, or the pointer of the
// static subschema and its declared type set.
type walkCase struct {
	name  string
	path  []Step
	ok    bool
	ptr   string
	types TypeSet
}

func checkWalks(t *testing.T, w *Walker, cases []walkCase) {
	t.Helper()
	for _, c := range cases {
		s, ok, err := w.Walk(c.path)
		if err != nil {
			t.Errorf("%s: Walk: %v", c.name, err)
			continue
		}
		if ok != c.ok || s.Pointer != c.ptr || s.Types != c.types {
			t.Errorf("%s: Walk = (%q, %s, %v), want (%q, %s, %v)", c.name, s.Pointer, s.Types, ok, c.ptr, c.types, c.ok)
		}
		if ok {
			if at, _ := lookup(w.doc, s.Pointer); at == nil || canonical(at) != canonical(s.Schema) {
				t.Errorf("%s: Schema is not the schema at %q", c.name, s.Pointer)
			}
		}
	}
}

// TestWalkProperties: a member step is answered by properties.
func TestWalkProperties(t *testing.T) {
	w := walker(t, `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"a/b~c": {"type": "integer"},
			"0": {"type": "boolean"},
			"address": {"properties": {"city": {"type": ["string", "null"]}}}
		}
	}`)
	checkWalks(t, w, []walkCase{
		{"root", nil, true, "", TypeObject},
		{"member", []Step{Member("name")}, true, "/properties/name", TypeString},
		{"escaped member", []Step{Member("a/b~c")}, true, "/properties/a~1b~0c", TypeInteger},
		{"member named 0", []Step{Member("0")}, true, "/properties/0", TypeBoolean},
		{"nested, type array", []Step{Member("address"), Member("city")}, true, "/properties/address/properties/city", TypeString | TypeNull},
		{"no type", []Step{Member("address")}, true, "/properties/address", 0},
		{"undeclared member", []Step{Member("missing")}, false, "", 0},
		{"index on an object schema", []Step{Index(0)}, false, "", 0},
		{"below a leaf", []Step{Member("name"), Member("x")}, false, "", 0},
	})
}

// TestWalkPrefixItemsAndItems: an index step is answered by prefixItems
// when it covers the index, and by items for any other index.
func TestWalkPrefixItemsAndItems(t *testing.T) {
	w := walker(t, `{
		"properties": {
			"pair": {"prefixItems": [{"type": "string"}, {"type": "number"}], "items": {"type": "boolean"}},
			"prefixOnly": {"prefixItems": [{"type": "string"}]},
			"rows": {"type": "array", "items": {"type": "object", "properties": {"id": {"type": "integer"}}}},
			"open": {"items": true}
		}
	}`)
	checkWalks(t, w, []walkCase{
		{"first prefix", []Step{Member("pair"), Index(0)}, true, "/properties/pair/prefixItems/0", TypeString},
		{"second prefix", []Step{Member("pair"), Index(1)}, true, "/properties/pair/prefixItems/1", TypeNumber},
		{"beyond the prefix", []Step{Member("pair"), Index(2)}, true, "/properties/pair/items", TypeBoolean},
		{"far beyond", []Step{Member("pair"), Index(1000)}, true, "/properties/pair/items", TypeBoolean},
		{"prefix only, covered", []Step{Member("prefixOnly"), Index(0)}, true, "/properties/prefixOnly/prefixItems/0", TypeString},
		{"prefix only, beyond", []Step{Member("prefixOnly"), Index(1)}, false, "", 0},
		{"items, any index", []Step{Member("rows"), Index(7), Member("id")}, true, "/properties/rows/items/properties/id", TypeInteger},
		{"a CSV row schema", []Step{Member("rows"), Index(0)}, true, "/properties/rows/items", TypeObject},
		{"member on an array schema", []Step{Member("rows"), Member("0")}, false, "", 0},
		{"items true", []Step{Member("open"), Index(3)}, true, "/properties/open/items", 0},
		{"below a boolean schema", []Step{Member("open"), Index(3), Member("x")}, false, "", 0},
		{"negative index", []Step{Member("rows"), Index(-1)}, false, "", 0},
	})
}

// TestWalkRef: the walk goes through $ref to its target, by JSON Pointer
// fragment, by anchor and by another resource's $id, resolving each
// reference against the base URI where it stands (ADR-003 D5 item 3). A
// schema's own keywords answer a step first; its $ref target answers what
// they do not.
func TestWalkRef(t *testing.T) {
	w := walker(t, `{
		"properties": {
			"byPointer": {"$ref": "#/$defs/person"},
			"byAnchor": {"$ref": "#addr"},
			"byDynamicAnchor": {"$ref": "#tree"},
			"byID": {"$ref": "https://example.org/money.json"},
			"encoded": {"$ref": "#/$defs/a%20b"},
			"chain": {"$ref": "#/$defs/hop"},
			"siblings": {"properties": {"own": {"type": "number"}}, "$ref": "#/$defs/person"},
			"overridden": {"type": "string", "$ref": "#/$defs/person"},
			"sub": {"$id": "sub/", "properties": {"rel": {"$ref": "item.json"}}}
		},
		"$defs": {
			"person": {"type": "object", "properties": {"name": {"type": "string"}, "own": {"type": "boolean"}}},
			"address": {"$anchor": "addr", "type": "object", "properties": {"zip": {"type": "string"}}},
			"tree": {"$dynamicAnchor": "tree", "type": "array", "items": {"$ref": "#tree"}},
			"money": {
				"$id": "https://example.org/money.json",
				"type": "object",
				"properties": {"amount": {"type": "number"}, "currency": {"$ref": "#/$defs/currency"}},
				"$defs": {"currency": {"type": "string"}}
			},
			"a b": {"type": "null"},
			"hop": {"$ref": "#/$defs/hop2"},
			"hop2": {"type": "integer"},
			"item": {"$id": "sub/item.json", "type": "boolean"}
		}
	}`)
	checkWalks(t, w, []walkCase{
		{"pointer, at the location", []Step{Member("byPointer")}, true, "/properties/byPointer", TypeObject},
		{"pointer, through", []Step{Member("byPointer"), Member("name")}, true, "/$defs/person/properties/name", TypeString},
		{"anchor", []Step{Member("byAnchor"), Member("zip")}, true, "/$defs/address/properties/zip", TypeString},
		{"dynamic anchor as a plain one", []Step{Member("byDynamicAnchor"), Index(2), Index(0)}, true, "/$defs/tree/items", TypeArray},
		{"another resource's $id", []Step{Member("byID"), Member("amount")}, true, "/$defs/money/properties/amount", TypeNumber},
		{"resolved against that resource", []Step{Member("byID"), Member("currency")}, true, "/$defs/money/properties/currency", TypeString},
		{"percent-encoded pointer", []Step{Member("encoded")}, true, "/properties/encoded", TypeNull},
		{"a chain of $refs", []Step{Member("chain")}, true, "/properties/chain", TypeInteger},
		{"own properties first", []Step{Member("siblings"), Member("own")}, true, "/properties/siblings/properties/own", TypeNumber},
		{"then the $ref target", []Step{Member("siblings"), Member("name")}, true, "/$defs/person/properties/name", TypeString},
		{"own type first", []Step{Member("overridden")}, true, "/properties/overridden", TypeString},
		{"relative to an embedded $id", []Step{Member("sub"), Member("rel")}, true, "/properties/sub/properties/rel", TypeBoolean},
	})
}

// TestWalkRelativeRef: "money.json" above resolves against the file's own
// retrieval URI, metais:///inputs/money.json, which is no resource there,
// so the walk reports it unresolved rather than answering for it.
func TestWalkRelativeRef(t *testing.T) {
	w := walker(t, `{"properties": {"relative": {"$ref": "money.json#/$defs/currency"}}}`)
	if _, ok, err := w.Walk([]Step{Member("relative")}); ok || !errors.Is(err, ErrUnresolved) {
		t.Errorf("Walk = %v, %v; want ErrUnresolved", ok, err)
	}
}

// TestWalkEvaluatesNoApplicator pins D14 item 1's "and through nothing
// else" false first: a member, element or type declared only under allOf,
// anyOf, oneOf, a conditional, dependentSchemas, patternProperties,
// additionalProperties, contains or $dynamicRef is not found by the walk,
// so its answer depends on no data.
func TestWalkEvaluatesNoApplicator(t *testing.T) {
	w := walker(t, `{
		"$defs": {"t": {"$dynamicAnchor": "t", "type": "string"}},
		"properties": {
			"all": {"allOf": [{"type": "integer", "properties": {"x": {"type": "integer"}}}]},
			"any": {"anyOf": [{"type": "integer", "properties": {"x": {"type": "integer"}}}]},
			"one": {"oneOf": [{"type": "integer", "properties": {"x": {"type": "integer"}}}]},
			"cond": {"if": {"type": "integer"}, "then": {"type": "integer", "properties": {"x": {"type": "integer"}}}, "else": {"type": "integer"}},
			"dep": {"dependentSchemas": {"x": {"type": "integer", "properties": {"x": {"type": "integer"}}}}},
			"pattern": {"patternProperties": {"^x$": {"type": "integer"}}},
			"additional": {"additionalProperties": {"type": "integer"}},
			"contains": {"contains": {"type": "integer"}},
			"not": {"not": {"type": "integer", "properties": {"x": {"type": "integer"}}}},
			"dynamic": {"$dynamicRef": "#t"},
			"unevaluated": {"unevaluatedProperties": {"type": "integer"}, "unevaluatedItems": {"type": "integer"}}
		}
	}`)
	var cases []walkCase
	for _, p := range []string{"all", "any", "one", "cond", "dep", "pattern", "additional", "contains", "not", "dynamic", "unevaluated"} {
		cases = append(cases,
			walkCase{p + ": no type", []Step{Member(p)}, true, "/properties/" + p, 0},
			walkCase{p + ": no member", []Step{Member(p), Member("x")}, false, "", 0},
			walkCase{p + ": no element", []Step{Member(p), Index(0)}, false, "", 0},
		)
	}
	checkWalks(t, w, cases)
}

// TestWalkCombinedSchema walks the combined schema by an inputs pointer:
// through the input's property, its $ref to the embedded resource, and on
// into the input's own schema, which is how a path set (ADR-003 D17) and a
// CSV input (D11 item 8) find their schema.
func TestWalkCombinedSchema(t *testing.T) {
	decl := Declarations{Inputs: []Input{
		{Name: "vendor", Title: "Vendor", Schema: "inputs/vendor.schema.json", Required: true},
		{Name: "criteria", Title: "Criteria", Schema: "inputs/criteria.schema.yaml", Required: true},
		{Name: "notes", Title: "Notes", Schema: "inputs/notes.json"},
	}}
	files := []File{
		{Path: "inputs/vendor.schema.json", Root: parse(t, `{"$id": "https://example.org/vendor", "type": "object", "properties": {"name": {"type": "string"}, "address": {"$ref": "common.json#/$defs/address"}}}`)},
		{Path: "inputs/common.json", Root: parse(t, `{"$id": "https://example.org/common.json", "$defs": {"address": {"type": "object", "properties": {"zip": {"type": "string"}}}}}`)},
		{Path: "inputs/criteria.schema.yaml", Root: parse(t, `{"type": "array", "items": {"type": "object", "properties": {"score": {"type": "integer"}, "met": {"type": "boolean"}}}}`)},
		{Path: "inputs/notes.json", Root: value.Bool(true)},
	}
	b := mustBundle(t, decl, files)
	w, err := NewWalker(b, CombinedURI)
	if err != nil {
		t.Fatalf("NewWalker: %v", err)
	}
	checkWalks(t, w, []walkCase{
		{"the inputs object", nil, true, "", TypeObject},
		{"an input's root", []Step{Member("vendor")}, true, "/properties/vendor", TypeObject},
		{"into its schema", []Step{Member("vendor"), Member("name")}, true, "/$defs/inputs~1vendor.schema.json/properties/name", TypeString},
		{"on to another file", []Step{Member("vendor"), Member("address"), Member("zip")}, true, "/$defs/inputs~1common.json/$defs/address/properties/zip", TypeString},
		{"a CSV input's row", []Step{Member("criteria"), Index(4)}, true, "/$defs/inputs~1criteria.schema.yaml/items", TypeObject},
		{"a CSV column", []Step{Member("criteria"), Index(4), Member("score")}, true, "/$defs/inputs~1criteria.schema.yaml/items/properties/score", TypeInteger},
		{"an input of any value", []Step{Member("notes")}, true, "/properties/notes", 0},
		{"no such input", []Step{Member("other")}, false, "", 0},
	})
}

// TestWalkRefusals: what the compiler refuses, the walk refuses too
// rather than guessing an answer.
func TestWalkRefusals(t *testing.T) {
	for _, c := range []struct {
		name string
		src  string
		path []Step
		want error
	}{
		{"unresolved pointer", `{"$ref": "#/$defs/none"}`, nil, ErrUnresolved},
		{"unresolved anchor", `{"properties": {"a": {"$ref": "#none"}}}`, []Step{Member("a"), Member("b")}, ErrUnresolved},
		{"unresolved resource", `{"$ref": "https://example.org/x.json"}`, nil, ErrUnresolved},
		{"the meta-schema", `{"$ref": "https://json-schema.org/draft/2020-12/schema"}`, nil, ErrUnresolved},
		{"cycle, declared type", `{"$ref": "#"}`, nil, ErrRefCycle},
		{"cycle, step", `{"$defs": {"a": {"$ref": "#/$defs/b"}, "b": {"$ref": "#/$defs/a"}}, "$ref": "#/$defs/a"}`, []Step{Member("x")}, ErrRefCycle},
		{"type a number", `{"type": 1}`, nil, ErrMalformed},
		{"unknown type", `{"type": ["string", "date"]}`, nil, ErrMalformed},
		{"type array of a number", `{"type": ["string", 1]}`, nil, ErrMalformed},
		{"$ref a number", `{"$ref": 1}`, nil, ErrMalformed},
		{"$ref not a URI reference", `{"$ref": "%zz"}`, nil, ErrMalformed},
		{"properties an array", `{"properties": []}`, []Step{Member("a")}, ErrMalformed},
		{"prefixItems an object", `{"prefixItems": {}}`, []Step{Index(0)}, ErrMalformed},
		{"items an array", `{"items": [{}]}`, []Step{Index(0)}, ErrMalformed},
	} {
		w, err := NewWalker(parse(t, c.src), RetrievalURI("inputs/s.json"))
		if err == nil {
			_, _, err = w.Walk(c.path)
		}
		if !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", c.name, err, c.want)
		}
	}
}

// TestNewWalkerRefusals: the index refuses what makes a reference
// ambiguous or unreadable (ADR-003 D5 item 6).
func TestNewWalkerRefusals(t *testing.T) {
	for _, c := range []struct {
		name string
		src  string
		want error
	}{
		{"two resources, one $id", `{"$defs": {"a": {"$id": "x.json"}, "b": {"$id": "x.json"}}}`, ErrDuplicateURI},
		{"an $id equal to the root's", `{"$id": "https://example.org/s", "$defs": {"a": {"$id": "https://example.org/s"}}}`, ErrDuplicateURI},
		{"two anchors, one name", `{"$defs": {"a": {"$anchor": "x"}, "b": {"$anchor": "x"}}}`, ErrDuplicateURI},
		{"$id not a string", `{"$defs": {"a": {"$id": true}}}`, ErrMalformed},
		{"$id with a fragment", `{"$defs": {"a": {"$id": "a.json#x"}}}`, ErrMalformed},
		{"$anchor not a string", `{"$anchor": 1}`, ErrMalformed},
		{"allOf an object", `{"allOf": {}}`, ErrMalformed},
		{"$defs an array", `{"$defs": []}`, ErrMalformed},
	} {
		if _, err := NewWalker(parse(t, c.src), RetrievalURI("inputs/s.json")); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", c.name, err, c.want)
		}
	}
	// One schema may carry an $anchor and a $dynamicAnchor of one name.
	if _, err := NewWalker(parse(t, `{"$anchor": "x", "$dynamicAnchor": "x"}`), RetrievalURI("inputs/s.json")); err != nil {
		t.Errorf("one schema, two anchors of one name: %v", err)
	}
}

// TestWalkIndexesOnlySchemas: an $id or $anchor inside a value that is not
// a schema (const, enum, default, an x- annotation) names nothing.
func TestWalkIndexesOnlySchemas(t *testing.T) {
	w := walker(t, `{
		"const": {"$id": "https://example.org/c", "$anchor": "c"},
		"enum": [{"$id": "https://example.org/e"}],
		"x-note": {"$anchor": "n", "$ref": "#/$defs/q"},
		"$defs": {"q": {"type": "string"}},
		"properties": {
			"a": {"$ref": "https://example.org/c"},
			"b": {"$ref": "#c"},
			"c": {"$ref": "#/x-note"}
		}
	}`)
	for _, p := range []string{"a", "b"} {
		if _, _, err := w.Walk([]Step{Member(p)}); !errors.Is(err, ErrUnresolved) {
			t.Errorf("%s: err = %v, want ErrUnresolved", p, err)
		}
	}
	// A pointer may still name such a location, which takes the base of
	// the schema around it to resolve its own $ref.
	if s, ok, err := w.Walk([]Step{Member("c")}); err != nil || !ok || s.Types != TypeString {
		t.Errorf("c: Walk = %v, %v, %v", s, ok, err)
	}
}

func TestTypeSetString(t *testing.T) {
	for set, want := range map[TypeSet]string{
		0:                         `[]`,
		TypeString:                `["string"]`,
		TypeInteger | TypeBoolean: `["boolean","integer"]`,
		TypeNull | TypeBoolean | TypeObject | TypeArray | TypeNumber | TypeString | TypeInteger: `["null","boolean","object","array","number","string","integer"]`,
	} {
		if got := set.String(); got != want {
			t.Errorf("%08b.String() = %s, want %s", uint8(set), got, want)
		}
	}
	if !(TypeString | TypeNull).Has(TypeNull) || TypeString.Has(TypeString|TypeNull) {
		t.Error("Has")
	}
}
