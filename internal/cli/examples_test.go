// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// examplePackagesDir is where the example packages live: one source
// directory per package, at the repository's root (plan §13).
const examplePackagesDir = "../../examples"

// sampleModes are the --optional modes every example previews in: every
// optional field filled, and none (PRD SC4).
var sampleModes = []string{"all", "none"}

// TestExamplePackages is SC4's harness (plan §12): every package under
// examples/ checks and tests green, packs into a package that checks and
// tests green again, and previews from sample data with every optional
// field filled and with none, into output that is what render reports
// and what the same data, kept by sample and validated, renders. With no
// package there, it reports that it ran over none.
func TestExamplePackages(t *testing.T) {
	testenv.Home(t)
	dirs, err := exampleDirs(examplePackagesDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) == 0 {
		t.Skipf("%s holds no example package, so the harness ran over none", examplePackagesDir)
	}
	for _, dir := range dirs {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			for _, p := range exampleHarness(t, dir) {
				t.Error(p)
			}
		})
	}
}

// exampleDirs are the directories directly under root, in name order: a
// file there is not a package, and a missing root holds none.
func exampleDirs(root string) ([]string, error) {
	ents, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, e := range ents {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(root, e.Name()))
		}
	}
	return dirs, nil
}

// exampleHarness runs SC4's steps over the source directory src and
// returns every problem, each naming its step:
//
//   - check reports no error and no warning;
//   - test passes every example, and there is at least one;
//   - each example's inputs render each file of its expected/ with that
//     file's executable bit, which test does not compare;
//   - pack writes the package it reports, which checks and tests green;
//   - for each of sampleModes, render --sample writes exactly the files
//     its result lists, with their sizes, digests and modes; sample writes
//     data in the same mode that validate accepts; and render of that
//     data writes the same bytes as the preview;
//   - and nothing is written into src.
func exampleHarness(t *testing.T, src string) []string {
	t.Helper()
	var ps []string
	add := func(step, format string, args ...any) {
		ps = append(ps, step+": "+fmt.Sprintf(format, args...))
	}
	before := treeState(t, src)
	work := testenv.Dir(t)

	// green runs argv and reports a run that is not ok, or whose findings
	// hold an error or a warning.
	green := func(step string, argv ...string) (ran, bool) {
		r := runTool(t, append(argv, "--json")...)
		if r.doc == nil || !r.doc.OK || r.status != 0 {
			add(step, "status %d, findings %q", r.status, r.codes())
			return r, false
		}
		var sum struct{ Errors, Warnings int }
		if err := json.Unmarshal(r.doc.Summary, &sum); err != nil {
			add(step, "the summary: %v", err)
			return r, false
		}
		if sum.Errors+sum.Warnings > 0 {
			add(step, "%d errors and %d warnings: %q", sum.Errors, sum.Warnings, r.codes())
		}
		return r, true
	}
	passes := func(step string, pkg string) {
		r, ok := green(step, "test", pkg)
		if !ok {
			return
		}
		var res testResult
		if err := json.Unmarshal(r.doc.Result, &res); err != nil {
			add(step, "the result: %v", err)
			return
		}
		if len(res.Examples) == 0 {
			add(step, "the package has no example to test")
		}
		for _, e := range res.Examples {
			if e.Status != "pass" {
				add(step, "example %s: %s, %+v", e.Name, e.Status, e.Differences)
			}
		}
	}

	green("check", "check", src)
	passes("test", src)
	expectedModes(t, src, add)

	packed := filepath.Join(work, "packed.metais")
	if r, ok := green("pack", "pack", src, "--out", packed); ok {
		var res packResult
		b, err := os.ReadFile(packed)
		switch {
		case json.Unmarshal(r.doc.Result, &res) != nil:
			add("pack", "the result: %s", r.doc.Result)
		case err != nil:
			add("pack", "the package it reports: %v", err)
		case int64(len(b)) != res.Size:
			add("pack", "wrote %d bytes and reports %d", len(b), res.Size)
		default:
			green("check of the packed package", "check", packed)
			passes("test of the packed package", packed)
		}
	}

	for _, mode := range sampleModes {
		preview := filepath.Join(work, "preview-"+mode)
		r, ok := green("render --sample --optional "+mode, "render", src, "--sample", "--optional", mode, "--out", preview)
		if !ok {
			continue
		}
		var res renderResult
		if err := json.Unmarshal(r.doc.Result, &res); err != nil {
			add("render --sample --optional "+mode, "the result: %v", err)
			continue
		}
		for _, d := range writtenAsListed(preview, res.Files) {
			add("render --sample --optional "+mode, "%s", d)
		}
		data := filepath.Join(work, "data-"+mode)
		if _, ok := green("sample --optional "+mode, "sample", src, "--optional", mode, "--to", data); !ok {
			continue
		}
		if _, ok := green("validate of sample --optional "+mode, "validate", src, "--data", data); !ok {
			continue
		}
		rendered := filepath.Join(work, "rendered-"+mode)
		if _, ok := green("render of sample --optional "+mode, "render", src, "--data", data, "--out", rendered); !ok {
			continue
		}
		if a, b := treeState(t, preview), treeState(t, rendered); !maps.EqualFunc(a, b, bytes.Equal) {
			add("render of sample --optional "+mode, "writes other files than the preview from the same data: %q and %q", sortedKeys(b), sortedKeys(a))
		}
	}

	if after := treeState(t, src); !maps.EqualFunc(before, after, bytes.Equal) {
		add("the source directory", "the harness's commands changed it: %q became %q", sortedKeys(before), sortedKeys(after))
	}
	return ps
}

