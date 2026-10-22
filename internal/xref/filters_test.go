// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package xref

import (
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/liquid"
	"metais.org/metais/internal/manifest"
)

// catalog is ADR-010 D4 item 2's list of the standard filters, Shopify's
// 5.14.0 standardfilters.rb, as that ADR writes it. The standard catalog
// is this list without date, which internal/template registers in Wave 5,
// and with escape's alias h, which ADR-004's Context lists.
const catalog = `abs append at_least at_most
     base64_decode base64_encode base64_url_safe_decode
     base64_url_safe_encode capitalize ceil compact concat date default
     divided_by downcase escape escape_once find find_index first floor has
     join last lstrip map minus modulo newline_to_br plus prepend reject
     remove remove_first remove_last replace replace_first replace_last
     reverse round rstrip size slice sort sort_natural split squish strip
     strip_html strip_newlines sum times truncate truncatewords uniq upcase
     url_decode url_encode where`

// TestFilterTableCoversTheCatalog holds the signature table to ADR-007 D3
// item 10: a signature for every filter of the standard catalog, and the
// property-name arguments of map, where, sort, sort_natural and sum.
func TestFilterTableCoversTheCatalog(t *testing.T) {
	want := []string{"h"}
	for _, f := range strings.Fields(catalog) {
		if f != "date" {
			want = append(want, f)
		}
	}
	slices.Sort(want)
	if got := Filters(); !slices.Equal(got, want) {
		t.Errorf("the table names\n\t%q\nthe catalog\n\t%q", got, want)
	}
	var props []string
	for name, sig := range signatures {
		if sig.property {
			props = append(props, name)
		}
	}
	slices.Sort(props)
	if want := []string{"map", "sort", "sort_natural", "sum", "where"}; !slices.Equal(props, want) {
		t.Errorf("property-name arguments: %q, want %q", props, want)
	}
	for name, sig := range signatures {
		if sig.kind == 0 || (sig.kind == sigScalar) != (sig.scalar != 0) {
			t.Errorf("%s: malformed signature %+v", name, sig)
		}
	}
}

// TestFilterTableMeetsCore is the meeting with W3-4: the signature table
// names exactly the filters of liquid.Core() (ADR-007 D3 item 10). The
// filters Wave 5 adds through Extend — date and METAIS's own — are not in
// Core, and the table gives each of them unknown until it names them.
func TestFilterTableMeetsCore(t *testing.T) {
	if got, want := Filters(), liquid.Core().FilterNames(); !slices.Equal(got, want) {
		t.Errorf("the table names\n\t%q\nCore holds\n\t%q", got, want)
	}
}

const rows = `{"type": "object", "properties": {
	"rows": {"type": "array", "items": {"type": "object", "properties": {"value": {"type": "number"}, "name": {"type": "string"}, "tags": {"type": "array", "items": {"type": "object", "properties": {"t": {}}}}}}},
	"more": {"type": "array", "items": {"type": "object", "properties": {"extra": {}}}},
	"text": {"type": "string"},
	"obj": {"type": "object", "properties": {"k": {}}}
}}`

// pair is a tuple whose elements declare different properties, so the
// element first gives and the ones last gives can be told apart.
const pair = `{"type": "object", "properties": {
	"pair": {"type": "array", "prefixItems": [{"type": "object", "properties": {"a": {}}}, {"type": "object", "properties": {"b": {}}}], "items": false}
}}`

