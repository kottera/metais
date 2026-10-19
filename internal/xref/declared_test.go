// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package xref

import "testing"

// metricsSchema is the worked resolution's schema (ADR-007 D3): metrics
// is {type: object, properties: {rows: {$ref: "#/$defs/rows"}}}, and
// $defs/rows is {type: array, items: {type: object, properties: {value:
// {type: number}}}}.
const metricsSchema = `{
	"type": "object",
	"properties": {"rows": {"$ref": "#/$defs/rows"}},
	"$defs": {"rows": {"type": "array", "items": {"type": "object", "properties": {"value": {"type": "number"}}}}}
}`

// TestWorkedResolution is ADR-007 D3's worked resolution:
// inputs.metrics.rows[0].value is declared, and .vaule stops at the last
// step with template.xref.unknown_field, declared names value, suggestion
// value.
func TestWorkedResolution(t *testing.T) {
	r := one(metricsSchema, "{{ inputs.metrics.rows[0].value }}").run(t)
	expect(t, r)
	r = one(metricsSchema, "{{ inputs.metrics.rows[0].vaule }}").run(t)
	expect(t, r, `template.xref.unknown_field templates/report.md.liquid:1:26 declared=["value"] declared_count=1 input="metrics" output="report" property="vaule" suggestion="value" related=inputs/metrics.json#/$defs/rows/items`)
}

// xcase is a template read against one input's schema and the codes its
// findings have, sorted.
type xcase struct {
	name, schema, text string
	want               []string
}

func runCases(t *testing.T, cases []xcase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := codes(one(c.schema, c.text).run(t))
			if !equalCodes(got, c.want) {
				t.Errorf("%s: codes %q, want %q", c.text, got, c.want)
			}
		})
	}
}

