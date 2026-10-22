// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/jsonx"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/template"
	"metais.org/metais/internal/testenv"
)

// The benchmarks below are ADR-010 D15 item 1's for Liquid: the parse and
// the render of the three fixtures in testdata/bench/, in the dialect a
// package's templates are parsed and rendered in (internal/template's,
// for markdown_table), under ADR-009's default limits:
//
//   - table: a 10,000-row table through markdown_table, whose render is
//     D15 item 2's 200 ms budget;
//   - site: a 500-page site, the page template rendered once for each
//     item of a collection and the index once, through render partials;
//   - nested: conditionals nested 24 deep, inside a loop of 2,000 items.
//
// The data is built here, the same on every run. Each render benchmark
// reports cpu-ns/op beside ns/op, since a budget is CPU time (ADR-010
// D15 item 2). testdata/bench/baseline.txt is a sandbox figure, not a
// budget; docs/dev/benchmarks.md is how the module's benchmarks are run.

const benchDir = "testdata/bench"

// benchFixture is one fixture: its templates by the name a render tag
// gives them, the one each render starts from, and the renders it is
// given, each with its globals.
type benchFixture struct {
	sources map[string][]byte
	roots   []string
	renders []benchRender
}

type benchRender struct {
	root    string
	globals value.Object
}

// benchFixtures are the three fixtures, by name.
func benchFixtures(b testing.TB) map[string]benchFixture {
	b.Helper()
	read := func(names ...string) map[string][]byte {
		out := map[string][]byte{}
		for _, n := range names {
			text, err := os.ReadFile(filepath.Join(benchDir, filepath.FromSlash(n)))
			if err != nil {
				b.Fatal(err)
			}
			out[strings.TrimPrefix(n, "site/")] = text
		}
		return out
	}

	var rows strings.Builder
	rows.WriteString(`{"rows": [`)
	for i := range 10_000 {
		if i > 0 {
			rows.WriteString(",")
		}
		fmt.Fprintf(&rows, `{"id": %d, "name": "Item %d | part %d", "status": %q, "amount": %d.%02d}`,
			i, i, i%97, []string{"open", "held", "closed"}[i%3], i*13%10_000, i%100)
	}
	rows.WriteString("]}")

	var pages strings.Builder
	pages.WriteString(`{"site": {"title": "Handbook", "summary": "Every page of the handbook."}, "pages": [`)
	for i := range 500 {
		if i > 0 {
			pages.WriteString(",")
		}
		fmt.Fprintf(&pages, `{"index": %d, "prev": %d, "slug": "page-%03d", "title": "  Page %d  ", "summary": "The %d%s page, which says what page %d says at some length.", "tags": [`,
			i, max(i-1, 0), i, i, i, "th", i)
		for j := range i % 4 {
			if j > 0 {
				pages.WriteString(",")
			}
			fmt.Fprintf(&pages, `"tag-%d"`, (i+j)%17)
		}
		pages.WriteString(`], "sections": [`)
		for j := range 4 {
			if j > 0 {
				pages.WriteString(",")
			}
			note := "null"
			if (i+j)%3 == 0 {
				note = fmt.Sprintf(`"note %d.%d"`, i, j)
			}
			fmt.Fprintf(&pages, `{"heading": "section %d of page %d", "body": "line one of %d.%d\nline two\nline three", "note": %s}`, j, i, i, j, note)
		}
		pages.WriteString("]}")
	}
	pages.WriteString("]}")

	var cases strings.Builder
	cases.WriteString(`{"cases": [`)
	for i := range 2_000 {
		if i > 0 {
			cases.WriteString(",")
		}
		fmt.Fprintf(&cases, `{"n": %d, "k": %d, "s": "s%d", "skip": [%d, %d]}`, i%30, i%7, i%7, i%11, i%13)
	}
	cases.WriteString("]}")

	site := read("site/page.md.liquid", "site/index.md.liquid", "site/partials/header.md.liquid",
		"site/partials/section.md.liquid", "site/partials/link.md.liquid", "site/partials/footer.md.liquid")
	siteInputs := inputs(b, pages.String())
	in, _ := siteInputs.Get("inputs")
	pageList, _ := in.(value.Object).Get("pages")
	pageItems := pageList.(value.Array)
	var siteRenders []benchRender
	for i := range pageItems.Len() {
		siteRenders = append(siteRenders, benchRender{"page.md.liquid", with(b, siteInputs, "item", pageItems.At(i))})
	}
	siteRenders = append(siteRenders, benchRender{"index.md.liquid", siteInputs})

	return map[string]benchFixture{
		"table": {
			sources: read("table.md.liquid"),
			roots:   []string{"table.md.liquid"},
			renders: []benchRender{{"table.md.liquid", inputs(b, rows.String())}},
		},
		"site": {
			sources: site,
			roots:   []string{"page.md.liquid", "index.md.liquid", "partials/header.md.liquid", "partials/section.md.liquid", "partials/link.md.liquid", "partials/footer.md.liquid"},
			renders: siteRenders,
		},
		"nested": {
			sources: read("nested.md.liquid"),
			roots:   []string{"nested.md.liquid"},
			renders: []benchRender{{"nested.md.liquid", inputs(b, cases.String())}},
		},
	}
}

