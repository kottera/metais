// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"io/fs"
	"maps"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostenv"
	"github.com/kottera/metais/internal/testenv"
)

// copyTree copies the tree at src into a new directory and returns it.
func copyTree(t testing.TB, src string) string {
	t.Helper()
	dst := testenv.Dir(t)
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

// treeState is every file beneath dir, by slash-separated path, with its
// bytes, and every directory, with a nil value.
func treeState(t testing.TB, dir string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == dir {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			out[rel+"/"] = nil
			return nil
		}
		b, err := os.ReadFile(p)
		out[rel] = b
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// testRun reads metais.test/1's result from r.
func testRun(t *testing.T, r ran) testResult {
	t.Helper()
	var res testResult
	if err := jsonUnmarshal(r.doc.Result, &res); err != nil {
		t.Fatalf("result %s: %v", r.doc.Result, err)
	}
	return res
}

// TestTestResult is ADR-008 D9 test item 2: each example's status, with
// pass, fail and no-expected, and each difference's path and kind, a text
// difference's line among them.
func TestTestResult(t *testing.T) {
	r := runTool(t, "test", "--json", filepath.Join(vectorsDir, "aut/test-examples/package"))
	if r.status != finding.ExitFindings {
		t.Fatalf("status %d: %s", r.status, r.stdout)
	}
	res := testRun(t, r)
	var got []string
	for _, e := range res.Examples {
		got = append(got, e.Name+":"+e.Status)
	}
	if want := []string{"a:fail", "b:pass", "c:no-expected", "d:fail"}; !slices.Equal(got, want) {
		t.Errorf("examples %q, want %q", got, want)
	}
	if d := res.Examples[0].Differences; len(d) != 1 || d[0] != (testDifference{Path: "notes.md", Kind: "missing"}) || res.Update != nil {
		t.Errorf("a's differences %+v, update %+v", d, res.Update)
	}
	r = runTool(t, "test", "--json", filepath.Join(vectorsDir, "aut/test-compare/package"), "--example", "mixed")
	res = testRun(t, r)
	var kinds []string
	for _, d := range res.Examples[0].Differences {
		line := 0
		if d.Line != nil {
			line = *d.Line
		}
		kinds = append(kinds, d.Path+":"+d.Kind+":"+string(rune('0'+line)))
	}
	if want := []string{"report.md:differs:1", "site/M.md:unexpected:0", "site/old.md:missing:0", "zz/notes.txt:missing:0"}; !slices.Equal(kinds, want) {
		t.Errorf("differences %q, want %q", kinds, want)
	}
	if !slices.Equal(r.doc.Next, []Next{nextTestAgain, nextTestUpdate}) {
		t.Errorf("next %+v", r.doc.Next)
	}
}

// TestTestRefusals is ADR-008 D16 item 2 for test's own refusals: each is
// triggered, refuses the run with its status and next step, and writes
// nothing.
func TestTestRefusals(t *testing.T) {
	src := copyTree(t, filepath.Join(vectorsDir, "aut/test-compare/package"))
	packed := filepath.Join(vectorsDir, "aut/test-packed/package.metais")
	before := treeState(t, src)
	for _, c := range []struct {
		name   string
		args   []string
		code   finding.Code
		status int
		params map[string]any
	}{
		{"unknown example", []string{"test", src, "--example", "equal", "--example", "nope"}, finding.CodeUsageUnknownExample, finding.ExitUsage,
			map[string]any{"argument": float64(6), "examples": []any{"equal", "mixed"}}},
		{"update on a packed package", []string{"test", packed, "--update"}, finding.CodeUsageNotSourceDir, finding.ExitUsage,
			map[string]any{"argument": float64(2)}},
		{"unknown example, written with =", []string{"test", src, "--example=nope"}, finding.CodeUsageUnknownExample, finding.ExitUsage,
			map[string]any{"argument": float64(3)}},
		{"dry run without update", []string{"test", src, "--dry-run"}, finding.CodeUsageMissingFlag, finding.ExitUsage,
			map[string]any{"command": "test", "flag": "--update"}},
		{"standard input", []string{"test", "-"}, finding.CodeUsagePackageFromStdin, finding.ExitUsage, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := runTool(t, append(c.args, "--json")...)
			if r.status != c.status || !slices.Equal(r.codes(), []string{string(c.code)}) || string(r.doc.Result) != "null" {
				t.Fatalf("status %d, codes %q, result %s", r.status, r.codes(), r.doc.Result)
			}
			for k, v := range c.params {
				if got := r.doc.Findings[0].Params[k]; !equalJSON(got, v) {
					t.Errorf("%s = %v, want %v", k, got, v)
				}
			}
			if h := runTool(t, c.args...); h.stdout != "" || !strings.Contains(h.stderr, string(c.code)) {
				t.Errorf("human: stdout %q, stderr %q", h.stdout, h.stderr)
			}
		})
	}
	if after := treeState(t, src); !maps.EqualFunc(before, after, bytes.Equal) {
		t.Error("a refused test changed the source directory")
	}
}

func equalJSON(a, b any) bool {
	switch x := a.(type) {
	case []any:
		y, ok := b.([]any)
		return ok && slices.EqualFunc(x, y, equalJSON)
	}
	return a == b
}

// TestTestUpdate is ADR-007 D5 item 5 and ADR-008 D9 test item 3:
// --update --dry-run lists what --update writes and removes, and writes
// nothing; --update makes each example's expected/ exactly its render,
// removing the stale files and the directories they empty and nothing
// else, and a test after it passes; an example whose inputs are refused is
// not updated.
func TestTestUpdate(t *testing.T) {
	src := copyTree(t, filepath.Join(vectorsDir, "aut/test-compare/package"))
	if err := os.MkdirAll(filepath.Join(src, "examples/bad/inputs"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"examples/bad/inputs/report.json": `{"title": 1}` + "\n",
		"examples/bad/expected/stale.md":  "stale\n",
	} {
		p := filepath.Join(src, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	before := treeState(t, src)
	want := testUpdate{
		Written: []string{"examples/mixed/expected/report.md", "examples/mixed/expected/site/M.md"},
		Removed: []string{"examples/mixed/expected/site/old.md", "examples/mixed/expected/zz/notes.txt"},
	}

	r := runTool(t, "test", src, "--update", "--dry-run", "--json")
	res := testRun(t, r)
	if r.status != finding.ExitFindings || res.Update == nil || !res.Update.DryRun ||
		!slices.Equal(res.Update.Written, want.Written) || !slices.Equal(res.Update.Removed, want.Removed) {
		t.Fatalf("--dry-run: status %d, update %+v", r.status, res.Update)
	}
	if after := treeState(t, src); !maps.EqualFunc(before, after, bytes.Equal) {
		t.Fatal("--dry-run changed the source directory")
	}
	h := runTool(t, "test", src, "--update", "--dry-run")
	for _, line := range []string{"Would write examples/mixed/expected/report.md\n", "Would remove examples/mixed/expected/zz/notes.txt\n"} {
		if !strings.Contains(h.stdout, line) {
			t.Errorf("--dry-run's human report lacks %q:\n%s", line, h.stdout)
		}
	}

	h = runTool(t, "test", src, "--update")
	for _, line := range []string{"Wrote examples/mixed/expected/site/M.md\n", "Removed examples/mixed/expected/site/old.md\n"} {
		if !strings.Contains(h.stdout, line) {
			t.Errorf("--update's human report lacks %q:\n%s", line, h.stdout)
		}
	}
	// Only bad's inputs fail it: mixed was updated and passes.
	if h.status != finding.ExitFindings || !strings.Contains(h.stdout, "mixed: pass\n") || !strings.Contains(h.stdout, "bad: fail\n") {
		t.Errorf("--update: status %d:\n%s", h.status, h.stdout)
	}
	after := treeState(t, src)
	changed := map[string]bool{}
	for p := range maps.Keys(before) {
		if b, ok := after[p]; !ok || !bytes.Equal(b, before[p]) {
			changed[p] = true
		}
	}
	for p := range maps.Keys(after) {
		if _, ok := before[p]; !ok {
			changed[p] = true
		}
	}
	wantChanged := []string{
		"examples/mixed/expected/report.md", "examples/mixed/expected/site/M.md", "examples/mixed/expected/site/old.md",
		"examples/mixed/expected/zz/", "examples/mixed/expected/zz/notes.txt",
	}
	if got := slices.Sorted(maps.Keys(changed)); !slices.Equal(got, wantChanged) {
		t.Errorf("--update changed %q, want %q", got, wantChanged)
	}
	if got := string(after["examples/mixed/expected/report.md"]); got != "# M\n" {
		t.Errorf("report.md after --update: %q", got)
	}
	if r := runTool(t, "test", src, "--example", "mixed", "--example", "equal", "--json"); r.status != finding.ExitOK {
		t.Errorf("test after --update: status %d, findings %q", r.status, r.codes())
	}
}

// TestTestUpdateNewExpected is --update on an example with no expected/:
// it is created, holding the render, and a test after it passes.
func TestTestUpdateNewExpected(t *testing.T) {
	src := copyTree(t, filepath.Join(vectorsDir, "aut/test-examples/package"))
	r := runTool(t, "test", src, "--example", "c", "--update", "--json")
	if res := testRun(t, r); r.status != finding.ExitOK || !slices.Equal(res.Update.Written, []string{"examples/c/expected/report.md"}) {
		t.Fatalf("status %d, update %+v", r.status, res.Update)
	}
	if b, err := os.ReadFile(filepath.Join(src, "examples/c/expected/report.md")); err != nil || string(b) != "# C\n" {
		t.Fatalf("the new expected/: %q, %v", b, err)
	}
	if r := runTool(t, "test", src, "--example", "c", "--json"); r.status != finding.ExitOK || testRun(t, r).Examples[0].Status != "pass" {
		t.Errorf("test after --update: status %d", r.status)
	}
}

// TestTestUpdateWriteFailed is --update meeting a file where expected/
// should be a directory: it is refused as a destination that is not a
// directory, and nothing is removed.
func TestTestUpdateWriteFailed(t *testing.T) {
	src := copyTree(t, filepath.Join(vectorsDir, "aut/test-examples/package"))
	if err := os.WriteFile(filepath.Join(src, "examples/c/expected"), []byte("a file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := treeState(t, src)
	r := runTool(t, "test", src, "--example", "c", "--update", "--json")
	if r.status != finding.ExitDestination || !slices.Equal(r.codes(), []string{string(finding.CodeDestinationNotDirectory)}) {
		t.Fatalf("status %d, codes %q", r.status, r.codes())
	}
	if after := treeState(t, src); !maps.EqualFunc(before, after, bytes.Equal) {
		t.Error("a refused --update changed the source directory")
	}
}

// removedOutsideExpected is the update confinement detector (ADR-007 D5
// item 5, RULINGS R1-006): it runs test --update under faults on a source
// directory whose example's expected/ holds a stale file, and returns every
// file that existed before outside examples/<name>/expected/ and is gone
// after.
func removedOutsideExpected(t *testing.T, faults fault.Set) []string {
	t.Helper()
	src := copyTree(t, filepath.Join(vectorsDir, "aut/test-now/package"))
	// The stale file is alone in a directory of expected/, so its removal
	// empties that directory, which --update removes too (R-W5-11).
	stale := filepath.Join(src, "examples/with/expected/sub/stale.md")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := treeState(t, src)
	runIn(t, hostenv.Display{NoColor: true}, failingReader{t}, faults, "test", src, "--update", "--json")
	after := treeState(t, src)
	for _, p := range []string{"examples/with/expected/sub/stale.md", "examples/with/expected/sub/"} {
		if _, ok := after[p]; ok {
			t.Fatalf("--update left %s in expected/", p)
		}
	}
	if _, ok := after["examples/with/expected/"]; !ok {
		t.Fatal("--update removed expected/ itself")
	}
	var gone []string
	for p := range before {
		if _, ok := after[p]; !ok && !strings.HasPrefix(p, "examples/with/expected/") {
			gone = append(gone, p)
		}
	}
	slices.Sort(gone)
	return gone
}

// TestUpdateConfinement is R1-006's safety property: test --update removes
// nothing outside examples/<name>/expected/. Its control, in the same run,
// is the fault at internal/cli's seam, UpdateBeyondExpected, under which
// --update reckons its removals over the whole example; the detector must
// see the example's own files go, and the directory it empties outside
// expected/, so a detector gone blind fails here. Directories are watched
// as files are: removedOutsideExpected lists each with a trailing "/".
func TestUpdateConfinement(t *testing.T) {
	testenv.Home(t)
	if gone := removedOutsideExpected(t, 0); len(gone) > 0 {
		t.Errorf("--update removed outside expected/: %q", gone)
	}
	t.Run("UpdateBeyondExpected", func(t *testing.T) {
		gone := removedOutsideExpected(t, fault.UpdateBeyondExpected)
		if !slices.Contains(gone, "examples/with/example.yaml") || !slices.Contains(gone, "examples/with/inputs/report.json") ||
			!slices.Contains(gone, "examples/with/inputs/") {
			t.Errorf("UpdateBeyondExpected went undetected: %q", gone)
		}
	})
}

// TestTestScaffold is ADR-007 D1's golden path: the scaffold init writes
// checks with no finding, and test passes it.
func TestTestScaffold(t *testing.T) {
	dir := filepath.Join(testenv.Dir(t), "report")
	for _, argv := range [][]string{{"init", dir}, {"check", dir}, {"test", dir}} {
		r := runTool(t, append(argv, "--json")...)
		if r.status != finding.ExitOK || len(r.doc.Findings) != 0 {
			t.Fatalf("metais %s: status %d, findings %q", argv[0], r.status, r.codes())
		}
		if argv[0] == "test" {
			if res := testRun(t, r); len(res.Examples) != 1 || res.Examples[0].Status != "pass" {
				t.Errorf("test of the scaffold: %+v", res.Examples)
			}
		}
	}
}

// TestTestDiffCaps is ADR-007 D5 item 4's caps: a text difference's diff
// is at most 16 KiB in JSON and 40 lines in human output, each marked
// when truncated.
func TestTestDiffCaps(t *testing.T) {
	src := copyTree(t, filepath.Join(vectorsDir, "aut/test-now/package"))
	var long strings.Builder
	for i := range 4000 {
		long.WriteString(strings.Repeat("x", i%50) + "\n")
	}
	if err := os.WriteFile(filepath.Join(src, "examples/with/expected/report.md"), []byte(long.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	r := runTool(t, "test", src, "--example", "with", "--json")
	p := r.doc.Findings[0].Params
	diff, _ := p["diff"].(string)
	if len(diff) > maxDiffJSON || p["diff_truncated"] != true || !strings.HasSuffix(diff, diffTruncatedNotice) {
		t.Errorf("the JSON diff: %d bytes, truncated %v", len(diff), p["diff_truncated"])
	}
	h := runTool(t, "test", src, "--example", "with")
	i := strings.Index(h.stdout, "    --- expected/report.md")
	if i < 0 {
		t.Fatalf("no diff in:\n%s", h.stdout)
	}
	shown := strings.SplitAfter(h.stdout[i:], "\n")
	if len(shown) < maxDiffHuman+1 || shown[maxDiffHuman] != "    "+diffTruncatedNotice {
		t.Errorf("the human diff is not cut at %d lines with the notice:\n%s", maxDiffHuman, strings.Join(shown[:min(len(shown), maxDiffHuman+2)], ""))
	}
}

// TestEditScript holds the diff's edit script to what it must be: the
// kept and removed lines are the expected file, the kept and added lines
// the rendered one, for random edits and for edits beyond the search's
// bound.
func TestEditScript(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for n := range 200 {
		var a []string
		for range rng.IntN(40) {
			a = append(a, string(rune('a'+rng.IntN(4)))+"\n")
		}
		b := slices.Clone(a)
		edits := rng.IntN(10)
		if n%50 == 0 {
			edits = 3 * maxEditDistance
		}
		for range edits {
			switch i := rng.IntN(len(b) + 1); rng.IntN(2) {
			case 0:
				b = slices.Insert(b, i, string(rune('a'+rng.IntN(4)))+"\n")
			default:
				if i < len(b) {
					b = slices.Delete(b, i, i+1)
				}
			}
		}
		var gotA, gotB []string
		for _, e := range editScript(a, b) {
			if e.op != '+' {
				gotA = append(gotA, e.line)
			}
			if e.op != '-' {
				gotB = append(gotB, e.line)
			}
		}
		if !slices.Equal(gotA, a) || !slices.Equal(gotB, b) {
			t.Fatalf("case %d: the script does not rebuild its files", n)
		}
	}
	d, cut := unifiedDiff([]byte("a\nb\nc\n"), []byte("a\nB\nc"), "x", "y", maxDiffJSON)
	want := "--- x\n+++ y\n@@ -1,3 +1,3 @@\n a\n-b\n-c\n+B\n+c\n\\ No newline at end of file\n"
	if d != want || cut {
		t.Errorf("unifiedDiff = %q, want %q", d, want)
	}
}
