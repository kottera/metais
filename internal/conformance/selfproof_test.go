// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kottera/metais/internal/testenv"
)

// fixtureIDs are every fixture vector.
var fixtureIDs = []string{fixRenderTree, fixBounded, fixRefuse, fixValidate, fixRenderStream, fixRenderDry, fixRepeatable}

// selfProofWall is the wall clock the self-proof runs every process
// under, clean and faulty alike, so the wall fault is caught in seconds.
const selfProofWall = 8 * time.Second

// fault is one fault of ADR-011 D18 item 1: the fake breaks one thing,
// on one fixture vector, and the runner must fail it on one check.
type fault struct {
	name     string
	vector   string
	check    string
	adapters []string
}

var both = []string{"cli", "exec"}

// faults is one fault per check the runner makes.
var faults = []fault{
	{"ok", fixValidate, checkOK, both},
	{"exit", fixRenderTree, checkExit, []string{"cli"}},
	{"finding-order", fixValidate, checkFindings, both},
	{"finding-member", fixValidate, checkFindings, both},
	{"finding-extra", fixValidate, checkFindings, both},
	{"finding-missing", fixValidate, checkFindings, both},
	{"tree-extra", fixRenderTree, checkTree, both},
	{"tree-missing", fixRenderTree, checkTree, both},
	{"tree-changed", fixRenderTree, checkTree, both},
	{"tree-emptydir", fixRenderTree, checkTree, both},
	{"tree-link", fixRenderTree, checkTree, both},
	{"executable", fixRenderTree, checkTree, both},
	{"zip-order", fixRenderTree, checkContainer, both},
	{"zip-content", fixRenderTree, checkContainer, both},
	{"zip-method", fixRenderTree, checkContainer, both},
	{"zip-header", fixRenderTree, checkContainer, both},
	{"write-outside", fixRenderTree, checkSnapshot, both},
	{"tmp-left", fixRenderTree, checkSnapshot, both},
	{"sentinel-stdout", fixRenderTree, checkSentinel, both},
	{"sentinel-stderr", fixRenderTree, checkSentinel, both},
	{"sentinel-file", fixRenderTree, checkSentinel, both},
	{"sentinel-zip", fixRenderTree, checkSentinel, both},
	{"sentinel-zip-nested", fixRenderTree, checkSentinel, both},
	{"sentinel-tar", fixRenderStream, checkSentinel, both},
	{"stream", fixRenderStream, checkStream, both},
	{"listing", fixRenderDry, checkListing, both},
	{"ab", fixRenderTree, checkAB, both},
	{"repeatable", fixRepeatable, checkRepeat, both},
	{"cpu", fixBounded, checkCPU, both},
	{"rss", fixBounded, checkRSS, both},
	{"wall", fixRenderTree, checkWall, both},
	{"driver-malformed", fixRenderTree, checkDriver, []string{"exec"}},
	{"driver-exit", fixRenderTree, checkDriver, []string{"exec"}},
}

// fakeProgram makes the fake for fault below base and returns its path:
// this test binary, linked as metais in a directory named for the fault.
func fakeProgram(t testing.TB, base, fault string) string {
	t.Helper()
	dir := filepath.Join(base, fakeDirPrefix+fault)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, toolName)
	if err := os.Symlink(testBinary(t), p); err != nil {
		t.Fatal(err)
	}
	return p
}

// fixtureRunner makes a runner over the fixture suite against the fake
// for fault, through adapter kind, with both environments whatever the
// test flags say: the self-proof's detectors and controls run in every go
// test.
func fixtureRunner(t testing.TB, suite, kind, fault string) *runner {
	t.Helper()
	a, err := newAdapter(kind, fakeProgram(t, testenv.Dir(t), fault), suite)
	if err != nil {
		t.Fatal(err)
	}
	r := newRunner(t, suite, a)
	r.short = false
	r.wall = selfProofWall
	return r
}

// runVector runs every case of the fixture vector id and returns its
// problems.
func runVector(t testing.TB, r *runner, suite, id string) []problem {
	t.Helper()
	v, err := readVector(filepath.Join(suite, "vectors", filepath.FromSlash(id)))
	if err != nil {
		t.Fatal(err)
	}
	var ps []problem
	for ci := range v.Cases {
		res := r.runCase(v, &v.Cases[ci], testenv.Dir(t))
		if res.skipped != "" {
			t.Fatalf("%s/%s skipped: %s", id, v.Cases[ci].Name, res.skipped)
		}
		ps = append(ps, res.problems...)
	}
	return ps
}

// TestSelfProof is the runner's own proof (ADR-011 D18): the clean fake
// passes every fixture vector through each adapter, and each faulty fake
// fails its vector on the one check its fault breaks. It needs no vector
// of the suite.
func TestSelfProof(t *testing.T) {
	suite := writeFixture(t, testenv.Dir(t))
	for _, kind := range both {
		t.Run("clean/"+kind, func(t *testing.T) {
			t.Parallel()
			r := fixtureRunner(t, suite, kind, "clean")
			for _, id := range fixtureIDs {
				for _, p := range runVector(t, r, suite, id) {
					t.Errorf("the clean fake fails %s: %s", id, p)
				}
			}
		})
	}
	for _, f := range faults {
		for _, kind := range f.adapters {
			t.Run(f.name+"/"+kind, func(t *testing.T) {
				t.Parallel()
				r := fixtureRunner(t, suite, kind, f.name)
				ps := runVector(t, r, suite, f.vector)
				if !slices.ContainsFunc(ps, func(p problem) bool { return p.check == f.check }) {
					var got []string
					for _, p := range ps {
						got = append(got, p.String())
					}
					t.Errorf("the fake with fault %s passed the %s check of %s; the runner reported:\n%s",
						f.name, f.check, f.vector, strings.Join(got, "\n"))
				}
			})
		}
	}
}

// TestSelfProofCoversEveryCheck holds the fault table to the runner's
// checks: every check but the harness's own has a fault, so a check added
// without one fails here.
func TestSelfProofCoversEveryCheck(t *testing.T) {
	checks := []string{checkOK, checkExit, checkFindings, checkTree, checkStream, checkListing, checkSnapshot,
		checkSentinel, checkAB, checkRepeat, checkCPU, checkRSS, checkWall, checkDriver, checkContainer}
	for _, c := range checks {
		if !slices.ContainsFunc(faults, func(f fault) bool { return f.check == c }) {
			t.Errorf("no fault breaks the %s check", c)
		}
	}
}
