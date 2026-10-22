// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/testenv"
)

// refusedBy is the limit the first finding of r names, with its code, or
// "" when r succeeded.
func refusedBy(r ran) (code, limit string) {
	if r.doc == nil || len(r.doc.Findings) == 0 {
		return "", ""
	}
	f := r.doc.Findings[0]
	l, _ := f.Params["limit"].(string)
	return f.Code, l
}

// TestOutputValues is ADR-009 Amendment A6 item 3, pinned false first: the
// values decoded from one validated output, JSON or YAML, its root
// included, are counted against output-values, per output and not summed,
// and an output of one value more is refused with limit.exceeded.output
// naming the limit.
func TestOutputValues(t *testing.T) {
	// Each output decodes to 6 values: the array and five items.
	pkg := transformPackage(t, testenv.Dir(t),
		"  - id: json\n    template: a.json.liquid\n    validate: references/s.json\n"+
			"  - id: yaml\n    template: a.yaml.liquid\n    validate: references/s.json\n",
		map[string]string{
			"references/s.json":       `{"type": "array"}`,
			"templates/a.json.liquid": "[1, 2, 3, 4, 5]\n",
			"templates/a.yaml.liquid": "[1, 2, 3, 4, 5]\n",
		})
	for _, c := range []struct {
		bound int
		want  string
	}{{6, ""}, {5, "limit.exceeded.output output-values"}} {
		t.Run(strconv.Itoa(c.bound), func(t *testing.T) {
			r := runTool(t, "render", pkg, "--set", "/x=a", "--out", filepath.Join(testenv.Dir(t), "out"), "--json",
				"--limit", "output-values="+strconv.Itoa(c.bound))
			var got []string
			for _, f := range r.doc.Findings {
				l, _ := f.Params["limit"].(string)
				got = append(got, f.Code+" "+l)
			}
			want := []string{}
			if c.want != "" {
				want = []string{c.want, c.want}
			}
			if strings.Join(got, "\n") != strings.Join(want, "\n") {
				t.Errorf("findings %q, want %q", got, want)
			}
		})
	}
}

// TestOutputYAMLSize is ADR-009 Amendment A6 item 3, pinned false first: a
// validated YAML output longer than yaml-size is refused with
// limit.exceeded.output naming yaml-size before it is parsed; one of
// exactly yaml-size is validated.
func TestOutputYAMLSize(t *testing.T) {
	doc := "[1, 2, 3]\n"
	pkg := transformPackage(t, testenv.Dir(t), "  - id: yaml\n    template: a.yaml.liquid\n    validate: references/s.json\n",
		map[string]string{"references/s.json": `{"type": "array"}`, "templates/a.yaml.liquid": doc})
	for _, c := range []struct {
		bound int
		want  string
	}{{len(doc), ""}, {len(doc) - 1, "yaml-size"}} {
		t.Run(strconv.Itoa(c.bound), func(t *testing.T) {
			r := runTool(t, "render", pkg, "--set", "/x=a", "--out", filepath.Join(testenv.Dir(t), "out"), "--json",
				"--limit", "yaml-size="+strconv.Itoa(c.bound))
			code, limit := refusedBy(r)
			if c.want == "" {
				if r.status != finding.ExitOK {
					t.Fatalf("status %d, codes %q", r.status, r.codes())
				}
				return
			}
			if code != "limit.exceeded.output" || limit != c.want {
				t.Errorf("%s naming %q, want limit.exceeded.output naming %s", code, limit, c.want)
			}
		})
	}
}

// TestExampleMetadataManifestSize is ADR-009 Amendment A6 item 2, pinned
// false first: example.yaml is bounded by manifest-size whichever form it
// is read in, so one in its JSON form longer than manifest-size is
// example.metadata, as its YAML form is, and is not read whole first.
func TestExampleMetadataManifestSize(t *testing.T) {
	pkg := writePackage(t, testenv.Dir(t), "  - id: a\n    template: a.txt.liquid\n", map[string]string{"a.txt.liquid": "a"})
	pad := strings.Repeat("x", 8<<10)
	for name, text := range map[string]string{
		"json": `{"title": "Big", "description": "` + pad + `"}`,
		"yaml": "title: Big\ndescription: " + pad + "\n",
	} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(pkg, "examples", "big")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "example.yaml"), []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
			r := runTool(t, "test", pkg, "--json", "--limit", "manifest-size=4KiB")
			var got []string
			for _, f := range r.doc.Findings {
				reason, _ := f.Params["reason"].(string)
				got = append(got, f.Code+" "+reason)
			}
			if len(got) != 1 || got[0] != "example.metadata syntax" {
				t.Errorf("findings %q, want example.metadata syntax", got)
			}
		})
	}
}
