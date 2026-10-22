// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
)

// The suite's tripwires of ADR-011 D14 that read the vectors:
// TestVectorsWellFormed, TestNoSentinelInSuite and
// TestHostileVectorsOpenEverywhere, each with a control that turns it
// red. TestSuiteTreePortable needs ADR-005 D12's portable-path predicate
// and lands with it.

// statementIndex reads spec/statements.json beside suite, in the form
// internal/guarantees generates it: each statement's since, by
// identifier. A missing index is nil, no error.
func statementIndex(suite string) (map[string]string, error) {
	b, err := os.ReadFile(filepath.Join(suite, "..", "statements.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var index struct {
		Statements []struct {
			ID    string `json:"id"`
			Since string `json:"since"`
		} `json:"statements"`
	}
	if err := json.Unmarshal(b, &index); err != nil {
		return nil, fmt.Errorf("statements.json: %w", err)
	}
	idx := map[string]string{}
	for _, e := range index.Statements {
		if e.Since == "" {
			e.Since = "1.0"
		}
		idx[e.ID] = e.Since
	}
	return idx, nil
}

// vectorsWellFormed returns every way a vector of suite breaks ADR-011
// D9: vector.json fails its schema; the id differs from the path; the
// area is not the first statement's; since is not the statements'
// highest; a referenced file is missing or a file is unreferenced; a
// package.metais lacks byte-level; an expected findings file fails its
// schema; a processor vector runs an operation only an authoring tool
// performs.
func vectorsWellFormed(suite string) []string {
	dirs, err := vectorDirs(suite)
	if err != nil {
		return []string{err.Error()}
	}
	if len(dirs) == 0 {
		return nil
	}
	set, err := suiteSchemas(suite)
	if err != nil {
		return []string{err.Error()}
	}
	idx, err := statementIndex(suite)
	if err != nil {
		return []string{err.Error()}
	}
	var bad []string
	for _, d := range dirs {
		rel := vectorPath(suite, d)
		add := func(format string, args ...any) { bad = append(bad, rel+": "+fmt.Sprintf(format, args...)) }
		raw, err := os.ReadFile(filepath.Join(d, "vector.json"))
		if err != nil {
			add("%v", err)
			continue
		}
		if err := set.validator(vectorSchema)(raw); err != nil {
			add("vector.json fails its schema: %v", err)
			continue
		}
		v, err := readVector(d)
		if err != nil {
			add("%v", err)
			continue
		}
		if v.ID != rel {
			add("id %q is not its path", v.ID)
		}
		if area, _, _ := strings.Cut(v.ID, "/"); area != strings.ToLower(v.Statements[0][4:7]) {
			add("area %q is not its first statement's, %s", area, v.Statements[0])
		}
		switch {
		case idx == nil:
			add("spec/statements.json is missing, so since cannot be checked")
		default:
			highest := ""
			for _, s := range v.Statements {
				since, ok := idx[s]
				if !ok {
					add("cites %s, which spec/statements.json does not list", s)
					continue
				}
				if highest == "" || laterRevision(since, highest) {
					highest = since
				}
			}
			if highest != "" && v.Since != highest {
				add("since %s is not its statements' highest, %s", v.Since, highest)
			}
		}
		if op := authoringStep(v); v.Class == "processor" && op != "" {
			add("class processor, and a step runs %s, which only an authoring tool performs", op)
		}
		bad = append(bad, referenceProblems(rel, v, set)...)
	}
	return bad
}

// authoringStep is the first operation of v that only an authoring tool
// performs (chapter 15, the operations): check, test, pack, unpack,
// sample, and a render with sample. "" when v has none, so a processor
// claim can run it.
func authoringStep(v *vector) string {
	for _, c := range v.Cases {
		for _, st := range c.Steps {
			switch {
			case slices.Contains([]string{"check", "test", "pack", "unpack", "sample"}, st.Op):
				return st.Op
			case st.Op == "render" && st.args.Sample != nil:
				return "render with sample"
			}
		}
	}
	return ""
}

// referenceProblems checks that every file v names exists, but a path a
// case's setup lists as absent, and every file of its directory is named
// (ADR-011 D9 item 2, A4 item 1).
func referenceProblems(rel string, v *vector, set *schemaSet) []string {
	var bad []string
	add := func(format string, args ...any) { bad = append(bad, rel+": "+fmt.Sprintf(format, args...)) }
	referenced := map[string]bool{"vector.json": true}
	// name marks the file or tree p as referenced, and reports whether it
	// exists.
	name := func(p string) (exists bool) {
		root := filepath.Join(v.dir, filepath.FromSlash(p))
		fi, err := os.Lstat(root)
		if err != nil {
			return false
		}
		if !fi.IsDir() {
			referenced[path.Clean(p)] = true
			return true
		}
		filepath.WalkDir(root, func(q string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				r, _ := filepath.Rel(v.dir, q)
				referenced[filepath.ToSlash(r)] = true
			}
			return nil
		})
		return true
	}
	packed := false
	names := map[string]bool{}
	for _, c := range v.Cases {
		if names[c.Name] {
			add("case %q is not unique", c.Name)
		}
		names[c.Name] = true
		if c.Setup != nil && c.Setup.Copy != nil && *c.Setup.Copy != "" && !name(*c.Setup.Copy) {
			add("case %s: setup copies %s, which is missing", c.Name, *c.Setup.Copy)
		}
		var written []string
		for si, st := range c.Steps {
			for _, p := range []string{st.args.Package, st.args.Source, st.args.Data, st.args.InputsJSON} {
				if c.Setup != nil && slices.Contains(c.Setup.Absent, p) {
					continue // ADR-011 A4 item 1: it must not exist
				}
				if p == "" || name(p) {
					packed = packed || path.Base(p) == "package.metais"
					continue
				}
				if !slices.ContainsFunc(written, func(w string) bool { return p == w || strings.HasPrefix(p, w+"/") }) {
					add("case %s step %d: %s is missing, and no earlier step writes it", c.Name, si+1, p)
				}
			}
			files := []string{st.Expect.Findings}
			if st.Expect.Listing != nil {
				files = append(files, st.Expect.Listing.Equals)
			}
			for _, f := range files {
				if f != "" && !name(f) {
					add("case %s step %d: %s is missing", c.Name, si+1, f)
				}
			}
			if st.Expect.Findings != "" {
				if b, err := os.ReadFile(filepath.Join(v.dir, filepath.FromSlash(st.Expect.Findings))); err == nil {
					if err := set.validator(findingsSchema)(b); err != nil {
						add("case %s step %d: %s fails its schema: %v", c.Name, si+1, st.Expect.Findings, err)
					}
				}
			}
			if s := st.Expect.Stream; s != nil && !name(s.Equals) {
				add("case %s step %d: %s is missing", c.Name, si+1, s.Equals)
			}
			for _, te := range st.Expect.Trees {
				if te.Equals != "" && !name(te.Equals) {
					add("case %s step %d: %s is missing", c.Name, si+1, te.Equals)
				}
				written = append(written, te.Path)
			}
			written = append(written, st.args.Out, st.args.To)
		}
	}
	filepath.WalkDir(v.dir, func(q string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		r, _ := filepath.Rel(v.dir, q)
		r = filepath.ToSlash(r)
		if !referenced[r] {
			add("%s is named by nothing in vector.json", r)
		}
		return nil
	})
	if packed && !slices.Contains(v.Tags, "byte-level") {
		add("a package.metais is committed, and the vector is not tagged byte-level")
	}
	return bad
}

