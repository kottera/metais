// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"context"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/author"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostenv"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/render"
	"github.com/kottera/metais/internal/testenv"
)

// scaffoldFiles is what init reports writing, in order.
var scaffoldFiles = []string{
	".gitattributes", "metais.yaml", "inputs/greeting.schema.json", "inputs/greeting.md",
	"templates/greeting.md.liquid", "examples/basic/example.yaml",
	"examples/basic/inputs/greeting.json", "examples/basic/expected/greeting.md",
}

// readInit reads the result of a metais.init/1 document.
func readInit(t *testing.T, r ran) initResult {
	t.Helper()
	var res initResult
	if r.doc == nil || jsonUnmarshal(r.doc.Result, &res) != nil {
		t.Fatalf("metais %q printed no init result: %q", r.args, r.stdout)
	}
	return res
}

// TestInit is init's success, in JSON and in human mode: the scaffold is
// written into a directory created with its parents, the name taken from
// the operand, and the files listed in order.
func TestInit(t *testing.T) {
	root := testenv.Dir(t)
	dir := filepath.Join(root, "a", "acme-report")
	r := runTool(t, "init", dir, "--json")
	if r.status != 0 || !r.doc.OK || len(r.doc.Findings) != 0 || len(r.doc.Next) != 1 || r.doc.Next[0].Run != "metais check <dir>" {
		t.Fatalf("metais init --json: status %d, %s", r.status, r.stdout)
	}
	if res := readInit(t, r); res.DryRun || !slices.Equal(res.Files, scaffoldFiles) {
		t.Errorf("result %+v", res)
	}
	b, err := os.ReadFile(filepath.Join(dir, "metais.yaml"))
	if err != nil || !bytes.Contains(b, []byte("\nname: \"acme-report\"\n")) {
		t.Errorf("metais.yaml does not take the name from the operand: %q, %v", b, err)
	}

	dir2 := filepath.Join(root, "b")
	r = runTool(t, "init", dir2, "--name", "other-name")
	if r.status != 0 || r.stderr != "\nNext:\n  metais check <dir>  — Checks the package, which reports no finding until you edit it.\n" {
		t.Fatalf("metais init: status %d, stderr %q", r.status, r.stderr)
	}
	want := "Wrote 8 files:\n  " + strings.Join(scaffoldFiles, "\n  ") + "\n\nEdit inputs/ and templates/ to make the package yours.\n"
	if r.stdout != want {
		t.Errorf("metais init printed %q, want %q", r.stdout, want)
	}
	if b, _ := os.ReadFile(filepath.Join(dir2, "metais.yaml")); !bytes.Contains(b, []byte("\nname: \"other-name\"\n")) {
		t.Errorf("metais.yaml does not take the name from --name: %q", b)
	}
}

// TestInitScaffoldPinned is ADR-007 D1 item 7: the scaffold, exactly as
// init writes it, checks with no finding at all, tests with its one
// example passing, and packs, with no finding either, into a package that
// checks with none. Nothing of it writes into the scaffold.
func TestInitScaffoldPinned(t *testing.T) {
	testenv.Home(t)
	dir := filepath.Join(testenv.Dir(t), "report")
	out := testenv.Dir(t)
	steps := [][]string{
		{"init", dir},
		{"check", dir},
		{"test", dir},
		{"pack", dir, "--out", out},
		{"check", filepath.Join(out, "report-0.1.0.metais")},
	}
	var written map[string][]byte
	for i, argv := range steps {
		r := runTool(t, append(argv, "--json")...)
		if r.status != finding.ExitOK || r.doc == nil || !r.doc.OK || len(r.doc.Findings) != 0 {
			t.Fatalf("metais %s: status %d, findings %q", argv[0], r.status, r.codes())
		}
		switch {
		case i == 0:
			written = treeState(t, dir)
		case argv[0] == "test":
			if res := testRun(t, r); len(res.Examples) != 1 || res.Examples[0].Name != "basic" || res.Examples[0].Status != "pass" {
				t.Errorf("test of the scaffold: %+v", res.Examples)
			}
		case argv[0] == "pack":
			var res packResult
			result(t, r, &res)
			b, err := os.ReadFile(filepath.Join(out, "report-0.1.0.metais"))
			if err != nil || res.File != "report-0.1.0.metais" || int64(len(b)) != res.Size {
				t.Errorf("pack of the scaffold: result %+v, the file: %d bytes, %v", res, len(b), err)
			}
		}
	}
	if got := treeState(t, dir); !maps.EqualFunc(got, written, bytes.Equal) {
		t.Errorf("check, test or pack wrote into the scaffold: %q, as init wrote it %q", sortedKeys(got), sortedKeys(written))
	}
}