// expectedModes renders the inputs of each example of the source
// directory src that holds expected/, and reports, through add, each file
// render lists whose executable bit is not that of the file of the same
// path in expected/. Test compares an example's files by their bytes
// alone (MTS-AUT-040), so the modes its expected/ carries are held here.
// A checkout on Windows keeps no mode to compare.
func expectedModes(t *testing.T, src string, add func(step, format string, args ...any)) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	examples, err := exampleDirs(filepath.Join(src, "examples"))
	if err != nil {
		add("expected modes", "%v", err)
		return
	}
	for _, ex := range examples {
		expected := filepath.Join(ex, "expected")
		if _, err := os.Stat(expected); err != nil {
			continue
		}
		step := "expected modes of " + filepath.Base(ex)
		argv := []string{"render", src, "--out", filepath.Join(testenv.Dir(t), "out"), "--json"}
		if _, err := os.Stat(filepath.Join(ex, "inputs")); err == nil {
			argv = append(argv, "--data", filepath.Join(ex, "inputs"))
		}
		r := runTool(t, argv...)
		var res renderResult
		if r.doc == nil || r.status != 0 || json.Unmarshal(r.doc.Result, &res) != nil {
			add(step, "render: status %d, findings %q", r.status, r.codes())
			continue
		}
		for _, f := range res.Files {
			fi, err := os.Stat(filepath.Join(expected, filepath.FromSlash(f.Path)))
			if err != nil {
				continue
			}
			if want := fi.Mode()&0o100 != 0; f.Executable != want {
				add(step, "%s renders executable %t, and expected/ holds it with mode %v", f.Path, f.Executable, fi.Mode())
			}
		}
	}
}

// writtenAsListed compares the tree at dir with render's list of the
// files it wrote: every file listed is there, with its size, digest and
// executable bit, and nothing else is.
func writtenAsListed(dir string, files []renderFile) []string {
	var ds []string
	listed := map[string]bool{}
	for _, f := range files {
		listed[f.Path] = true
		p := filepath.Join(dir, filepath.FromSlash(f.Path))
		b, err := os.ReadFile(p)
		if err != nil {
			ds = append(ds, fmt.Sprintf("%s is listed and cannot be read: %v", f.Path, err))
			continue
		}
		sum := sha256.Sum256(b)
		if int64(len(b)) != f.Size || "sha256:"+hex.EncodeToString(sum[:]) != f.Digest {
			ds = append(ds, fmt.Sprintf("%s holds %d bytes that are not the %d bytes of %s listed", f.Path, len(b), f.Size, f.Digest))
		}
		if fi, err := os.Stat(p); err == nil && runtime.GOOS != "windows" && (fi.Mode()&0o100 != 0) != f.Executable {
			ds = append(ds, fmt.Sprintf("%s has mode %v, and is listed executable %t", f.Path, fi.Mode(), f.Executable))
		}
	}
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		if !listed[filepath.ToSlash(rel)] {
			ds = append(ds, fmt.Sprintf("%s is written and not listed", filepath.ToSlash(rel)))
		}
		return nil
	})
	return ds
}