// inputs is the globals {"inputs": the JSON text j}, as a template sees
// them.
func inputs(b testing.TB, j string) value.Object {
	b.Helper()
	doc, fs := jsonx.Parse([]byte(j), jsonx.Source{Input: finding.Subject{Kind: finding.SubjectInput, Name: "bench"}}, limits.Defaults())
	if len(fs) > 0 {
		b.Fatalf("the fixture's data does not parse: %v", fs[0])
	}
	return with(b, value.Object{}, "inputs", liquid.Data(doc.Value))
}

// with is o with the member name set to v.
func with(b testing.TB, o value.Object, name string, v value.Value) value.Object {
	b.Helper()
	var members []value.Member
	for n, v := range o.All() {
		if n != name {
			members = append(members, value.Member{Name: n, Value: v})
		}
	}
	out, err := value.NewObject(append(members, value.Member{Name: name, Value: v})...)
	if err != nil {
		b.Fatal(err)
	}
	return out
}

// parse parses every template of f, as a package's templates are parsed.
func (f benchFixture) parse(b testing.TB) map[string]*liquid.Template {
	lim := template.ParseLimits(limits.Defaults())
	out := map[string]*liquid.Template{}
	for _, name := range f.roots {
		t, errs := liquid.Parse(template.Dialect(), liquid.Source{Name: name, Text: f.sources[name]}, lim)
		if len(errs) > 0 {
			b.Fatalf("%s: %v", name, errs[0])
		}
		out[name] = t
	}
	return out
}

// benchPartials resolves a render tag to a parsed fixture template.
type benchPartials map[string]*liquid.Template

func (p benchPartials) Partial(name string) (*liquid.Template, *liquid.Error) {
	if t, ok := p[name]; ok {
		return t, nil
	}
	return nil, &liquid.Error{Code: liquid.CodeInternal, Detail: "no partial " + name}
}

// render renders every render of f from ts to w, each under a meter of
// its own with ADR-009's default limits, as a command renders an output.
func (f benchFixture) render(b testing.TB, ts map[string]*liquid.Template, w io.Writer) {
	lim := limits.Defaults()
	for _, r := range f.renders {
		env := &liquid.Env{
			Dialect:  template.Dialect(),
			Globals:  r.globals,
			Partials: benchPartials(ts),
			Meter: liquid.NewMeter(liquid.MeterLimits{
				RenderWork:     lim.Value(limits.RenderWork),
				OutputSize:     lim.Value(limits.OutputSize),
				AssignBytes:    lim.Value(limits.AssignBytes),
				LoopIterations: lim.Value(limits.LoopIterations),
			}),
			Limits: liquid.RenderLimits{
				PartialDepth: int(lim.Value(limits.PartialDepth)),
				ValueSize:    lim.Value(limits.ValueSize),
				ArraySize:    lim.Value(limits.ArraySize),
			},
		}
		if err := liquid.Render(w, ts[r.root], env); err != nil {
			b.Fatalf("%s: %v", r.root, err)
		}
	}
}

// BenchmarkParse is the parse of each fixture's every template.
func BenchmarkParse(b *testing.B) {
	testenv.Home(b)
	fixtures := benchFixtures(b)
	for _, name := range []string{"table", "site", "nested"} {
		f := fixtures[name]
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				f.parse(b)
			}
		})
	}
}

// BenchmarkRender is the render of each fixture's every render, parsed
// once beforehand. Render/table is D15 item 2's 200 ms budget.
func BenchmarkRender(b *testing.B) {
	testenv.Home(b)
	fixtures := benchFixtures(b)
	for _, name := range []string{"table", "site", "nested"} {
		f := fixtures[name]
		b.Run(name, func(b *testing.B) {
			ts := f.parse(b)
			var out bytes.Buffer
			f.render(b, ts, &out)
			b.SetBytes(int64(out.Len()))
			b.ReportAllocs()
			cpu := startCPU()
			for b.Loop() {
				out.Reset()
				f.render(b, ts, &out)
			}
			cpu.report(b)
		})
	}
}

// TestBenchFixtures holds the benchmarks' fixtures to what they are
// there to measure, in every go test, so that a benchmark never times a
// fixture that stopped doing its work: each parses and renders, the
// table has its 10,000 rows, the site its 500 pages and its index, and
// the nested conditionals reach their deepest level and their else.
func TestBenchFixtures(t *testing.T) {
	testenv.Home(t)
	fixtures := benchFixtures(t)
	got := map[string]string{}
	for name, f := range fixtures {
		var out bytes.Buffer
		f.render(t, f.parse(t), &out)
		got[name] = out.String()
	}
	if n := strings.Count(got["table"], "\n| "); n != 10_002 {
		t.Errorf("the table has %d lines of cells, want a header, its delimiter line and 10,000 rows", n)
	}
	if !strings.Contains(got["table"], "Amount (EUR)") || !strings.Contains(got["table"], "Item 9999 \\| part 8") {
		t.Errorf("the table lacks its renamed header or its last row, escaped")
	}
	if n := len(fixtures["site"].renders); n != 501 {
		t.Errorf("the site renders %d times, want 500 pages and the index", n)
	}
	for _, want := range []string{"# Page 499\n", "## 4. Section 3 of page 499", "> NOTE 498.0", "Page 500 of 500 · [previous](page-498.md)", "- [  Page 499  ](page-499.md) (tag-6)"} {
		if !strings.Contains(got["site"], want) {
			t.Errorf("the site lacks %q", want)
		}
	}
	for _, want := range []string{"a24", "e5", "z1", "k", "h"} {
		if !strings.Contains(got["nested"], want) {
			t.Errorf("the nested conditionals never render %q", want)
		}
	}
}