func equalCodes(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

const (
	unknownField = "template.xref.unknown_field"
	notAnObject  = "template.xref.not_an_object"
	notAnArray   = "template.xref.not_an_array"
	outOfRange   = "template.xref.index_out_of_range"
	dynamicRef   = "template.xref.dynamic_reference"
	unknownVar   = "template.xref.unknown_variable"
)

// TestDeclared holds each step of ADR-007 D3 item 4's "declared" to its
// wording.
func TestDeclared(t *testing.T) {
	runCases(t, []xcase{
		// 4.3: properties[p].
		{"properties", `{"type": "object", "properties": {"a": {"type": "string"}}}`, `{{ inputs.metrics.a }}`, nil},
		// 4.3: every patternProperties schema whose pattern matches p.
		{"patternProperties match", `{"type": "object", "patternProperties": {"^x_": {"type": "string"}}}`, `{{ inputs.metrics.x_a }}`, nil},
		{"patternProperties no match", `{"type": "object", "patternProperties": {"^x_": {"type": "string"}}}`, `{{ inputs.metrics.y }}`, []string{unknownField}},
		{"patternProperties unanchored", `{"type": "object", "patternProperties": {"b": {}}}`, `{{ inputs.metrics.abc }}`, nil},
		// 4.3: additionalProperties when present and not false, true included.
		{"additionalProperties true", `{"type": "object", "properties": {"a": {}}, "additionalProperties": true}`, `{{ inputs.metrics.z }}`, nil},
		{"additionalProperties schema", `{"type": "object", "additionalProperties": {"type": "object", "properties": {"v": {}}}}`, `{{ inputs.metrics.any.v }}`, nil},
		{"additionalProperties schema checked", `{"type": "object", "additionalProperties": {"type": "object", "properties": {"v": {}}}}`, `{{ inputs.metrics.any.w }}`, []string{unknownField}},
		{"additionalProperties false", `{"type": "object", "properties": {"a": {}}, "additionalProperties": false}`, `{{ inputs.metrics.z }}`, []string{unknownField}},
		// 4.4: an absent additionalProperties declares nothing.
		{"additionalProperties absent", `{"type": "object", "properties": {"a": {}}}`, `{{ inputs.metrics.z }}`, []string{unknownField}},
		// 4.3: additionalProperties applies only to a name properties and
		// patternProperties do not declare.
		{"additionalProperties false beside properties", `{"type": "object", "properties": {"a": {"type": "object", "properties": {"b": {}}}}, "additionalProperties": false}`, `{{ inputs.metrics.a.b }}`, nil},
		// 4.3: unevaluatedProperties when there is no additionalProperties.
		{"unevaluatedProperties", `{"type": "object", "unevaluatedProperties": {"type": "string"}}`, `{{ inputs.metrics.z }}`, nil},
		{"unevaluatedProperties false", `{"type": "object", "unevaluatedProperties": false}`, `{{ inputs.metrics.z }}`, []string{unknownField}},
		{"additionalProperties before unevaluatedProperties", `{"type": "object", "additionalProperties": false, "unevaluatedProperties": true}`, `{{ inputs.metrics.z }}`, []string{unknownField}},
		// 4.2: an open schema declares everything below it.
		{"open: true", `true`, `{{ inputs.metrics.a.b[3].c }}`, nil},
		{"open: annotations only", `{"title": "T", "description": "D", "$comment": "c", "examples": [{}], "default": {}, "deprecated": false, "readOnly": false, "writeOnly": false, "$anchor": "x", "$defs": {"q": {"type": "string"}}}`, `{{ inputs.metrics.a.b }}`, nil},
		{"open: an empty property schema", `{"type": "object", "properties": {"a": {}}}`, `{{ inputs.metrics.a.b.c }}`, nil},
		{"not open: a type", `{"type": "object"}`, `{{ inputs.metrics.a }}`, []string{unknownField}},
		{"not open: false", `{"type": "object", "properties": {"a": false}}`, `{{ inputs.metrics.a.b }}`, []string{notAnObject}},
		// 4.1: through $ref, and a recursive $ref, cut by a visited set.
		{"$ref", `{"$ref": "#/$defs/o", "$defs": {"o": {"type": "object", "properties": {"a": {}}}}}`, `{{ inputs.metrics.a }}`, nil},
		{"recursive $ref", `{"$ref": "#/$defs/node", "$defs": {"node": {"type": "object", "properties": {"name": {"type": "string"}, "children": {"type": "array", "items": {"$ref": "#/$defs/node"}}}}}}`, `{{ inputs.metrics.children[0].children[1].name }}{{ inputs.metrics.children[0].nmae }}`, []string{unknownField}},
		{"$ref cycle", `{"$ref": "#/$defs/a", "$defs": {"a": {"$ref": "#/$defs/b"}, "b": {"$ref": "#/$defs/a", "type": "object", "properties": {"x": {}}}}}`, `{{ inputs.metrics.x }}{{ inputs.metrics.y }}`, []string{unknownField}},
		{"$ref to another file", `{"type": "object", "properties": {"addr": {"$ref": "address.json"}}}`, `{{ inputs.metrics.addr.city }}`, nil},
		{"$ref to another file checked", `{"type": "object", "properties": {"addr": {"$ref": "address.json"}}}`, `{{ inputs.metrics.addr.town }}`, []string{unknownField}},
		// 4.1: allOf, anyOf, oneOf, if, then, else, dependentSchemas.
		{"allOf", `{"allOf": [{"type": "object", "properties": {"a": {}}}, {"properties": {"b": {}}}]}`, `{{ inputs.metrics.a }}{{ inputs.metrics.b }}`, nil},
		{"anyOf", `{"type": "object", "anyOf": [{"properties": {"a": {}}}, {"properties": {"b": {}}}]}`, `{{ inputs.metrics.b }}`, nil},
		{"a field in one oneOf branch", `{"type": "object", "oneOf": [{"properties": {"kind": {"const": "a"}, "x": {}}}, {"properties": {"kind": {"const": "b"}, "y": {}}}]}`, `{{ inputs.metrics.y }}`, nil},
		{"a field in then", `{"type": "object", "properties": {"kind": {}}, "if": {"properties": {"kind": {"const": "a"}}}, "then": {"properties": {"x": {}}}, "else": {"properties": {"y": {}}}}`, `{{ inputs.metrics.x }}{{ inputs.metrics.y }}`, nil},
		{"dependentSchemas", `{"type": "object", "properties": {"a": {}}, "dependentSchemas": {"a": {"properties": {"b": {}}}}}`, `{{ inputs.metrics.b }}`, nil},
		// 4.1: not is excluded: a property named in not must not match.
		{"a field only in not", `{"type": "object", "properties": {"a": {}}, "not": {"properties": {"b": {}}, "required": ["b"]}}`, `{{ inputs.metrics.b }}`, []string{unknownField}},
		// 4.5: Liquid's command names, and a declared property winning.
		{"size of an object", `{"type": "object", "properties": {"a": {}}}`, `{{ inputs.metrics.size }}`, nil},
		{"size of a string", `{"type": "object", "properties": {"s": {"type": "string"}}}`, `{{ inputs.metrics.s.size }}`, nil},
		{"size of a number", `{"type": "object", "properties": {"n": {"type": "number"}}}`, `{{ inputs.metrics.n.size }}`, []string{notAnObject}},
		{"first and last of an array", `{"type": "array", "items": {"type": "object", "properties": {"v": {}}}}`, `{{ inputs.metrics.first.v }}{{ inputs.metrics.last.v }}{{ inputs.metrics.first.w }}`, []string{unknownField}},
		{"first of a string", `{"type": "object", "properties": {"s": {"type": "string"}}}`, `{{ inputs.metrics.s.first.size }}`, nil},
		{"first of an object", `{"type": "object", "properties": {"a": {}}}`, `{{ inputs.metrics.first }}`, []string{unknownField}},
		{"a declared size wins", `{"type": "object", "properties": {"size": {"type": "object", "properties": {"w": {}}}}}`, `{{ inputs.metrics.size.w }}{{ inputs.metrics.size.h }}`, []string{unknownField}},
		{"a declared first wins", `{"type": "array", "items": {"type": "string"}, "properties": {"first": {"type": "object", "properties": {"k": {}}}}}`, `{{ inputs.metrics.first.k }}`, nil},
		// 4.6: not_an_object when the node's types exclude object.
		{"a property of a string", `{"type": "object", "properties": {"s": {"type": "string"}}}`, `{{ inputs.metrics.s.x }}`, []string{notAnObject}},
		{"a property of an array", `{"type": "array", "items": {}}`, `{{ inputs.metrics.x }}`, []string{notAnObject}},
		// Union semantics: an error only when every alternative rejects.
		{"one alternative declares it", `{"type": "object", "properties": {"a": {"type": "object", "properties": {"p": {}}}, "b": {"type": "object", "properties": {"q": {}}}}}`, `{% assign x = inputs.metrics.a %}{% assign x = inputs.metrics.b %}{{ x.q }}`, nil},
		{"no alternative declares it", `{"type": "object", "properties": {"a": {"type": "object", "properties": {"p": {}}}, "b": {"type": "object", "properties": {"q": {}}}}}`, `{% assign x = inputs.metrics.a %}{% assign x = inputs.metrics.b %}{{ x.r }}`, []string{unknownField}},
		{"a string literal index is a property", `{"type": "object", "properties": {"a b": {}}}`, `{{ inputs.metrics["a b"] }}{{ inputs.metrics["a c"] }}`, []string{unknownField}},
	})
}

// TestDeclaredTypes holds a node's types to ADR-007 D3 item 5: all when
// absent, intersected through $ref and allOf, united across anyOf and
// oneOf, and not narrowed by if, then or else.
func TestDeclaredTypes(t *testing.T) {
	for _, c := range []struct {
		name, schema, types string
	}{
		{"absent", `{"properties": {"s": {"enum": [1]}}}`, ""},
		{"a type", `{"properties": {"s": {"type": "string"}}}`, `["string"]`},
		{"anyOf unites", `{"properties": {"s": {"anyOf": [{"type": "string"}, {"type": "integer"}]}}}`, `["string","integer"]`},
		{"oneOf unites", `{"properties": {"s": {"oneOf": [{"type": "null"}, {"type": "number"}]}}}`, `["null","number"]`},
		{"allOf intersects", `{"properties": {"s": {"type": ["string", "number", "object"], "allOf": [{"type": ["string", "integer"]}]}}}`, `["string","integer"]`},
		{"$ref intersects", `{"properties": {"s": {"type": ["string", "boolean"], "$ref": "#/$defs/b"}}, "$defs": {"b": {"type": ["boolean", "object", "null"]}}}`, `["boolean"]`},
		{"if does not narrow", `{"properties": {"s": {"type": ["string", "boolean"], "if": {"type": "string"}, "then": {"type": "string"}, "else": {"type": "string"}}}}`, `["boolean","string"]`},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := one(`{"type": "object", `+c.schema[1:], `{{ inputs.metrics.s.p }}`).run(t)
			if c.types == "" {
				// Every type includes object: the name is undeclared.
				if got := codes(r); !equalCodes(got, []string{unknownField}) {
					t.Errorf("codes %q, want unknown_field", got)
				}
				return
			}
			if len(r.Findings) != 1 || string(r.Findings[0].Code()) != notAnObject {
				t.Fatalf("findings %q, want one not_an_object", describeAll(t, r))
			}
			got := describe(t, r.Findings[0])
			if want := "types=" + c.types; !contains(got, want) {
				t.Errorf("%s, want %s", got, want)
			}
		})
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// TestDeclaredFindingNames holds unknown_field to ADR-007 D3 item 4.6: it
// names the template location, the input, the schema location as a
// related location, the declared names there and the nearest within
// edit distance 2; and to not_an_object's types.
func TestDeclaredFindingNames(t *testing.T) {
	s := `{"type": "object", "properties": {"zeta": {}, "alpha": {}, "beta": {"type": ["string", "null"]}}, "allOf": [{"properties": {"gamma": {}}}]}`
	r := one(s, "line one\n  {{ inputs.metrics.alpah }}{{ inputs.metrics.omega }}{{ inputs.metrics.beta.x }}").run(t)
	expect(t, r,
		`template.xref.unknown_field templates/report.md.liquid:2:20 declared=["alpha","beta","gamma","zeta"] declared_count=4 input="metrics" output="report" property="alpah" suggestion="alpha" related=inputs/metrics.json#`,
		`template.xref.unknown_field templates/report.md.liquid:2:46 declared=["alpha","beta","gamma","zeta"] declared_count=4 input="metrics" output="report" property="omega" related=inputs/metrics.json#`,
		`template.xref.not_an_object templates/report.md.liquid:2:77 input="metrics" output="report" property="x" types=["null","string"] related=inputs/metrics.json#/properties/beta`,
	)
}

// TestDeclaredBoundsTheNames: unknown_field lists at most 20 declared
// names, and counts them all.
func TestDeclaredBoundsTheNames(t *testing.T) {
	props := ""
	for i := range 25 {
		if i > 0 {
			props += ", "
		}
		props += `"p` + string(rune('a'+i)) + `": {}`
	}
	r := one(`{"type": "object", "properties": {`+props+`}}`, `{{ inputs.metrics.zz }}`).run(t)
	if len(r.Findings) != 1 {
		t.Fatalf("findings %q", describeAll(t, r))
	}
	d := describe(t, r.Findings[0])
	if !contains(d, "declared_count=25") || !contains(d, `"pt"]`) || contains(d, `"pu"`) {
		t.Errorf("%s: want the first 20 of 25 names", d)
	}
}
