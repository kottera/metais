// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package author

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/jsonx"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/yamlx"
)

var update = flag.Bool("update", false, "rewrite the scaffold's golden files")

// goldenName is the name the golden scaffold is written for.
const goldenName = "acme-report"

// scaffoldOrder is ADR-007 D1 item 4's list, in its order.
var scaffoldOrder = []string{
	".gitattributes",
	"metais.yaml",
	"inputs/greeting.schema.json",
	"inputs/greeting.md",
	"templates/greeting.md.liquid",
	"examples/basic/example.yaml",
	"examples/basic/inputs/greeting.json",
	"examples/basic/expected/greeting.md",
}

// goldenForms are init's scaffolds (ADR-007 D1 item 4, A1), each with the
// directory holding it for goldenName byte for byte, and its files in the
// order init writes them.
var goldenForms = []struct {
	form  Form
	dir   string
	order []string
}{
	{Form{}, filepath.Join("testdata", "scaffold", goldenName), scaffoldOrder},
	{Form{HTML: true}, filepath.Join("testdata", "scaffold", "html", goldenName), []string{
		".gitattributes",
		"metais.yaml",
		"inputs/greeting.schema.json",
		"inputs/greeting.md",
		"templates/greeting.html.liquid",
		"examples/basic/example.yaml",
		"examples/basic/inputs/greeting.json",
		"examples/basic/expected/greeting.html",
	}},
	{Form{Minimal: true}, filepath.Join("testdata", "scaffold", "minimal", goldenName), []string{
		".gitattributes",
		"metais.yaml",
		"inputs/data.schema.json",
		"inputs/data.md",
		"templates/document.md.liquid",
	}},
	{Form{Minimal: true, HTML: true}, filepath.Join("testdata", "scaffold", "minimal-html", goldenName), []string{
		".gitattributes",
		"metais.yaml",
		"inputs/data.schema.json",
		"inputs/data.md",
		"templates/page.html.liquid",
	}},
}

// readGolden returns the golden scaffold in dir, its files in order.
func readGolden(t *testing.T, dir string, order []string) []File {
	t.Helper()
	var out []File
	for _, p := range order {
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(p)))
		if err != nil {
			t.Fatalf("%v (run go test -run TestScaffoldGolden -args -update)", err)
		}
		out = append(out, File{p, b})
	}
	return out
}

// diffScaffold returns the first difference between got and want: a path,
// an order or a byte.
func diffScaffold(got, want []File) error {
	if len(got) != len(want) {
		return fmt.Errorf("%d files, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i].Path != want[i].Path {
			return fmt.Errorf("file %d is %s, want %s", i, got[i].Path, want[i].Path)
		}
		if !bytes.Equal(got[i].Data, want[i].Data) {
			return fmt.Errorf("%s differs:\ngot:\n%s\nwant:\n%s", got[i].Path, got[i].Data, want[i].Data)
		}
	}
	return nil
}

// TestScaffoldGolden is W3-8.1, and W2-6.1's and W2-6.3's forms: each
// scaffold is its files, in its order, equal byte for byte to the
// committed golden files, and no golden file goes stale.
func TestScaffoldGolden(t *testing.T) {
	for _, g := range goldenForms {
		got := Scaffold(goldenName, g.form)
		if *update {
			if err := os.RemoveAll(g.dir); err != nil {
				t.Fatal(err)
			}
			for _, f := range got {
				p := filepath.Join(g.dir, filepath.FromSlash(f.Path))
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, f.Data, 0o644); err != nil {
					t.Fatal(err)
				}
			}
		}
		var paths []string
		for _, f := range got {
			paths = append(paths, f.Path)
		}
		if !slices.Equal(paths, g.order) {
			t.Fatalf("%+v: the scaffold writes %q, want %q", g.form, paths, g.order)
		}
		if err := diffScaffold(got, readGolden(t, g.dir, g.order)); err != nil {
			t.Errorf("%+v: %v", g.form, err)
		}
		var onDisk []string
		err := filepath.WalkDir(g.dir, func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				rel, _ := filepath.Rel(g.dir, p)
				onDisk = append(onDisk, filepath.ToSlash(rel))
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		slices.Sort(onDisk)
		if want := slices.Sorted(slices.Values(g.order)); !slices.Equal(onDisk, want) {
			t.Errorf("%s holds %q, want %q", g.dir, onDisk, want)
		}
	}
}

