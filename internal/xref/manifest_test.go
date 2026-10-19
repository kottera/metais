// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package xref

import (
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/manifest"
)

// TestManifestExpressions holds path, when and for_each to ADR-007 D3 item
// 9: each analysed in its output's context, located at the declaration's
// manifest pointer, for_each without item and permitting arrays.
func TestManifestExpressions(t *testing.T) {
	out := tpl("page", "page.md.liquid")
	out.ForEach = "inputs.metrics.r | where: \"a\""
	out.When = "item.c and inputs.metrics.nope"
	out.Path = "{% assign slug = item.c %}{{ slug.zz }}/{{ output.path }}.md"
	fx := fixture{
		inputs:    []fxInput{in("metrics", tuple)},
		templates: map[string]string{"page.md.liquid": `{{ item.c }}`},
		outputs:   []manifest.Output{out},
	}
	expect(t, fx.run(t),
		`template.xref.unknown_field metais.yaml#/outputs/0/when declared=["bare","m","map","o","r","s","t","u"] declared_count=8 input="metrics" output="page" property="nope" related=inputs/metrics.json#`,
		`template.xref.unknown_variable metais.yaml#/outputs/0/path name="output" output="page"`,
	)

	// for_each is analysed without item, and must permit an array.
	out = tpl("page", "page.md.liquid")
	out.ForEach = "item"
	fx.outputs = []manifest.Output{out}
	expect(t, fx.run(t),
		`template.xref.unknown_variable metais.yaml#/outputs/0/for_each name="item" output="page"`,
		`template.xref.unused_input metais.yaml#/inputs/0`,
	)
	out.ForEach = "inputs.metrics.o"
	fx.outputs = []manifest.Output{out}
	expect(t, fx.run(t), `template.xref.for_each_not_array metais.yaml#/outputs/0/for_each types=["object"]`)
	out.ForEach = "inputs.metrics.s | upcase"
	fx.outputs = []manifest.Output{out}
	expect(t, fx.run(t), `template.xref.for_each_not_array metais.yaml#/outputs/0/for_each types=["string"]`)
	out.ForEach = "inputs.metrics.bare.z"
	fx.outputs = []manifest.Output{out}
	expect(t, fx.run(t))
}

// TestManifestContainer: a container with for_each gives item to every
// entry, and the entries' expressions and templates are analysed in it.
func TestManifestContainer(t *testing.T) {
	entry := tpl("", "doc.xml.liquid")
	entry.Path = "word/{{ item.c }}.xml"
	entry.When = "fan_out.first"
	c := container("docx", "{{ fan_out.index }}.docx", entry, mirror("", "parts"))
	c.ForEach = "inputs.metrics.m"
	fx := fixture{
		inputs: []fxInput{in("metrics", tuple)},
		templates: map[string]string{
			"doc.xml.liquid":      `{{ item.c }}{{ item.zz }}`,
			"parts/a.xml.liquid":  `{{ item.c }}`,
			"parts/_p.xml.liquid": `{{ item.c }}`,
		},
		outputs: []manifest.Output{c},
	}
	expect(t, fx.run(t),
		`template.xref.unknown_field templates/doc.xml.liquid:1:20 declared=["c"] declared_count=1 input="metrics" output="docx" property="zz" suggestion="c" related=inputs/metrics.json#/properties/m/items`,
		`template.xref.unused_template templates/parts/_p.xml.liquid`,
	)
}

// TestManifestContexts: a template used by several outputs is analysed
// once per distinct context, and each finding names the output whose
// context produced it.
func TestManifestContexts(t *testing.T) {
	each := tpl("pages", "card.md.liquid")
	each.ForEach = "inputs.metrics.m"
	each.Path = "{{ fan_out.index }}.md"
	fx := fixture{
		inputs:    []fxInput{in("metrics", tuple)},
		templates: map[string]string{"card.md.liquid": `{{ item.c }}{{ inputs.metrics.zz }}`},
		outputs:   []manifest.Output{tpl("one", "card.md.liquid"), tpl("two", "card.md.liquid"), each},
	}
	r := fx.run(t)
	expect(t, r,
		`template.xref.unknown_field templates/card.md.liquid:1:30 declared=["bare","m","map","o","r","s","t","u"] declared_count=8 input="metrics" output="one" property="zz" suggestion="m" related=inputs/metrics.json#`,
		`template.xref.unknown_variable templates/card.md.liquid:1:4 name="item" output="one"`,
	)
	// card in the context of one and two, card in that of pages, and
	// pages' for_each and path.
	if r.Contexts != 4 {
		t.Errorf("contexts = %d, want 4", r.Contexts)
	}
}

