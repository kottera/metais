// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/testenv"
)

// packSource copies the source directory of the vector id's tree dir into
// a new directory and returns it.
func packSource(t *testing.T, id, dir string) string {
	t.Helper()
	src := filepath.Join(vectorsDir, filepath.FromSlash(id), filepath.FromSlash(dir))
	to := filepath.Join(testenv.Dir(t), "src")
	if err := os.CopyFS(to, os.DirFS(src)); err != nil {
		t.Fatal(err)
	}
	return to
}

// packOf runs metais pack --json over argv and returns the run and its
// result, failing unless it packed.
func packOf(t *testing.T, argv ...string) (ran, packResult) {
	t.Helper()
	r := runTool(t, append([]string{"pack", "--json"}, argv...)...)
	if !r.doc.OK || r.status != 0 {
		t.Fatalf("metais pack %q: status %d, %s", argv, r.status, r.stdout)
	}
	var res packResult
	if err := json.Unmarshal(r.doc.Result, &res); err != nil {
		t.Fatal(err)
	}
	return r, res
}

// TestPackDefaultName is ADR-008 D9 pack item 2 and item 4: without
// --out, the package is <name>-<version>.metais in the working directory,
// and the result names it; with an --out naming an existing directory it
// is that file inside it, named in the result too; with an --out naming
// the file, the result names none, since the caller's path is never
// echoed.
func TestPackDefaultName(t *testing.T) {
	testenv.Home(t)
	src := packSource(t, "aut/pack-refused", "package")
	work := testenv.Dir(t)
	if _, err := loadSchemas(); err != nil {
		// Compiled before the working directory moves from the schemas.
		t.Fatal(err)
	}
	t.Chdir(work)
	_, res := packOf(t, src)
	if res.File != "refused-1.0.0.metais" || res.DryRun {
		t.Errorf("result %+v", res)
	}
	b, err := os.ReadFile(filepath.Join(work, "refused-1.0.0.metais"))
	if err != nil || int64(len(b)) != res.Size {
		t.Fatalf("the default file: %v, %d bytes, result %d", err, len(b), res.Size)
	}
	dir := filepath.Join(work, "dist")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	_, res = packOf(t, src, "--out", dir)
	if got, err := os.ReadFile(filepath.Join(dir, "refused-1.0.0.metais")); err != nil || !bytes.Equal(got, b) || res.File != "refused-1.0.0.metais" {
		t.Errorf("into a directory: %v, equal %v, result %+v", err, bytes.Equal(got, b), res)
	}
	sentinel := filepath.Join(work, "CALLER-SENTINEL.metais")
	r, res := packOf(t, src, "--out", sentinel)
	if got, err := os.ReadFile(sentinel); err != nil || !bytes.Equal(got, b) || res.File != "" || strings.Contains(r.stdout, "SENTINEL") || strings.Contains(r.stdout, "\"file\"") {
		t.Errorf("into a named file: %v, equal %v, result %+v, stdout %s", err, bytes.Equal(got, b), res, r.stdout)
	}
	r = runTool(t, "pack", src, "--out", sentinel)
	if r.status != 0 || strings.Contains(r.stdout+r.stderr, "SENTINEL") || !strings.HasPrefix(r.stdout, "Packed the package: ") {
		t.Errorf("human, into a named file: status %d, stdout %q, stderr %q", r.status, r.stdout, r.stderr)
	}
	// A directory at the default name is not one to write in: without
	// --out, the file is the name in the working directory.
	os.Remove(filepath.Join(work, "refused-1.0.0.metais"))
	if err := os.Mkdir(filepath.Join(work, "refused-1.0.0.metais"), 0o755); err != nil {
		t.Fatal(err)
	}
	r = runTool(t, "pack", "--json", src)
	if r.status != finding.ExitDestination || !slices.Equal(r.codes(), []string{string(finding.CodeDestinationNothingWritten)}) {
		t.Errorf("a directory at the default name: status %d, codes %q", r.status, r.codes())
	}
	if names, _ := os.ReadDir(filepath.Join(work, "refused-1.0.0.metais")); len(names) != 0 {
		t.Errorf("pack wrote into the directory at the default name: %v", names)
	}
	os.Remove(filepath.Join(work, "refused-1.0.0.metais"))
	r = runTool(t, "pack", src)
	if r.status != 0 || !strings.HasPrefix(r.stdout, "Packed refused-1.0.0.metais: "+strconv.FormatInt(res.Size, 10)+" bytes, 7 entries, package digest "+res.Digest+"\n") {
		t.Errorf("human: status %d, stdout %q", r.status, r.stdout)
	}
}