// TestScaffoldOneByte is W3-8.1's control: the comparison TestScaffoldGolden
// makes fails on a change of any one byte of any file.
func TestScaffoldOneByte(t *testing.T) {
	want := readGolden(t, goldenForms[0].dir, scaffoldOrder)
	for i, f := range want {
		for j := range f.Data {
			got := slices.Clone(want)
			got[i].Data = slices.Clone(f.Data)
			got[i].Data[j] ^= 0x01
			if diffScaffold(got, want) == nil {
				t.Fatalf("a change of byte %d of %s goes unnoticed", j, f.Path)
			}
		}
		got := slices.Clone(want)
		got[i].Data = append(slices.Clone(f.Data), 'x')
		if diffScaffold(got, want) == nil {
			t.Fatalf("a byte added to %s goes unnoticed", f.Path)
		}
	}
}

// TestScaffoldText is ADR-007 D1 item 4's encoding: every file is UTF-8
// with LF line endings and a final newline.
func TestScaffoldText(t *testing.T) {
	for _, name := range []string{goldenName, "x", "org.example.vpat-acr"} {
		for _, g := range goldenForms {
			for _, f := range Scaffold(name, g.form) {
				if !utf8.Valid(f.Data) || bytes.ContainsRune(f.Data, '\r') || !bytes.HasSuffix(f.Data, []byte("\n")) {
					t.Errorf("%s for %s, %+v, is not UTF-8 text with LF endings and a final newline", f.Path, name, g.form)
				}
			}
		}
	}
}

// TestScaffoldTemplateExact pins the template to ADR-007 D1 item 4's text.
func TestScaffoldTemplateExact(t *testing.T) {
	const want = "# Hello, {{ inputs.greeting.name }}\n{%- if inputs.greeting.note %}\n\n{{ inputs.greeting.note }}\n{%- endif %}\n"
	if got := file(t, Scaffold(goldenName, Form{}), "templates/greeting.md.liquid"); got != want {
		t.Errorf("the template is %q, want %q", got, want)
	}
	if got := file(t, Scaffold(goldenName, Form{}), "examples/basic/expected/greeting.md"); got != "# Hello, Ada Lovelace\n\nWelcome.\n" {
		t.Errorf("the expected output is %q", got)
	}
	if got := file(t, Scaffold(goldenName, Form{}), ".gitattributes"); got != "* -text\n" {
		t.Errorf(".gitattributes is %q", got)
	}
}

func file(t *testing.T, fs []File, path string) string {
	t.Helper()
	for _, f := range fs {
		if f.Path == path {
			return string(f.Data)
		}
	}
	t.Fatalf("the scaffold has no %s", path)
	return ""
}

// TestScaffoldReads holds every file each scaffold writes to the reader
// of its kind: metais.yaml is a manifest Build accepts, naming the
// package, for every kind of name the rule accepts, including those a YAML
// reader would take for another type unquoted; and the JSON and YAML files
// parse under their profiles.
func TestScaffoldReads(t *testing.T) {
	long := strings.Repeat("a", 63) + "." + strings.Repeat("b", 63)
	for _, g := range goldenForms {
		for _, name := range []string{goldenName, "x", "1.0", "true", "null", "0x10", "1e3", "org.example.vpat-acr", long} {
			if !ValidName(name) {
				t.Fatalf("%q does not satisfy the name rule", name)
			}
			m := readManifest(t, Scaffold(name, g.form))
			if m.Name != name || m.Title != Title(name) || m.Version != "0.1.0" || len(m.Inputs) != 1 || len(m.Outputs) != 1 {
				t.Errorf("metais.yaml for %q, %+v, reads as %+v", name, g.form, m)
			}
		}
		for _, f := range Scaffold(goldenName, g.form) {
			switch path.Ext(f.Path) {
			case ".json":
				in := finding.Subject{Kind: finding.SubjectInput, Name: "greeting"}
				if _, fds := jsonx.Parse(f.Data, jsonx.Source{Input: in}, limits.Defaults()); len(fds) > 0 {
					t.Errorf("%+v %s: %v", g.form, f.Path, fds[0].Message())
				}
			case ".yaml":
				in := finding.Subject{Kind: finding.SubjectInput, Name: "example"}
				if _, fds := yamlx.Parse(f.Data, yamlx.Source{Subject: in}, limits.Defaults()); len(fds) > 0 {
					t.Errorf("%+v %s: %v", g.form, f.Path, fds[0].Message())
				}
			}
		}
	}
}

