// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/regexprofile"
)

// refChain is a schema file whose root refers to /$defs/s0, each of whose
// n-1 definitions refers to the next: a chain of n references at one
// instance location, ending in an empty schema.
func refChain(n int) string {
	defs := map[string]any{}
	for i := range n - 1 {
		defs["s"+strconv.Itoa(i)] = map[string]any{"$ref": "#/$defs/s" + strconv.Itoa(i+1)}
	}
	defs["s"+strconv.Itoa(n-1)] = map[string]any{}
	b, err := json.Marshal(map[string]any{"$ref": "#/$defs/s0", "$defs": defs})
	if err != nil {
		panic(err)
	}
	return string(b)
}

// wantNoObserved checks that fs is one limit.package finding of limit at
// entry#ptr, with no observed value.
func wantNoObserved(t *testing.T, fs []finding.Finding, limit, entry, ptr string) {
	t.Helper()
	if len(fs) != 1 {
		t.Fatalf("%d findings, want one %s:\n%s", len(fs), limit, describe(fs))
	}
	f := fs[0]
	l, _ := f.Location()
	if f.Code() != finding.CodeLimitPackage || l.File != entry || !l.HasPointer || l.Pointer != ptr {
		t.Fatalf("got %s at %s#%s, want limit.package at %s#%s", f.Code(), l.File, l.Pointer, entry, ptr)
	}
	if got := param(t, f, "limit"); got != limit {
		t.Errorf("limit is %q, want %q", got, limit)
	}
	if _, ok := f.Param("observed"); ok {
		t.Error("an unbounded chain observes a value")
	}
}

// TestRefDepth is ref-depth at its boundary (ADR-009 D3): a chain of
// references at one instance location passes at its own length, and one
// reference fewer refuses it with limit.package at the first reference,
// in the order the pre-check walks, that begins a chain longer than the
// bound, observing that chain's length. The walk visits the root's $defs
// before its $ref, so the first such reference is a definition's.
func TestRefDepth(t *testing.T) {
	files := map[string]string{"inputs/s.json": refChain(5)}
	if _, fs := compileFiles(t, lower(t, "ref-depth=5"), files); len(fs) > 0 {
		t.Fatalf("at the bound:\n%s", describe(fs))
	}
	_, fs := compileFiles(t, lower(t, "ref-depth=4"), files)
	wantBreach(t, fs, "ref-depth", "inputs/s.json", "/$ref", 5)
	_, fs = compileFiles(t, lower(t, "ref-depth=3"), files)
	wantBreach(t, fs, "ref-depth", "inputs/s.json", "/$defs/s0/$ref", 4)
}

// TestRefDepthChains holds ref-depth to chains of subschemas that each
// apply where the one before applies: the in-place applicators carry a
// chain on without counting, a reference to another file counts, and an
// applicator that moves to a member or an item starts a new chain.
func TestRefDepthChains(t *testing.T) {
	for _, c := range []struct {
		name  string
		files map[string]string
		hops  int64
		ptr   string
	}{
		{"in-place applicators", map[string]string{"inputs/s.json": `{"allOf": [{"anyOf": [{"$ref": "#/$defs/a"}]}],
			"$defs": {"a": {"oneOf": [{"not": {"$ref": "#/$defs/b"}}]}, "b": {"if": {"$ref": "#/$defs/c"}},
			"c": {"dependentSchemas": {"x": {"then": {"else": {"$ref": "#/$defs/d"}}}}}, "d": {}}}`}, 4, "/allOf/0/anyOf/0/$ref"},
		{"across files", map[string]string{
			"inputs/a.json": `{"$ref": "b.json"}`,
			"inputs/b.json": `{"$ref": "c.json#/$defs/x"}`,
			"inputs/c.json": `{"$defs": {"x": {"$ref": "#/$defs/y"}, "y": {"$ref": "#y"}, "z": {"$anchor": "y"}}}`,
		}, 4, "/$ref"},
		{"a member starts a new chain", map[string]string{"inputs/s.json": `{"$ref": "#/$defs/a",
			"$defs": {"a": {"properties": {"p": {"$ref": "#/$defs/b"}}, "$ref": "#/$defs/b"}, "b": {"$ref": "#/$defs/c"}, "c": {}}}`}, 3, "/$ref"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, fs := compileFiles(t, lower(t, fmt.Sprintf("ref-depth=%d", c.hops)), c.files); len(fs) > 0 {
				t.Fatalf("at the bound:\n%s", describe(fs))
			}
			_, fs := compileFiles(t, lower(t, fmt.Sprintf("ref-depth=%d", c.hops-1)), c.files)
			entry := "inputs/s.json"
			if _, ok := c.files[entry]; !ok {
				entry = "inputs/a.json"
			}
			wantBreach(t, fs, "ref-depth", entry, c.ptr, c.hops)
		})
	}
}

