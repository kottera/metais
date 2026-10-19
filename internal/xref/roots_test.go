// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package xref

import (
	"testing"

	"github.com/kottera/metais/internal/manifest"
)

const objectAB = `{"type": "object", "properties": {"a": {"type": "string"}, "b": {"type": "array", "items": {"type": "object", "properties": {"v": {}}}}}}`

// TestRoots holds each root of ADR-007 D3 item 3 to its wording.
func TestRoots(t *testing.T) {
	runCases(t, []xcase{
		{"inputs", objectAB, `{{ inputs.metrics.a }}{{ inputs.metrics.b[0].v }}`, nil},
		{"package", objectAB, `{{ package.metais }}{{ package.name }}{{ package.version }}{{ package.title }}{{ package.description }}{{ package.guide }}{{ package.license }}{{ package.homepage }}{{ package.digest }}{{ package.authors[0].name }}{{ package.authors[0].email }}{{ package.authors[0].url }}{{ inputs.metrics }}`, nil},
		{"package is closed", objectAB, `{{ package.inputs }}{{ inputs.metrics }}`, []string{unknownField}},
		{"now", objectAB, `{{ now }}{{ now.size }}{{ inputs.metrics }}`, nil},
		{"sources", objectAB, `{{ sources.metrics.origin }}{{ inputs.metrics }}`, nil},
		{"sources is closed", objectAB, `{{ sources.metrics.form }}{{ inputs.metrics }}`, []string{unknownField}},
		{"output in a template body", objectAB, `{{ output.path }}{{ output.root }}{{ inputs.metrics }}`, nil},
		{"output is closed", objectAB, `{{ output.name }}{{ inputs.metrics }}`, []string{unknownField}},
		{"item outside for_each", objectAB, `{{ item.a }}{{ inputs.metrics }}`, []string{unknownVar}},
		{"fan_out outside for_each", objectAB, `{{ fan_out.index }}{{ inputs.metrics }}`, []string{unknownVar}},
		{"forloop inside for", objectAB, `{% for r in inputs.metrics.b %}{{ forloop.index }}{{ forloop.first }}{{ forloop.name }}{{ forloop.length }}{% for s in inputs.metrics.b %}{{ forloop.parentloop.index0 }}{% endfor %}{% endfor %}`, nil},
		{"forloop is closed", objectAB, `{% for r in inputs.metrics.b %}{{ forloop.idx }}{% endfor %}`, []string{unknownField}},
		{"forloop outside for", objectAB, `{% for r in inputs.metrics.b %}{% endfor %}{{ forloop.index }}`, []string{unknownVar}},
		{"forloop in a for's else", objectAB, `{% for r in inputs.metrics.b %}{% else %}{{ forloop.index }}{% endfor %}`, []string{unknownVar}},
		{"tablerowloop inside tablerow", objectAB, `{% tablerow r in inputs.metrics.b cols: 2 %}{{ tablerowloop.col }}{{ tablerowloop.row }}{{ r.v }}{% endtablerow %}`, nil},
		{"tablerowloop outside tablerow", objectAB, `{% for r in inputs.metrics.b %}{{ tablerowloop.col }}{% endfor %}`, []string{unknownVar}},
		{"a for variable outside its for", objectAB, `{% for r in inputs.metrics.b %}{% endfor %}{{ r.v }}`, []string{unknownVar}},
		{"a block binding shadows a global", objectAB, `{% for inputs in inputs.metrics.b %}{{ inputs.v }}{{ inputs.w }}{% endfor %}`, []string{unknownField}},
		{"anything else", objectAB, `{{ vendor.name }}{{ inputs.metrics }}`, []string{unknownVar}},
		{"a bracketed root", objectAB, `{{ ["inputs"].metrics.zzz }}`, []string{dynamicRef}},
	})
}

// TestRootFindings: an unknown root names the nearest known name within
// edit distance 2, and a bracketed root is a dynamic reference that uses
// every input.
func TestRootFindings(t *testing.T) {
	r := one(objectAB, `{{ inptus.metrics.a }}{% assign total = 1 %}{{ totl }}{{ xyzzy }}`).run(t)
	expect(t, r,
		`template.xref.unknown_variable templates/report.md.liquid:1:4 name="inptus" output="report" suggestion="inputs"`,
		`template.xref.unknown_variable templates/report.md.liquid:1:48 name="totl" output="report" suggestion="total"`,
		`template.xref.unknown_variable templates/report.md.liquid:1:58 name="xyzzy" output="report"`,
		`template.xref.unused_input metais.yaml#/inputs/0`,
	)
	r = one(objectAB, `{% assign k = "metrics" %}{{ [k].zzz }}`).run(t)
	expect(t, r, `template.xref.dynamic_reference templates/report.md.liquid:1:30 output="report"`)
	r = one(objectAB, `{% assign k = "metrics" %}{{ inputs[k].zzz }}`).run(t)
	expect(t, r, `template.xref.dynamic_reference templates/report.md.liquid:1:36 output="report"`)
}

// TestRootsUnderForEach: item and fan_out are bound in the template, path
// and when of a for_each output, and item is the element of its value
// (ADR-007 D3 items 3 and 9).
func TestRootsUnderForEach(t *testing.T) {
	out := tpl("page", "page.md.liquid")
	out.ForEach = "inputs.metrics.b"
	out.Path = "pages/{{ fan_out.index }}-{{ item.v }}.md"
	out.When = "item.v and fan_out.first"
	fx := fixture{
		inputs:    []fxInput{in("metrics", objectAB)},
		templates: map[string]string{"page.md.liquid": `{{ item.v }}{{ fan_out.previous.v }}{{ fan_out.next.w }}{{ fan_out.length }}{{ output.path }}`},
		outputs:   []manifest.Output{out},
	}
	expect(t, fx.run(t), `template.xref.unknown_field templates/page.md.liquid:1:52 declared=["v"] declared_count=1 input="metrics" output="page" property="w" suggestion="v" related=inputs/metrics.json#/properties/b/items`)
}
