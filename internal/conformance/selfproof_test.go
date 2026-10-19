// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kottera/metais/internal/testenv"
)

// fixtureIDs are every fixture vector.
var fixtureIDs = []string{fixRenderTree, fixBounded, fixRefuse, fixValidate, fixRenderStream, fixRenderDry, fixRepeatable}

// selfProofWall is the wall clock the self-proof runs its processes
// under, clean and faulty alike, so the wall fault is caught in seconds.
// The rows that prove a resource bound are the exception (rowWall).
const selfProofWall = 5 * time.Second

// accruing are the faults whose proof needs the fake to accrue a resource
// past its bound: CPU time and resident memory accrue only as fast as the
// host schedules the fake, so under a wall clock they would race it.
var accruing = []string{"cpu", "rss"}

// rowWall is the wall clock the self-proof's row for fault runs under:
// short for every row but an accruing one, which runs under no wall clock
// that can fire before the test's own deadline. The runner checks a
// process's bounds only as it finds them when the process ends, and a
// starved fake killed by the wall clock has not yet used what its fault
// burns, so a short wall would make that row's verdict a matter of
// scheduler share. Past the deadline go test fails the whole binary, so
// the row ends with its own check or with the test's timeout, never with
// the wall check.
func rowWall(t *testing.T, fault string, short time.Duration) time.Duration {
	if !slices.Contains(accruing, fault) {
		return short
	}
	if d, ok := t.Deadline(); ok {
		return time.Until(d) + time.Minute
	}
	return 24 * time.Hour
}

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
	{"ab-clock", fixRenderTree, checkAB, both},
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
// test. Its wall clock is selfProofWall.
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