// TestRefDepthCycle refuses a cycle of references that consumes no
// instance, at every bound, with no observed length, before the library
// sees it; a cycle that moves to an item each time round is a recursive
// schema, and passes.
func TestRefDepthCycle(t *testing.T) {
	for _, c := range []struct {
		name, schema, ptr string
	}{
		{"two definitions", `{"$ref": "#/$defs/a", "$defs": {"a": {"$ref": "#/$defs/b"}, "b": {"allOf": [{"$ref": "#/$defs/a"}]}}}`, "/$defs/a/$ref"},
		{"the root", `{"anyOf": [{"$ref": "#"}]}`, "/anyOf/0/$ref"},
		{"a dynamic reference", `{"$dynamicAnchor": "n", "not": {"$dynamicRef": "#n"}}`, "/not/$dynamicRef"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, fs := compileFiles(t, limits.Defaults(), map[string]string{"inputs/s.json": c.schema})
			wantNoObserved(t, fs, "ref-depth", "inputs/s.json", c.ptr)
		})
	}
	if _, fs := compileFiles(t, lower(t, "ref-depth=1"), map[string]string{
		"inputs/s.json": `{"items": {"$ref": "#"}, "properties": {"p": {"$dynamicRef": "#n"}}, "$dynamicAnchor": "n"}`,
	}); len(fs) > 0 {
		t.Errorf("a recursive schema:\n%s", describe(fs))
	}
}

// workSchema costs 12 units against workInstance: the root (1 unit, and 2
// for its members), /properties/a (1, and 2 for its items),
// /properties/a/items twice (1 each), /additionalProperties for "b" (1),
// and /allOf/0 at the root (1, and 2 for its members).
const (
	workSchema   = `{"properties": {"a": {"items": {"type": "integer"}}}, "additionalProperties": {"type": "string"}, "allOf": [{"required": ["a"]}]}`
	workInstance = `{"a": [1, 2], "b": "x"}`
	workUnits    = 12
)

// TestSchemaWork is schema-work at its boundary (ADR-009 D3): a
// validation of exactly the bound passes, and one unit fewer refuses it
// with a *regexprofile.LimitError naming schema-work and no failures,
// before the library evaluates anything.
func TestSchemaWork(t *testing.T) {
	files := map[string]string{"inputs/s.json": workSchema}
	for _, c := range []struct {
		bound  int64
		refuse bool
	}{{workUnits, false}, {workUnits - 1, true}} {
		t.Run(strconv.FormatInt(c.bound, 10), func(t *testing.T) {
			comp, fs := compileFiles(t, lower(t, fmt.Sprintf("schema-work=%d", c.bound)), files)
			if len(fs) > 0 {
				t.Fatal(describe(fs))
			}
			s, _ := comp.Schema("inputs/s.json")
			fails, err := s.Validate(mustValue(t, workInstance))
			var le *regexprofile.LimitError
			switch {
			case !c.refuse && (err != nil || len(fails) > 0):
				t.Fatalf("within the bound: %v, %v", fails, err)
			case c.refuse && (!errors.As(err, &le) || le.Limit != limits.SchemaWork || fails != nil):
				t.Fatalf("past the bound: %v, %v; want schema-work and no failures", fails, err)
			}
		})
	}
}

// TestSchemaWorkCommand holds schema-work to one command: validations
// through one compiler draw on one budget, so the second of two that
// each fit alone is refused.
func TestSchemaWorkCommand(t *testing.T) {
	comp, fs := compileFiles(t, lower(t, fmt.Sprintf("schema-work=%d", workUnits+workUnits/2)), map[string]string{"inputs/s.json": workSchema})
	if len(fs) > 0 {
		t.Fatal(describe(fs))
	}
	s, _ := comp.Schema("inputs/s.json")
	if _, err := s.Validate(mustValue(t, workInstance)); err != nil {
		t.Fatalf("the first validation: %v", err)
	}
	var le *regexprofile.LimitError
	if _, err := s.Validate(mustValue(t, workInstance)); !errors.As(err, &le) || le.Limit != limits.SchemaWork {
		t.Fatalf("the second validation: %v, want schema-work", err)
	}
}