// TestExampleHarness holds the harness to what it is there to see: the
// scaffold init writes passes it with nothing to report, and a package
// that warns in check, one whose example fails, and one whose preview
// cannot be made each fail on the step that shows it, and a written tree
// that is not the one render lists is reported. A missing examples/
// directory, an empty one and a file there each hold no package.
func TestExampleHarness(t *testing.T) {
	testenv.Home(t)
	scaffold := func(t *testing.T) string {
		dir := filepath.Join(testenv.Dir(t), "greeting")
		if r := runTool(t, "init", dir); r.status != 0 {
			t.Fatalf("metais init: status %d, %s", r.status, r.stderr)
		}
		return dir
	}
	write := func(t *testing.T, p, s string) {
		t.Helper()
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("the scaffold passes", func(t *testing.T) {
		if ps := exampleHarness(t, scaffold(t)); len(ps) != 0 {
			t.Errorf("the scaffold fails the harness:\n%s", strings.Join(ps, "\n"))
		}
	})
	for _, c := range []struct {
		name  string
		edit  func(t *testing.T, dir string)
		steps []string // the steps that must report, each a prefix of a problem
	}{
		{"a check warning", func(t *testing.T, dir string) {
			write(t, filepath.Join(dir, "notes.txt"), "not packed\n")
		}, []string{"check: ", "pack: "}},
		{"a failing example", func(t *testing.T, dir string) {
			write(t, filepath.Join(dir, "examples/basic/expected/greeting.md"), "# Hello, someone else\n")
		}, []string{"test: ", "pack: "}},
		{"an expected file whose mode is not the render's", func(t *testing.T, dir string) {
			if runtime.GOOS == "windows" {
				t.Skip("a checkout on Windows keeps no mode")
			}
			if err := os.Chmod(filepath.Join(dir, "examples/basic/expected/greeting.md"), 0o755); err != nil {
				t.Fatal(err)
			}
		}, []string{"expected modes of basic: "}},
		{"no example", func(t *testing.T, dir string) {
			if err := os.RemoveAll(filepath.Join(dir, "examples")); err != nil {
				t.Fatal(err)
			}
		}, []string{"test: "}},
		{"a preview that cannot be made", func(t *testing.T, dir string) {
			// A required property no string can satisfy: the generator
			// cannot make it, in either mode.
			b, err := os.ReadFile(filepath.Join(dir, "inputs/greeting.schema.json"))
			if err != nil {
				t.Fatal(err)
			}
			s := strings.Replace(string(b), `"minLength": 1,`, `"minLength": 1, "pattern": "^a$", "const": "b",`, 1)
			if s == string(b) {
				t.Fatal("the scaffold's schema has no minLength to edit")
			}
			write(t, filepath.Join(dir, "inputs/greeting.schema.json"), s)
		}, []string{"render --sample --optional all: ", "render --sample --optional none: "}},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := scaffold(t)
			c.edit(t, dir)
			ps := exampleHarness(t, dir)
			t.Logf("the harness reported:\n%s", strings.Join(ps, "\n"))
			for _, step := range c.steps {
				if !slices.ContainsFunc(ps, func(p string) bool { return strings.HasPrefix(p, step) }) {
					t.Errorf("no problem from %q; the harness reported:\n%s", strings.TrimSuffix(step, ": "), strings.Join(ps, "\n"))
				}
			}
		})
	}

	t.Run("a written tree that is not what render lists", func(t *testing.T) {
		dir := testenv.Dir(t)
		write(t, filepath.Join(dir, "a.md"), "a\n")
		write(t, filepath.Join(dir, "extra.md"), "extra\n")
		listed := func(p, content string) renderFile {
			sum := sha256.Sum256([]byte(content))
			return renderFile{Path: p, Size: int64(len(content)), Digest: "sha256:" + hex.EncodeToString(sum[:])}
		}
		a, extra := listed("a.md", "a\n"), listed("extra.md", "extra\n")
		if ds := writtenAsListed(dir, []renderFile{a, extra}); len(ds) != 0 {
			t.Errorf("the tree as listed: %q", ds)
		}
		if ds := writtenAsListed(dir, []renderFile{a, listed("extra.md", "other\n")}); len(ds) != 1 || !strings.HasPrefix(ds[0], "extra.md holds") {
			t.Errorf("a file whose digest is not the one listed: %q", ds)
		}
		if ds := writtenAsListed(dir, []renderFile{a}); len(ds) != 1 || ds[0] != "extra.md is written and not listed" {
			t.Errorf("a file written and not listed: %q", ds)
		}
		if ds := writtenAsListed(dir, []renderFile{a, listed("b.md", ""), extra}); len(ds) != 1 || !strings.HasPrefix(ds[0], "b.md is listed and cannot be read") {
			t.Errorf("a file listed and not written: %q", ds)
		}
		if runtime.GOOS != "windows" {
			a.Executable = true
			if ds := writtenAsListed(dir, []renderFile{a, extra}); len(ds) != 1 || !strings.HasPrefix(ds[0], "a.md has mode") {
				t.Errorf("a file listed executable and written without the bit: %q", ds)
			}
		}
	})

	t.Run("where there is no package", func(t *testing.T) {
		root := testenv.Dir(t)
		write(t, filepath.Join(root, "README.md"), "Not a package.\n")
		for _, dir := range []string{filepath.Join(root, "missing"), testenv.Dir(t), root} {
			if dirs, err := exampleDirs(dir); err != nil || len(dirs) != 0 {
				t.Errorf("%s holds packages %q, %v", dir, dirs, err)
			}
		}
		if err := os.Mkdir(filepath.Join(root, "one"), 0o755); err != nil {
			t.Fatal(err)
		}
		if dirs, err := exampleDirs(root); err != nil || !slices.Equal(dirs, []string{filepath.Join(root, "one")}) {
			t.Errorf("a directory with one package directory holds %q, %v", dirs, err)
		}
	})
}