// runVector runs every case of the fixture vector id in sandboxes below
// base and returns its problems. A skipped case is a harness problem: no
// fixture requires anything.
func runVector(r *runner, suite, id, base string) []problem {
	v, err := readVector(filepath.Join(suite, "vectors", filepath.FromSlash(id)))
	if err != nil {
		return []problem{{check: checkHarness, harness: true, msg: err.Error()}}
	}
	var ps []problem
	for ci := range v.Cases {
		dir := filepath.Join(base, id, v.Cases[ci].Name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return append(ps, problem{check: checkHarness, harness: true, msg: err.Error()})
		}
		res := r.runCase(v, &v.Cases[ci], dir)
		if res.skipped != "" {
			ps = append(ps, problem{check: checkHarness, harness: true, msg: v.Cases[ci].Name + " skipped: " + res.skipped})
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
	t.Parallel()
	selfProof(t, selfProofWall, true, func(fault) bool { return true })
}

// TestSelfProofAccruingRowsIgnoreWall starves the accruing faults of the
// time they need: their rows run with the self-proof's wall clock at half
// the CPU bound the cpu fault exceeds, which a fake burning on one thread
// cannot outrun. Each must still fail on its own check, so its verdict
// does not hang on how much of the host the fake was given.
func TestSelfProofAccruingRowsIgnoreWall(t *testing.T) {
	t.Parallel()
	selfProof(t, fakeCPUBound()/2, false, func(f fault) bool { return slices.Contains(accruing, f.name) })
}

// selfProof runs the clean fake, when clean is set, and every fault keep
// selects, through each of its adapters, with short as every row's wall
// clock but an accruing one's (rowWall).
//
// The runs go concurrently, each a few processes that mostly wait out
// environment B's two seconds, and are reported afterwards, one subtest
// per run.
func selfProof(t *testing.T, short time.Duration, clean bool, keep func(fault) bool) {
	suite := writeFixture(t, testenv.Dir(t))
	type run struct {
		name   string
		f      *fault // nil for the clean fake
		vector string
		r      *runner
		base   string
		ps     []problem
		took   time.Duration
	}
	var runs []*run
	for _, kind := range both {
		if !clean {
			break
		}
		r := fixtureRunner(t, suite, kind, "clean")
		r.wall = short
		for _, id := range fixtureIDs {
			runs = append(runs, &run{name: "clean/" + kind + "/" + id, vector: id, r: r, base: testenv.Dir(t)})
		}
	}
	for i := range faults {
		if !keep(faults[i]) {
			continue
		}
		for _, kind := range faults[i].adapters {
			r := fixtureRunner(t, suite, kind, faults[i].name)
			r.wall = rowWall(t, faults[i].name, short)
			runs = append(runs, &run{name: faults[i].name + "/" + kind, f: &faults[i], vector: faults[i].vector,
				r: r, base: testenv.Dir(t)})
		}
	}
	var wg sync.WaitGroup
	for _, rn := range runs {
		wg.Go(func() {
			start := time.Now()
			defer func() { rn.took = time.Since(start) }()
			rn.ps = runVector(rn.r, suite, rn.vector, rn.base)
		})
	}
	wg.Wait()
	for _, rn := range runs {
		t.Run(rn.name, func(t *testing.T) {
			t.Logf("ran in %v under a %v wall clock", rn.took.Round(time.Millisecond), rn.r.wall.Round(time.Millisecond))
			for _, p := range rn.ps {
				t.Log(p)
			}
			if rn.f == nil {
				for _, p := range rn.ps {
					t.Errorf("the clean fake fails: %s", p)
				}
				return
			}
			if !slices.ContainsFunc(rn.ps, func(p problem) bool { return p.check == rn.f.check }) {
				var got []string
				for _, p := range rn.ps {
					got = append(got, p.String())
				}
				t.Errorf("the fake with fault %s passed the %s check of %s; the runner reported:\n%s",
					rn.f.name, rn.f.check, rn.f.vector, strings.Join(got, "\n"))
			}
		})
	}
}

// TestBStartsTwoSecondsAfterA pins ADR-011 D12 item 4's offset directly:
// a fake that writes the clock to the nanosecond into every document
// shows that each clock reading of B's processes is at least two seconds
// after every one of A's, through each adapter. The ab-clock row proves
// the offset catches a fault; this proves the offset, which a row of
// second resolution can see only when A and B straddle a second.
func TestBStartsTwoSecondsAfterA(t *testing.T) {
	t.Parallel()
	suite := writeFixture(t, testenv.Dir(t))
	v, err := readVector(filepath.Join(suite, "vectors", filepath.FromSlash(fixRenderTree)))
	if err != nil {
		t.Fatal(err)
	}
	clock := regexp.MustCompile(`clock (\d+)`)
	readings := func(t *testing.T, run *envRun) []time.Time {
		t.Helper()
		var out []time.Time
		for _, st := range run.steps {
			for _, inv := range st.invs {
				m := clock.FindSubmatch(inv.stdout)
				if m == nil {
					t.Fatalf("%s wrote no clock reading: %q", inv.label, inv.stdout)
				}
				n, err := strconv.ParseInt(string(m[1]), 10, 64)
				if err != nil {
					t.Fatal(err)
				}
				out = append(out, time.Unix(0, n))
			}
		}
		if len(out) == 0 {
			t.Fatal("no invocation ran")
		}
		return out
	}
	for _, kind := range both {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			r := fixtureRunner(t, suite, kind, "clock")
			dir := testenv.Dir(t)
			res := r.runCase(v, &v.Cases[0], dir)
			a, b := res.runs[envA], res.runs[envB]
			if a == nil || b == nil || a.fatal() || b.fatal() {
				t.Fatalf("a run did not finish: %v", res.problems)
			}
			last := slices.MaxFunc(readings(t, a), time.Time.Compare)
			first := slices.MinFunc(readings(t, b), time.Time.Compare)
			if gap := first.Sub(last); gap < 2*time.Second {
				t.Errorf("B's first clock reading is %v after A's last, want at least 2s", gap)
			}
		})
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