// TestVectorsWellFormed checks every vector of the suite (ADR-011 D14).
func TestVectorsWellFormed(t *testing.T) {
	for _, b := range vectorsWellFormed(defaultSuite()) {
		t.Error(b)
	}
}

// TestVectorsWellFormedControl runs the same checks over the fixture
// suite, which passes, and over one break of each rule, which each fail.
func TestVectorsWellFormedControl(t *testing.T) {
	suite := writeFixture(t, testenv.Dir(t))
	if bad := vectorsWellFormed(suite); len(bad) > 0 {
		t.Fatalf("the fixture suite is not well formed:\n%s", strings.Join(bad, "\n"))
	}
	edit := func(t *testing.T, id string, f func(map[string]any)) {
		t.Helper()
		p := filepath.Join(suite, "vectors", filepath.FromSlash(id), "vector.json")
		var m map[string]any
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatal(err)
		}
		f(m)
		if b, err = json.Marshal(m); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []struct {
		name    string
		want    string
		breakIt func(t *testing.T)
	}{
		{"schema", "fails its schema", func(t *testing.T) {
			edit(t, fixValidate, func(m map[string]any) { m["class"] = "viewer" })
		}},
		{"files outside a repeatable step", "fails its schema", func(t *testing.T) {
			edit(t, fixRenderTree, func(m map[string]any) {
				st := m["cases"].([]any)[0].(map[string]any)["steps"].([]any)[0].(map[string]any)
				st["expect"].(map[string]any)["trees"] = []any{map[string]any{"path": "out", "files": []string{}}}
			})
		}},
		{"id", "is not its path", func(t *testing.T) { edit(t, fixValidate, func(m map[string]any) { m["id"] = "fix/other" }) }},
		{"area", "is not its first statement's", func(t *testing.T) {
			edit(t, fixValidate, func(m map[string]any) { m["statements"] = []string{"MTS-OTH-001", "MTS-FIX-001"} })
		}},
		{"since", "is not its statements' highest", func(t *testing.T) {
			edit(t, fixValidate, func(m map[string]any) { m["since"] = "1.1" })
		}},
		{"unknown statement", "does not list", func(t *testing.T) {
			edit(t, fixValidate, func(m map[string]any) { m["statements"] = []string{"MTS-FIX-999"} })
		}},
		{"missing file", "expected/findings.json is missing", func(t *testing.T) {
			os.Remove(filepath.Join(suite, "vectors", "fix", "validate", "expected", "findings.json"))
		}},
		{"missing package", "package is missing", func(t *testing.T) {
			os.RemoveAll(filepath.Join(suite, "vectors", "fix", "validate", "package"))
		}},
		{"unreferenced file", "stray.txt is named by nothing", func(t *testing.T) {
			os.WriteFile(filepath.Join(suite, "vectors", "fix", "validate", "stray.txt"), []byte("x\n"), 0o644)
		}},
		{"findings schema", "fails its schema", func(t *testing.T) {
			os.WriteFile(filepath.Join(suite, "vectors", "fix", "validate", "expected", "findings.json"),
				[]byte(`[{"code": "input.fake_invalid", "message": "a message is the implementation's"}]`), 0o644)
		}},
		{"class", "which only an authoring tool performs", func(t *testing.T) {
			edit(t, fixValidate, func(m map[string]any) {
				st := m["cases"].([]any)[0].(map[string]any)["steps"].([]any)[0].(map[string]any)
				st["op"], st["args"] = "check", map[string]any{"source": "package"}
			})
		}},
		{"byte-level", "not tagged byte-level", func(t *testing.T) {
			dir := filepath.Join(suite, "vectors", "fix", "validate")
			os.WriteFile(filepath.Join(dir, "package.metais"), []byte("PK"), 0o644)
			edit(t, fixValidate, func(m map[string]any) {
				st := m["cases"].([]any)[0].(map[string]any)["steps"].([]any)[0].(map[string]any)
				st["args"].(map[string]any)["package"] = "package.metais"
			})
			os.RemoveAll(filepath.Join(dir, "package"))
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			suite = writeFixture(t, testenv.Dir(t))
			c.breakIt(t)
			bad := vectorsWellFormed(suite)
			if !slices.ContainsFunc(bad, func(b string) bool { return strings.Contains(b, c.want) }) {
				t.Errorf("a vector with a broken %s passed; reported %q", c.name, bad)
			}
		})
	}
}

