// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package scenario

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

// TestTheSet holds the committed set to plan §5's amended table: its nine
// scenarios, each prompt's declaration under the Wave 1 brief's ruling on
// R-UX-8, and the fixtures each starts from.
func TestTheSet(t *testing.T) {
	set, err := Load(os.DirFS(filepath.Join("..", "scenarios")))
	if err != nil {
		t.Fatal(err)
	}
	type row struct {
		name   string
		kind   Kind
		output bool
		as     []string
	}
	report := []string{"report.metais"}
	want := []row{
		{"A1-new-package", Authoring, false, nil},
		{"A2-edit-package", Authoring, false, []string{"invoice.metais"}},
		{"A3-data-to-js", Authoring, false, nil},
		{"C0-fill-ambiguous", Consuming, false, report},
		{"C1-render-real", Consuming, true, report},
		{"C2-render-new-fictional", Consuming, true, report},
		{"C3-preview", Consuming, true, report},
		{"C4-what-it-needs", Consuming, false, report},
		{"C5-script", Consuming, true, report},
	}
	var got []row
	for _, s := range set {
		var as []string
		for _, f := range s.Fixtures {
			as = append(as, f.As)
			if f.Example != "" {
				if _, err := os.Stat(filepath.Join("..", "..", "..", "..", "examples", f.Example, "metais.yaml")); err != nil {
					t.Errorf("%s: example %s: %v", s.Name(), f.Example, err)
				}
			}
		}
		got = append(got, row{s.Name(), s.Kind, s.AsksForOutput, as})
		for _, f := range s.Fixtures {
			if !strings.Contains(s.Prompt, f.As) {
				t.Errorf("%s: the prompt does not name its fixture %s", s.Name(), f.As)
			}
		}
	}
	if !slices.EqualFunc(got, want, func(a, b row) bool {
		return a.name == b.name && a.kind == b.kind && a.output == b.output && slices.Equal(a.as, b.as)
	}) {
		t.Errorf("the set is\n\t%v\nwant\n\t%v", got, want)
	}
}

func TestLoadRefuses(t *testing.T) {
	good := `{"id": "C1", "slug": "render-real", "kind": "consuming", "asks_for_output": true, "fixtures": [{"example": "dora-incident-report", "as": "report.metais"}]}`
	set := func(dir, decl, prompt string, extra ...string) fstest.MapFS {
		m := fstest.MapFS{dir + "/scenario.json": {Data: []byte(decl)}}
		if prompt != "" {
			m[dir+"/prompt.txt"] = &fstest.MapFile{Data: []byte(prompt)}
		}
		for _, e := range extra {
			m[e] = &fstest.MapFile{Data: []byte("x")}
		}
		return m
	}
	if _, err := Load(set("C1-render-real", good, "render this file: report.metais\n")); err != nil {
		t.Fatalf("a good scenario: %v", err)
	}
	for _, c := range []struct {
		name string
		fsys fstest.MapFS
		want string
	}{
		{"no prompt", set("C1-render-real", good, ""), "prompt.txt"},
		{"an empty prompt", set("C1-render-real", good, "\n"), "prompt.txt is empty"},
		{"an unknown field", set("C1-render-real", strings.Replace(good, `"kind"`, `"outcome": 1, "kind"`, 1), "p"), "unknown field"},
		{"a misnamed directory", set("C1-render", good, "p"), "not named C1-render-real"},
		{"a bad id", set("X1-render-real", strings.Replace(good, `"C1"`, `"X1"`, 1), "p"), "not C<n> or A<n>"},
		{"a kind against its id", set("C1-render-real", strings.Replace(good, "consuming", "authoring", 1), "p"), "does not match id"},
		{"an unknown kind", set("C1-render-real", strings.Replace(good, "consuming", "reading", 1), "p"), "neither"},
		{"no fixtures field", set("C1-render-real", strings.Replace(good, `"fixtures": [{"example": "dora-incident-report", "as": "report.metais"}]`, `"fixtures": null`, 1), "p"), "fixtures is missing"},
		{"a fixture naming both", set("C1-render-real", strings.Replace(good, `"as"`, `"source": "x", "as"`, 1), "p"), "neither or both"},
		{"a fixture naming neither", set("C1-render-real", strings.Replace(good, `"example": "dora-incident-report", `, "", 1), "p"), "neither or both"},
		{"a source with no package", set("C1-render-real", strings.Replace(good, `"example"`, `"source"`, 1), "p"), "holds no metais.yaml"},
		{"a path for an example", set("C1-render-real", strings.Replace(good, "dora-incident-report", "../dora", 1), "p"), "not a directory name"},
		{"a fixture's file name", set("C1-render-real", strings.Replace(good, "report.metais", "../report.metais", 1), "p"), "not a file name ending .metais"},
		{"a file at the root", set("C1-render-real", good, "p", "README.md"), "not a scenario directory"},
		{"two scenarios with one id", func() fstest.MapFS {
			m := set("C1-render-real", good, "p")
			m["C1-other/scenario.json"] = &fstest.MapFile{Data: []byte(strings.Replace(good, "render-real", "other", 1))}
			m["C1-other/prompt.txt"] = &fstest.MapFile{Data: []byte("p")}
			return m
		}(), "a second scenario C1"},
		{"an empty set", fstest.MapFS{}, "no scenario"},
		{"a rewording with no transcript", set("C1-render-real", strings.Replace(good, `"fixtures"`, `"rewordings": [{"date": "2026-10-22", "was": "render it", "ambiguity": "which file"}], "fixtures"`, 1), "p"), "rewording 0 needs"},
		{"a rewording with no date", set("C1-render-real", strings.Replace(good, `"fixtures"`, `"rewordings": [{"date": "soon", "was": "render it", "ambiguity": "which file", "transcript": "t"}], "fixtures"`, 1), "p"), "rewording 0 needs"},
		{"a rewording to itself", set("C1-render-real", strings.Replace(good, `"fixtures"`, `"rewordings": [{"date": "2026-10-22", "was": "p", "ambiguity": "which file", "transcript": "t"}], "fixtures"`, 1), "p"), "the prompt it was is the prompt it is"},
	} {
		if _, err := Load(c.fsys); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: Load = %v, want an error containing %q", c.name, err, c.want)
		}
	}
	// A rewording that says what changed and why loads.
	rw := strings.Replace(good, `"fixtures"`, `"rewordings": [{"date": "2026-10-22", "was": "render it", "ambiguity": "which file", "transcript": "scorecard-2026-10-22-baseline, C1 haiku"}], "fixtures"`, 1)
	if s, err := Load(set("C1-render-real", rw, "p")); err != nil || len(s[0].Rewordings) != 1 {
		t.Errorf("a rewording: %v", err)
	}
	// A source with its package loads.
	src := set("A2-edit-package", `{"id": "A2", "slug": "edit-package", "kind": "authoring", "asks_for_output": false, "fixtures": [{"source": "invoice", "as": "invoice.metais"}]}`, "p", "A2-edit-package/invoice/metais.yaml")
	if _, err := Load(src); err != nil {
		t.Errorf("a source fixture: %v", err)
	}
}
