// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// The runner's own proof of the absent and unreadable paths of the setup,
// and of the unreadable-files probe (ADR-011 A4; D18): each harness error
// of A4 items 1 and 3 is reported before anything runs, never a pass or a
// skip; a path listed absent is absent when the first step runs, and one
// the setup makes is a harness error; an unreadable path refuses this
// process a read once the setup is in place, is recorded by the snapshot,
// and has its mode back after the run; and a host whose probe fails skips
// the case with the reason.

// fixtureVector reads the fixture vector id from suite.
func fixtureVector(t *testing.T, suite, id string) *vector {
	t.Helper()
	v, err := readVector(filepath.Join(suite, "vectors", filepath.FromSlash(id)))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// TestSetupHarnessErrors is A4 items 1 and 3: an unreadable path in a
// vector that does not require unreadable-files, a path listed absent
// that the vector directory holds, and one that no step names are each a
// harness error, before any sandbox is made.
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
	} {
		t.Run(c.name, func(t *testing.T) {
			v := fixtureVector(t, suite, fixValidate)
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
