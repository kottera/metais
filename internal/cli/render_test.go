// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/author"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/testenv"
)

// renderPackage writes init's scaffold below root and returns its path and
// its example's data directory.
func renderPackage(t *testing.T, root string) (pkg, data string) {
	t.Helper()
	pkg = filepath.Join(root, "pkg")
	if _, err := author.Init(pkg, "pkg", false); err != nil {
		t.Fatal(err)
	}
	return pkg, filepath.Join(pkg, "examples", "basic", "inputs")
}

// readRender reads the result of a metais.render/1 document.
func readRender(t *testing.T, r ran) renderResult {
	t.Helper()
	var res renderResult
	if r.doc == nil || jsonUnmarshal(r.doc.Result, &res) != nil {
		t.Fatalf("metais %q printed no render result: %q", r.args, r.stdout)
	}
	return res
}

// TestRender is render's success, in JSON and in human mode: the scaffold
// renders its example's expected output into a new directory, created
// with its parents, and nothing else is left beside it.
func TestRender(t *testing.T) {
	root := testenv.Dir(t)
	pkg, data := renderPackage(t, root)
	out := filepath.Join(root, "a", "b", "out")
	r := runTool(t, "render", pkg, "--data", data, "--out", out, "--json")
	if r.status != 0 || !r.doc.OK || len(r.doc.Findings) != 0 {
		t.Fatalf("metais render --json: status %d, %s", r.status, r.stdout)
	}
	res := readRender(t, r)
	if res.Out != "directory" || res.DryRun || res.Only != nil || res.Sample != nil || len(res.Files) != 1 {
		t.Fatalf("result %+v", res)
	}
	f := res.Files[0]
	if f.Path != "greeting.md" || f.Output != "greeting" || f.Source != "template" || f.Kind != "markdown" || f.Status != "create" || f.Executable {
		t.Errorf("file %+v", f)
	}
	got, err := os.ReadFile(filepath.Join(out, "greeting.md"))
	want, _ := os.ReadFile(filepath.Join(pkg, "examples", "basic", "expected", "greeting.md"))
	if err != nil || string(got) != string(want) || int64(len(got)) != f.Size {
		t.Errorf("greeting.md is %q, %v; want %q", got, err, want)
	}
	entries, _ := os.ReadDir(filepath.Join(root, "a", "b"))
	if len(entries) != 1 || entries[0].Name() != "out" {
		t.Errorf("beside out: %v", entries)
	}

	out2 := filepath.Join(root, "out2")
	r = runTool(t, "render", pkg, "--data", data, "--out", out2)
	if r.status != 0 || r.stderr != "" || r.stdout != "32\t"+f.Digest+"\tgreeting.md\n" {
		t.Errorf("metais render: status %d, stdout %q, stderr %q", r.status, r.stdout, r.stderr)
	}
}

