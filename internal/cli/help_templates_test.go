// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/jsonx"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/template"
)

// filterDrift is what keeps docs from documenting exactly catalog: a
// filter of the catalog no row documents, a row naming no filter of the
// catalog, and a filter documented twice.
func filterDrift(catalog []string, docs []filterDoc) []string {
	var out []string
	seen := map[string]bool{}
	for _, d := range docs {
		if seen[d.Name] {
			out = append(out, "documented twice: "+d.Name)
		}
		seen[d.Name] = true
		if !slices.Contains(catalog, d.Name) {
			out = append(out, "documents no filter of the dialect: "+d.Name)
		}
	}
	for _, name := range catalog {
		if !seen[name] {
			out = append(out, "undocumented: "+name)
		}
	}
	return out
}

// TestTemplatesTopicFilters: metais help templates documents every filter
// of the dialect templates are rendered in, and nothing else, so a filter
// added to the engine without its line in the help fails here.
func TestTemplatesTopicFilters(t *testing.T) {
	catalog := template.Dialect().FilterNames()
	for _, p := range filterDrift(catalog, filterDocs()) {
		t.Error(p)
	}

	// Pinned false: a filter the dialect gains and the help lacks, a row
	// for a filter it lacks, and a row written twice are each reported.
	docs := filterDocs()
	cases := []struct {
		catalog []string
		docs    []filterDoc
		want    string
	}{
		{append(slices.Clone(catalog), "zz_new_filter"), docs, "undocumented: zz_new_filter"},
		{catalog, append(slices.Clone(docs), filterDoc{Name: "zz_gone"}), "documents no filter of the dialect: zz_gone"},
		{catalog, append(slices.Clone(docs), docs[0]), "documented twice: " + docs[0].Name},
	}
	for _, c := range cases {
		if got := filterDrift(c.catalog, c.docs); !slices.Contains(got, c.want) {
			t.Errorf("filterDrift = %q, want it to report %q", got, c.want)
		}
	}

	out := runTool(t, "help", "templates").stdout
	for _, name := range catalog {
		if !strings.Contains(out, "\n  "+name+" ") && !strings.Contains(out, "\n  "+name+"\n") {
			t.Errorf("metais help templates lists no filter %s", name)
		}
	}
	for i, l := range strings.Split(out, "\n") {
		if n := utf8.RuneCountInString(l); n > helpWidth {
			t.Errorf("line %d is %d columns, beyond %d: %q", i+1, n, helpWidth, l)
		}
	}
	// The examples lead the page (ADR-008 A3): the first worked example
	// comes before the environment's heading.
	first := strings.Index(out, strings.Split(leadExamples[0].src, "\n")[0])
	if first < 0 || first > strings.Index(out, "The environment:") {
		t.Errorf("the worked examples do not lead the page")
	}
}

// examplePartials is the one partial the worked examples render.
type examplePartials map[string]*liquid.Template

func (p examplePartials) Partial(name string) (*liquid.Template, *liquid.Error) {
	return p[name], nil
}

// TestTemplatesTopicExamples: every example the help gives renders, in the
// dialect a package's templates are rendered in and over the inputs the
// help states, exactly what the help says it gives; the escaping examples
// in an output whose escape is html.
func TestTemplatesTopicExamples(t *testing.T) {
	doc, fs := jsonx.Parse([]byte(templateInputs), jsonx.Source{Input: finding.Subject{Kind: finding.SubjectInput, Name: "d"}}, limits.Defaults())
	if len(fs) > 0 {
		t.Fatalf("templateInputs: %v", fs)
	}
	globals := liquid.Data(doc.Value).(value.Object)
	d := template.Dialect()
	parse := func(name, src string) *liquid.Template {
		t.Helper()
		tpl, errs := liquid.Parse(d, liquid.Source{Name: name, Text: []byte(src)}, liquid.ParseLimits{})
		if len(errs) > 0 {
			t.Errorf("%s: %v", src, errs[0])
			return nil
		}
		return tpl
	}
	check := func(e templateExample) {
		t.Helper()
		tpl := parse("t", e.src)
		if tpl == nil {
			return
		}
		env := &liquid.Env{Dialect: d, Globals: globals, Meter: liquid.NewMeter(liquid.MeterLimits{})}
		if e.partial != "" {
			env.Partials = examplePartials{"row.md": parse("row.md", e.partial)}
		}
		if e.html {
			env.Escape = liquid.EscapeHTML
		}
		var b strings.Builder
		if err := liquid.Render(&b, tpl, env); err != nil {
			t.Errorf("%s: %v", e.src, err)
			return
		}
		if b.String() != e.gives {
			t.Errorf("%s\ngives %q\nhelp  %q", e.src, b.String(), e.gives)
		}
	}
	for _, f := range filterDocs() {
		if !strings.Contains(f.Example, "| "+f.Name) {
			t.Errorf("%s's example %s does not apply it", f.Name, f.Example)
		}
		check(templateExample{src: f.Example, gives: f.Gives})
	}
	for _, e := range leadExamples {
		check(e)
	}
	for _, e := range escapeExamples {
		if !e.html {
			t.Errorf("escaping example %s is not rendered in an html output", e.src)
		}
		check(e)
	}
}
