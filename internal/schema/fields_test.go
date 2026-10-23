// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"fmt"
	"strconv"
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

// TestFieldsCuts is ADR-008 A7 item 5 and A16 for a line's width: a
// constraint, a title and a condition longer than MaxShown bytes are cut,
// and a line's constraints at MaxConstraints, and each mark names the
// command that prints the value whole.
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
	mark := func(n int) string {
		return "[…+" + strconv.Itoa(n) + " bytes; metais schema <package> in prints it whole]"
	}
	// The quote before the é's puts the cut inside a character, so it
	// falls a byte earlier, before it.
	if c, want := root.Constraints[0], `const="`+strings.Repeat("é", 511)+mark(1027); c != want {
		t.Errorf("the const constraint is not cut at %d bytes: %d bytes, ending %q", MaxShown, len(c), c[len(c)-60:])
	}
	if got, want := root.Constraints[len(root.Constraints)-1], "[…; metais schema <package> in prints every constraint]"; got != want || len(root.Constraints) != 5 {
		t.Errorf("the constraints are not cut at %d bytes: %d of them, the last %q", MaxConstraints, len(root.Constraints), got)
	}
	if want := strings.Repeat("é", 512) + mark(1024); *root.Title != want {
		t.Errorf("the title is not cut at %d bytes: %d bytes", MaxShown, len(*root.Title))
	}
	if w := cond.When[0]; !strings.HasSuffix(w, " bytes; metais schema <package> in prints it whole]") || len(w) > len(`/in matches `)+MaxShown+len(mark(9999)) {
		t.Errorf("the condition is not cut: %d bytes, ending %q", len(w), w[len(w)-60:])
	}
}

// ruleLines is each field of fs that states a rule, its pointer then its
// rules in words, one line each.
func ruleLines(fs []Field) []string {
	var out []string
	for _, f := range fs {
		for _, r := range f.Rules {
			out = append(out, f.Pointer+" "+r.Words())
		}
	}
	return out
}

// TestFieldsRules is ADR-003 A2 item 10's listing: each uniqueKeys and
// coverage is a rule of the line of the array whose schema carries it,
// stated in words, a composite key's pointers together; it is reached
// through $ref and allOf as a constraint is, on an array nested in an
// item, and under a condition; a rule a field reaches along two paths is
// stated once; and the keyword stays among the line's constraints. A
// schema with neither keyword states no rule.
func TestFieldsRules(t *testing.T) {
	w := walker(t, `{
		"type": "object",
		"properties": {
			"criteria": {"$ref": "#/$defs/table"},
			"clauses": {"type": "array", "uniqueKeys": ["/standard", "/clause", "/edition"]},
			"sections": {"type": "array", "items": {"properties": {
				"rows": {"allOf": [{"$ref": "#/$defs/keyed"}, {"$ref": "#/$defs/keyed"}]}
			}}},
			"stages": {"if": {"minItems": 2}, "then": {"coverage": {"key": "/n", "values": [1, 2.5, "3", null]}}},
			"plain": {"type": "array", "uniqueItems": true, "items": {"type": "string"}}
		},
		"$defs": {
			"table": {
				"type": "array",
				"uniqueKeys": ["/criterion"],
				"coverage": {"key": "/criterion", "values": ["1.1.1", "1.2.1"]}
			},
			"keyed": {"uniqueKeys": ["/id"]}
		}
	}`)
	fs := w.Fields("in", true, 64)
	got, want := ruleLines(fs), []string{
		"/in/clauses no two rows share `/standard`, `/clause` and `/edition`",
		"/in/criteria rows must include, at `/criterion`, each of: \"1.1.1\", \"1.2.1\"",
		"/in/criteria no two rows share `/criterion`",
		"/in/sections/*/rows no two rows share `/id`",
		"/in/stages rows must include, at `/n`, each of: 1, 2.5, \"3\", null",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("rules:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	for _, f := range fs {
		switch f.Pointer {
		case "/in/criteria":
			if strings.Join(f.Constraints, " ") != `coverage={"key":"/criterion","values":["1.1.1","1.2.1"]} uniqueKeys=["/criterion"]` {
				t.Errorf("the keywords are not among %s's constraints: %v", f.Pointer, f.Constraints)
			}
			if r := f.Rules[0]; r.Keyword != "coverage" || strings.Join(r.Pointers, " ") != "/criterion" || strings.Join(r.Values, " ") != `"1.1.1" "1.2.1"` {
				t.Errorf("the coverage rule is %+v", r)
			}
			if r := f.Rules[1]; r.Keyword != "uniqueKeys" || strings.Join(r.Pointers, " ") != "/criterion" || r.Values != nil {
				t.Errorf("the uniqueKeys rule is %+v", r)
			}
		case "/in/stages":
			if len(f.Rules) > 0 && len(f.When) == 0 {
				t.Errorf("the conditional rule is on a line with no condition")
			}
		case "/in/plain", "/in/plain/*", "/in":
			if len(f.Rules) != 0 {
				t.Errorf("%s states rules it has not: %v", f.Pointer, f.Rules)
			}
		}
	}
	if fs := walker(t, `{"type": "array", "uniqueItems": true, "items": {"type": "object"}}`).Fields("in", true, 64); len(ruleLines(fs)) != 0 {
		t.Errorf("a schema with neither keyword states rules: %v", ruleLines(fs))
	}
}

// TestFieldsRulesWhole is ADR-003 A2 item 10's bound: a rule's values are
// never cut, where the same coverage as a constraint is. The list is the
// smallest past both of the listing's cuts: longer than one constraint's
// MaxShown bytes, and than a line's MaxConstraints.
func TestFieldsRulesWhole(t *testing.T) {
	var values []string
	for i := 0; len(strings.Join(values, ", ")) <= MaxConstraints; i++ {
		values = append(values, strconv.Quote(fmt.Sprintf("criterion %d.%d.%d", i/100+1, i/10%10+1, i%10+1)))
	}
	w := walker(t, `{"type": "array", "coverage": {"key": "/criterion", "values": [`+strings.Join(values, ",")+`]}}`)
	fs := w.Fields("in", true, 64)
	if len(fs) != 1 || len(fs[0].Rules) != 1 {
		t.Fatalf("Fields: %v", fieldLines(fs))
	}
	if c := fs[0].Constraints[0]; !strings.HasSuffix(c, "prints it whole]") {
		t.Errorf("the constraint is not cut at %d bytes: %d bytes", MaxShown, len(c))
	}
	want := "rows must include, at `/criterion`, each of: " + strings.Join(values, ", ")
	if got := fs[0].Rules[0].Words(); got != want {
		t.Errorf("the rule is cut: %d bytes, want %d, ending %q", len(got), len(want), got[max(0, len(got)-60):])
	}
}
