// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
)

// The runner's own proof of the absent, unreadable and renamed paths of
// the setup and its file-size limit, and of their probes (ADR-011 A4, A6;
// D18): each harness error of A4 items 1 and 3 and A6 items 1, 2 and 4 is
// reported before anything runs, never a pass or a skip; a path listed
// absent is absent when the first step runs, and one the setup makes is a
// harness error; an unreadable path refuses this process a read once the
// setup is in place, is recorded by the snapshot, and has its mode back
// after the run; a renamed path is in place under its new name; a process
// a step starts under the limit is refused a write past it; and a host
// whose probe fails skips the case with the reason.

// fixtureVector reads the fixture vector id from suite.
func fixtureVector(t *testing.T, suite, id string) *vector {
	t.Helper()
	v, err := readVector(filepath.Join(suite, "vectors", filepath.FromSlash(id)))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// TestSetupHarnessErrors is A4 items 1 and 3 and A6 items 1, 2 and 4: an
// unreadable path, a rename or a file-size limit in a vector that does not
// require what makes it so; a rename to a name the committed tree could
// hold, or to more than one segment; and a path listed absent that the
// vector directory holds, or that no step names, are each a harness error,
// before any sandbox is made.
func TestSetupHarnessErrors(t *testing.T) {
	t.Parallel()
	suite := writeFixture(t, testenv.Dir(t))
	r := fixtureRunner(t, suite, "cli", "clean")
	for _, c := range []struct {
		name  string
		setup setup
		want  string
	}{
		{"unreadable without the requirement", setup{Unreadable: []string{"package/fake.json"}},
			"the setup makes a path unreadable, and the vector does not require unreadable-files"},
		{"absent and held", setup{Absent: []string{"package"}},
			"the setup lists package as absent, and the vector directory holds it"},
		{"absent and named by no step", setup{Absent: []string{"elsewhere"}},
			"the setup lists elsewhere as absent, and no step names it"},
		{"a rename without the requirement", setup{Renames: []rename{{"data/first.json", "a:b.json"}}},
			"the setup renames a path, and the vector does not require nonportable-names"},
		{"a rename to a portable name", setup{Renames: []rename{{"data/first.json", "ab.json"}}},
			`the setup renames data/first.json to "ab.json", which is not one segment holding a colon or a backslash`},
		{"a rename to two segments", setup{Renames: []rename{{"data/first.json", "a/b:c.json"}}},
			`the setup renames data/first.json to "a/b:c.json", which is not one segment holding a colon or a backslash`},
		{"a file-size limit without the requirement", setup{FileSizeLimit: 300},
			"the setup sets a file-size limit, and the vector does not require file-size-limit"},
	} {
		t.Run(c.name, func(t *testing.T) {
			v := fixtureVector(t, suite, fixValidate)
			if c.setup.Renames != nil && !strings.Contains(c.want, "require") {
				v.Requires = []string{"nonportable-names"}
			}
			v.Cases[0].Setup = &c.setup
			base := testenv.Dir(t)
			res := r.runCase(v, &v.Cases[0], base)
			if res.skipped != "" || len(res.runs) > 0 || !slices.ContainsFunc(res.problems, func(p problem) bool {
				return p.harness && p.msg == c.want
			}) {
				t.Errorf("skipped %q, runs %d, problems %v", res.skipped, len(res.runs), res.problems)
			}
			if entries, _ := os.ReadDir(base); len(entries) > 0 {
				t.Errorf("a case refused before it ran made %d entries", len(entries))
			}
		})
	}
}

// TestAbsentPath is A4 item 1 through a run: a step's path listed absent
// is left out, the vector with it well formed although the vector
// directory does not hold it, and the case runs; one the setup itself
// makes is a harness error before the first step.
func TestAbsentPath(t *testing.T) {
	t.Parallel()
	suite := writeFixture(t, testenv.Dir(t))
	r := fixtureRunner(t, suite, "cli", "clean")
	set, err := suiteSchemas(suite)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("left out", func(t *testing.T) {
		v := fixtureVector(t, suite, fixRenderTree)
		v.Cases[0].Setup = &setup{Absent: []string{"out"}}
		if res := r.runCase(v, &v.Cases[0], testenv.Dir(t)); len(res.problems) > 0 || res.skipped != "" {
			t.Errorf("skipped %q, problems %v", res.skipped, res.problems)
		}
		if ps := referenceProblems(v.ID, v, set); len(ps) > 0 {
			t.Errorf("the vector is not well formed: %v", ps)
		}
	})
	t.Run("made by the setup", func(t *testing.T) {
		v := fixtureVector(t, suite, fixRenderTree)
		v.Cases[0].Setup = &setup{Absent: []string{"out"}, Dirs: []string{"out"}}
		res := r.runCase(v, &v.Cases[0], testenv.Dir(t))
		if !slices.ContainsFunc(res.problems, func(p problem) bool {
			return p.harness && strings.Contains(p.msg, "the setup lists out as absent, and the work directory holds it before the first step")
		}) {
			t.Errorf("problems %v", res.problems)
		}
	})
	t.Run("named and missing", func(t *testing.T) {
		v := fixtureVector(t, suite, fixValidate)
		v.Cases[0].Steps[0].args.Data = "absent-data"
		if ps := referenceProblems(v.ID, v, set); !slices.ContainsFunc(ps, func(p string) bool { return strings.Contains(p, "absent-data is missing") }) {
			t.Errorf("a missing path not listed absent passed: %v", ps)
		}
		v.Cases[0].Setup = &setup{Absent: []string{"absent-data"}}
		if ps := referenceProblems(v.ID, v, set); slices.ContainsFunc(ps, func(p string) bool { return strings.Contains(p, "absent-data") }) {
			t.Errorf("a missing path listed absent is reported: %v", ps)
		}
	})
}

// TestUnreadableSandboxIsSealedAndUnsealed holds the sandbox to A4 item
// 2's pre-state: after the setup the file has mode 0000 and this process
// cannot read it, and the snapshot records it; after the run its mode is
// back.
func TestUnreadableSandboxIsSealedAndUnsealed(t *testing.T) {
	t.Parallel()
	if ok, why := probeRequirement(testenv.Dir(t), "unreadable-files"); !ok {
		t.Skipf("requires unreadable-files: %s", why)
	}
	suite := writeFixture(t, testenv.Dir(t))
	v := fixtureVector(t, suite, fixValidate)
	v.Requires = []string{"unreadable-files"}
	v.Cases[0].Setup = &setup{Unreadable: []string{"data", "package/fake.json"}}
	sb, err := newSandbox(testenv.Dir(t), envB)
	if err != nil {
		t.Fatal(err)
	}
	if err := sb.applySetup(v, &v.Cases[0]); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"data", "package/fake.json"} {
		p := filepath.Join(sb.work, filepath.FromSlash(rel))
		if fi, err := os.Lstat(p); err != nil || fi.Mode().Perm() != unreadableMode {
			t.Errorf("%s: %v, %v", rel, fi, err)
		}
		if f, err := os.Open(p); err == nil {
			f.Close()
			t.Errorf("%s was opened", rel)
		}
	}
	snap, err := sb.snapshot()
	if err != nil {
		t.Fatalf("the snapshot of a sealed sandbox: %v", err)
	}
	for _, rel := range []string{"work/data", "work/package/fake.json"} {
		if snap[rel].kind != "sealed" {
			t.Errorf("the snapshot records %s as %+v", rel, snap[rel])
		}
	}
	if _, ok := snap["work/data/first.json"]; ok {
		t.Error("the snapshot read into a sealed directory")
	}
	if err := sb.unlock(); err != nil {
		t.Fatal(err)
	}
	for rel, mode := range map[string]os.FileMode{"data": 0o755, "package/fake.json": 0o644} {
		if fi, _ := os.Lstat(filepath.Join(sb.work, filepath.FromSlash(rel))); fi.Mode().Perm() != mode {
			t.Errorf("after unlock %s has mode %v", rel, fi.Mode().Perm())
		}
	}
}

