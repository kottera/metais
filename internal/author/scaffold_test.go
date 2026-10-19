// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package author

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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

// goldenDir holds the scaffold for goldenName, byte for byte.
var goldenDir = filepath.Join("testdata", "scaffold", goldenName)

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

// readGolden returns the golden scaffold's files in scaffoldOrder.
func readGolden(t *testing.T) []File {
	t.Helper()
	var out []File
	for _, p := range scaffoldOrder {
		b, err := os.ReadFile(filepath.Join(goldenDir, filepath.FromSlash(p)))
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

// TestScaffoldGolden is W3-8.1: the scaffold is ADR-007 D1 item 4's eight
// files, in its order, equal byte for byte to the committed golden files.
func TestScaffoldGolden(t *testing.T) {
	got := Scaffold(goldenName)
	if *update {
		for _, f := range got {
			p := filepath.Join(goldenDir, filepath.FromSlash(f.Path))
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
	if !slices.Equal(paths, scaffoldOrder) {
		t.Fatalf("the scaffold writes %q, want %q", paths, scaffoldOrder)
	}
	if err := diffScaffold(got, readGolden(t)); err != nil {
		t.Error(err)
	}
	// No golden file goes stale.
	var onDisk []string
	err := filepath.WalkDir(goldenDir, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(goldenDir, p)
			onDisk = append(onDisk, filepath.ToSlash(rel))
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(onDisk)
	want := slices.Sorted(slices.Values(scaffoldOrder))
	if !slices.Equal(onDisk, want) {
		t.Errorf("the golden directory holds %q, want %q", onDisk, want)
	}
}

// TestScaffoldOneByte is W3-8.1's control: the comparison TestScaffoldGolden
// makes fails on a change of any one byte of any file.
func TestScaffoldOneByte(t *testing.T) {
	want := readGolden(t)
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
		for _, f := range Scaffold(name) {
			if !utf8.Valid(f.Data) || bytes.ContainsRune(f.Data, '\r') || !bytes.HasSuffix(f.Data, []byte("\n")) {
				t.Errorf("%s for %s is not UTF-8 text with LF endings and a final newline", f.Path, name)
			}
		}
	}
}

// TestScaffoldTemplateExact pins the template to ADR-007 D1 item 4's text.
func TestScaffoldTemplateExact(t *testing.T) {
	const want = "# Hello, {{ inputs.greeting.name }}\n{%- if inputs.greeting.note %}\n\n{{ inputs.greeting.note }}\n{%- endif %}\n"
	if got := file(t, Scaffold(goldenName), "templates/greeting.md.liquid"); got != want {
		t.Errorf("the template is %q, want %q", got, want)
	}
	if got := file(t, Scaffold(goldenName), "examples/basic/expected/greeting.md"); got != "# Hello, Ada Lovelace\n\nWelcome.\n" {
		t.Errorf("the expected output is %q", got)
	}
	if got := file(t, Scaffold(goldenName), ".gitattributes"); got != "* -text\n" {
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

// TestScaffoldReads holds every file the scaffold writes to the reader of
// its kind: metais.yaml is a manifest Build accepts, naming the package,
// for every kind of name the rule accepts, including those a YAML reader
// would take for another type unquoted; and the JSON and YAML files parse
// under their profiles.
func TestScaffoldReads(t *testing.T) {
	long := strings.Repeat("a", 63) + "." + strings.Repeat("b", 63)
	for _, name := range []string{goldenName, "x", "1.0", "true", "null", "0x10", "1e3", "org.example.vpat-acr", long} {
		if !ValidName(name) {
			t.Fatalf("%q does not satisfy the name rule", name)
		}
		fs := Scaffold(name)
		doc, fds := yamlx.Parse([]byte(file(t, fs, "metais.yaml")), yamlx.Source{Subject: finding.Subject{Kind: finding.SubjectManifest}}, limits.Defaults())
		if len(fds) > 0 {
			t.Fatalf("metais.yaml for %q: %v", name, fds[0].Message())
		}
		m, err := manifest.Build(doc.Value)
		if err != nil {
			t.Fatalf("metais.yaml for %q: %v", name, err)
		}
		if m.Name != name || m.Title != Title(name) || m.Version != "0.1.0" || len(m.Inputs) != 1 || len(m.Outputs) != 1 {
			t.Errorf("metais.yaml for %q reads as %+v", name, m)
		}
	}
	fs := Scaffold(goldenName)
	for _, p := range []string{"inputs/greeting.schema.json", "examples/basic/inputs/greeting.json"} {
		in := finding.Subject{Kind: finding.SubjectInput, Name: "greeting"}
		if _, fds := jsonx.Parse([]byte(file(t, fs, p)), jsonx.Source{Input: in}, limits.Defaults()); len(fds) > 0 {
			t.Errorf("%s: %v", p, fds[0].Message())
		}
	}
	in := finding.Subject{Kind: finding.SubjectInput, Name: "example"}
	if _, fds := yamlx.Parse([]byte(file(t, fs, "examples/basic/example.yaml")), yamlx.Source{Subject: in}, limits.Defaults()); len(fds) > 0 {
		t.Errorf("example.yaml: %v", fds[0].Message())
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
