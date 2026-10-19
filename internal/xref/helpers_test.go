// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package xref

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/jsonx"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid"
	"github.com/kottera/metais/internal/liquid/ast"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/schema"
	"github.com/kottera/metais/internal/testenv"
)

// stubFilter and stubTag stand for the definitions internal/template
// adds in Wave 5: the analysis reads only their names, through the tree
// the parser builds.
type stubFilter string

func (f stubFilter) Name() string                  { return string(f) }
func (stubFilter) Check(*ast.Filter) *liquid.Error { return nil }
func (stubFilter) Apply(liquid.FilterContext, value.Value, []liquid.Arg) (value.Value, *liquid.Error) {
	panic("the analysis applies no filter")
}

type stubTag string

func (t stubTag) Name() string               { return string(t) }
func (stubTag) Check(*ast.Tag) *liquid.Error { return nil }
func (stubTag) Render(liquid.TagContext, *ast.Tag) *liquid.Error {
	panic("the analysis renders no tag")
}

// dialect is the standard catalog, liquid.Core(), extended as Wave 5's
// internal/template extends it: METAIS's json filter, which the signature
// table does not name, METAIS's tags, and the reserved names.
func dialect(t testing.TB) *liquid.Dialect {
	t.Helper()
	spec := liquid.Spec{Filters: []liquid.FilterDef{stubFilter("json")}}
	for _, name := range []string{"embed", "asset", "provenance"} {
		spec.Tags = append(spec.Tags, stubTag(name))
	}
	spec.Reserved = []string{"inputs", "package", "item", "fan_out", "output", "now", "sources", "forloop", "tablerowloop"}
	d, err := liquid.Core().Extend(spec)
	if err != nil {
		t.Fatalf("Extend: %v", err)
	}
	return d
}

// fixture is a package, written as text.
type fixture struct {
	// inputs are declared in order, each with its schema file
	// inputs/<name>.json; files are further schema files by path.
	inputs []fxInput
	files  map[string]string
	// templates are by path relative to templates/; broken names those
	// that are listed as not parsing.
	templates map[string]string
	broken    []string
	assets    []string
	outputs   []manifest.Output
}

type fxInput struct {
	name, schema string
	collection   bool
}

// in declares an input with schema text s.
func in(name, s string) fxInput { return fxInput{name: name, schema: s} }

// tpl, mirror and container are output declarations.
func tpl(id, source string) manifest.Output {
	return manifest.Output{Form: manifest.OutputTemplate, ID: id, Source: source}
}

func mirror(id, dir string) manifest.Output {
	return manifest.Output{Form: manifest.OutputMirror, ID: id, Source: dir}
}

func container(id, path string, entries ...manifest.Output) manifest.Output {
	return manifest.Output{Form: manifest.OutputContainer, ID: id, Path: path, Container: &manifest.Container{Format: "zip", Entries: entries}}
}

// pointers gives each declaration its manifest pointer.
func pointers(os []manifest.Output, prefix string) []manifest.Output {
	out := slices.Clone(os)
	for i := range out {
		out[i].Pointer = fmt.Sprintf("%s/%d", prefix, i)
		if out[i].Container != nil {
			c := *out[i].Container
			c.Entries = pointers(c.Entries, out[i].Pointer+"/container/entries")
			out[i].Container = &c
		}
	}
	return out
}

// jsonValue returns the value of the JSON text src.
func jsonValue(t testing.TB, file, src string) value.Value {
	t.Helper()
	from := jsonx.Source{
		Input: finding.Subject{Kind: finding.SubjectSchema, Name: file},
		At:    finding.Location{Origin: finding.OriginPackage, File: file},
	}
	doc, fs := jsonx.Parse([]byte(src), from, limits.Defaults())
	if len(fs) > 0 {
		t.Fatalf("%s: %v", file, fs)
	}
	return doc.Value
}