// TestUnreadableUnmetIsSkipped is A4 item 3 on a host that lacks
// unreadable-files, which a probe that fails stands for: the case is
// skipped with the reason, never run and never counted a pass.
func TestUnreadableUnmetIsSkipped(t *testing.T) {
	t.Parallel()
	suite := writeFixture(t, testenv.Dir(t))
	r := fixtureRunner(t, suite, "cli", "clean")
	r.probe = func(dir, req string) (bool, string) {
		return req != "unreadable-files", "this host's file modes do not refuse a read"
	}
	v := fixtureVector(t, suite, fixValidate)
	v.Requires = []string{"unreadable-files"}
	v.Cases[0].Setup = &setup{Unreadable: []string{"package/fake.json"}}
	res := r.runCase(v, &v.Cases[0], testenv.Dir(t))
	if res.skipped != "requires unreadable-files: this host's file modes do not refuse a read" || len(res.problems) > 0 || len(res.runs) > 0 {
		t.Errorf("skipped %q, problems %v, runs %d", res.skipped, res.problems, len(res.runs))
	}
}

// TestRenamedPath holds the sandbox to A6 item 1's pre-state: after the
// setup the file is in its directory under its new name, holding a colon,
// and not under its old one.
func TestRenamedPath(t *testing.T) {
	t.Parallel()
	if ok, why := probeRequirement(testenv.Dir(t), "nonportable-names"); !ok {
		t.Skipf("requires nonportable-names: %s", why)
	}
	suite := writeFixture(t, testenv.Dir(t))
	v := fixtureVector(t, suite, fixValidate)
	v.Requires = []string{"nonportable-names"}
	v.Cases[0].Setup = &setup{Renames: []rename{{"data/first.json", "fir:st.json"}}}
	sb, err := newSandbox(testenv.Dir(t), envB)
	if err != nil {
		t.Fatal(err)
	}
	if err := sb.applySetup(v, &v.Cases[0]); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(sb.work, "data"))
	if err != nil || len(entries) != 1 || entries[0].Name() != "fir:st.json" {
		t.Errorf("the data directory holds %v, %v", entries, err)
	}
}