// TestRenderRefusals is each refusal of render's own line and
// destination, each writing nothing.
func TestRenderRefusals(t *testing.T) {
	root := testenv.Dir(t)
	pkg, data := renderPackage(t, root)
	file := filepath.Join(root, "file")
	full := filepath.Join(root, "full")
	empty := filepath.Join(root, "empty")
	for _, d := range []string{full, empty} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{file, filepath.Join(full, "x")} {
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []struct {
		args   []string
		status int
		code   string
	}{
		{[]string{"render", pkg, "--json"}, 3, "usage.missing_flag"},
		{[]string{"render", "-", "--out", "o", "--json"}, 3, "usage.package_from_stdin"},
		{[]string{"render", pkg, "--out", "o", "--now", "2026-10-19", "--json"}, 3, "usage.bad_value"},
		{[]string{"render", pkg, "--out", "o", "--data", filepath.Join(root, "absent"), "--json"}, 3, "usage.data_dir_unreadable"},
		{[]string{"render", pkg, "--data", data, "--out", file, "--json"}, 5, "output.destination.not_directory"},
		{[]string{"render", pkg, "--data", data, "--out", filepath.Join(file, "o"), "--json"}, 5, "output.destination.not_directory"},
		{[]string{"render", pkg, "--data", data, "--out", full, "--json"}, 5, "output.destination.not_empty"},
		{[]string{"render", pkg, "--data", data, "--out", empty, "--json"}, 5, "output.destination.write_failed"},
		{[]string{"render", pkg, "--out", filepath.Join(root, "o"), "--json"}, 1, "input.missing"},
	} {
		before := snapshotNames(t, root)
		r := runTool(t, c.args...)
		if r.status != c.status || !slices.Equal(r.codes(), []string{c.code}) {
			t.Errorf("metais %q: status %d, codes %q; want %d, %s", c.args, r.status, r.codes(), c.status, c.code)
		}
		if after := snapshotNames(t, root); !slices.Equal(before, after) {
			t.Errorf("metais %q changed the tree: %q, then %q", c.args, before, after)
		}
	}
}

// snapshotNames is every path below root, sorted.
func snapshotNames(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(out)
	return out
}

// TestRenderNowArgument is usage.bad_value's argument for --now, given as
// --now value and as --now=value: the position of the value refused.
func TestRenderNowArgument(t *testing.T) {
	for _, args := range [][]string{
		{"render", "p", "--out", "o", "--now", "x", "--json"},
		{"render", "p", "--out", "o", "--now=x", "--json"},
	} {
		r := runTool(t, args...)
		want := float64(slices.IndexFunc(args, func(a string) bool { return strings.HasSuffix(a, "x") }) + 1)
		if len(r.doc.Findings) != 1 || r.doc.Findings[0].Params["argument"] != want {
			t.Errorf("metais %q: %+v, want argument %v", args, r.doc.Findings, want)
		}
	}
}

// TestRenderLeavesWhatIsNotItsOwn is the second of the Wave 4 brief's
// hostfs constraints for render (W4-3.10): whatever a failed render
// removes is its own. A staging directory left beside the output
// directory, with a file in it, is not the render's: the render that meets
// it fails, and leaves it, its file and everything else as they were, and
// creates no output directory.
func TestRenderLeavesWhatIsNotItsOwn(t *testing.T) {
	root := testenv.Dir(t)
	pkg, data := renderPackage(t, root)
	parent := filepath.Join(root, "parent")
	staging := filepath.Join(parent, hostfs.StagingName("out"))
	if err := os.MkdirAll(staging, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "theirs.txt"), []byte("not the render's\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := snapshotNames(t, root)
	r := runTool(t, "render", pkg, "--data", data, "--out", filepath.Join(parent, "out"), "--json")
	if r.status != 5 || !slices.Equal(r.codes(), []string{"output.destination.write_failed"}) {
		t.Fatalf("status %d, codes %q", r.status, r.codes())
	}
	if after := snapshotNames(t, root); !slices.Equal(before, after) {
		t.Errorf("the failed render changed the tree: %q, then %q", before, after)
	}
	if b, err := os.ReadFile(filepath.Join(staging, "theirs.txt")); err != nil || string(b) != "not the render's\n" {
		t.Errorf("the staging directory's file is %q, %v", b, err)
	}
}

// A render whose write fails part way, after the staging directory exists
// and a file is written beneath it, aborts: neither the staging directory,
// nor the output directory, nor an ancestor the render created is left
// (ADR-005 D14 item 1.5). The second planned path runs through the first,
// a file, which no plan that passed the path checks holds, so the create
// fails beneath the staging directory.
func TestRenderCommitAbortsAFailedWrite(t *testing.T) {
	testenv.Home(t)
	parent := testenv.Dir(t)
	plan := []*planned{
		{path: "a.txt", bytes: []byte("A")},
		{path: "a.txt/b.txt", bytes: []byte("B")},
	}
	f, ok := commit(filepath.Join(parent, "x", "out"), plan, &session{})
	if ok {
		t.Fatal("the commit succeeded")
	}
	if f.Code() != finding.CodeDestinationWriteFailed {
		t.Errorf("finding %s, want %s", f.Code(), finding.CodeDestinationWriteFailed)
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	var left []string
	for _, e := range entries {
		left = append(left, e.Name())
	}
	if len(left) != 0 {
		t.Errorf("the failed render left %q", left)
	}
}
