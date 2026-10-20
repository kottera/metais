// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/testenv"
)

// vectorsDir is the conformance suite's vectors, from this package.
const vectorsDir = "../../spec/conformance/vectors"

// TestInspectSourceAndPacked is ADR-008 D16 item 4 for inspect (W3-1.5):
// a fixture's source directory and its committed package, packed from it,
// give the same result apart from source, the digest included (ADR-001 D6
// item 3).
func TestInspectSourceAndPacked(t *testing.T) {
	testenv.Home(t)
	for _, pair := range [][2]string{
		{"pkg/opens-source/package", "pkg/opens-packed/package.metais"},
		{"dat/combined-schema-source/package", "dat/combined-schema-packed/package.metais"},
		{"man/canonical-form-source/package", "man/canonical-form-packed/package.metais"},
	} {
		results := [2]map[string]any{}
		for i, p := range pair {
			r := runTool(t, "inspect", "--json", filepath.Join(vectorsDir, p))
			if !r.doc.OK {
				t.Fatalf("metais inspect %s: %s", p, r.stdout)
			}
			if err := json.Unmarshal(r.doc.Result, &results[i]); err != nil {
				t.Fatal(err)
			}
			if r.status != 0 {
				t.Errorf("metais inspect %s exited %d", p, r.status)
			}
		}
		src, packed := results[0]["package"].(map[string]any), results[1]["package"].(map[string]any)
		if src["source"] != "directory" || packed["source"] != "packed" {
			t.Errorf("%s: source %v and %v", pair[0], src["source"], packed["source"])
		}
		delete(src, "source")
		delete(packed, "source")
		a, _ := json.Marshal(results[0])
		b, _ := json.Marshal(results[1])
		if !bytes.Equal(a, b) {
			t.Errorf("inspect of %s and of %s differ apart from source:\n%s\n%s", pair[0], pair[1], a, b)
		}
	}
}

// TestInspectRefusalsTriggered is ADR-008 D16 item 2 for inspect: every
// refusal it declares is triggered by inspect over some package of the
// conformance suite, or by a line of lineCases, or by a path made here
// that the suite's vectors make in their setup. runTool holds every run
// to its declaration and to a next step that parses.
func TestInspectRefusalsTriggered(t *testing.T) {
	testenv.Home(t)
	seen := map[string]bool{}
	pkgs, _ := filepath.Glob(filepath.Join(vectorsDir, "*", "*", "package*"))
	if len(pkgs) == 0 {
		t.Fatal("the suite holds no package")
	}
	for _, p := range pkgs {
		args := []string{"inspect", "--json", p}
		if b, err := os.ReadFile(filepath.Join(filepath.Dir(p), "vector.json")); err == nil && bytes.Contains(b, []byte(`"limits"`)) {
			var v struct {
				Cases []struct {
					Steps []struct {
						Args struct {
							Limits map[string]json.Number `json:"limits"`
						} `json:"args"`
					} `json:"steps"`
				} `json:"cases"`
			}
			json.Unmarshal(b, &v)
			for name, val := range v.Cases[0].Steps[0].Args.Limits {
				args = append(args, "--limit", name+"="+val.String())
			}
		}
		r := runTool(t, args...)
		for _, c := range r.codes() {
			seen[c] = true
			if r.status != finding.ExitRefused {
				t.Errorf("metais %q refused with %s and exit %d", args, c, r.status)
			}
		}
	}
	// What git cannot commit: an empty directory and a link, made here.
	for _, mk := range []func(dir string) error{
		func(dir string) error { return os.Mkdir(filepath.Join(dir, "assets"), 0o755) },
		func(dir string) error {
			return os.Symlink("out.txt.liquid", filepath.Join(dir, "templates", "link.liquid"))
		},
	} {
		dir := guidePackage(t, []string{"guide"})
		if err := mk(dir); err != nil {
			t.Fatal(err)
		}
		for _, code := range runTool(t, "inspect", "--json", dir).codes() {
			seen[code] = true
		}
	}
	for _, c := range lineCases {
		if c.json {
			for _, code := range runTool(t, c.args...).codes() {
				seen[code] = true
			}
		}
	}
	// What the suite's vectors make in their setup (ADR-011 A4): a path
	// that names nothing, and a directory this process may not read.
	for _, code := range runTool(t, "inspect", "--json", filepath.Join(testenv.Dir(t), "absent")).codes() {
		seen[code] = true
	}
	sealed := testenv.Dir(t)
	if err := os.Chmod(sealed, 0o000); err != nil {
		t.Fatal(err)
	}
	for _, code := range runTool(t, "inspect", "--json", sealed).codes() {
		seen[code] = true
	}
	if err := os.Chmod(sealed, 0o755); err != nil {
		t.Fatal(err)
	}
	if os.Geteuid() == 0 || runtime.GOOS == "windows" {
		// A directory's mode refuses no one here; pkg/operand-not-readable
		// proves it where the host's modes refuse a read.
		seen["package.not_readable"] = true
	}
	for _, rf := range lookup(table, "inspect").Refusals {
		code := string(rf.Code)
		if !seen[code] && code != string(finding.CodeUsageBadValue) &&
			code != string(finding.CodeUsageRepeatedFlag) && code != string(finding.CodeUsageUnexpectedOperand) &&
			code != string(finding.CodeUsageUnknownFlag) && code != string(finding.CodeLimitInvalid) {
			t.Errorf("inspect declares %s, which nothing triggers", code)
		}
	}
	if r := runTool(t, "inspect", "--json", "--limit", "nope=1", filepath.Join(vectorsDir, "pkg/opens-source/package")); !slices.Equal(r.codes(), []string{string(finding.CodeLimitInvalid)}) || r.status != finding.ExitUsage {
		t.Errorf("a bad --limit: %s", r.stdout)
	}
	if r := runTool(t, "inspect", "--json", filepath.Join(testenv.Dir(t), "absent")); !slices.Equal(r.codes(), []string{"package.not_found"}) || r.status != finding.ExitRefused {
		t.Errorf("an absent package: %s", r.stdout)
	}
}

