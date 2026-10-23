// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package author

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/fault"
	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid"
	"metais.org/metais/internal/loader"
	templates "metais.org/metais/internal/template"
	"metais.org/metais/internal/testenv"
)

const checkManifest = `metais: "1.0"
name: example
version: 1.0.0
title: Example
description: An example package.
guide: Supply a report.
inputs:
  - name: report
    title: Report
    schema: inputs/report.schema.json
outputs:
  - id: report
    template: report.md.liquid
`

// checkFiles is a package with nothing to report, by entry.
func checkFiles() map[string]string {
	return map[string]string{
		"metais.yaml":                checkManifest,
		"inputs/report.schema.json":  `{"type": "object", "properties": {"title": {"type": "string"}}}`,
		"templates/report.md.liquid": "# {{ inputs.report.title }}\n",
	}
}

// writeSource writes files as a source directory and returns it.
func writeSource(t *testing.T, files map[string]string) string {
	t.Helper()
	testenv.Home(t)
	dir := filepath.Join(testenv.Dir(t), "pkg")
	for _, name := range slices.Sorted(maps.Keys(files)) {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(files[name]), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// runCheck checks files as a source directory at the default limits.
func runCheck(t *testing.T, files map[string]string) []finding.Finding {
	t.Helper()
	return Check(writeSource(t, files), limits.Defaults(), fault.Set(0))
}

// line is a finding as one line: its code, file, pointer and params.
func line(t *testing.T, f finding.Finding) string {
	t.Helper()
	var js []struct {
		Code     string
		Location *struct {
			File    string
			Pointer *string
		}
		Params map[string]any
	}
	if err := json.Unmarshal(finding.AppendJSON(nil, f), &js); err != nil || len(js) != 1 {
		t.Fatalf("decode finding: %v", err)
	}
	j := js[0]
	var b strings.Builder
	b.WriteString(j.Code)
	if l := j.Location; l != nil && l.File != "" {
		fmt.Fprintf(&b, " %s", l.File)
		if l.Pointer != nil {
			fmt.Fprintf(&b, "#%s", *l.Pointer)
		}
	}
	for _, k := range slices.Sorted(maps.Keys(j.Params)) {
		v, _ := json.Marshal(j.Params[k])
		fmt.Fprintf(&b, " %s=%s", k, v)
	}
	return b.String()
}

func lines(t *testing.T, fs []finding.Finding) []string {
	t.Helper()
	var out []string
	for _, f := range fs {
		out = append(out, line(t, f))
	}
	return out
}

func codesOf(fs []finding.Finding) []string {
	var out []string
	for _, f := range fs {
		out = append(out, string(f.Code()))
	}
	return out
}

// TestCheckClean: a package with nothing to report gives no finding, and
// the scaffold init writes is one (ADR-007 D1 item 7).
func TestCheckClean(t *testing.T) {
	if fs := runCheck(t, checkFiles()); len(fs) > 0 {
		t.Errorf("findings %q", lines(t, fs))
	}
	testenv.Home(t)
	dir := filepath.Join(testenv.Dir(t), "scaffold")
	if _, err := Init(dir, "scaffold", Form{}, false); err != nil {
		t.Fatal(err)
	}
	if fs := Check(dir, limits.Defaults(), 0); len(fs) > 0 {
		t.Errorf("init's scaffold: findings %q", lines(t, fs))
	}
}

// TestCheckStepsRunPastFailures is ADR-007 D2 item 2: each step runs
// whatever the steps before it found, and a step whose input could not be
// made is named, once, in report.step_skipped, in step order.
func TestCheckStepsRunPastFailures(t *testing.T) {
	files := checkFiles()
	files["metais.yaml"] = strings.Replace(checkManifest, "title: Example\n", "", 1)
	files["inputs/report.schema.json"] = `{"type": "object", "properties": {"title": {"type": "string", "default": 1}}}`
	files["templates/report.md.liquid"] = "{{ inputs.report.title | }}\n"
	files["examples/readme.md"] = "x\n"
	got := codesOf(runCheck(t, files))
	want := []string{"manifest.required", "schema.invalid_default", "template.syntax", "example.layout", "report.step_skipped"}
	if !slices.Equal(got, want) {
		t.Errorf("codes %q, want %q", got, want)
	}

	// Schemas that do not compile skip nothing: the cross-reference runs
	// with every input unknown, and the compiled form's failure is the
	// schema's finding.
	files = checkFiles()
	files["inputs/report.schema.json"] = `{"$ref": "../../outside.json"}`
	files["templates/report.md.liquid"] = "{{ inptus.report.title }}\n"
	got = codesOf(runCheck(t, files))
	want = []string{"schema.ref_escapes", "template.xref.unknown_variable", "template.xref.unused_input"}
	if !slices.Equal(got, want) {
		t.Errorf("schemas that do not compile: codes %q, want %q", got, want)
	}
}

// TestCheckStepSkippedNames is the skipped steps by name: a source
// directory's adds the limits of the compiled files, which a packed
// package's read has already held.
func TestCheckStepSkippedNames(t *testing.T) {
	files := checkFiles()
	files["metais.yaml"] = strings.Replace(checkManifest, "guide: Supply a report.\n", "", 1)
	fs := runCheck(t, files)
	last := line(t, fs[len(fs)-1])
	want := `report.step_skipped steps=["limits of the compiled files","manifest paths","guide links","combined schema","manifest expressions","cross-reference","compiled form"]`
	if last != want {
		t.Errorf("last finding %s, want %s", last, want)
	}
}

// TestCheckRefusedRead is MTS-AUT-003: a package whose read is refused is
// refused with the findings the loader gives every command, and nothing
// else.
func TestCheckRefusedRead(t *testing.T) {
	files := checkFiles()
	files[".compiled/manifest.json"] = "{}"
	files["templates/broken.liquid"] = "{{ |"
	dir := writeSource(t, files)
	_, want := loader.Open(dir, loader.Options{Limits: limits.Defaults()}, 0)
	got := Check(dir, limits.Defaults(), 0)
	if !slices.Equal(lines(t, got), lines(t, want)) || len(got) == 0 {
		t.Errorf("check %q, the loader %q", lines(t, got), lines(t, want))
	}
}

// TestCheckWritesNothing: check reads the package in place and writes
// nothing, to it or beside it.
func TestCheckWritesNothing(t *testing.T) {
	files := checkFiles()
	files["templates/report.md.liquid"] = "{{ inputs.report.nope }}\n"
	dir := writeSource(t, files)
	before := tree(t, filepath.Dir(dir))
	if fs := Check(dir, limits.Defaults(), 0); len(fs) == 0 {
		t.Fatal("no finding")
	}
	if after := tree(t, filepath.Dir(dir)); !slices.Equal(before, after) {
		t.Errorf("the tree changed: %q, then %q", before, after)
	}
}

// TestCheckSourceTree is step 1's warnings: OS litter beneath a layout
// tree, and a top-level name pack leaves out. Top-level dotfiles, a
// .metais file and litter outside the layout trees are not reported.
func TestCheckSourceTree(t *testing.T) {
	files := checkFiles()
	files["assets/.DS_Store"] = "x"
	files["templates/._report.md.liquid"] = "x"
	files["references/sub/Thumbs.db"] = "x"
	files["Thumbs.db"] = "x"
	files[".gitignore"] = "x"
	files["old.metais"] = "x"
	got := lines(t, runCheck(t, files))
	want := []string{
		"package.layout.not_packed Thumbs.db",
		"package.layout.os_litter assets/.DS_Store",
		"package.layout.os_litter references/sub/Thumbs.db",
		"package.layout.os_litter templates/._report.md.liquid",
		"template.xref.unused_template templates/._report.md.liquid",
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("findings %q, want %q", got, want)
	}
}

// TestGuideLinks is markdownLinks and packagePath: inline links and
// images, reference definitions, and the targets that are not relative.
func TestGuideLinks(t *testing.T) {
	src := "See [a](references/a.md) and ![b](<assets/b c.png> \"t\") and [c](https://x.org/y) and [d](#top).\n" +
		"`[e](code.md)` and ``[f](code2.md)`` stay code, [g](references/(g).md) is balanced.\n" +
		"```\n[h](fenced.md)\n```\n" +
		"[ref]: ./references/r.md#part\n" +
		"   [ref2]: <references/r2.md>\n" +
		"[i](/absolute.md) [j](mailto:a@b.c) [k](references/k%20l.md?q=1)\n"
	got := markdownLinks(src)
	want := []string{"references/a.md", "assets/b c.png", "https://x.org/y", "#top", "references/(g).md",
		"./references/r.md#part", "references/r2.md", "/absolute.md", "mailto:a@b.c", "references/k%20l.md?q=1"}
	if !slices.Equal(got, want) {
		t.Errorf("links %q, want %q", got, want)
	}
	var rel []string
	for _, l := range got {
		if p, ok := packagePath(l); ok {
			rel = append(rel, p)
		}
	}
	wantRel := []string{"references/a.md", "assets/b c.png", "references/(g).md", "references/r.md", "references/r2.md", "references/k l.md"}
	if !slices.Equal(rel, wantRel) {
		t.Errorf("package paths %q, want %q", rel, wantRel)
	}
}

// TestCheckGuideLinks is step 4's warning: one per distinct missing
// target, none for an entry or a directory holding one.
func TestCheckGuideLinks(t *testing.T) {
	files := checkFiles()
	files["metais.yaml"] = strings.Replace(checkManifest, "guide: Supply a report.",
		"guide: \"Read [n](references/notes.md), [d](references/), [x](references/x.md) and [x again](references/x.md).\"", 1)
	files["references/notes.md"] = "# Notes\n"
	got := lines(t, runCheck(t, files))
	want := []string{`manifest.guide_link_missing metais.yaml#/guide target="references/x.md"`}
	if !slices.Equal(got, want) {
		t.Errorf("findings %q, want %q", got, want)
	}
}

// TestSchemaStatics is step 5: defaults, examples, consts and enum
// members validated against the object holding them, at every depth, and
// required names held to D3 item 4's declared.
func TestSchemaStatics(t *testing.T) {
	files := checkFiles()
	files["inputs/report.schema.json"] = `{
	  "type": "object",
	  "required": ["title", "titel", "viaRef", "viaAllOf", "viaPattern_x", "viaThen", "inNot"],
	  "properties": {
	    "title": {"type": "string", "default": "ok", "examples": ["ok"]},
	    "list": {"type": "array", "items": {"type": "integer", "default": "one", "const": 2, "maximum": 1}},
	    "kind": {"enum": ["a", "b"], "type": "string", "default": "a"}
	  },
	  "allOf": [{"$ref": "#/$defs/more"}, {"properties": {"viaAllOf": true}}],
	  "patternProperties": {"^viaPattern_": {}},
	  "if": {"type": "object"}, "then": {"properties": {"viaThen": {}}},
	  "not": {"properties": {"inNot": {}}},
	  "$defs": {
	    "more": {"properties": {"viaRef": {}}},
	    "open": {"type": "object", "required": ["anything"], "additionalProperties": {"type": "string"}},
	    "closed": {"type": "object", "required": ["nothing"], "additionalProperties": false},
	    "opened": {"allOf": [true], "required": ["x"]}
	  }
	}`
	got := lines(t, runCheck(t, files))
	want := []string{
		"schema.invalid_default inputs/report.schema.json#/properties/list/items/default",
		"schema.unsatisfiable inputs/report.schema.json#/properties/list/items/const",
		"schema.required_not_declared inputs/report.schema.json#/required/1 name=\"titel\"",
		"schema.required_not_declared inputs/report.schema.json#/required/6 name=\"inNot\"",
		"schema.required_not_declared inputs/report.schema.json#/$defs/closed/required/0 name=\"nothing\"",
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("findings\n\t%s\nwant\n\t%s", strings.Join(got, "\n\t"), strings.Join(want, "\n\t"))
	}
}

// TestSchemaStaticsBeneathArrayApplicators is step 5 beneath each keyword
// whose value is an array of schemas: a default that fails the object
// holding it is found beneath allOf, anyOf, oneOf and prefixItems.
func TestSchemaStaticsBeneathArrayApplicators(t *testing.T) {
	files := checkFiles()
	files["inputs/report.schema.json"] = `{
	  "type": "object",
	  "properties": {
	    "pair": {"type": "array", "prefixItems": [{"type": "integer", "default": "p"}]}
	  },
	  "allOf": [{"properties": {"a": {"type": "integer"}}}, {"type": "object", "default": "a"}],
	  "anyOf": [{"type": "object", "default": "b"}, {}],
	  "oneOf": [{"type": "object", "default": "c"}]
	}`
	got := lines(t, runCheck(t, files))
	want := []string{
		"schema.invalid_default inputs/report.schema.json#/allOf/1/default",
		"schema.invalid_default inputs/report.schema.json#/anyOf/0/default",
		"schema.invalid_default inputs/report.schema.json#/oneOf/0/default",
		"schema.invalid_default inputs/report.schema.json#/properties/pair/prefixItems/0/default",
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("findings\n\t%s\nwant\n\t%s", strings.Join(got, "\n\t"), strings.Join(want, "\n\t"))
	}
}

// TestParseHintCauses is ADR-007 D2 item 4: each cause, and the order of
// preference among them.
func TestParseHintCauses(t *testing.T) {
	const w = `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>`
	for _, c := range []struct {
		name, path, src, cause string
	}{
		{"grammar", "a.md.liquid", "{{ 1 | }}", loader.CauseGrammar},
		{"quotes", "a.md.liquid", "{{ “x” }}", loader.CauseQuotes},
		{"quotes win over the grammar in any template", "a.md.liquid", "{% if a == ‘x’ %}{% endif %}", loader.CauseQuotes},
		{"an entity outside OOXML is the grammar", "a.md.liquid", "{{ a &lt; b }}", loader.CauseGrammar},
		{"an entity in OOXML", "a.xml.liquid", w + "{% if a &lt; 3 %}{% endif %}</w:t></w:r></w:p></w:body></w:document>", loader.CauseEscaped},
		{"an entity inside a string literal is not the cause", "a.xml.liquid", w + "{{ \"&lt;\" | }}</w:t></w:r></w:p></w:body></w:document>", loader.CauseGrammar},
		{"markup in OOXML", "a.xml.liquid", w + "{{ a.</w:t></w:r><w:r><w:t>b }}</w:t></w:r></w:p></w:body></w:document>", loader.CauseMarkup},
		{"markup wins over quotes", "a.xml.liquid", w + "{{ “a</w:t></w:r><w:r><w:t>b” }}</w:t></w:r></w:p></w:body></w:document>", loader.CauseMarkup},
		{"a partial of an OOXML template is OOXML", "row.liquid", "{% if a &lt; 3 %}{% endif %}", loader.CauseEscaped},
	} {
		t.Run(c.name, func(t *testing.T) {
			files := checkFiles()
			files["templates/"+c.path] = c.src
			if c.path == "row.liquid" {
				files["templates/doc.xml.liquid"] = w + `{% render "row.liquid" %}</w:t></w:r></w:p></w:body></w:document>`
			}
			var causes []string
			for _, f := range runCheck(t, files) {
				if f.Code() == "template.syntax" {
					v, _ := f.Param("cause")
					s, _ := v.Str()
					causes = append(causes, s)
				}
			}
			if !slices.Equal(causes, []string{c.cause}) {
				t.Errorf("causes %q, want %q", causes, c.cause)
			}
		})
	}
}

// TestCheckOneParse is W4-4.7: check parses a template with the parse the
// manifest's expressions use, Dialect() under ParseLimits(), and reports
// exactly the errors that parse gives. Dialect() is liquid.Core() with
// METAIS's filters and date (W5-1), and with METAIS's embed, asset and
// provenance tags known by name, which no expression may hold (ADR-004 D16 item 5).
func TestCheckOneParse(t *testing.T) {
	want := append(liquid.Core().FilterNames(), "csv", "date", "duration", "json", "json_script", "markdown_table", "safe", "slug", "yaml")
	slices.Sort(want)
	if got := loader.Dialect().FilterNames(); !slices.Equal(got, want) {
		t.Errorf("the dialect's filters are %q, want %q", got, want)
	}
	if got, want := loader.Dialect().FilterNames(), templates.Dialect().FilterNames(); !slices.Equal(got, want) {
		t.Errorf("the expressions' filters are %q, the templates' %q", got, want)
	}
	for _, tag := range []string{"embed", "asset", "provenance"} {
		if _, ok := loader.Dialect().Tag(tag); !ok {
			t.Errorf("the dialect does not know the %s tag", tag)
		}
	}
	srcs := []string{"{{ a | nosuchfilter }}", "{% include 'x' %}", "{{ 1 | divided_by: 0 }}", "{% for %}", "{{ 99999999999999999999 }}", "{{ 0 | date: '%Q' }}"}
	for _, src := range srcs {
		files := checkFiles()
		files["templates/report.md.liquid"] = src
		_, errs := liquid.Parse(loader.Dialect(), liquid.Source{Name: "templates/report.md.liquid", Text: []byte(src)}, loader.ParseLimits(limits.Defaults()))
		var want []string
		for _, e := range errs {
			l := finding.Location{Origin: finding.OriginPackage, File: "templates/report.md.liquid", Line: e.Line, Column: e.Col}
			want = append(want, line(t, loader.TemplateFinding(e, finding.Subject{Kind: finding.SubjectTemplate, Name: "templates/report.md.liquid"}, l, limits.Defaults(), loader.CauseGrammar)))
		}
		var got []string
		for _, f := range runCheck(t, files) {
			if strings.HasPrefix(string(f.Code()), "template.") && !strings.HasPrefix(string(f.Code()), "template.xref.") {
				got = append(got, line(t, f))
			}
		}
		if len(want) == 0 || !slices.Equal(got, want) {
			t.Errorf("%q: check %q, the parse %q", src, got, want)
		}
	}
}

// TestCheckTemplateFindings is step 6: not UTF-8 is reported and not
// parsed; a CR LF is located at its first occurrence.
func TestCheckTemplateFindings(t *testing.T) {
	files := checkFiles()
	files["templates/report.md.liquid"] = "# {{ inputs.report.title }}\nä\r\nb\r\n"
	files["templates/bin.liquid"] = "ok\n\xff{{ |"
	var got []string
	for _, f := range runCheck(t, files) {
		l, _ := f.Location()
		got = append(got, fmt.Sprintf("%s %s:%d:%d", f.Code(), l.File, l.Line, l.Column))
	}
	want := []string{"template.not_utf8 templates/bin.liquid:2:1", "template.crlf templates/report.md.liquid:2:2", "template.xref.unused_template templates/bin.liquid:0:0"}
	if !slices.Equal(got, want) {
		t.Errorf("findings %q, want %q", got, want)
	}
}

// TestCheckExamples is step 9.
func TestCheckExamples(t *testing.T) {
	files := checkFiles()
	files["examples/basic/inputs/report.json"] = "{}"
	files["examples/basic/expected/report.md"] = "#\n"
	files["examples/basic/example.yaml"] = "title: Basic\n"
	files["examples/basic/example.yml"] = "x"
	files["examples/basic/inputs.json"] = "x"
	files["examples/2bad/inputs/report.json"] = "{}"
	files["examples/2bad/other"] = "{}"
	files["examples/README.md"] = "x"
	got := lines(t, runCheck(t, files))
	want := []string{
		`example.layout examples/2bad/inputs/report.json reason="name"`,
		`example.layout examples/README.md reason="entry"`,
		`example.layout examples/basic/example.yml reason="entry"`,
		`example.layout examples/basic/inputs.json reason="entry"`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("findings %q, want %q", got, want)
	}
}

// TestCheckCompiledLimits is step 3: README.md, the compiled files and
// the digest file count toward the entries limit of the package pack
// would write (ADR-001 A1 item 1).
func TestCheckCompiledLimits(t *testing.T) {
	dir := writeSource(t, checkFiles())
	lim := limits.Defaults()
	if f, ok := lim.Lower("entries=7"); !ok {
		t.Fatal(f)
	}
	got := lines(t, Check(dir, lim, 0))
	if want := []string{`limit.package bound=7 limit="entries" lowered=true observed=8`}; !slices.Equal(got, want) {
		t.Errorf("findings %q, want %q", got, want)
	}
	lim = limits.Defaults()
	if f, ok := lim.Lower("entries=8"); !ok {
		t.Fatal(f)
	}
	if fs := Check(dir, lim, 0); len(fs) > 0 {
		t.Errorf("at the bound: %q", lines(t, fs))
	}
}

// TestCheckOutputPaths is MTS-AUT-080, the plan orchestrator's Wave 7
// ruling: check refuses an output path the manifest writes literally that
// every render refuses, with the render's finding, alone and together; a
// path template's path, a declaration with a when, and an asset's root
// destination are left to the render.
func TestCheckOutputPaths(t *testing.T) {
	files := checkFiles()
	files["metais.yaml"] = checkManifest + `    path: ../escape.txt
  - id: computed
    template: report.md.liquid
    path: "{{ package.name }}/../escape.txt"
  - id: first
    template: report.md.liquid
    path: a.md
  - id: again
    template: report.md.liquid
    path: a.md
  - id: sometimes
    template: report.md.liquid
    path: a.md
    when: package.name == "example"
  - id: everything
    asset: .
    path: .
`
	files["assets/logo.txt"] = "logo\n"
	want := []string{
		`output.path.shape metais.yaml#/outputs/0/path path="../escape.txt" reason="dot_segment"`,
		`output.path.duplicate metais.yaml#/outputs/3/path path="a.md"`,
	}
	if got := lines(t, runCheck(t, files)); !slices.Equal(got, want) {
		t.Errorf("findings\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	files["metais.yaml"] = strings.Replace(files["metais.yaml"], "path: ../escape.txt", "path: escape.txt", 1)
	files["metais.yaml"] = strings.Replace(files["metais.yaml"], "  - id: again\n    template: report.md.liquid\n    path: a.md\n", "", 1)
	if fs := runCheck(t, files); len(fs) > 0 {
		t.Errorf("portable paths: findings %q", lines(t, fs))
	}
}