// sentinelInTree returns every file below root whose name or content
// holds the sentinel token, case-insensitively.
func sentinelInTree(root string) []string {
	var bad []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == root && errors.Is(err, fs.ErrNotExist) {
				return filepath.SkipDir
			}
			return err
		}
		rel, _ := filepath.Rel(root, p)
		if sentinelAt([]byte(rel)) != "" {
			bad = append(bad, filepath.ToSlash(rel)+": its name holds the sentinel token")
		}
		if !d.Type().IsRegular() {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if bytes.Contains(bytes.ToUpper(b), []byte(sentinel)) {
			bad = append(bad, filepath.ToSlash(rel)+": holds the sentinel token")
		}
		return nil
	})
	if err != nil {
		bad = append(bad, err.Error())
	}
	return bad
}

// TestNoSentinelInSuite fails when any file under spec/conformance/ holds
// the sentinel token (ADR-011 D14): a suite file carrying it would make
// the scan's hit mean nothing.
func TestNoSentinelInSuite(t *testing.T) {
	for _, b := range sentinelInTree(defaultSuite()) {
		t.Error(b)
	}
}

// TestNoSentinelInSuiteControl plants the token in a copy of the fixture
// suite, in a file's content in mixed case and in a file's name, and each
// is caught; the fixture suite itself is clean.
func TestNoSentinelInSuiteControl(t *testing.T) {
	suite := writeFixture(t, testenv.Dir(t))
	if bad := sentinelInTree(suite); len(bad) > 0 {
		t.Fatalf("the fixture suite: %q", bad)
	}
	content := filepath.Join(suite, "vectors", "fix", "validate", "data", "first.json")
	if err := os.WriteFile(content, []byte(`{"x": "mtsSentinelCwd"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	named := filepath.Join(suite, "vectors", "fix", "validate", "data", strings.ToLower(sentinel)+".json")
	if err := os.WriteFile(named, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	bad := sentinelInTree(suite)
	for _, want := range []string{"first.json: holds the sentinel token", ".json: its name holds the sentinel token"} {
		if !slices.ContainsFunc(bad, func(b string) bool { return strings.Contains(b, want) }) {
			t.Errorf("a planted token was not caught (%s); reported %q", want, bad)
		}
	}
}

// hostileOpenProblems returns every security vector of suite whose open
// steps the cli adapter a would not run through every command the tool
// declares as opening a package (ADR-011 D16 item 3). The opening set is
// read from the tool, through help --json, only when such a vector
// exists.
func hostileOpenProblems(suite string, a *cliAdapter, base string) []string {
	dirs, err := vectorDirs(suite)
	if err != nil {
		return []string{err.Error()}
	}
	var bad []string
	for _, d := range dirs {
		v, err := readVector(d)
		if err != nil {
			bad = append(bad, err.Error())
			continue
		}
		if !slices.Contains(v.Tags, "security") {
			continue
		}
		for _, c := range v.Cases {
			for si, st := range c.Steps {
				if st.Op != "open" {
					continue
				}
				where := fmt.Sprintf("%s/%s step %d", v.ID, c.Name, si+1)
				set, err := a.opening(base)
				if err != nil {
					bad = append(bad, where+": the opening set: "+err.Error())
					continue
				}
				plan, err := openPlan(set, st.args.Package, (&sandbox{}).path)
				if err != nil {
					bad = append(bad, where+": "+err.Error())
					continue
				}
				if missing := planCovers(set, plan); len(missing) > 0 {
					bad = append(bad, fmt.Sprintf("%s: not run through %s", where, strings.Join(missing, ", ")))
				}
			}
		}
	}
	return bad
}

// TestHostileVectorsOpenEverywhere fails when a security vector's open
// step is not run through every command that opens a package, as the
// tool's own declarations say (ADR-011 D14, D16 item 3).
func TestHostileVectorsOpenEverywhere(t *testing.T) {
	a := &cliAdapter{path: builtTool}
	for _, b := range hostileOpenProblems(defaultSuite(), a, testenv.Dir(t)) {
		t.Error(b)
	}
	// The set is read from the binary built from source, whatever the
	// suite holds; internal/cli's TestOpeningSetAsTheRunnerReadsIt holds
	// that reading to the declarations.
	if _, err := a.opening(testenv.Dir(t)); err != nil {
		t.Errorf("the opening set of the built tool: %v", err)
	}
}

// TestHostileVectorsOpenEverywhereControl proves the tripwire against the
// fakes: the clean fake's declared commands are each run for the fixture
// suite's hostile open step, which the runner then runs through every one
// of them; a fake that declares a command the adapter cannot run, and a
// plan that leaves one out, are each caught.
func TestHostileVectorsOpenEverywhereControl(t *testing.T) {
	t.Parallel()
	suite := writeFixture(t, testenv.Dir(t))
	clean := &cliAdapter{path: fakeProgram(t, testenv.Dir(t), "clean")}
	if bad := hostileOpenProblems(suite, clean, testenv.Dir(t)); len(bad) > 0 {
		t.Errorf("the clean fake: %q", bad)
	}
	set, err := clean.opening(testenv.Dir(t))
	if err != nil {
		t.Fatal(err)
	}
	r := fixtureRunner(t, suite, "cli", "clean")
	v, err := readVector(filepath.Join(suite, "vectors", filepath.FromSlash(fixRefuse)))
	if err != nil {
		t.Fatal(err)
	}
	r.short = true
	res := r.runCase(v, &v.Cases[0], testenv.Dir(t))
	var ran []string
	for _, inv := range res.runs[envA].steps[0].invs {
		ran = append(ran, inv.label)
	}
	if !slices.Equal(ran, set) || len(res.problems) > 0 {
		t.Errorf("the hostile open step ran %q with problems %v; want every declared opening command, %q", ran, res.problems, set)
	}

	unmapped := &cliAdapter{path: fakeProgram(t, testenv.Dir(t), "help-unmapped")}
	if bad := hostileOpenProblems(suite, unmapped, testenv.Dir(t)); !slices.ContainsFunc(bad, func(b string) bool {
		return strings.Contains(b, `"peek"`)
	}) {
		t.Errorf("a declared opening command the adapter cannot run passed; reported %q", bad)
	}
	plan, err := openPlan(set, "package", (&sandbox{}).path)
	if err != nil {
		t.Fatal(err)
	}
	if missing := planCovers(set, plan[:len(plan)-1]); !slices.Equal(missing, set[len(set)-1:]) {
		t.Errorf("a plan without %s is reported missing %q", set[len(set)-1], missing)
	}
	if _, err := openPlan(nil, "package", (&sandbox{}).path); err == nil {
		t.Error("an empty opening set gives a plan")
	}
}

// blackBoxViolations returns every module package in deps, the test
// closure of this package, that the runner may not link: anything but
// the runner itself and internal/testenv (ADR-011 D11 item 2, R2-166).
func blackBoxViolations(deps []string) []string {
	const module = "metais.org/metais"
	allowed := []string{module + "/internal/conformance", module + "/internal/testenv"}
	var bad []string
	for _, d := range deps {
		pkg, _, _ := strings.Cut(d, " ")
		pkg = strings.TrimSuffix(pkg, ".test")
		if (pkg == module || strings.HasPrefix(pkg, module+"/")) && !slices.Contains(allowed, pkg) {
			bad = append(bad, fmt.Sprintf("the runner links %s; it drives the implementation as a black box (ADR-011 D11 item 2)", pkg))
		}
	}
	return bad
}

// TestBlackBox fails when the runner's test binary links any package of
// the implementation, internal/cli included.
func TestBlackBox(t *testing.T) {
	cmd, err := goCommand(testenv.Dir(t), "list", "-deps", "-test", "-f", "{{.ImportPath}}", ".")
	if err != nil {
		t.Fatal(err)
	}
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	deps := strings.Split(strings.TrimSpace(string(out)), "\n")
	if !slices.ContainsFunc(deps, func(d string) bool { return strings.HasSuffix(d, "/internal/testenv") }) {
		t.Fatalf("go list reported no internal/testenv in %q; the check has gone blind", deps)
	}
	for _, b := range blackBoxViolations(deps) {
		t.Error(b)
	}
	if bad := blackBoxViolations(append(deps, "metais.org/metais/internal/cli")); len(bad) != 1 {
		t.Errorf("a closure linking internal/cli: reported %q", bad)
	}
}