// TestInitCurrentDirectory is init with no operand: into the working
// directory, with --name, and nothing echoed of where that is.
func TestInitCurrentDirectory(t *testing.T) {
	dir := testenv.Dir(t)
	t.Chdir(dir)
	r := runTool(t, "init", "--name", "here", "--json")
	if r.status != 0 || !slices.Equal(readInit(t, r).Files, scaffoldFiles) {
		t.Fatalf("metais init --name here: %d %s", r.status, r.stdout)
	}
	if _, err := os.Stat(filepath.Join(dir, "metais.yaml")); err != nil {
		t.Error(err)
	}
}

// TestInitDryRun is W3-8.4 through the command line: --dry-run reports
// the files and writes nothing.
func TestInitDryRun(t *testing.T) {
	root := testenv.Dir(t)
	r := runTool(t, "init", filepath.Join(root, "x"), "--dry-run", "--json")
	if res := readInit(t, r); r.status != 0 || !res.DryRun || !slices.Equal(res.Files, scaffoldFiles) {
		t.Errorf("metais init --dry-run: %d %s", r.status, r.stdout)
	}
	r = runTool(t, "init", filepath.Join(root, "x"), "--dry-run")
	if !strings.HasPrefix(r.stdout, "Would write 8 files; --dry-run wrote nothing:\n  .gitattributes\n") {
		t.Errorf("metais init --dry-run printed %q", r.stdout)
	}
	if es, _ := os.ReadDir(root); len(es) != 0 {
		t.Errorf("a dry run wrote %v", es)
	}
}

// TestInitNextOnceCheckExists is ADR-008 D14 item 3: init's next step is
// check, with <dir> when the caller named a directory.
func TestInitNextOnceCheckExists(t *testing.T) {
	withCheck := table
	for _, c := range []struct {
		args []string
		next string
	}{
		{[]string{"init", "pkg", "--json"}, "metais check <dir>"},
		{[]string{"init", "--name", "pkg", "--json"}, "metais check"},
	} {
		t.Chdir(testenv.Dir(t))
		var stdout, stderr bytes.Buffer
		status := run(context.Background(), Invocation{Args: c.args, Stdin: failingReader{t}, Stdout: &stdout, Stderr: &stderr,
			Options: Options{Display: hostenv.Display{NoColor: true}}}, withCheck)
		var doc document
		if status != 0 || jsonUnmarshal(stdout.Bytes(), &doc) != nil || len(doc.Next) != 1 || doc.Next[0].Run != c.next {
			t.Errorf("metais %q: status %d, next %+v, want %q", c.args, status, doc.Next, c.next)
		}
	}
}

// initRefusal is one line init refuses, and how.
type initRefusal struct {
	name   string
	args   func(root string) []string
	status int
	code   finding.Code
}

// initRefusals trigger every refusal init declares beyond its line's
// (ADR-008 D16 item 2).
var initRefusals = []initRefusal{
	{"no operand", func(string) []string { return []string{"init"} }, finding.ExitUsage, finding.CodeUsageNameRequired},
	{"dot", func(r string) []string {
		return []string{"init", filepath.Join(r, "a") + string(filepath.Separator) + "."}
	}, finding.ExitUsage, finding.CodeUsageNameRequired},
	{"dot dot", func(r string) []string {
		return []string{"init", filepath.Join(r, "a") + string(filepath.Separator) + ".."}
	}, finding.ExitUsage, finding.CodeUsageNameRequired},
	{"a bad --name", func(r string) []string { return []string{"init", filepath.Join(r, "a"), "--name", "Acme Report"} }, finding.ExitUsage, finding.CodeUsageNameInvalid},
	{"a bad directory name", func(r string) []string { return []string{"init", filepath.Join(r, "Acme Report")} }, finding.ExitUsage, finding.CodeUsageNameInvalid},
	{"not empty", func(r string) []string { return []string{"init", filepath.Join(r, "full")} }, finding.ExitDestination, finding.CodeDestinationNotEmpty},
	{"not empty, dry run", func(r string) []string { return []string{"init", filepath.Join(r, "full"), "--dry-run"} }, finding.ExitDestination, finding.CodeDestinationNotEmpty},
	{"a file", func(r string) []string { return []string{"init", filepath.Join(r, "full", "notes")} }, finding.ExitDestination, finding.CodeDestinationNotDirectory},
	{"beneath a file", func(r string) []string { return []string{"init", filepath.Join(r, "full", "notes", "pkg")} }, finding.ExitDestination, finding.CodeDestinationNotDirectory},
}

