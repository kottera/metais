// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"fmt"
	"strings"
	"testing"
)

// fieldLines is fs, one line each: pointer, type, required, constraints,
// title, when, and the same-as and beyond-value-depth marks.
func fieldLines(fs []Field) []string {
	out := make([]string, len(fs))
	for i, f := range fs {
		title := "-"
		if f.Title != nil {
			title = *f.Title
		}
		typ := "any"
		if len(f.Type) > 0 {
			typ = strings.Join(f.Type, "|")
		}
		line := fmt.Sprintf("%s %s %v [%s] %s", f.Pointer, typ, f.Required, strings.Join(f.Constraints, " "), title)
		if len(f.When) > 0 {
			line += " when " + strings.Join(f.When, " and ")
		}
		if f.SameAs != "" {
			line += " same-as " + f.SameAs
		}
		if f.BeyondValueDepth {
			line += " beyond"
		}
		out[i] = line
	}
	return out
}

func checkFields(t *testing.T, got []Field, want []string) {
	t.Helper()
	g := fieldLines(got)
	if strings.Join(g, "\n") != strings.Join(want, "\n") {
		t.Errorf("Fields:\n%s\nwant:\n%s", strings.Join(g, "\n"), strings.Join(want, "\n"))
	}
}

// TestFieldsReachEveryDeclaration is ADR-008 A7 item 4: the listing
// follows $ref and allOf into a field's own line; walks properties,
// patternProperties, additionalProperties, prefixItems, items and the
// unevaluated forms to the fields beneath; lists a name required and not
// declared; and lists what anyOf, oneOf, then, else and dependentSchemas
// declare under their conditions.
func TestFieldsReachEveryDeclaration(t *testing.T) {
	w := walker(t, `{
		"title": "Root",
		"$ref": "#/$defs/thing",
		"$defs": {
			"thing": {
				"type": "object",
				"required": ["name", "id"],
				"properties": {
					"name": {"$ref": "#/$defs/line", "maxLength": 9, "title": "Name"},
					"tags": {"type": "array", "items": {"type": "string", "x-note": "ignored"}},
					"pair": {"prefixItems": [{"type": "integer"}], "unevaluatedItems": {"type": "null"}},
					"map": {"patternProperties": {"^a": {"type": "boolean"}}, "additionalProperties": {"type": "number"}},
					"open": {"unevaluatedProperties": {"const": 1}},
					"shut": {"additionalProperties": false, "default": {}}
				},
				"allOf": [{"properties": {"name": {"minLength": 1}, "kind": {"enum": ["a", "b"]}}}],
				"anyOf": [{"properties": {"x": {"type": "string"}}}, {"required": ["y"]}],
				"oneOf": [{"title": "One", "type": "object"}],
				"if": {"properties": {"kind": {"const": "a"}}},
				"then": {"required": ["a"]},
				"else": {"properties": {"a": false}},
				"dependentSchemas": {"name": {"required": ["kind"]}}
			},
			"line": {"type": "string", "pattern": "^\\S"}
		}
	}`)
	checkFields(t, w.Fields("in", true, 64), []string{
		`/in object true [] Root`,
		`/in/map any false [] -`,
		`/in/map/* boolean false [name-matches="^a"] -`,
		`/in/map/* number false [] -`,
		`/in/name string true [maxLength=9 pattern="^\\S" minLength=1] Name`,
		`/in/open any false [] -`,
		`/in/open/* any false [const=1] -`,
		`/in/pair any false [] -`,
		`/in/pair/0 integer false [] -`,
		`/in/pair/* null false [] -`,
		`/in/shut any false [additionalProperties=false default={}] -`,
		`/in/tags array false [] -`,
		`/in/tags/* string false [] -`,
		`/in/id any true [] -`,
		`/in/kind any false [enum=["a","b"]] -`,
		`/in/x string false [] - when /in matches anyOf/0`,
		`/in/y any true [] - when /in matches anyOf/1`,
		`/in object false [] One when /in matches oneOf/0`,
		`/in/a any true [] - when /in matches {"properties":{"kind":{"const":"a"}}}`,
		`/in/a any false [false] - when /in does not match {"properties":{"kind":{"const":"a"}}}`,
		`/in/kind any true [] - when /in/name is present`,
	})
}

// TestFieldsCombined is ADR-008 A7 item 2: with no input, the root is
// not listed and its members are, each pointer beginning with the
// member's name; a condition at the root names it as "".
func TestFieldsCombined(t *testing.T) {
	w := walker(t, `{
		"type": "object",
		"required": ["a"],
		"properties": {"a": {"type": "string", "title": "A"}, "b": {"type": "integer"}},
		"if": {"required": ["b"]},
		"then": {"required": ["a"]}
	}`)
	checkFields(t, w.Fields("", false, 64), []string{
		`/a string true [] A`,
		`/b integer false [] -`,
		`/a any true [] - when "" matches {"required":["b"]}`,
	})
}