// TestFilters holds each kind of analysis signature to ADR-007 D3 item 10.
func TestFilters(t *testing.T) {
	runCases(t, []xcase{
		// A property-name argument is resolved on the element and checked.
		{"map: a declared name", rows, `{{ inputs.metrics.rows | map: "value" }}`, nil},
		{"map: a wrong name", rows, `{{ inputs.metrics.rows | map: "vaule" }}`, []string{unknownField}},
		{"where", rows, `{{ inputs.metrics.rows | where: "name", "x" }}{{ inputs.metrics.rows | where: "nmae", "x" }}`, []string{unknownField}},
		{"sort", rows, `{{ inputs.metrics.rows | sort: "value" }}{{ inputs.metrics.rows | sort: "valeu" }}`, []string{unknownField}},
		{"sort_natural", rows, `{{ inputs.metrics.rows | sort_natural: "name" }}{{ inputs.metrics.rows | sort_natural: "naem" }}`, []string{unknownField}},
		{"sum", rows, `{{ inputs.metrics.rows | sum: "value" }}{{ inputs.metrics.rows | sum: "vlue" }}`, []string{unknownField}},
		{"sort without a property", rows, `{{ inputs.metrics.rows | sort }}`, nil},
		{"a property name that is no literal", rows, `{% assign p = "x" %}{{ inputs.metrics.rows | map: p }}`, nil},
		// Preserves its input.
		{"sort preserves", rows, `{% assign s = inputs.metrics.rows | sort: "value" | reverse | uniq | compact | slice: 0, 2 %}{{ s[0].name }}{{ s[0].nope }}`, []string{unknownField}},
		{"where preserves", rows, `{% assign w = inputs.metrics.rows | where: "name", "x" %}{{ w.first.value }}{{ w.first.nope }}`, []string{unknownField}},
		// default gives the union with its argument.
		{"default", rows, `{% assign d = inputs.metrics.obj | default: inputs.metrics.rows[0] %}{{ d.k }}{{ d.value }}{{ d.nope }}`, []string{unknownField}},
		// concat gives the union of element values.
		{"concat", rows, `{% assign c = inputs.metrics.rows | concat: inputs.metrics.more %}{{ c[0].value }}{{ c[1].extra }}{{ c.first.nope }}`, []string{unknownField}},
		// first and last give the element.
		{"first and last", rows, `{{ inputs.metrics.rows | first | map: "value" }}{% assign l = inputs.metrics.rows | last %}{{ l.name }}{{ l.nope }}`, []string{unknownField}},
		// Of a tuple, first is [0], its first element alone, and last is
		// [-1], which may be any of them (ADR-007 D3 item 6).
		{"first of a tuple", pair, `{% assign f = inputs.metrics.pair | first %}{{ f.a }}{{ f.b }}`, []string{unknownField}},
		{"last of a tuple", pair, `{% assign l = inputs.metrics.pair | last %}{{ l.a }}{{ l.b }}`, nil},
		{"first of a string", rows, `{% assign f = inputs.metrics.text | first %}{{ f.size }}{{ f.x }}`, []string{notAnObject}},
		// map projects a property.
		{"map projects", rows, `{% assign tags = inputs.metrics.rows | map: "tags" %}{{ tags[0][0].t }}{{ tags[0][0].nope }}`, []string{unknownField}},
		// A scalar of a stated kind.
		{"a number", rows, `{% assign n = inputs.metrics.rows | size %}{{ n.x }}`, []string{notAnObject}},
		{"a string", rows, `{% assign s = inputs.metrics.text | upcase %}{{ s.size }}{{ s.x }}`, []string{notAnObject}},
		{"a boolean", rows, `{% assign b = inputs.metrics.rows | has: "name", "x" %}{{ b.x }}`, []string{notAnObject}},
		// unknown, for a table entry so marked and for a filter the table
		// does not name.
		{"split", rows, `{% assign s = inputs.metrics.text | split: "," %}{{ s.anything }}`, nil},
		{"a filter the table does not name", rows, `{% assign j = inputs.metrics.obj | json %}{{ j.anything }}`, nil},
		// Every argument is analysed as an expression.
		{"arguments", rows, `{{ inputs.metrics.text | append: inputs.metrics.nope }}{{ inputs.metrics.text | json: indent: inputs.metrics.nope2 }}`, []string{unknownField, unknownField}},
	})
}

// TestFilterPropertyLocated: a wrong property-name argument is located at
// the literal, and resolved on the element of every alternative.
func TestFilterPropertyLocated(t *testing.T) {
	fx := one(rows, `{{ inputs.metrics.rows | map: "vaule" }}`)
	expect(t, fx.run(t),
		`template.xref.unknown_field templates/report.md.liquid:1:31 declared=["name","tags","value"] declared_count=3 input="metrics" output="report" property="vaule" suggestion="value" related=inputs/metrics.json#/properties/rows/items`)
	// In a manifest expression too.
	out := tpl("page", "report.md.liquid")
	out.ForEach = `inputs.metrics.rows | sort: "nmae"`
	fx.outputs = []manifest.Output{out}
	fx.templates["report.md.liquid"] = `{{ item.name }}`
	expect(t, fx.run(t),
		`template.xref.unknown_field metais.yaml#/outputs/0/for_each declared=["name","tags","value"] declared_count=3 input="metrics" output="page" property="nmae" suggestion="name" related=inputs/metrics.json#/properties/rows/items`)
}