// TestInspectGuide is ADR-008 D6 items 2 and 9: a guide of at most 60
// lines and 6144 bytes is shown whole; a longer one is cut to its first 40
// lines, within 4096 bytes, with --guide named as the next step; and
// --guide prints the package and the whole guide alone.
func TestInspectGuide(t *testing.T) {
	cases := []struct {
		name         string
		guide        []string
		excerptLines int
		truncated    bool
	}{
		{"60 lines", lines(60, "line"), 60, false},
		{"61 lines", lines(61, "line"), 40, true},
		{"6144 bytes", lines(6, strings.Repeat("x", 1023)), 6, false},
		{"6145 bytes", append(lines(6, strings.Repeat("x", 1023)), "y"), 4, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := guidePackage(t, c.guide)
			r := runTool(t, "inspect", "--json", dir)
			var res inspectResult
			if err := json.Unmarshal(r.doc.Result, &res); err != nil {
				t.Fatalf("%v: %s", err, r.stdout)
			}
			g := res.Guide
			if g.TotalLines != len(c.guide) || g.ExcerptLines != c.excerptLines || g.Truncated != c.truncated {
				t.Errorf("guide: %d of %d lines, truncated %v", g.ExcerptLines, g.TotalLines, g.Truncated)
			}
			if c.truncated && (len(g.Text) > 4096 || !slices.ContainsFunc(r.doc.Next, func(n Next) bool { return n.Run == "metais inspect <package> --guide" })) {
				t.Errorf("a cut guide of %d bytes, next %v", len(g.Text), r.doc.Next)
			}
			whole := runTool(t, "inspect", "--json", "--guide", dir)
			var all map[string]json.RawMessage
			json.Unmarshal(whole.doc.Result, &all)
			if len(all) != 2 || all["package"] == nil || !strings.Contains(string(all["guide"]), `"truncated":false`) {
				t.Errorf("--guide: %s", whole.doc.Result)
			}
			if h := runTool(t, "inspect", "--guide", dir); h.stdout != strings.Join(c.guide, "\n")+"\n" {
				t.Errorf("--guide's human output is not the guide alone")
			}
		})
	}
}

func lines(n int, s string) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = s
	}
	return out
}