// readManifest is the manifest model of the scaffold fs's metais.yaml.
func readManifest(t *testing.T, fs []File) *manifest.Manifest {
	t.Helper()
	doc, fds := yamlx.Parse([]byte(file(t, fs, "metais.yaml")), yamlx.Source{Subject: finding.Subject{Kind: finding.SubjectManifest}}, limits.Defaults())
	if len(fds) > 0 {
		t.Fatalf("metais.yaml: %v", fds[0].Message())
	}
	m, err := manifest.Build(doc.Value)
	if err != nil {
		t.Fatalf("metais.yaml: %v", err)
	}
	return m
}

// TestScaffoldMinimal is ADR-007 A1 item 1: --minimal writes the layout,
// one input and one template, and nothing of the greeting or its example.
func TestScaffoldMinimal(t *testing.T) {
	for _, html := range []bool{false, true} {
		fs := Scaffold(goldenName, Form{Minimal: true, HTML: html})
		for _, f := range fs {
			if strings.HasPrefix(f.Path, "examples/") || strings.Contains(strings.ToLower(f.Path+string(f.Data)), "greeting") {
				t.Errorf("html %v: %s holds the greeting or its example", html, f.Path)
			}
		}
		var templates []string
		for _, f := range fs {
			if strings.HasPrefix(f.Path, "templates/") {
				templates = append(templates, f.Path)
			}
		}
		if m := readManifest(t, fs); len(m.Inputs) != 1 || m.Inputs[0].Name != "data" || len(templates) != 1 {
			t.Errorf("html %v: inputs %+v, templates %q, want the one input data and one template", html, m.Inputs, templates)
		}
	}
}

// TestScaffoldHTML is ADR-007 A1 item 2: with --html, the scaffold's one
// output is an HTML template whose output declares escape: html, and that
// writes its data into a <script> with json_script and never with json.
func TestScaffoldHTML(t *testing.T) {
	for _, minimal := range []bool{false, true} {
		fs := Scaffold(goldenName, Form{Minimal: minimal, HTML: true})
		m := readManifest(t, fs)
		o := m.Outputs[0]
		if o.Escape != "html" || !strings.HasSuffix(o.Source, ".html.liquid") {
			t.Fatalf("minimal %v: the output is %+v, want an .html.liquid template with escape html", minimal, o)
		}
		tpl := file(t, fs, "templates/"+o.Source)
		script := regexp.MustCompile(`<script type="application/json" id="[a-z]+">\{\{ inputs\.[a-z]+ \| json_script \}\}</script>`)
		if !script.MatchString(tpl) || strings.Contains(tpl, "| json }}") {
			t.Errorf("minimal %v: the template writes no <script> through json_script:\n%s", minimal, tpl)
		}
	}
	// Without --html, nothing is escaped.
	for _, minimal := range []bool{false, true} {
		if o := readManifest(t, Scaffold(goldenName, Form{Minimal: minimal})).Outputs[0]; o.Escape != "none" {
			t.Errorf("minimal %v without --html: escape %q", minimal, o.Escape)
		}
	}
}

// TestTitle is ADR-007 D1 item 4's title: the name with - and _ as spaces,
// each word capitalised.
func TestTitle(t *testing.T) {
	for name, want := range map[string]string{
		"acme-report":          "Acme Report",
		"x":                    "X",
		"status-widget":        "Status Widget",
		"org.example.vpat-acr": "Org.example.vpat Acr",
		"a_b-c":                "A B C",
		"1st-run":              "1st Run",
	} {
		if got := Title(name); got != want {
			t.Errorf("Title(%q) = %q, want %q", name, got, want)
		}
	}
}
