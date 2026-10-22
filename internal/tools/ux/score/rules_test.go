// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package score

import (
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"
)

// TestOwnRender: a zip tool or a zip library that reads an archive the
// session rendered extracts nothing of the package and is not counted,
// whether a command line or a program the session wrote reads it. A
// command that also names a .metais file, or a file the session started
// with, is still counted (W3-7). The probes are
// c6-opus-unzips-its-own-render, whose line-51 call reads out/acr.docx,
// which its render wrote, with that call changed, a script written before
// it, and for a fixture other than the package, one added to its record.
func TestOwnRender(t *testing.T) {
	f := fixtureNamed(t, "c6-opus-unzips-its-own-render")
	const line = 51
	command := func(c string) func([]string) []string {
		return func(l []string) []string {
			l[line] = withCommand(t, l[line], c)
			return l
		}
	}
	// script is the call at line 51 changed to run peek.py, which a Write
	// just before it writes with text.
	script := func(text string) func([]string) []string {
		return func(l []string) []string {
			use, _ := json.Marshal(map[string]any{"type": "assistant", "message": map[string]any{"id": "m-peek", "content": []any{
				map[string]any{"type": "tool_use", "id": "t-peek", "name": "Write", "input": map[string]any{"file_path": "/ux/run/work/peek.py", "content": text}}}}})
			res, _ := json.Marshal(map[string]any{"type": "user", "message": map[string]any{"content": []any{
				map[string]any{"type": "tool_result", "tool_use_id": "t-peek", "content": "File created successfully at: /ux/run/work/peek.py"}}}})
			l[line] = withCommand(t, l[line], "python3 peek.py | head -c 1200")
			return append(l[:line:line], append([]string{string(use), string(res)}, l[line:]...)...)
		}
	}
	for _, c := range []struct {
		name string
		edit func([]string) []string
		// fixture is a file the session started with, added to the
		// record's fixtures beside the package.
		fixture string
		// counted is the words no-zip-extraction's failure holds, or ""
		// for a read that is not counted.
		counted string
	}{
		{"as recorded", func(l []string) []string { return l }, "", ""},
		{"the package beside the render", command("unzip -p accessibility-conformance-report.metais out/acr.docx"), "", "unzip extracts the archive"},
		{"a .metais beside the render", command("unzip -p copy.metais out/acr.docx"), "", "unzip extracts the archive"},
		{"a fixture beside the render", command("unzip -p vpat.docx out/acr.docx"), "vpat.docx", "unzip extracts the archive"},
		{"a zip library on the command line", command(`python3 -I -c "import zipfile; print(zipfile.ZipFile('out/acr.docx').read('word/document.xml'))"`), "", ""},
		{"a zip library on the command line, and a fixture", command(`python3 -I -c "import zipfile; print(zipfile.ZipFile('out/acr.docx').read('word/document.xml'), open('vpat.docx').read())"`), "vpat.docx", "parses the archive with zipfile"},
		{"a script the session wrote", script("import zipfile\nprint(zipfile.ZipFile('out/acr.docx').read('word/document.xml'))\n"), "", ""},
		{"a script the session wrote, of the package", script("import zipfile\nprint(zipfile.ZipFile('accessibility-conformance-report.metais').read('metais.yaml'))\n"), "", "which the session wrote to parse the archive with zipfile"},
		{"a script the session wrote, of a .metais beside the render", script("import zipfile\nz = zipfile.ZipFile('out/acr.docx')\nprint(z.read('word/document.xml'), zipfile.ZipFile('copy.metais').read('metais.yaml'))\n"), "", "which the session wrote to parse the archive with zipfile"},
	} {
		in := probe(t, f, c.edit)
		if c.fixture != "" {
			m := in.Run.(fstest.MapFS)
			var rec map[string]any
			if err := json.Unmarshal(m["record.json"].Data, &rec); err != nil {
				t.Fatal(err)
			}
			rec["fixtures"] = append(rec["fixtures"].([]any), map[string]any{"path": c.fixture, "size": 1, "sha256": strings.Repeat("0", 64)})
			b, _ := json.Marshal(rec)
			m["record.json"] = &fstest.MapFile{Data: b}
		}
		o, err := Score(in)
		if err != nil {
			t.Fatal(err)
		}
		res := ruleResult(t, o, RuleNoZip)
		if res.Pass != (c.counted == "") || !strings.Contains(res.Detail, c.counted) {
			t.Errorf("%s: no-zip-extraction %+v, want counted for %q", c.name, res, c.counted)
		}
	}
}