// refusalRoot makes the tree initRefusals run against: full/ holds a
// file, notes.
func refusalRoot(t *testing.T) string {
	t.Helper()
	root := testenv.Dir(t)
	if err := os.Mkdir(filepath.Join(root, "full"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "full", "notes"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// TestInitRefusals is ADR-008 D16 item 2 for init: each refusal it
// declares is triggered, with its status, a next step that parses, and
// nothing written; runTool holds every run to its declaration.
func TestInitRefusals(t *testing.T) {
	seen := map[finding.Code]bool{}
	for _, c := range initRefusals {
		root := refusalRoot(t)
		t.Chdir(root)
		for _, mode := range [][]string{{"--json"}, nil} {
			r := runTool(t, append(c.args(root), mode...)...)
			if r.status != c.status {
				t.Errorf("%s: status %d, want %d", c.name, r.status, c.status)
			}
			if mode != nil {
				if !slices.Equal(r.codes(), []string{string(c.code)}) || r.doc.OK || string(r.doc.Result) != "null" || len(r.doc.Next) == 0 {
					t.Errorf("%s: %s", c.name, r.stdout)
				}
				seen[c.code] = true
			} else if r.stdout != "" || !strings.Contains(r.stderr, string(c.code)) || !strings.Contains(r.stderr, "Next:\n  metais init") {
				t.Errorf("%s in human mode: stdout %q, stderr %q", c.name, r.stdout, r.stderr)
			}
		}
		want := []string{"full", "full/notes"}
		if tr := walk(t, root); !slices.Equal(tr, want) {
			t.Errorf("%s: the refusal left %q", c.name, tr)
		}
	}
	for _, rf := range lookup(table, "init").Refusals {
		e, _ := finding.Lookup(rf.Code)
		if e.Class == finding.ClassUsage && strings.HasPrefix(string(rf.Code), "usage.name_") && !seen[rf.Code] {
			t.Errorf("no line triggers %s", rf.Code)
		}
		// A failed write needs a host that refuses it: TestInitWriteFailed and
		// TestInitFailedListing prove the two.
		failed := rf.Code == finding.CodeDestinationWriteFailed || rf.Code == finding.CodeDestinationNothingWritten
		if e.Class == finding.ClassDestination && !failed && !seen[rf.Code] {
			t.Errorf("no line triggers %s", rf.Code)
		}
	}
}

// walk lists every entry beneath root, "/"-separated, in byte order.
func walk(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(p string, _ os.DirEntry, err error) error {
		if err == nil && p != root {
			rel, _ := filepath.Rel(root, p)
			out = append(out, filepath.ToSlash(rel))
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(out)
	return out
}

// TestInitNameParams pins usage.name_invalid's parameters and next step: a
// name from --name gets a suggestion, also written into the next step; a
// name from the directory gets none, because no part of a caller's path
// is printed.
func TestInitNameParams(t *testing.T) {
	root := testenv.Dir(t)
	r := runTool(t, "init", filepath.Join(root, "a"), "--json", "--name", "Acme Report")
	p := r.doc.Findings[0].Params
	if p["source"] != "name" || p["argument"] != float64(4) || p["suggestion"] != "acme-report" {
		t.Errorf("params %v", p)
	}
	if len(r.doc.Next) != 1 || r.doc.Next[0].Run != "metais init <dir> --name acme-report" {
		t.Errorf("next %+v", r.doc.Next)
	}
	r = runTool(t, "init", "--name=!!!", "--json")
	if p := r.doc.Findings[0].Params; p["argument"] != float64(2) || p["suggestion"] != nil || r.doc.Next[0] != nextInitValidName {
		t.Errorf("params %v, next %+v", p, r.doc.Next)
	}
	r = runTool(t, "--json", "init", filepath.Join(root, "Acme Report"))
	if p := r.doc.Findings[0].Params; p["source"] != "directory" || p["argument"] != float64(3) || p["suggestion"] != nil || r.doc.Next[0] != nextInitValidName {
		t.Errorf("params %v, next %+v", p, r.doc.Next)
	}
}

// TestInitNoCallerPath is W3-8.6 (R1-011, ADR-008 D11 item 5): no part of
// the directory the caller named reaches any output, on success or on a
// refusal, in JSON or in human mode.
func TestInitNoCallerPath(t *testing.T) {
	root := filepath.Join(testenv.Dir(t), "SENTINEL-root")
	if err := os.MkdirAll(filepath.Join(root, "full"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "full", "x"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, argv := range [][]string{
		{"init", filepath.Join(root, "sentinel-pkg")},
		{"init", filepath.Join(root, "SENTINEL Pkg")},
		{"init", filepath.Join(root, "SENTINEL") + string(filepath.Separator) + ".."},
		{"init", filepath.Join(root, "full")},
		{"init", filepath.Join(root, "full", "x")},
		{"init", filepath.Join(root, "dry"), "--dry-run"},
	} {
		for _, mode := range [][]string{{"--json"}, nil} {
			r := runTool(t, append(slices.Clone(argv), mode...)...)
			if out := strings.ToLower(r.stdout + r.stderr); strings.Contains(out, "sentinel") || strings.Contains(out, strings.ToLower(root)) {
				t.Errorf("metais %q printed a part of its path:\n%s", argv, r.stdout+r.stderr)
			}
		}
	}
}

// TestInitFailedListing is W3-8.5's report: a failed write is
// output.destination.write_failed carrying the files already written, in
// order, which human mode lists after the finding.
func TestInitFailedListing(t *testing.T) {
	s := &session{table: table, line: line{cmd: lookup(table, "init")}}
	written := scaffoldFiles[:3]
	o := s.initFailed(&author.WriteError{Op: "create", Reason: hostfs.Exists, Written: written})
	if len(o.findings) != 1 || o.findings[0].Code() != finding.CodeDestinationWriteFailed {
		t.Fatalf("findings %v", o.findings)
	}
	if v, _ := o.findings[0].Param("reason"); v.Type() != finding.TypeString {
		t.Errorf("reason %v", v)
	} else if s, _ := v.Str(); s != "exists" {
		t.Errorf("reason %q, want exists", s)
	}
	want := "\nWritten before the failure, and left in place:\n  .gitattributes\n  metais.yaml\n  inputs/greeting.schema.json\n"
	if string(o.human) != want {
		t.Errorf("human %q, want %q", o.human, want)
	}
	if len(o.next) != 1 || o.next[0] != nextInitAgain {
		t.Errorf("next %+v", o.next)
	}
	for r, want := range map[hostfs.Reason]string{
		hostfs.PermissionDenied: "permission_denied", hostfs.NotFound: "not_found", hostfs.IOError: "io_error",
		hostfs.Exists: "exists", hostfs.Link: "exists", hostfs.NotDirectory: "exists",
		hostfs.Changed: "io_error", hostfs.BadName: "io_error",
	} {
		if got := render.WriteReason(r); got != want {
			t.Errorf("render.WriteReason(%s) = %s, want %s", r, got, want)
		}
	}
	// Names found in a directory are host text, neutralised as findings
	// neutralise it (ADR-006 D9 item 9).
	o = s.initFailed(&author.NotEmptyError{Entries: []string{"a\x1b[31m", "b‮"}, More: true})
	if got := string(o.human); got != "\nFound in the directory:\n  a\\u001b[31m\n  b\\u202e\n  and more\n" {
		t.Errorf("human %q", got)
	}
}

// TestInitWriteFailed is output.destination.nothing_written through the
// command line: a directory init may not write into stops it before it
// creates anything, with status 5, nothing written and nothing removed. It needs a host that
// refuses the write; one that does not (an administrator, a platform
// without POSIX modes) is reported as skipped by its probe.
func TestInitWriteFailed(t *testing.T) {
	root := testenv.Dir(t)
	locked := filepath.Join(root, "locked")
	if err := os.Mkdir(locked, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o755) })
	if f, err := os.Create(filepath.Join(locked, "probe")); err == nil {
		f.Close()
		t.Skip("requires a directory this process may not write into; this host lets it write")
	}
	r := runTool(t, "init", filepath.Join(locked, "pkg"), "--json")
	if r.status != finding.ExitDestination || !slices.Equal(r.codes(), []string{string(finding.CodeDestinationNothingWritten)}) {
		t.Fatalf("status %d, %s", r.status, r.stdout)
	}
	if p := r.doc.Findings[0].Params; p["reason"] != "permission_denied" || len(p) != 1 {
		t.Errorf("params %v", p)
	}
}