// guidePackage writes a source directory whose guide is guide's lines.
func guidePackage(t *testing.T, guide []string) string {
	dir := testenv.Dir(t)
	var g strings.Builder
	for _, l := range guide {
		g.WriteString("  " + l + "\n")
	}
	files := map[string]string{
		"metais.yaml": "metais: \"1.0\"\nname: guide\nversion: 1.0.0\ntitle: Guide\ndescription: A long guide.\nguide: |-\n" + g.String() +
			"inputs: []\noutputs:\n  - id: out\n    template: out.txt.liquid\n",
		"templates/out.txt.liquid": "out\n",
	}
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestInspectHuman is ADR-008 D6 item 10 and D11 item 3: the summary's
// layout, top to bottom, on standard output with its Next: block after it,
// and nothing on standard error.
func TestInspectHuman(t *testing.T) {
	testenv.Home(t)
	r := runTool(t, "inspect", filepath.Join(vectorsDir, "pkg/opens-source/package"))
	if r.status != 0 || r.stderr != "" {
		t.Fatalf("status %d, stderr %q", r.status, r.stderr)
	}
	if !strings.HasPrefix(r.stdout, "Status report — org.example.status-report 2.1.0\nA weekly status report") {
		t.Errorf("the summary does not begin with its title line:\n%s", r.stdout)
	}
	at := -1
	for _, s := range []string{"\nGuide\n", "\nInputs\n", "\nOutputs\n", "\nReferences\n", "\nAssets\n", "\nExamples\n"} {
		i := strings.Index(r.stdout, s)
		if i <= at {
			t.Errorf("section %q is missing or out of order", strings.TrimSpace(s))
		}
		at = i
	}
	for _, want := range []string{
		"METAIS 1.0, source directory, sha256:",
		"  project — Project (required; project.json, project.yaml, project.yml)",
		"  items — Items (optional; items/)",
		"  references/notes.txt (42 B)",
		"  references/standards/practice.md (71 B) — Reporting practice",
		"  basic: project (with expected outputs)",
	} {
		if !strings.Contains(r.stdout, want) {
			t.Errorf("the summary lacks %q:\n%s", want, r.stdout)
		}
	}
}

func TestHumanSize(t *testing.T) {
	for n, want := range map[int64]string{0: "0 B", 1023: "1023 B", 1024: "1.0 KiB", 1536: "1.5 KiB", 1 << 20: "1.0 MiB", 3 << 30: "3.0 GiB"} {
		if got := humanSize(n); got != want {
			t.Errorf("humanSize(%d) = %q, want %q", n, got, want)
		}
	}
}

// TestInspectOpensTheScaffold is W3-1's meeting with W3-8: the source
// directory metais init writes opens under inspect, with ok and no
// finding.
func TestInspectOpensTheScaffold(t *testing.T) {
	testenv.Home(t)
	dir := filepath.Join(testenv.Dir(t), "new-package")
	if r := runTool(t, "init", dir, "--json"); r.status != 0 {
		t.Fatalf("metais init: %s", r.stdout)
	}
	r := runTool(t, "inspect", "--json", dir)
	if r.status != 0 || !r.doc.OK || len(r.doc.Findings) != 0 {
		t.Errorf("metais inspect of init's scaffold: status %d, %s", r.status, r.stdout)
	}
}

// TestInspectExamplesInNameOrder is ADR-008 D6 items 7 and 8 (W7-C3):
// inspect lists a package's examples in name order, names compared by code
// point as ADR-004 compares strings, every run alike, and the Read next
// list's example step names the first of them. The package has seven examples, so a list built
// in map order differs from name order within a few runs.
func TestInspectExamplesInNameOrder(t *testing.T) {
	testenv.Home(t)
	p := filepath.Join(vectorsDir, "aut/test-containers/package")
	want := []string{"content", "equal", "method", "missing", "order", "unexpected", "unreadable"}
	for i := range 32 {
		r := runTool(t, "inspect", "--json", p)
		if !r.doc.OK {
			t.Fatalf("metais inspect %s: %s", p, r.stdout)
		}
		var res struct {
			Examples []struct{ Name string }
		}
		if err := json.Unmarshal(r.doc.Result, &res); err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, ex := range res.Examples {
			got = append(got, ex.Name)
		}
		if !slices.Equal(got, want) {
			t.Fatalf("run %d: examples %q, want %q", i, got, want)
		}
		step := "metais ls <package> examples/" + want[0] + "/ -r"
		if !slices.ContainsFunc(r.doc.Next, func(n Next) bool { return n.Run == step }) {
			t.Fatalf("run %d: next %v names no %q", i, r.doc.Next, step)
		}
	}
}

// TestInspectInputExamplesInNameOrder is ADR-008 D6 item 3 (W7-NB3): each
// input's entry lists the examples that supply it in name order, every run
// alike. The package's one input is supplied by eight examples, so a list
// built in map order differs from name order within a few runs.
func TestInspectInputExamplesInNameOrder(t *testing.T) {
	testenv.Home(t)
	p := filepath.Join(vectorsDir, "aut/test-example-yaml/package")
	want := []string{"json", "list", "member", "now", "plain", "syntax", "title", "type"}
	for i := range 32 {
		r := runTool(t, "inspect", "--json", p)
		if !r.doc.OK {
			t.Fatalf("metais inspect %s: %s", p, r.stdout)
		}
		var res struct {
			Inputs []struct {
				Name     string
				Examples []string
			}
		}
		if err := json.Unmarshal(r.doc.Result, &res); err != nil {
			t.Fatal(err)
		}
		if len(res.Inputs) != 1 || res.Inputs[0].Name != "report" {
			t.Fatalf("run %d: inputs %+v, want the one input report", i, res.Inputs)
		}
		if got := res.Inputs[0].Examples; !slices.Equal(got, want) {
			t.Fatalf("run %d: report's examples %q, want %q", i, got, want)
		}
	}
}