// TestPackStdout is ADR-008 D9 pack item 3 and D11 items 1 and 2: --out -
// writes the package's bytes to standard output and nothing else, the
// same bytes pack writes to a file, with the document on standard error
// after them; and it is refused when standard output is a terminal, with
// nothing written there.
func TestPackStdout(t *testing.T) {
	testenv.Home(t)
	src := packSource(t, "aut/pack-refused", "package")
	file := filepath.Join(testenv.Dir(t), "p.metais")
	_, want := packOf(t, src, "--out", file)
	b, _ := os.ReadFile(file)
	r := runTool(t, "pack", "--json", src, "--out", "-")
	if r.status != 0 || r.stdout != string(b) || !r.doc.OK {
		t.Fatalf("--out -: status %d, %d bytes on standard output, want %d", r.status, len(r.stdout), len(b))
	}
	var res packResult
	if err := json.Unmarshal(r.doc.Result, &res); err != nil || res.File != "-" || res.Digest != want.Digest || res.Size != int64(len(b)) {
		t.Errorf("--out - result %+v, %v", res, err)
	}
	r = runTool(t, "pack", src, "--out", "-")
	if r.status != 0 || r.stdout != string(b) || !strings.HasPrefix(r.stderr, "Packed the package to standard output: ") {
		t.Errorf("human --out -: status %d, stderr %q", r.status, r.stderr)
	}
	alwaysTerminal(t)
	for _, j := range []bool{true, false} {
		args := []string{"pack", src, "--out", "-"}
		if j {
			args = append(args, "--json")
		}
		r := runTool(t, args...)
		if r.status != finding.ExitUsage || r.stdout != "" || !strings.Contains(r.stderr, "usage.archive_to_terminal") {
			t.Errorf("metais %q to a terminal: status %d, stdout %q, stderr %q", args, r.status, r.stdout, r.stderr)
		}
	}
	// A dry run writes no bytes, so a terminal does not refuse it.
	r = runTool(t, "pack", "--json", src, "--out", "-", "--dry-run")
	if r.status != 0 || !r.doc.OK {
		t.Errorf("--out - --dry-run to a terminal: status %d, %s", r.status, r.stdout)
	}
}

// TestPackPackedOperand is W6-1.9: a packed package given as pack's
// operand is opened first, as every opening command opens it, and then
// refused with usage.not_a_source_directory, and nothing is written; a
// packed package that does not open is refused with the open's findings.
func TestPackPackedOperand(t *testing.T) {
	testenv.Home(t)
	root := testenv.Dir(t)
	pkg := filepath.Join(vectorsDir, "pkg/opens-packed/package.metais")
	before := snapshot(t, filepath.Dir(pkg))
	r := runTool(t, "pack", "--json", pkg)
	if r.status != finding.ExitUsage || !slices.Equal(r.codes(), []string{string(finding.CodeUsageNotSourceDir)}) {
		t.Errorf("a packed package: status %d, codes %q", r.status, r.codes())
	}
	hostile := filepath.Join(vectorsDir, "pkg/digest-mismatch/package.metais")
	want := runTool(t, "inspect", "--json", hostile)
	r = runTool(t, "pack", "--json", hostile, "--out", filepath.Join(root, "p.metais"))
	if r.doc.OK || !slices.Equal(r.codes(), want.codes()) || r.status != want.status {
		t.Errorf("a hostile packed package: %q with %d; inspect %q with %d", r.codes(), r.status, want.codes(), want.status)
	}
	if after := snapshot(t, filepath.Dir(pkg)); !slices.Equal(before, after) {
		t.Errorf("pack wrote beside the package:\n%q\n%q", before, after)
	}
	if names, _ := os.ReadDir(root); len(names) != 0 {
		t.Errorf("pack wrote %v", names)
	}
}

