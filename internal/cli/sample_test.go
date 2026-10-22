// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
)

// samplePackage copies the SMP vectors' package below root and returns
// its path: four inputs, one in each of the JSON, CSV and text forms and
// a YAML collection, under conditionals, patterns and formats.
func samplePackage(t *testing.T, root string) string {
	t.Helper()
	pkg := filepath.Join(root, "pkg")
	src := filepath.Join(vectorsDir, "smp", "sample-data-directory", "package")
	if err := os.CopyFS(pkg, os.DirFS(src)); err != nil {
		t.Fatal(err)
	}
	return pkg
}

// readSample reads the result of a metais.sample/1 document.
func readSample(t *testing.T, r ran) sampleResult {
	t.Helper()
	var res sampleResult
	if r.doc == nil || jsonUnmarshal(r.doc.Result, &res) != nil {
		t.Fatalf("metais %q printed no sample result: %q", r.args, r.stdout)
	}
	return res
}

// hostTree is every file below dir, by its slash-separated path, with its
// bytes, and every directory, with a trailing slash.
func hostTree(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			out[rel+"/"] = ""
			return nil
		}
		b, err := os.ReadFile(p)
		out[rel] = string(b)
		return err
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return out
}

func equalTrees(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || w != v {
			return false
		}
	}
	return true
}

// TestSampleTo is W5-10.1 and W5-10.6: sample --out writes a data directory
// in each input's first form, its JSON result names the options and every
// file relative to the destination, the data renders under render --data,
// and what render --sample previews is exactly what that render gives.
func TestSampleTo(t *testing.T) {
	root := testenv.Dir(t)
	pkg := samplePackage(t, root)
	data := filepath.Join(root, "data")
	r := runTool(t, "sample", pkg, "--out", data, "--json")
	if r.status != 0 || !r.doc.OK {
		t.Fatalf("metais sample: status %d, %s", r.status, r.stdout)
	}
	res := readSample(t, r)
	want := []string{"events/001.yaml", "events/002.yaml", "events/003.yaml", "lines.csv", "summary.txt", "vendor.json"}
	if res.Optional != "all" || res.Seed != 0 || res.Items != 3 || res.Example != nil || res.DryRun || !slices.Equal(res.Files, want) {
		t.Errorf("result %+v", res)
	}
	got := hostTree(t, data)
	for _, f := range want {
		if _, ok := got[f]; !ok {
			t.Errorf("%s was not written: %v", f, got)
		}
	}
	if len(got) != len(want)+2 { // the root and events/
		t.Errorf("the data directory holds %d entries: %v", len(got), got)
	}
	if !strings.HasPrefix(got["lines.csv"], "sku,qty,price\r\n") {
		t.Errorf("lines.csv: %q", got["lines.csv"])
	}

	if r := runTool(t, "validate", pkg, "--data", data, "--json"); r.status != 0 || len(r.doc.Findings) != 0 {
		t.Errorf("the data does not validate: %s", r.stdout)
	}
	saved, preview := filepath.Join(root, "saved"), filepath.Join(root, "preview")
	if r := runTool(t, "render", pkg, "--data", data, "--out", saved); r.status != 0 {
		t.Fatalf("render --data: %s%s", r.stdout, r.stderr)
	}
	r = runTool(t, "render", pkg, "--sample", "--out", preview, "--json")
	if r.status != 0 {
		t.Fatalf("render --sample: %s", r.stdout)
	}
	if s := readRender(t, r).Sample; s == nil || s.Optional != "all" || s.Seed != 0 || s.Items != 3 {
		t.Errorf("render --sample's sample is %+v", s)
	}
	if a, b := hostTree(t, saved), hostTree(t, preview); !equalTrees(a, b) {
		t.Errorf("render --sample previews %v, and render --data of sample's data gives %v", b, a)
	}

	// Human mode lists the files on standard output.
	r = runTool(t, "sample", pkg, "--out", filepath.Join(root, "human"))
	if r.status != 0 || !strings.HasPrefix(r.stdout, "Wrote 6 files:\n  events/001.yaml\n") {
		t.Errorf("human: status %d, stdout %q", r.status, r.stdout)
	}
}