// TestUse holds use to ADR-007 D3 item 11: an input is used when a
// reference in an effective template or manifest expression passes
// through inputs.<name>, uses inputs whole, or an embed names it.
func TestUse(t *testing.T) {
	ins := []fxInput{in("a", `{"type": "object"}`), in("b", `{"type": "array"}`), in("c", `{"type": "string"}`)}
	for _, c := range []struct {
		name   string
		text   string
		when   string
		unused []string
	}{
		{"none", `text`, "", []string{"a", "b", "c"}},
		{"through inputs.<name>, whatever follows", `{{ inputs.a.nope }}{% for x in inputs.b %}{% endfor %}`, "", []string{"c"}},
		{"a manifest expression", ``, "inputs.c", []string{"a", "b"}},
		{"inputs whole", `{{ inputs | json }}`, "", nil},
		{"a computed name", `{% assign k = "a" %}{{ inputs[k] }}`, "", nil},
		{"embed", `{% embed "c" %}`, "", []string{"a", "b"}},
		{"sources is no use", `{{ sources.a.origin }}`, "", []string{"a", "b", "c"}},
		{"a name only in a for variable that shadows inputs", `{% for inputs in (1..2) %}{{ inputs }}{% endfor %}`, "", []string{"a", "b", "c"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			out := tpl("main", "main.liquid")
			out.When = c.when
			fx := fixture{inputs: ins, templates: map[string]string{"main.liquid": c.text}, outputs: []manifest.Output{out}}
			var want []string
			for _, u := range c.unused {
				want = append(want, `template.xref.unused_input metais.yaml#/inputs/`+map[string]string{"a": "0", "b": "1", "c": "2"}[u])
			}
			got := describeAll(t, fx.run(t))
			var unused []string
			for _, g := range got {
				if contains(g, "unused_input") {
					unused = append(unused, g)
				}
			}
			if !equalCodes(unused, want) {
				t.Errorf("unused %q, want %q", unused, want)
			}
		})
	}
}

// TestUnusedTemplate holds an unused template to ADR-007 D3 item 11: a
// warning; still analysed, with every partial-parameter name unknown; its
// references count as no use.
func TestUnusedTemplate(t *testing.T) {
	fx := fixture{
		inputs: []fxInput{in("metrics", tuple), in("other", `{"type": "object"}`)},
		templates: map[string]string{
			"main.liquid":     `{{ inputs.metrics }}{% render "used.liquid" %}`,
			"used.liquid":     ``,
			"orphan.liquid":   `{{ inputs.other }}{{ inputs.metrics.zz }}{{ row.x }}{{ item.y }}`,
			"_partial.liquid": ``,
		},
		outputs: []manifest.Output{tpl("main", "main.liquid")},
	}
	expect(t, fx.run(t),
		`template.xref.unknown_field templates/orphan.liquid:1:36 declared=["bare","m","map","o","r","s","t","u"] declared_count=8 input="metrics" property="zz" suggestion="m" related=inputs/metrics.json#`,
		`template.xref.unused_input metais.yaml#/inputs/1`,
		`template.xref.unused_template templates/_partial.liquid`,
		`template.xref.unused_template templates/orphan.liquid`,
	)
}