// TestSchemaWorkCountsEveryBranch counts each applicator whatever another
// evaluation's outcome: every branch of anyOf and oneOf; if with both then
// and else, but for the one a boolean if never reaches; a dependent schema
// at the object it depends on; and patternProperties only for the members
// whose names match.
func TestSchemaWorkCountsEveryBranch(t *testing.T) {
	for _, c := range []struct {
		name, schema, instance string
		units                  int64
	}{
		{"anyOf and oneOf", `{"anyOf": [true, {}], "oneOf": [{}, false]}`, `1`, 5},
		{"if, then and else", `{"if": {}, "then": {}, "else": {}}`, `1`, 4},
		{"if true, and then", `{"if": true, "then": {}, "else": {}}`, `1`, 3},
		{"if false, and else", `{"if": false, "then": {}, "else": {}}`, `1`, 3},
		{"patternProperties", `{"patternProperties": {"^a": {}, "b": {}}, "additionalProperties": false}`, `{"ab": 1, "c": 2}`, 6},
		{"unevaluatedProperties", `{"properties": {"a": {}}, "unevaluatedProperties": {}}`, `{"a": 1, "b": 2}`, 5},
		{"prefixItems, items, contains and unevaluatedItems", `{"prefixItems": [{}], "contains": {}, "unevaluatedItems": {}}`, `[1, 2]`, 7},
		{"propertyNames", `{"propertyNames": {"maxLength": 3}}`, `{"a": 1, "b": 2}`, 5},
		{"dependentSchemas", `{"dependentSchemas": {"a": {}, "z": {}}}`, `{"a": 1}`, 4},
		{"a dynamic reference", `{"$dynamicAnchor": "n", "properties": {"p": {"$dynamicRef": "#n"}}}`, `{"p": {}}`, 4},
	} {
		t.Run(c.name, func(t *testing.T) {
			comp, fs := compileFiles(t, lower(t, fmt.Sprintf("schema-work=%d", c.units)), map[string]string{"inputs/s.json": c.schema})
			if len(fs) > 0 {
				t.Fatal(describe(fs))
			}
			s, _ := comp.Schema("inputs/s.json")
			if _, err := s.Validate(mustValue(t, c.instance)); err != nil {
				t.Fatalf("at %d units: %v", c.units, err)
			}
			comp, _ = compileFiles(t, lower(t, fmt.Sprintf("schema-work=%d", c.units-1)), map[string]string{"inputs/s.json": c.schema})
			s, _ = comp.Schema("inputs/s.json")
			var le *regexprofile.LimitError
			if _, err := s.Validate(mustValue(t, c.instance)); !errors.As(err, &le) || le.Limit != limits.SchemaWork {
				t.Fatalf("at %d units: %v, want schema-work", c.units-1, err)
			}
		})
	}
}

// TestSchemaWorkExponential is the case schema-work exists for: anyOf
// nested through references that consume no instance, each level doubling
// the evaluations. Thirty levels would take 2^30 evaluations; the count
// stops at the bound, so the validation is refused rather than run.
func TestSchemaWorkExponential(t *testing.T) {
	const levels = 30
	defs := map[string]any{}
	for i := range levels {
		next := map[string]any{"$ref": "#/$defs/s" + strconv.Itoa(i+1)}
		defs["s"+strconv.Itoa(i)] = map[string]any{"anyOf": []any{next, next}}
	}
	defs["s"+strconv.Itoa(levels)] = map[string]any{"type": "string"}
	b, _ := json.Marshal(map[string]any{"$ref": "#/$defs/s0", "$defs": defs})
	lim := lower(t, "schema-work=1000000")
	comp, fs := compileFiles(t, lim, map[string]string{"inputs/s.json": string(b)})
	if len(fs) > 0 {
		t.Fatal(describe(fs))
	}
	s, _ := comp.Schema("inputs/s.json")
	var le *regexprofile.LimitError
	if _, err := s.Validate(mustValue(t, `1`)); !errors.As(err, &le) || le.Limit != limits.SchemaWork {
		t.Fatalf("got %v, want schema-work", err)
	}
}

// TestCombinedSchemaDepth holds the combined schema to its files' depths:
// a file at schema-depth is embedded under $defs, a level below the
// combined schema's root, and the combined schema compiles all the same,
// since what it embeds is the files the pre-check has passed.
func TestCombinedSchemaDepth(t *testing.T) {
	c, fs := compileFiles(t, lower(t, "schema-depth=3"), map[string]string{"inputs/a.json": `{"items": {"items": {}}}`})
	if len(fs) > 0 {
		t.Fatal(describe(fs))
	}
	doc, err := Bundle(Declarations{Inputs: []Input{{Name: "a", Title: "A", Schema: "inputs/a.json", Required: true}}},
		[]File{{Path: "inputs/a.json", Root: c.docs["inputs/a.json"]}})
	if err != nil {
		t.Fatal(err)
	}
	if _, fs := c.Combined(doc); len(fs) > 0 {
		t.Fatal(describe(fs))
	}
}
