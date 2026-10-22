// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package xref

import (
	"testing"

	"metais.org/metais/internal/manifest"
)

// partials is a fixture whose output renders main.md.liquid, with the
// partials parts.
func partials(main string, parts map[string]string) fixture {
	ts := map[string]string{"main.md.liquid": main}
	for k, v := range parts {
		ts[k] = v
	}
	return fixture{
		inputs:    []fxInput{in("metrics", tuple)},
		templates: ts,
		outputs:   []manifest.Output{tpl("main", "main.md.liquid")},
	}
}

// TestLoops holds for and tablerow to ADR-007 D3 item 8: the variable is
// the element of the collection, a number over a range, and unknown over
// an object or a scalar.
func TestLoops(t *testing.T) {
	runCases(t, []xcase{
		{"over an array", tuple, `{% for x in inputs.metrics.r %}{{ x.a }}{{ x.c }}{% endfor %}`, nil},
		{"over an array, checked", tuple, `{% for x in inputs.metrics.m %}{{ x.zz }}{% endfor %}`, []string{unknownField}},
		{"over a range", tuple, `{% for i in (1..inputs.metrics.r.size) %}{{ i.x }}{% endfor %}`, []string{notAnObject}},
		{"over an object", tuple, `{% for x in inputs.metrics.o %}{{ x.anything }}{% endfor %}`, nil},
		{"over a string", tuple, `{% for x in inputs.metrics.s %}{{ x.anything }}{% endfor %}`, nil},
		{"the modifiers are analysed", tuple, `{% for x in inputs.metrics.r limit: inputs.metrics.lim offset: inputs.metrics.off %}{% endfor %}`, []string{unknownField, unknownField}},
		{"tablerow", tuple, `{% tablerow x in inputs.metrics.m cols: inputs.metrics.r.size %}{{ x.c }}{{ x.zz }}{% endtablerow %}`, []string{unknownField}},
	})
}

// TestRenderMissing: a render naming no template is
// template.xref.missing_partial.
func TestRenderMissing(t *testing.T) {
	r := partials(`{% render "parts/row.md.liquid" %}{{ inputs.metrics }}`, nil).run(t)
	expect(t, r, `template.xref.missing_partial templates/main.md.liquid:1:1 output="main" partial="parts/row.md.liquid"`)
}

// TestRenderScope holds a partial's scope to ADR-007 D3 item 8: the
// globals, its key: value parameters, and its with or for value under its
// alias, with forloop for for; none of the caller's locals.
func TestRenderScope(t *testing.T) {
	for _, c := range []struct {
		name, main, part string
		want             []string
	}{
		{"a parameter", `{% render "p.liquid", row: inputs.metrics.o.k %}`, `{{ row.x }}{{ row.y }}`, []string{unknownField}},
		{"with, aliased by default to the name's last segment, which no path can read", `{% render "p.liquid" with inputs.metrics.o.k %}`, `{{ p.x }}`, []string{"template.xref.partial_parameter_not_passed"}},
		{"with ... as", `{% render "p.liquid" with inputs.metrics.o.k as row %}`, `{{ row.x }}{{ row.y }}`, []string{unknownField}},
		{"for ... as binds the element and forloop", `{% render "p.liquid" for inputs.metrics.m as row %}`, `{{ row.c }}{{ forloop.index }}{{ row.zz }}`, []string{unknownField}},
		{"the globals are visible", `{% render "p.liquid" %}`, `{{ inputs.metrics.s }}{{ package.name }}{{ output.path }}{{ now }}`, nil},
		{"the caller's locals are not", `{% assign row = inputs.metrics.o %}{% render "p.liquid" %}`, `{{ row.k }}{{ inputs.metrics }}`, []string{"template.xref.partial_parameter_not_passed"}},
		{"forloop is not passed by a plain render", `{% for x in inputs.metrics.r %}{% render "p.liquid" %}{% endfor %}`, `{{ forloop.index }}`, []string{unknownVar}},
		{"a partial's own locals", `{% render "p.liquid" %}`, `{% assign v = inputs.metrics.o %}{{ v.k.x }}{{ v.k.y }}`, []string{unknownField}},
		{"a partial in a directory", `{% render "parts/row" with inputs.metrics.o %}`, ``, []string{unknownField}},
	} {
		t.Run(c.name, func(t *testing.T) {
			name := "p.liquid"
			if c.name == "a partial in a directory" {
				name = "parts/row"
				c.part = `{{ row.k.x }}{{ row.k.y }}`
			}
			got := codes(partials(c.main+`{{ inputs.metrics }}`, map[string]string{name: c.part}).run(t))
			if !equalCodes(got, c.want) {
				t.Errorf("codes %q, want %q", got, c.want)
			}
		})
	}
}