// TestMirror: a mirror renders every template beneath its directory but
// partials, whose name begins with _ at any depth, and templates an
// explicit output names, which are rendered once (ADR-005 D6).
func TestMirror(t *testing.T) {
	fx := fixture{
		inputs: []fxInput{in("metrics", tuple)},
		templates: map[string]string{
			"site/index.html.liquid":      `{{ output.path }}{{ inputs.metrics.a1 }}`,
			"site/_layout.html.liquid":    `{{ inputs.metrics.a2 }}`,
			"site/_inc/x.html.liquid":     `{{ inputs.metrics.a3 }}`,
			"site/deep/page.html.liquid":  `{{ inputs.metrics.a4 }}`,
			"site/article.html.liquid":    `{{ item.c }}`,
			"other/elsewhere.html.liquid": `{{ inputs.metrics.a5 }}`,
		},
		outputs: []manifest.Output{mirror("site", "site")},
	}
	art := tpl("articles", "site/article.html.liquid")
	art.ForEach = "inputs.metrics.m"
	art.Path = "{{ fan_out.index }}.html"
	fx.outputs = append(fx.outputs, art)
	// The mirrored templates are analysed for site, article for articles,
	// and the partials and the template outside the mirror are unused.
	expect(t, fx.run(t),
		`template.xref.unknown_field templates/other/elsewhere.html.liquid:1:18 declared=["bare","m","map","o","r","s","t","u"] declared_count=8 input="metrics" property="a5" suggestion="m" related=inputs/metrics.json#`,
		`template.xref.unknown_field templates/site/_inc/x.html.liquid:1:18 declared=["bare","m","map","o","r","s","t","u"] declared_count=8 input="metrics" property="a3" suggestion="m" related=inputs/metrics.json#`,
		`template.xref.unknown_field templates/site/_layout.html.liquid:1:18 declared=["bare","m","map","o","r","s","t","u"] declared_count=8 input="metrics" property="a2" suggestion="m" related=inputs/metrics.json#`,
		`template.xref.unknown_field templates/site/deep/page.html.liquid:1:18 declared=["bare","m","map","o","r","s","t","u"] declared_count=8 input="metrics" output="site" property="a4" suggestion="m" related=inputs/metrics.json#`,
		`template.xref.unknown_field templates/site/index.html.liquid:1:35 declared=["bare","m","map","o","r","s","t","u"] declared_count=8 input="metrics" output="site" property="a1" suggestion="m" related=inputs/metrics.json#`,
		`template.xref.unused_template templates/other/elsewhere.html.liquid`,
		`template.xref.unused_template templates/site/_inc/x.html.liquid`,
		`template.xref.unused_template templates/site/_layout.html.liquid`,
	)
}

// TestUnreadableLeavesUseUnknown: when a manifest expression or a
// template an output renders cannot be read, what it reads is unknown, so
// no input is called unused.
func TestUnreadableLeavesUseUnknown(t *testing.T) {
	out := tpl("main", "main.liquid")
	out.When = "inputs.metrics and"
	fx := fixture{inputs: []fxInput{in("metrics", tuple)}, templates: map[string]string{"main.liquid": ``}, outputs: []manifest.Output{out}}
	expect(t, fx.run(t))
	fx.outputs = []manifest.Output{tpl("main", "main.liquid")}
	fx.broken = []string{"main.liquid"}
	expect(t, fx.run(t))
	fx.broken = nil
	expect(t, fx.run(t), `template.xref.unused_input metais.yaml#/inputs/0`)
}

// TestManifestPositions: a finding about a manifest expression has the
// line of the expression's value when the manifest's reader kept
// positions.
func TestManifestPositions(t *testing.T) {
	out := tpl("main", "main.liquid")
	out.When = "inputs.metrics.nope_nope"
	fx := fixture{inputs: []fxInput{in("metrics", tuple)}, templates: map[string]string{"main.liquid": ``}, outputs: []manifest.Output{out}}
	p := fx.pkg(t)
	p.Positions = positions{"/outputs/0/when": {Line: 7, Column: 11}}
	r, err := Analyze(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Findings) != 1 {
		t.Fatalf("findings %q", describeAll(t, r))
	}
	if d := describe(t, r.Findings[0]); !contains(d, "metais.yaml:7:11#/outputs/0/when") {
		t.Errorf("%s: want line 7, column 11", d)
	}
}

// positions is a manifest reader's position table.
type positions map[string]finding.Pos

func (p positions) Value(ptr string) (finding.Pos, bool) { v, ok := p[ptr]; return v, ok }
func (p positions) Key(ptr string) (finding.Pos, bool)   { return finding.Pos{}, false }