// pkg builds the Package fx describes.
func (fx fixture) pkg(t testing.TB) Package {
	t.Helper()
	testenv.Home(t)
	d := dialect(t)
	m := &manifest.Manifest{Name: "fixture", Version: "1.0.0", Title: "Fixture"}
	var decl schema.Declarations
	var files []schema.File
	for i, x := range fx.inputs {
		path := "inputs/" + x.name + ".json"
		m.Inputs = append(m.Inputs, manifest.Input{
			Pointer: fmt.Sprintf("/inputs/%d", i), Name: x.name, Title: strings.ToUpper(x.name),
			Schema: path, Forms: []manifest.Form{manifest.FormJSON}, Collection: x.collection, Required: true,
		})
		decl.Inputs = append(decl.Inputs, schema.Input{Name: x.name, Title: strings.ToUpper(x.name), Schema: path, Required: true})
		files = append(files, schema.File{Path: path, Root: jsonValue(t, path, x.schema)})
	}
	for _, path := range slices.Sorted(maps.Keys(fx.files)) {
		files = append(files, schema.File{Path: path, Root: jsonValue(t, path, fx.files[path])})
	}
	combined, err := schema.Bundle(decl, files)
	if err != nil {
		t.Fatalf("Bundle: %v", err)
	}
	m.Outputs = pointers(fx.outputs, "/outputs")
	p := Package{Manifest: m, Schema: combined, Assets: fx.assets, Dialect: d, Limits: limits.Defaults()}
	for _, path := range slices.Sorted(maps.Keys(fx.templates)) {
		if slices.Contains(fx.broken, path) {
			p.Templates = append(p.Templates, Template{Path: path})
			continue
		}
		tm, errs := liquid.Parse(d, liquid.Source{Name: TemplatesDir + path, Text: []byte(fx.templates[path])}, liquid.ParseLimits{})
		if len(errs) > 0 {
			t.Fatalf("parse %s: %v", path, errs)
		}
		p.Templates = append(p.Templates, Template{Path: path, Root: tm.Root})
	}
	return p
}

// run analyses fx, failing t on a refusal.
func (fx fixture) run(t testing.TB) Result {
	t.Helper()
	r, err := Analyze(fx.pkg(t))
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	return r
}

// describe is a finding as one line: its code, location, params and
// related locations, as its JSON encoding gives them.
func describe(t testing.TB, f finding.Finding) string {
	t.Helper()
	var js []struct {
		Code     string
		Subject  struct{ Name string }
		Location *struct {
			File         string
			Line, Column int
			Pointer      *string
		}
		Params  map[string]any
		Related []struct {
			File    string
			Pointer *string
			Line    int
			Note    string
		}
	}
	if err := json.Unmarshal(finding.AppendJSON(nil, f), &js); err != nil || len(js) != 1 {
		t.Fatalf("decode finding: %v", err)
	}
	j := js[0]
	var b strings.Builder
	b.WriteString(j.Code)
	if l := j.Location; l != nil {
		fmt.Fprintf(&b, " %s", l.File)
		if l.Line > 0 {
			fmt.Fprintf(&b, ":%d:%d", l.Line, l.Column)
		}
		if l.Pointer != nil {
			fmt.Fprintf(&b, "#%s", *l.Pointer)
		}
	}
	for _, k := range slices.Sorted(maps.Keys(j.Params)) {
		v, _ := json.Marshal(j.Params[k])
		fmt.Fprintf(&b, " %s=%s", k, v)
	}
	for _, r := range j.Related {
		fmt.Fprintf(&b, " related=%s", r.File)
		if r.Line > 0 {
			fmt.Fprintf(&b, ":%d", r.Line)
		}
		if r.Pointer != nil {
			fmt.Fprintf(&b, "#%s", *r.Pointer)
		}
	}
	return b.String()
}

// describeAll describes every finding of r, sorted.
func describeAll(t testing.TB, r Result) []string {
	t.Helper()
	var out []string
	for _, f := range r.Findings {
		out = append(out, describe(t, f))
	}
	slices.Sort(out)
	return out
}

// expect fails t unless r's findings are exactly want, in any order.
func expect(t testing.TB, r Result, want ...string) {
	t.Helper()
	got := describeAll(t, r)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("findings:\n\t%s\nwant:\n\t%s", strings.Join(got, "\n\t"), strings.Join(want, "\n\t"))
	}
}

// codes returns the codes of r's findings, sorted.
func codes(r Result) []string {
	var out []string
	for _, f := range r.Findings {
		out = append(out, string(f.Code()))
	}
	slices.Sort(out)
	return out
}

// one is a fixture of one input, metrics, with schema s, and one template
// output rendering text.
func one(s, text string) fixture {
	return fixture{
		inputs:    []fxInput{in("metrics", s)},
		files:     map[string]string{"inputs/address.json": `{"type": "object", "properties": {"city": {"type": "string"}}}`},
		templates: map[string]string{"report.md.liquid": text},
		outputs:   []manifest.Output{tpl("report", "report.md.liquid")},
	}
}
