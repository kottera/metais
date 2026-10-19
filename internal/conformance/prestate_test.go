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

// The runner's own proof of the unwritable-destination pre-state and its
// probe (ADR-011 D9 items 4 and 6, A2; D18): the clean fake, which writes
// as its script says, is refused by the directory the setup locks; the
// same vector without the lock fails the clean fake, so it is the
// pre-state that refuses the write; a host whose probe fails skips the
// case and reports it, never counting it a pass; and a setup that locks a
// directory without the requirement is a harness error.

// unwritableVector reads fix/unwritable from the fixture suite.
func unwritableVector(t *testing.T, suite string) *vector {
	t.Helper()
	v, err := readVector(filepath.Join(suite, "vectors", filepath.FromSlash(fixUnwritable)))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// TestUnwritablePreState is the pre-state and its control, through each
// adapter, on a host the probe admits.
func TestUnwritablePreState(t *testing.T) {
	t.Parallel()
	if ok, why := probeRequirement(testenv.Dir(t), "unwritable-dirs"); !ok {
		t.Skipf("requires unwritable-dirs: %s", why)
	}
	suite := writeFixture(t, testenv.Dir(t))
	for _, kind := range both {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			r := fixtureRunner(t, suite, kind, "clean")
			if ps := runVector(r, suite, fixUnwritable, testenv.Dir(t)); len(ps) > 0 {
				t.Errorf("the clean fake fails fix/unwritable: %v", ps)
			}
			// The control: without the lock, the clean fake writes the
			// tree, and the step that expects the refusal fails.
			v := unwritableVector(t, suite)
			v.Cases[0].Setup.Unwritable = nil
			res := r.runCase(v, &v.Cases[0], testenv.Dir(t))
			if !slices.ContainsFunc(res.problems, func(p problem) bool { return p.check == checkOK }) {
				t.Errorf("without the unwritable directory the clean fake passed fix/unwritable: %v", res.problems)
			}
		})
	}
}

// TestUnwritableSandboxIsLockedAndUnlocked holds the sandbox to the
// pre-state: after the setup the directory has mode 0555 and refuses a
// file, beneath the work directory, while the package beside it was
// copied; after the run its mode is back, so the sandbox can be removed.
func TestUnwritableSandboxIsLockedAndUnlocked(t *testing.T) {
	t.Parallel()
	if ok, why := probeRequirement(testenv.Dir(t), "unwritable-dirs"); !ok {
		t.Skipf("requires unwritable-dirs: %s", why)
	}
	suite := writeFixture(t, testenv.Dir(t))
	v := unwritableVector(t, suite)
	sb, err := newSandbox(testenv.Dir(t), envB)
	if err != nil {
		t.Fatal(err)
	}
	if err := sb.applySetup(v, &v.Cases[0]); err != nil {
		t.Fatal(err)
	}
	locked := filepath.Join(sb.work, "locked")
	fi, err := os.Stat(locked)
	if err != nil || fi.Mode().Perm() != unwritableMode {
		t.Fatalf("the locked directory: %v, %v", fi, err)
	}
	if err := os.WriteFile(filepath.Join(locked, "x"), nil, 0o644); err == nil {
		t.Error("a file was created in the locked directory")
	}
	if _, err := os.Stat(filepath.Join(sb.work, "package", "fake.json")); err != nil {
		t.Errorf("the package was not copied: %v", err)
	}
	if err := sb.unlock(); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(locked); fi.Mode().Perm() != 0o755 {
		t.Errorf("after unlock the directory's mode is %v", fi.Mode().Perm())
	}
}

// TestUnmetRequirementIsSkippedAndReported is D9 item 6 on a host that
// lacks unwritable-dirs, which a probe that fails stands for: the case is
// skipped with the reason before any sandbox is made, the suite reports
// it as a skip in the record and not as a passing row, and the self-proof
// counts a skip as a problem.
func TestUnmetRequirementIsSkippedAndReported(t *testing.T) {
	t.Parallel()
	suite := writeFixture(t, testenv.Dir(t))
	// A suite of fix/unwritable alone.
	for _, id := range fixtureIDs {
		if err := os.RemoveAll(filepath.Join(suite, "vectors", filepath.FromSlash(id))); err != nil {
			t.Fatal(err)
		}
	}
	r := fixtureRunner(t, suite, "cli", "clean")
	asked := false
	r.probe = func(dir, req string) (bool, string) {
		asked = asked || req == "unwritable-dirs"
		return req != "unwritable-dirs", "this host's directory modes do not refuse a write"
	}
	v := unwritableVector(t, suite)
	base := testenv.Dir(t)
	res := r.runCase(v, &v.Cases[0], base)
	if !asked || res.skipped != "requires unwritable-dirs: this host's directory modes do not refuse a write" ||
		len(res.problems) > 0 || len(res.runs) > 0 {
		t.Errorf("the case: asked %v, skipped %q, problems %v, runs %d", asked, res.skipped, res.problems, len(res.runs))
	}
	if entries, _ := os.ReadDir(base); len(entries) > 0 {
		t.Errorf("a skipped case made %d entries", len(entries))
	}
	if ps := runVector(r, suite, fixUnwritable, testenv.Dir(t)); !slices.ContainsFunc(ps, func(p problem) bool {
		return p.harness && strings.Contains(p.msg, "skipped: requires unwritable-dirs")
	}) {
		t.Errorf("the self-proof does not count the skip as a problem: %v", ps)
	}
	r.rec = &recorder{path: filepath.Join(testenv.Dir(t), "record.json")}
	t.Run("suite", func(t *testing.T) { runSuite(t, r) })
	if !slices.ContainsFunc(r.rec.skipped, func(s recordSkip) bool { return s.Vector == fixUnwritable && s.Case == "locked" }) ||
		slices.ContainsFunc(r.rec.rows, func(row recordRow) bool { return row.Vector == fixUnwritable }) {
		t.Errorf("the record: skipped %v, rows %v", r.rec.skipped, r.rec.rows)
	}
}

// TestUnwritableNeedsTheRequirement is A2 item 3: a case whose setup
// locks a directory, in a vector that does not require unwritable-dirs,
// is a harness error before anything runs, never a pass or a skip.
func TestUnwritableNeedsTheRequirement(t *testing.T) {
	t.Parallel()
	suite := writeFixture(t, testenv.Dir(t))
	r := fixtureRunner(t, suite, "cli", "clean")
	v := unwritableVector(t, suite)
	v.Requires = nil
	res := r.runCase(v, &v.Cases[0], testenv.Dir(t))
	if res.skipped != "" || len(res.runs) > 0 || !slices.ContainsFunc(res.problems, func(p problem) bool {
		return p.harness && strings.Contains(p.msg, "does not require unwritable-dirs")
	}) {
		t.Errorf("skipped %q, runs %d, problems %v", res.skipped, len(res.runs), res.problems)
	}
}