// TestRenderParameterNotPassed holds an unpassed parameter to ADR-007 D3
// item 8: a warning naming the call site, once per call context, and the
// name is unknown below it, so an optional parameter is no error.
func TestRenderParameterNotPassed(t *testing.T) {
	main := "{% render \"card.liquid\", title: inputs.metrics.s %}\n{% render \"card.liquid\", title: inputs.metrics.s, note: inputs.metrics.o %}"
	r := partials(main, map[string]string{"card.liquid": `{{ title }}{% if note %}{{ note.k.x }}{{ note.anything }}{% endif %}`}).run(t)
	expect(t, r,
		`template.xref.partial_parameter_not_passed templates/card.liquid:1:18 name="note" output="main" related=templates/main.md.liquid:1`,
		`template.xref.partial_parameter_not_passed templates/card.liquid:1:28 name="note" output="main" related=templates/main.md.liquid:1`,
		`template.xref.partial_parameter_not_passed templates/card.liquid:1:42 name="note" output="main" related=templates/main.md.liquid:1`,
		`template.xref.unknown_field templates/card.liquid:1:46 declared=["k"] declared_count=1 input="metrics" output="main" property="anything" related=inputs/metrics.json#/properties/o`,
	)
	// Two call sites that pass different values, neither passing note:
	// each is named.
	main = "{% render \"card.liquid\", title: inputs.metrics.s %}\n{% render \"card.liquid\", title: 1 %}"
	r = partials(main, map[string]string{"card.liquid": `{{ title }}{{ note }}`}).run(t)
	expect(t, r,
		`template.xref.partial_parameter_not_passed templates/card.liquid:1:15 name="note" output="main" related=templates/main.md.liquid:1`,
		`template.xref.partial_parameter_not_passed templates/card.liquid:1:15 name="note" output="main" related=templates/main.md.liquid:2`,
	)
}

// TestRenderMemoised: a partial is analysed once per call context and
// bound values, and a recursive partial already on the stack with the same
// bindings is not entered again.
func TestRenderMemoised(t *testing.T) {
	main := `{% render "p.liquid", v: inputs.metrics.o %}{% render "p.liquid", v: inputs.metrics.o %}{% render "p.liquid", v: inputs.metrics.r %}`
	r := partials(main, map[string]string{"p.liquid": `{{ v }}`}).run(t)
	expect(t, r)
	if r.Contexts != 3 {
		t.Errorf("contexts = %d, want 3: main, and p with two distinct bindings", r.Contexts)
	}
	tree := `{{ node.name }}{{ node.nmae }}{% for c in node.children %}{% render "tree.liquid", node: c %}{% endfor %}`
	s := `{"$ref": "#/$defs/node", "$defs": {"node": {"type": "object", "properties": {"name": {}, "children": {"type": "array", "items": {"$ref": "#/$defs/node"}}}}}}`
	fx := fixture{
		inputs:    []fxInput{in("metrics", s)},
		templates: map[string]string{"main.md.liquid": `{% render "tree.liquid", node: inputs.metrics %}`, "tree.liquid": tree},
		outputs:   []manifest.Output{tpl("main", "main.md.liquid")},
	}
	r = fx.run(t)
	// node.nmae is read in two contexts, whose node is a different schema:
	// the input's root, and the children's items.
	expect(t, r,
		`template.xref.unknown_field templates/tree.liquid:1:23 declared=["children","name"] declared_count=2 input="metrics" output="main" property="nmae" suggestion="name" related=inputs/metrics.json#`,
		`template.xref.unknown_field templates/tree.liquid:1:23 declared=["children","name"] declared_count=2 input="metrics" output="main" property="nmae" suggestion="name" related=inputs/metrics.json#/$defs/node/properties/children/items`,
	)
	if r.Contexts > 4 {
		t.Errorf("recursive partial: %d contexts", r.Contexts)
	}
}

// TestRenderUnparsedPartial: a partial that did not parse is no missing
// partial, and leaves no input unused, since what it reads is unknown.
func TestRenderUnparsedPartial(t *testing.T) {
	fx := partials(`{% render "p.liquid" %}`, map[string]string{"p.liquid": `{{`})
	fx.broken = []string{"p.liquid"}
	expect(t, fx.run(t))
}