// TestPackReplace is ADR-007 D7 item 5's atomic replace, which no vector
// can observe: the package file already at the output is renamed over,
// never written into, so a reader holding it open still reads its old
// bytes whole; nothing is left beside it.
func TestPackReplace(t *testing.T) {
	testenv.Home(t)
	src := packSource(t, "aut/pack-refused", "package")
	out := filepath.Join(testenv.Dir(t), "out")
	if err := os.Mkdir(out, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(out, "refused-1.0.0.metais")
	old := []byte("the previous package\n")
	if err := os.WriteFile(target, old, 0o644); err != nil {
		t.Fatal(err)
	}
	held, err := os.Open(target)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	before, _ := held.Stat()
	packOf(t, src, "--out", out)
	after, err := os.Lstat(target)
	if err != nil || os.SameFile(before, after) || !after.Mode().IsRegular() {
		t.Errorf("the target was written in place, or is not a regular file: %v", err)
	}
	var b bytes.Buffer
	if _, err := b.ReadFrom(held); err != nil || !bytes.Equal(b.Bytes(), old) {
		t.Errorf("the file held open now reads %q", b.Bytes())
	}
	if names, _ := os.ReadDir(out); len(names) != 1 {
		t.Errorf("out holds %v", names)
	}
}

// TestPackDryRun is ADR-008 D9's --dry-run, read as unpack's (W6-1): it
// checks, tests, builds and places the package as pack does, reports the
// size, digest and entries pack would, with dry_run true and the next step
// that writes, and writes nothing anywhere; and it is refused as pack is,
// before writing.
func TestPackDryRun(t *testing.T) {
	testenv.Home(t)
	src := packSource(t, "aut/pack", "package")
	root := testenv.Dir(t)
	file := filepath.Join(root, "p.metais")
	_, want := packOf(t, src, "--out", file)
	os.Remove(file)
	before := snapshot(t, filepath.Dir(src))
	r, res := packOf(t, src, "--out", file, "--dry-run")
	if !res.DryRun || res.Size != want.Size || res.Digest != want.Digest || res.Entries != want.Entries || len(r.doc.Next) != 1 || r.doc.Next[0] != nextPackWrite {
		t.Errorf("dry run %+v, next %v; pack %+v", res, r.doc.Next, want)
	}
	h := runTool(t, "pack", src, "--out", file, "--dry-run")
	if h.status != 0 || !strings.Contains(h.stdout, "Would pack the package: ") || !strings.Contains(h.stdout, "\n  .compiled/digests.json\n") {
		t.Errorf("human dry run: %q", h.stdout)
	}
	if after := snapshot(t, filepath.Dir(src)); !slices.Equal(before, after) {
		t.Errorf("the dry run wrote:\n%q\n%q", before, after)
	}
	r = runTool(t, "pack", "--json", src, "--out", filepath.Join(src, "templates"), "--dry-run")
	if !slices.Contains(r.codes(), string(finding.CodeDestinationInPackage)) {
		t.Errorf("a dry run into a layout tree: %q", r.codes())
	}
}

// TestPackFailOnWarning is R2-134 under pack: --fail-on warning makes a
// warning fail pack, which then writes nothing.
func TestPackFailOnWarning(t *testing.T) {
	testenv.Home(t)
	src := packSource(t, "aut/pack", "package")
	file := filepath.Join(testenv.Dir(t), "p.metais")
	r := runTool(t, "pack", "--json", src, "--out", file, "--fail-on", "warning")
	if r.doc.OK || r.status != finding.ExitFindings || !slices.Contains(r.codes(), "package.layout.not_packed") {
		t.Errorf("status %d, codes %q", r.status, r.codes())
	}
	if _, err := os.Lstat(file); !os.IsNotExist(err) {
		t.Error("pack wrote under --fail-on warning")
	}
}

// TestPackRoundTripBytes is W6-1.5's half no vector can observe, and
// ADR-007 D8 item 5's: the scaffold init writes and the fixture of
// aut/pack, packed, unpacked and packed again by this build, give the same
// bytes.
func TestPackRoundTripBytes(t *testing.T) {
	testenv.Home(t)
	scaffold := filepath.Join(testenv.Dir(t), "acme-report")
	if r := runTool(t, "init", scaffold); r.status != 0 {
		t.Fatalf("init: %s", r.stderr)
	}
	for name, src := range map[string]string{"scaffold": scaffold, "fixture": packSource(t, "aut/pack", "package")} {
		t.Run(name, func(t *testing.T) {
			first := filepath.Join(testenv.Dir(t), "first.metais")
			_, a := packOf(t, src, "--out", first)
			to := filepath.Join(testenv.Dir(t), "src")
			if r := runTool(t, "unpack", first, "--to", to); r.status != 0 {
				t.Fatalf("unpack: %s", r.stderr)
			}
			again := filepath.Join(testenv.Dir(t), "again.metais")
			_, b := packOf(t, to, "--out", again)
			x, _ := os.ReadFile(first)
			y, _ := os.ReadFile(again)
			if !bytes.Equal(x, y) || a.Digest != b.Digest {
				t.Errorf("pack of the unpacked source is %d bytes, %s; the first pack %d bytes, %s", len(y), b.Digest, len(x), a.Digest)
			}
		})
	}
}

// TestPackRefusalsTriggered is ADR-008 D16 item 2 for pack: every refusal
// it declares is check's or test's, triggered by their tests and the
// suite's vectors, or one of its own, triggered here or by its vectors.
func TestPackRefusalsTriggered(t *testing.T) {
	own := map[finding.Code]bool{
		finding.CodeUsageNotSourceDir: true, finding.CodeUsageArchiveToTerminal: true, finding.CodeUsageBadValue: true,
		finding.CodeDestinationInPackage: true, finding.CodeDestinationNotDirectory: true,
		finding.CodeDestinationNothingWritten: true,
	}
	check, test := lookup(table, "check").Refusals, lookup(table, "test").Refusals
	for _, rf := range lookup(table, "pack").Refusals {
		in := func(rs []Refusal) bool { return slices.ContainsFunc(rs, func(r Refusal) bool { return r.Code == rf.Code }) }
		if !own[rf.Code] && !in(check) && !in(test) {
			t.Errorf("pack declares %s, which neither check, test nor its own tests trigger", rf.Code)
		}
	}
}

// TestPackOutputRefusals are the refusals of the output no vector can
// make: an --out whose name hostfs will not resolve, and one that is
// empty.
func TestPackOutputRefusals(t *testing.T) {
	testenv.Home(t)
	src := packSource(t, "aut/pack-refused", "package")
	for _, out := range []string{"", filepath.Join(testenv.Dir(t), "a:b.metais")} {
		r := runTool(t, "pack", "--json", src, "--out", out)
		if r.status != finding.ExitUsage || !slices.Equal(r.codes(), []string{string(finding.CodeUsageBadValue)}) {
			t.Errorf("--out %q: status %d, codes %q", out, r.status, r.codes())
		}
	}
}