// TestFileSizeLimit holds a step's process to A6 item 3: under the case's
// limit, a write past it is refused and the process goes on to report it,
// and the file holds no more than the limit; with no limit the command
// line is the step's own.
func TestFileSizeLimit(t *testing.T) {
	t.Parallel()
	if ok, why := probeRequirement(testenv.Dir(t), "file-size-limit"); !ok {
		t.Skipf("requires file-size-limit: %s", why)
	}
	sb, err := newSandbox(testenv.Dir(t), envB)
	if err != nil {
		t.Fatal(err)
	}
	head, err := exec.LookPath("head")
	if err != nil {
		t.Skipf("no head program to write a file: %v", err)
	}
	// The step's PATH is the sandbox's empty bin, so head is named whole.
	script := []string{"-c", head + " -c 600 /dev/zero > big || echo refused"}
	if path, args, err := sb.limited("/bin/sh", script); err != nil || path != "/bin/sh" || !slices.Equal(args, script) {
		t.Errorf("with no limit the command line is %q %q, %v", path, args, err)
	}
	sb.fsize = 300
	path, args, err := sb.limited("/bin/sh", script)
	if err != nil {
		t.Fatal(err)
	}
	p, err := runProc(path, args, sb.work, sb.environ(), nil, defaultWall)
	if err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(filepath.Join(sb.work, "big"))
	if p.exit != 0 || strings.TrimSpace(string(p.stdout)) != "refused" || err != nil || fi.Size() > 300 {
		t.Errorf("exit %d, stdout %q, stderr %q, the file %v, %v", p.exit, p.stdout, p.stderr, fi, err)
	}
}