// TestSampleOptions is ADR-008 D9 sample item 4 and D8 item 8: the
// options change the data, the same options give the same bytes on every
// run, and the result and render's sample name them.
func TestSampleOptions(t *testing.T) {
	root := testenv.Dir(t)
	pkg := samplePackage(t, root)
	run := func(dir string, extra ...string) map[string]string {
		t.Helper()
		r := runTool(t, append([]string{"sample", pkg, "--out", filepath.Join(root, dir), "--json"}, extra...)...)
		if r.status != 0 {
			t.Fatalf("metais sample %q: %s", extra, r.stdout)
		}
		res := readSample(t, r)
		if len(extra) > 0 && (res.Optional != "mixed" || res.Seed != 18446744073709551615 || res.Items != 1) {
			t.Errorf("result %+v", res)
		}
		return hostTree(t, filepath.Join(root, dir))
	}
	opts := []string{"--optional", "mixed", "--seed", "18446744073709551615", "--items", "1"}
	a, b, c := run("a", opts...), run("b", opts...), run("c")
	if !equalTrees(a, b) {
		t.Errorf("the same options gave different data:\n%v\n%v", a, b)
	}
	if equalTrees(a, c) {
		t.Errorf("other options gave the same data: %v", a)
	}
	if _, ok := a["events/002.yaml"]; ok {
		t.Errorf("--items 1 made two events: %v", a)
	}
	r := runTool(t, append([]string{"render", pkg, "--sample", "--out", filepath.Join(root, "r"), "--json"}, opts...)...)
	if s := readRender(t, r).Sample; s == nil || s.Optional != "mixed" || s.Seed != 18446744073709551615 || s.Items != 1 {
		t.Errorf("render --sample's sample is %+v", s)
	}
}

// TestSampleFixedPoints is W5-10.1's fixed points and W5-10.5's: values
// given with the supply flags are kept as given, the conditionals they
// select are honoured, and an object given where the schema makes none is
// kept and refused by validation, not replaced.
func TestSampleFixedPoints(t *testing.T) {
	root := testenv.Dir(t)
	pkg := samplePackage(t, root)
	data := filepath.Join(root, "data")
	r := runTool(t, "sample", pkg, "--out", data, "--set", "/vendor/kind=company", "--set-json", `/vendor/name="Acme"`, "--json")
	if r.status != 0 {
		t.Fatalf("metais sample: %s", r.stdout)
	}
	v := hostTree(t, data)["vendor.json"]
	for _, s := range []string{`"kind": "company"`, `"name": "Acme"`, `"registration": `} {
		if !strings.Contains(v, s) {
			t.Errorf("vendor.json lacks %s: %s", s, v)
		}
	}

	out := filepath.Join(root, "out")
	r = runTool(t, "render", pkg, "--sample", "--set", "/vendor/kind=company", "--set-json", `/vendor/name="Acme"`, "--out", out)
	b, _ := os.ReadFile(filepath.Join(out, "report.md"))
	if r.status != 0 || string(b) != "# Acme\n\nThe vendor is a company, registered.\n" {
		t.Errorf("render --sample: status %d, report %q, %s", r.status, b, r.stderr)
	}

	// The lines are a CSV table; an object given for them is kept, and no
	// CSV can hold it.
	nothing := filepath.Join(root, "nothing")
	r = runTool(t, "sample", pkg, "--out", nothing, "--set-json", "/lines/0/qty=0", "--json")
	if r.status != 1 || len(r.doc.Findings) != 1 || r.doc.Findings[0].Code != "schema.sample_invalid" || r.doc.Findings[0].Params["code"] != "form" {
		t.Errorf("an object for a CSV input: status %d, %s", r.status, r.stdout)
	}
	if _, err := os.Lstat(nothing); !os.IsNotExist(err) {
		t.Errorf("a refused sample wrote %s: %v", nothing, err)
	}
}

