// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
)

// transformPackage writes a source directory below root holding one input, x,
// a string, the manifest's outputs outputs (YAML, indented as the list's
// items) and files, each by its package path, and returns its path.
func transformPackage(t *testing.T, root, outputs string, files map[string]string) string {
	t.Helper()
	pkg := filepath.Join(root, "pkg")
	all := map[string]string{
		"inputs/x.schema.json": `{"type": "string"}`,
		"metais.yaml": "metais: \"1.0\"\nname: org.example.transforms\nversion: 1.0.0\ntitle: Transforms\n" +
			"description: A package whose outputs are transformed.\nguide: Render it.\n" +
			"inputs:\n  - name: x\n    title: X\n    schema: inputs/x.schema.json\noutputs:\n" + outputs,
	}
	for k, v := range files {
		all[k] = v
	}
	for name, text := range all {
		p := filepath.Join(pkg, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return pkg
}

// TestRenderCRLF is ADR-005 D10 on a written output: the whole file's line
// endings, the template's own and those a value placed, are CR LF, and
// the size and digest reported are of the converted bytes.
func TestRenderCRLF(t *testing.T) {
	root := testenv.Dir(t)
	pkg := transformPackage(t, root,
		"  - id: notes\n    template: notes.txt.liquid\n    newline: crlf\n  - id: plain\n    template: notes.txt.liquid\n    path: plain.txt\n",
		map[string]string{"templates/notes.txt.liquid": "one\r\ntwo\n{{ inputs.x }}\r"})
	out := filepath.Join(root, "out")
	r := runTool(t, "render", pkg, "--set", "/x=a\nb", "--out", out, "--json")
	if r.status != 0 {
		t.Fatalf("metais render: status %d, %s", r.status, r.stdout)
	}
	for name, want := range map[string]string{"notes.txt": "one\r\ntwo\r\na\r\nb\r", "plain.txt": "one\r\ntwo\na\nb\r"} {
		got, err := os.ReadFile(filepath.Join(out, name))
		if err != nil || string(got) != want {
			t.Errorf("%s is %q, %v; want %q", name, got, err, want)
		}
	}
	for _, f := range readRender(t, r).Files {
		if f.Path == "notes.txt" && f.Size != int64(len("one\r\ntwo\r\na\r\nb\r")) {
			t.Errorf("notes.txt's size is reported as %d", f.Size)
		}
	}
}

// TestRenderValidationRefuses is ADR-005 D11 item 4, pinned false first: an
// output its schema refuses, or whose text does not parse, refuses the
// render with every finding of every validated file, each located in its
// output, and nothing is written. A valid output beside them is not
// written either.
func TestRenderValidationRefuses(t *testing.T) {
	root := testenv.Dir(t)
	pkg := transformPackage(t, root,
		"  - id: json\n    template: data.json.liquid\n    validate: references/s.json\n"+
			"  - id: yaml\n    template: data.yaml.liquid\n    validate: references/s.json\n    newline: crlf\n"+
			"  - id: broken\n    template: broken.json.liquid\n    validate: references/s.json\n"+
			"  - id: good\n    template: good.json.liquid\n    validate: references/s.json\n"+
			"  - id: table\n    template: good.json.liquid\n    path: table.csv\n    validate: references/s.json\n",
		map[string]string{
			"references/s.json":            `{"type": "object", "properties": {"n": {"type": "integer"}}, "required": ["n"]}`,
			"templates/data.json.liquid":   "{\n  \"n\": \"{{ inputs.x }}\"\n}\n",
			"templates/data.yaml.liquid":   "m: 1\nn: [1]\n",
			"templates/broken.json.liquid": "{\"n\": 1,}\n",
			"templates/good.json.liquid":   "{\"n\": 1}\n",
		})
	out := filepath.Join(root, "out")
	r := runTool(t, "render", pkg, "--set", "/x=a", "--out", out, "--json")
	if r.status != 1 || r.doc == nil || r.doc.OK {
		t.Fatalf("metais render: status %d, %s", r.status, r.stdout)
	}
	if _, err := os.Lstat(out); !os.IsNotExist(err) {
		t.Errorf("a refused render wrote %s: %v", out, err)
	}
	if entries, _ := os.ReadDir(root); len(entries) != 1 {
		t.Errorf("a refused render left %v beside the package", entries)
	}
	var doc struct {
		Findings []struct {
			Code     string
			Subject  struct{ Kind, Name string }
			Location struct {
				Origin, File, Pointer string
				Line, Column          int
			}
			Params map[string]any
		}
	}
	if err := json.Unmarshal([]byte(r.stdout), &doc); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range doc.Findings {
		l := f.Location
		got = append(got, strings.Join([]string{f.Code, f.Subject.Kind, f.Subject.Name, l.Origin, l.File, l.Pointer,
			itoa(l.Line), itoa(l.Column), param(f.Params, "rule"), param(f.Params, "kind")}, " "))
	}
	slices.Sort(got)
	want := []string{
		"output.declaration.validate_kind output table package metais.yaml /outputs/4/validate 28 15  csv",
		"output.parse output broken output broken.json  1 9 json.syntax ",
		"output.type output json output data.json /n 2 8  ",
		"output.type output yaml output data.yaml /n 2 4  ",
	}
	if !slices.Equal(got, want) {
		t.Errorf("findings:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// itoa is n's text, or "" for 0, a location member that is absent.
func itoa(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

func param(p map[string]any, name string) string {
	s, _ := p[name].(string)
	return s
}