// TestFieldsRecursive is ADR-008 A7 item 5: a schema that holds itself
// is listed once, and the field that holds it again names where its
// fields are, so the listing ends. Without the mark it never would: one
// level of recursion is the smallest that can loop.
func TestFieldsRecursive(t *testing.T) {
	w := walker(t, `{
		"$ref": "#/$defs/node",
		"$defs": {"node": {"type": "object", "properties": {"child": {"$ref": "#/$defs/node"}, "name": {"type": "string"}}}}
	}`)
	checkFields(t, w.Fields("tree", true, 64), []string{
		`/tree object true [] -`,
		`/tree/child object false [] - same-as /tree`,
		`/tree/name string false [] -`,
	})
}

// TestFieldsSharedOnce is ADR-008 A7 item 5 along the axis of sharing: a
// schema with fields that two members name is expanded beneath the first
// and named by the second. Listed again at every path, a chain of n such
// schemas gives 2^(n+1)-1 lines; two links are the smallest chain where
// that differs from the 2n+1 lines listed once.
func TestFieldsSharedOnce(t *testing.T) {
	w := walker(t, `{
		"$ref": "#/$defs/n0",
		"$defs": {
			"n0": {"properties": {"l": {"$ref": "#/$defs/n1"}, "r": {"$ref": "#/$defs/n1"}}},
			"n1": {"properties": {"l": {"$ref": "#/$defs/n2"}, "r": {"$ref": "#/$defs/n2"}}},
			"n2": {"type": "string"}
		}
	}`)
	checkFields(t, w.Fields("x", true, 64), []string{
		`/x any true [] -`,
		`/x/l any false [] -`,
		`/x/l/l string false [] -`,
		`/x/l/r string false [] -`,
		`/x/r any false [] - same-as /x/l`,
	})
}

// TestFieldsValueDepth is ADR-008 A7 item 5 along the axis of depth: a
// field value-depth tokens beneath its input is listed, and the fields
// beneath it are not, since no input nests deeper. A depth of 1 is the
// smallest at which a schema nested one level further can show it.
func TestFieldsValueDepth(t *testing.T) {
	w := walker(t, `{"properties": {"a": {"properties": {"b": {"type": "string"}}}}}`)
	checkFields(t, w.Fields("in", true, 1), []string{
		`/in any true [] -`,
		`/in/a any false [] - beyond`,
	})
	checkFields(t, w.Fields("in", true, 2), []string{
		`/in any true [] -`,
		`/in/a any false [] -`,
		`/in/a/b string false [] -`,
	})
}

// TestFieldsCuts is ADR-008 A7 item 5 for a line's width: a constraint,
// a title and a condition longer than MaxShown bytes are cut and marked
// with what was cut, and a line's constraints at MaxConstraints, marked
// […].
func TestFieldsCuts(t *testing.T) {
	long := strings.Repeat("é", MaxShown) // 2 bytes each
	w := walker(t, `{
		"title": "`+long+`",
		"const": "`+long+`",
		"allOf": [
			{"enum": ["`+strings.Repeat("a", 900)+`"]}, {"enum": ["`+strings.Repeat("b", 900)+`"]},
			{"enum": ["`+strings.Repeat("c", 900)+`"]}, {"enum": ["`+strings.Repeat("d", 900)+`"]}
		],
		"if": {"const": "`+long+`"},
		"then": {"required": ["z"]}
	}`)
	fs := w.Fields("in", true, 64)
	if len(fs) != 2 {
		t.Fatalf("Fields: %v", fieldLines(fs))
	}
	root, cond := fs[0], fs[1]
	// The quote before the é's puts the cut inside a character, so it
	// falls a byte earlier, before it.
	if c, want := root.Constraints[0], `const="`+strings.Repeat("é", 511)+"[…+1027 bytes]"; c != want {
		t.Errorf("the const constraint is not cut at %d bytes: %d bytes, ending %q", MaxShown, len(c), c[len(c)-20:])
	}
	if got := root.Constraints[len(root.Constraints)-1]; got != "[…]" || len(root.Constraints) != 5 {
		t.Errorf("the constraints are not cut at %d bytes: %d of them, the last %q", MaxConstraints, len(root.Constraints), got)
	}
	if want := shown(long); *root.Title != want || len(want) > MaxShown+len("[…+9999 bytes]") {
		t.Errorf("the title is not cut at %d bytes: %d bytes", MaxShown, len(*root.Title))
	}
	if w := cond.When[0]; !strings.HasSuffix(w, "bytes]") || len(w) > len(`/in matches `)+MaxShown+len("[…+9999 bytes]") {
		t.Errorf("the condition is not cut: %d bytes", len(w))
	}
}
