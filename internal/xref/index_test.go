// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package xref

import "testing"

const tuple = `{"type": "object", "properties": {
	"t": {"type": "array", "prefixItems": [{"type": "object", "properties": {"a": {}}}, {"type": "object", "properties": {"b": {}}}], "items": false},
	"r": {"type": "array", "prefixItems": [{"type": "object", "properties": {"a": {}}}], "items": {"type": "object", "properties": {"c": {}}}},
	"m": {"type": "array", "items": {"type": "object", "properties": {"c": {}}}, "maxItems": 2},
	"u": {"type": "array"},
	"s": {"type": "string"},
	"o": {"type": "object", "properties": {"k": {"type": "object", "properties": {"x": {}}}}},
	"map": {"type": "object", "additionalProperties": {"type": "object", "properties": {"x": {}}}},
	"bare": {"type": "object", "properties": {"z": {}}, "required": ["z"]}
}}`

// TestIndex holds index and computed segments to ADR-007 D3 item 6.
func TestIndex(t *testing.T) {
	runCases(t, []xcase{
		{"[i] takes prefixItems[i]", tuple, `{{ inputs.metrics.t[0].a }}{{ inputs.metrics.t[1].b }}`, nil},
		{"[i] checks prefixItems[i]", tuple, `{{ inputs.metrics.t[0].b }}`, []string{unknownField}},
		{"[i] beyond prefixItems takes items", tuple, `{{ inputs.metrics.r[0].a }}{{ inputs.metrics.r[5].c }}`, nil},
		{"items: false beyond prefixItems", tuple, `{{ inputs.metrics.t[2].a }}`, []string{outOfRange}},
		{"maxItems", tuple, `{{ inputs.metrics.m[1].c }}{{ inputs.metrics.m[2].c }}`, []string{outOfRange}},
		{"a negative index takes every prefixItems schema and items", tuple, `{{ inputs.metrics.r[-1].a }}{{ inputs.metrics.r[-1].c }}{{ inputs.metrics.t[-1].b }}{{ inputs.metrics.m[-1].c }}`, nil},
		{"a negative index is checked", tuple, `{{ inputs.metrics.r[-1].q }}`, []string{unknownField}},
		{"an array with neither keyword is open", tuple, `{{ inputs.metrics.u[0].anything.at.all }}`, nil},
		{"an index of a string", tuple, `{{ inputs.metrics.s[0] }}`, []string{notAnArray}},
		{"an index of an object", tuple, `{{ inputs.metrics.o[0] }}`, []string{notAnArray}},
		{"a computed key on a map", tuple, `{% assign k = "q" %}{{ inputs.metrics.map[k].x }}`, nil},
		{"a computed key on a map is checked below", tuple, `{% assign k = "q" %}{{ inputs.metrics.map[k].y }}`, []string{unknownField}},
		{"a computed key on an object takes every property", tuple, `{% assign k = "q" %}{{ inputs.metrics.o[k].x }}`, nil},
		{"a computed index on an array", tuple, `{% assign k = 1 %}{{ inputs.metrics.r[k].c }}{{ inputs.metrics.r[k].a }}`, nil},
		{"a computed key leaving nothing", tuple, `{% assign k = "q" %}{{ inputs.metrics.s[k].x }}`, []string{dynamicRef}},
		{"a computed key on a record", tuple, `{% assign k = "index" %}{% for x in inputs.metrics.r %}{{ forloop[k] }}{% endfor %}`, nil},
		{"a computed key on inputs", tuple, `{% assign k = "metrics" %}{{ inputs[k].anything }}`, []string{dynamicRef}},
		{"a computed key on an open node", `{"type": "object", "properties": {"o": {}}}`, `{% assign k = "q" %}{{ inputs.metrics.o[k].x.y }}`, nil},
		{"the index expression is analysed", tuple, `{{ inputs.metrics.r[inputs.metrics.nope] }}`, []string{unknownField}},
	})
}

// TestLocals holds template-scoped bindings to ADR-007 D3 item 7: assign
// binds its value, capture a string, increment and decrement a number;
// a local is template-wide and flow-insensitive, its value the union of
// every binding, to a fixed point.
func TestLocals(t *testing.T) {
	runCases(t, []xcase{
		{"assign", tuple, `{% assign o = inputs.metrics.o %}{{ o.k.x }}`, nil},
		{"assign is checked", tuple, `{% assign o = inputs.metrics.o %}{{ o.k.y }}`, []string{unknownField}},
		{"capture binds a string", tuple, `{% capture c %}{{ inputs.metrics.s }}{% endcapture %}{{ c.size }}{{ c.x }}`, []string{notAnObject}},
		{"increment binds a number", tuple, `{% increment n %}{{ n.x }}{{ inputs.metrics }}`, []string{notAnObject}},
		{"decrement binds a number", tuple, `{% decrement n %}{{ n.size }}{{ inputs.metrics }}`, nil},
		{"a local read before its assign", tuple, `{{ o.k.y }}{% assign o = inputs.metrics.o %}`, []string{unknownField}},
		{"an assign outlives its block", tuple, `{% if true %}{% assign o = inputs.metrics.o %}{% endif %}{{ o.k.x }}{{ o.k.y }}`, []string{unknownField}},
		{"a local is the union of its bindings", tuple, `{% assign v = inputs.metrics.o.k %}{% assign v = inputs.metrics.r[0] %}{{ v.x }}{{ v.a }}{{ v.zz }}`, []string{unknownField}},
		{"a chain of locals reaches its fixed point", tuple, `{% assign c = b.k %}{% assign b = a %}{% assign a = inputs.metrics.o %}{{ c.x }}{{ c.y }}`, []string{unknownField}},
		{"an assign inside a loop sees the loop variable", tuple, `{% for row in inputs.metrics.r %}{% assign last = row %}{% endfor %}{{ last.c }}{{ last.a }}{{ last.zz }}`, []string{unknownField}},
		{"a self-referring local terminates", tuple, `{% assign x = inputs.metrics.o %}{% assign x = x.k %}{% assign x = x.x %}{{ x.anything }}`, nil},
		{"filters on assign", tuple, `{% assign t = inputs.metrics.s | upcase %}{{ t.size }}{{ t.x }}`, []string{notAnObject}},
	})
}

// TestSetBound holds an abstract value to at most 64 alternatives (ADR-007
// D3 item 2): a larger one becomes unknown, below which nothing is
// checked.
func TestSetBound(t *testing.T) {
	build := func(n int) (string, string) {
		props, text := "", "{% assign v = nil %}"
		for i := range n {
			if i > 0 {
				props += ", "
			}
			name := "p" + itoa(i)
			props += `"` + name + `": {"type": "object", "properties": {"q": {}}}`
			text += "{% assign v = inputs.metrics." + name + " %}"
		}
		return `{"type": "object", "properties": {` + props + `}}`, text + "{{ v.zz }}"
	}
	// 63 schema alternatives and nil: 64, each rejecting .zz.
	s, text := build(63)
	if got := codes(one(s, text).run(t)); !equalCodes(got, []string{unknownField}) {
		t.Errorf("64 alternatives: codes %q, want unknown_field", got)
	}
	// 64 and nil: 65, which is unknown.
	s, text = build(64)
	if got := codes(one(s, text).run(t)); len(got) != 0 {
		t.Errorf("65 alternatives: codes %q, want none", got)
	}
	if got := union(make([]set, 0)...); len(got) != 0 {
		t.Errorf("the empty union is %v", got)
	}
}