// TestSampleOverwrite is W5-10.2's --overwrite, which no vector can reach
// (ADR-011 D8's sample takes no overwrite), proved on the host: without it
// a directory that holds any entry, a hidden one included, is refused and
// left as it was; with it, each file sample writes replaces the file of
// that name, a file it does not write is left with its bytes, and nothing
// else is created.
func TestSampleOverwrite(t *testing.T) {
	root := testenv.Dir(t)
	pkg := samplePackage(t, root)
	data := filepath.Join(root, "data")
	for name, body := range map[string]string{"vendor.json": "old vendor\n", "notes.txt": "the author's\n", "events/keep.txt": "kept\n", ".hidden": "hidden\n"} {
		p := filepath.Join(data, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	before := hostTree(t, data)
	r := runTool(t, "sample", pkg, "--out", data, "--json")
	if r.status != 5 || len(r.doc.Findings) != 1 || r.doc.Findings[0].Code != "output.destination.not_empty" {
		t.Fatalf("without --overwrite: status %d, %s", r.status, r.stdout)
	}
	if !slices.ContainsFunc(r.doc.Next, func(n Next) bool { return strings.Contains(n.Run, "--overwrite") }) {
		t.Errorf("the refusal does not point at --overwrite: %+v", r.doc.Next)
	}
	if after := hostTree(t, data); !equalTrees(before, after) {
		t.Errorf("a refused sample changed the directory:\n%v\n%v", before, after)
	}
	hidden := filepath.Join(root, "hidden")
	if err := os.MkdirAll(filepath.Join(hidden, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if r := runTool(t, "sample", pkg, "--out", hidden, "--json"); r.status != 5 || r.doc.Findings[0].Code != "output.destination.not_empty" {
		t.Errorf("a hidden entry: status %d, %s", r.status, r.stdout)
	}

	r = runTool(t, "sample", pkg, "--out", data, "--overwrite", "--json")
	if r.status != 0 {
		t.Fatalf("with --overwrite: %s", r.stdout)
	}
	after := hostTree(t, data)
	for _, name := range []string{"notes.txt", "events/keep.txt", ".hidden"} {
		if after[name] != before[name] {
			t.Errorf("%s was changed or removed: %q", name, after[name])
		}
	}
	if after["vendor.json"] == before["vendor.json"] || !strings.HasPrefix(after["vendor.json"], "{\n") {
		t.Errorf("vendor.json was not replaced: %q", after["vendor.json"])
	}
	for _, f := range readSample(t, r).Files {
		if _, ok := after[f]; !ok {
			t.Errorf("%s was not written", f)
		}
	}
	if len(after) != len(before)+len(readSample(t, r).Files)-1 {
		t.Errorf("--overwrite left %d entries from %d: %v", len(after), len(before), after)
	}

	// A link where sample writes a file is refused, by path, and nothing is
	// written.
	linked := filepath.Join(root, "linked")
	if err := os.Mkdir(linked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "elsewhere"), filepath.Join(linked, "vendor.json")); err != nil {
		t.Skipf("no symbolic link here: %v", err)
	}
	r = runTool(t, "sample", pkg, "--out", linked, "--overwrite", "--json")
	if r.status != 5 || len(r.doc.Findings) != 1 || r.doc.Findings[0].Code != "output.destination.link" || r.doc.Findings[0].Params["path"] != "vendor.json" {
		t.Errorf("a link in the way: status %d, %s", r.status, r.stdout)
	}
	if got := hostTree(t, linked); len(got) != 2 {
		t.Errorf("a refused write left %v", got)
	}
}

// TestSampleEmptyCollection is a collection with no item: its directory
// is written, so the data reads back as the empty collection it is
// (ADR-003 D15), and is listed in files. Into an existing directory it is
// W5-V2.A: the directory is checked with the files, before anything is
// written, prepared beneath .metais-staging, and arrives by the commit.
func TestSampleEmptyCollection(t *testing.T) {
	root := testenv.Dir(t)
	pkg := samplePackage(t, root)
	want := []string{"events/", "lines.csv", "summary.txt", "vendor.json"}
	for _, dir := range []string{"new", "empty", "full"} {
		data := filepath.Join(root, dir)
		line := []string{"sample", pkg, "--out", data, "--items", "0", "--json"}
		if dir != "new" {
			if err := os.Mkdir(data, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		if dir == "full" {
			// The verifier's case: an existing directory written with
			// --overwrite.
			if err := os.WriteFile(filepath.Join(data, "keep"), nil, 0o644); err != nil {
				t.Fatal(err)
			}
			line = append(line, "--overwrite")
		}
		r := runTool(t, line...)
		if r.status != 0 {
			t.Fatalf("%s: %s", dir, r.stdout)
		}
		if res := readSample(t, r); !slices.Equal(res.Files, want) {
			t.Errorf("%s: files %q, want %q", dir, res.Files, want)
		}
		if info, err := os.Lstat(filepath.Join(data, "events")); err != nil || !info.IsDir() {
			t.Errorf("%s: events/ is %v, %v", dir, info, err)
		}
		if _, err := os.Lstat(filepath.Join(data, ".metais-staging")); err == nil {
			t.Errorf("%s: the staging directory is left", dir)
		}
		// full's keep is no input, so only the others validate.
		if r := runTool(t, "validate", pkg, "--data", data, "--json"); dir != "full" && (r.status != 0 || len(r.doc.Findings) != 0) {
			t.Errorf("%s: %s", dir, r.stdout)
		}
		// Written again, the directory is there, and is left as it is.
		if dir == "full" {
			if r := runTool(t, line...); r.status != 0 || !slices.Equal(readSample(t, r).Files, want) {
				t.Errorf("again: status %d, %s", r.status, r.stdout)
			}
		}
	}

	// A file where the collection's directory goes is refused with the
	// other checks, and nothing is written: the directory is part of the
	// plan, not made after the commit.
	in := filepath.Join(root, "in-the-way")
	if err := os.Mkdir(in, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(in, "events"), []byte("a file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := hostTree(t, in)
	for _, dry := range []bool{true, false} {
		line := []string{"sample", pkg, "--out", in, "--items", "0", "--overwrite", "--json"}
		if dry {
			line = append(line, "--dry-run")
		}
		r := runTool(t, line...)
		if r.status != 5 || len(r.doc.Findings) != 1 || r.doc.Findings[0].Code != "output.destination.not_directory" || r.doc.Findings[0].Params["path"] != "events/" {
			t.Errorf("dry run %v: a file in the way: status %d, %s", dry, r.status, r.stdout)
		}
		if after := hostTree(t, in); !equalTrees(before, after) {
			t.Errorf("dry run %v: a refused write changed the directory: %v", dry, after)
		}
	}
}

// TestSampleReportsEveryRefusal is W5-V2.B: every planned path the checks
// refuse is reported, each with render's and extract's code and its path,
// together with a staging directory left in the way, before anything is
// written, on a dry run as on the write.
func TestSampleReportsEveryRefusal(t *testing.T) {
	root := testenv.Dir(t)
	pkg := samplePackage(t, root)
	data := filepath.Join(root, "data")
	for _, d := range []string{"vendor.json", "lines.csv", ".metais-staging"} {
		if err := os.MkdirAll(filepath.Join(data, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	before := hostTree(t, data)
	for _, dry := range []bool{false, true} {
		line := []string{"sample", pkg, "--out", data, "--overwrite", "--json"}
		if dry {
			line = append(line, "--dry-run")
		}
		r := runTool(t, line...)
		type got struct{ code, path, reason string }
		var gots []got
		for _, f := range r.doc.Findings {
			p, _ := f.Params["path"].(string)
			reason, _ := f.Params["reason"].(string)
			gots = append(gots, got{f.Code, p, reason})
		}
		want := []got{
			{"output.destination.not_regular_file", "lines.csv", "is_directory"},
			{"output.destination.not_regular_file", "vendor.json", "is_directory"},
			{"output.destination.staging_exists", "", ""},
		}
		if r.status != 5 || !slices.Equal(gots, want) {
			t.Errorf("dry run %v: status %d, findings %+v, want %+v", dry, r.status, gots, want)
		}
		if after := hostTree(t, data); !equalTrees(before, after) {
			t.Errorf("dry run %v: a refused write changed the directory: %v", dry, after)
		}
	}
}

// TestSampleHumanCount is W5-V2.C: the human heading agrees in number with
// the files it lists, as extract's does.
func TestSampleHumanCount(t *testing.T) {
	pkg, _ := declarationPackage(t, "  - id: t\n    template: t.txt.liquid\n")
	to := filepath.Join(testenv.Dir(t), "data")
	if r := runTool(t, "sample", pkg, "--out", to, "--dry-run"); r.status != 0 || r.stdout != "Would write 1 file:\n  pages.json\n" {
		t.Errorf("dry run: status %d, %q", r.status, r.stdout)
	}
	if r := runTool(t, "sample", pkg, "--out", to); r.status != 0 || r.stdout != "Wrote 1 file:\n  pages.json\n" {
		t.Errorf("status %d, %q", r.status, r.stdout)
	}
}

// TestSampleExample is W5-10.3: --example writes examples/<name>/inputs/
// and example.yaml into the source directory, which still checks; it is
// refused when the example exists at all, with a packed package, and with
// --overwrite.
func TestSampleExample(t *testing.T) {
	root := testenv.Dir(t)
	pkg := samplePackage(t, root)
	r := runTool(t, "sample", pkg, "--example", "preview", "--seed", "4", "--json")
	if r.status != 0 {
		t.Fatalf("metais sample --example: %s", r.stdout)
	}
	res := readSample(t, r)
	if res.Example == nil || *res.Example != "preview" || res.Seed != 4 || !slices.Contains(res.Files, "example.yaml") || !slices.Contains(res.Files, "inputs/vendor.json") {
		t.Errorf("result %+v", res)
	}
	if len(r.doc.Next) != 1 || !strings.HasPrefix(r.doc.Next[0].Run, "metais test <package> --example") {
		// With metais test declared, the next step runs the example.
		t.Errorf("next %+v", r.doc.Next)
	}
	ex := hostTree(t, filepath.Join(pkg, "examples", "preview"))
	if !strings.Contains(ex["example.yaml"], "title: Sample data\n") || !strings.Contains(ex["example.yaml"], "--seed 4, --optional all and --items 3") ||
		!strings.Contains(ex["example.yaml"], "metais test --example preview --update") {
		t.Errorf("example.yaml: %q", ex["example.yaml"])
	}
	if _, ok := ex["expected/"]; ok {
		t.Errorf("sample wrote expected/: %v", ex)
	}
	if r := runTool(t, "check", pkg, "--json"); r.status != 0 || len(r.doc.Findings) != 0 {
		t.Errorf("the source directory does not check: %s", r.stdout)
	}
	entries, _ := os.ReadDir(filepath.Join(pkg, "examples"))
	if len(entries) != 1 {
		t.Errorf("examples/ holds %v", entries)
	}

	before := hostTree(t, pkg)
	r = runTool(t, "sample", pkg, "--example", "preview", "--json")
	if r.status != 5 || r.doc.Findings[0].Code != "output.destination.nothing_written" || r.doc.Findings[0].Params["reason"] != "exists" {
		t.Errorf("an example that exists: status %d, %s", r.status, r.stdout)
	}
	if err := os.MkdirAll(filepath.Join(pkg, "examples", "bare"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "examples", "bare", "example.yaml"), []byte("title: Bare\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before = hostTree(t, pkg)
	if r := runTool(t, "sample", pkg, "--example", "bare", "--json"); r.status != 5 || r.doc.Findings[0].Params["reason"] != "exists" {
		t.Errorf("an example with no inputs: status %d, %s", r.status, r.stdout)
	}
	if after := hostTree(t, pkg); !equalTrees(before, after) {
		t.Errorf("a refused --example changed the source directory")
	}

	packed := filepath.Join(vectorsDir, "pkg", "opens-packed", "package.metais")
	r = runTool(t, "sample", packed, "--example", "x", "--json")
	if r.status != 3 || r.codes()[0] != "usage.not_a_source_directory" || !strings.Contains(r.doc.Next[0].Run, "unpack") {
		t.Errorf("a packed package: status %d, %s", r.status, r.stdout)
	}
}

// TestSampleDryRun is ADR-008 D9's --dry-run (R2-127): the data is made
// and validated, the files listed, and nothing written anywhere.
func TestSampleDryRun(t *testing.T) {
	root := testenv.Dir(t)
	pkg := samplePackage(t, root)
	before := hostTree(t, root)
	for _, line := range [][]string{
		{"sample", pkg, "--out", filepath.Join(root, "data"), "--dry-run", "--json"},
		{"sample", pkg, "--example", "dry", "--dry-run", "--json"},
	} {
		r := runTool(t, line...)
		res := readSample(t, r)
		if r.status != 0 || !res.DryRun || len(res.Files) < 6 {
			t.Errorf("metais %q: status %d, %+v", line, r.status, res)
		}
	}
	r := runTool(t, "sample", pkg, "--out", filepath.Join(root, "data"), "--dry-run")
	if !strings.HasPrefix(r.stdout, "Would write 6 files:\n") {
		t.Errorf("human: %q", r.stdout)
	}
	if after := hostTree(t, root); !equalTrees(before, after) {
		t.Errorf("a dry run wrote something")
	}

	// A dry run is refused as the write would be.
	full := filepath.Join(root, "full")
	if err := os.MkdirAll(filepath.Join(full, "vendor.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(pkg, "examples", "taken"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "examples", "taken", "example.yaml"), []byte("title: Taken\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before = hostTree(t, root)
	for _, tc := range []struct {
		line []string
		code string
	}{
		{[]string{"sample", pkg, "--out", full, "--dry-run"}, "output.destination.not_empty"},
		{[]string{"sample", pkg, "--out", full, "--overwrite", "--dry-run"}, "output.destination.not_regular_file"},
		{[]string{"sample", pkg, "--example", "taken", "--dry-run"}, "output.destination.nothing_written"},
	} {
		r := runTool(t, append(tc.line, "--json")...)
		if r.status != 5 || len(r.codes()) != 1 || r.codes()[0] != tc.code {
			t.Errorf("metais %q: status %d, %v", tc.line, r.status, r.codes())
		}
	}
	if after := hostTree(t, root); !equalTrees(before, after) {
		t.Errorf("a refused dry run wrote something")
	}
}

// TestSampleRefusals is each refusal of sample's line and of render
// --sample's, each writing nothing: the flags that exclude each other or
// need --sample, a count or name refused, and the unsatisfiable field.
func TestSampleRefusals(t *testing.T) {
	root := testenv.Dir(t)
	pkg := samplePackage(t, root)
	to := filepath.Join(root, "data")
	for _, tc := range []struct {
		line   []string
		status int
		code   string
	}{
		{[]string{"sample", pkg}, 3, "usage.missing_operand"},
		{[]string{"sample", pkg, "--out", to, "--example", "x"}, 3, "usage.flag_excluded"},
		{[]string{"sample", pkg, "--example", "x", "--overwrite"}, 3, "usage.flag_excluded"},
		{[]string{"sample", pkg, "--out", ""}, 3, "usage.bad_value"},
		{[]string{"sample", pkg, "--example", "Not-A-Name"}, 3, "usage.bad_value"},
		{[]string{"sample", pkg, "--out", to, "--seed", "18446744073709551616"}, 3, "usage.bad_value"},
		{[]string{"sample", pkg, "--out", to, "--seed=+1"}, 3, "usage.bad_value"},
		{[]string{"sample", pkg, "--out", to, "--items", "-1"}, 3, "usage.bad_value"},
		{[]string{"sample", pkg, "--out", to, "--optional", "some"}, 3, "usage.bad_value"},
		{[]string{"sample", "-", "--out", to}, 3, "usage.package_from_stdin"},
		{[]string{"sample", pkg, "--out", to, "--set", "/nothing=1"}, 1, "input.set_pointer_invalid"},
		{[]string{"render", pkg, "--sample", "--data", to, "--out", to}, 3, "usage.flag_excluded"},
		{[]string{"render", pkg, "--seed", "1", "--out", to}, 3, "usage.flag_requires"},
		{[]string{"render", pkg, "--optional", "none", "--out", to}, 3, "usage.flag_requires"},
		{[]string{"render", pkg, "--items", "2", "--out", to}, 3, "usage.flag_requires"},
		{[]string{"render", pkg, "--sample", "--seed", "x", "--out", to}, 3, "usage.bad_value"},
	} {
		r := runTool(t, append(tc.line, "--json")...)
		if r.status != tc.status || len(r.codes()) == 0 || r.codes()[0] != tc.code {
			t.Errorf("metais %q: status %d, %v; want %d, %s", tc.line, r.status, r.codes(), tc.status, tc.code)
		}
		if _, err := os.Lstat(to); !os.IsNotExist(err) {
			t.Fatalf("metais %q wrote %s", tc.line, to)
		}
	}
}

// TestSampleUnsatisfiable is W5-10.4: every field the generator cannot
// satisfy is reported, by sample and by render --sample, and nothing is
// written; and render --sample is an authoring command (W5-10.5): a
// defect of the package fails it with status 1, where render without it
// is refused with status 4.
func TestSampleUnsatisfiable(t *testing.T) {
	root := testenv.Dir(t)
	pkg := filepath.Join(root, "pkg")
	if err := os.CopyFS(pkg, os.DirFS(filepath.Join(vectorsDir, "smp", "sample-unsatisfiable", "package"))); err != nil {
		t.Fatal(err)
	}
	to := filepath.Join(root, "data")
	for _, line := range [][]string{{"sample", pkg, "--out", to}, {"render", pkg, "--sample", "--out", to}} {
		r := runTool(t, append(line, "--json")...)
		if r.status != 1 || !slices.Equal(r.codes(), []string{"schema.sample_unsatisfiable", "schema.sample_unsatisfiable"}) {
			t.Errorf("metais %q: status %d, %s", line, r.status, r.stdout)
		}
		if _, err := os.Lstat(to); !os.IsNotExist(err) {
			t.Errorf("metais %q wrote %s", line, to)
		}
		// The next step is the input's field listing (W2-NB1).
		if len(r.doc.Next) != 1 || r.doc.Next[0].Run != "metais schema <package> <name> --fields" {
			t.Errorf("metais %q: next %v", line, r.doc.Next)
		}
	}
	// The listing's line for the field the finding names gives the
	// constraints no value meets, which is what an examples value must
	// meet.
	fields := runTool(t, "schema", pkg, "vendor", "--fields")
	if !strings.Contains(fields.stdout, "/vendor/name\tstring\trequired\tmaxLength=3 minLength=5\t") {
		t.Errorf("the field listing does not give the unsatisfiable field's constraints:\n%s", fields.stdout)
	}

	broken := filepath.Join(root, "broken")
	if err := os.CopyFS(broken, os.DirFS(filepath.Join(vectorsDir, "smp", "sample-data-directory", "package"))); err != nil {
		t.Fatal(err)
	}
	tmpl := filepath.Join(broken, "templates", "report.md.liquid")
	b, _ := os.ReadFile(tmpl)
	if err := os.WriteFile(tmpl, append(b, []byte("{{ 1 | divided_by: 0 }}\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := runTool(t, "render", broken, "--sample", "--out", to, "--json"); r.status != 1 {
		t.Errorf("render --sample of a template defect: status %d, %s", r.status, r.stdout)
	}
	data := filepath.Join(root, "broken-data")
	if r := runTool(t, "sample", broken, "--out", data); r.status != 0 {
		t.Fatalf("metais sample: %s", r.stderr)
	}
	if r := runTool(t, "render", broken, "--data", data, "--out", to, "--json"); r.status != 4 {
		t.Errorf("render of a template defect: status %d, %s", r.status, r.stdout)
	}
}

// TestSampleWriteFailed is a write that fails part way through a staged
// data directory and a staged example: nothing of it is left.
func TestSampleWriteFailed(t *testing.T) {
	root := testenv.Dir(t)
	pkg := samplePackage(t, root)
	parent := filepath.Join(root, "ro")
	if err := os.Mkdir(parent, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(parent, 0o755) })
	if f, err := os.Create(filepath.Join(parent, "probe")); err == nil {
		f.Close()
		t.Skip("this process writes into a directory of mode 0555")
	}
	r := runTool(t, "sample", pkg, "--out", filepath.Join(parent, "data"), "--json")
	if r.status != 5 || r.codes()[0] != "output.destination.nothing_written" {
		t.Errorf("status %d, %s", r.status, r.stdout)
	}
	if got := hostTree(t, parent); len(got) != 1 {
		t.Errorf("a failed write left %v", got)
	}
}
