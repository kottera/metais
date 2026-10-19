// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// The checks the runner makes, each of which the self-proof breaks once
// (ADR-011 D18 item 1).
const (
	checkOK        = "ok"
	checkExit      = "exit"
	checkFindings  = "findings"
	checkTree      = "tree"
	checkStream    = "stream"
	checkListing   = "listing"
	checkSnapshot  = "nothing-else-changes"
	checkSentinel  = "sentinel"
	checkAB        = "A/B"
	checkRepeat    = "repeatable"
	checkCPU       = "cpu"
	checkRSS       = "memory"
	checkWall      = "wall"
	checkDriver    = "driver"
	checkHarness   = "harness"
	checkContainer = "container"
)

// problem is one failure of a step: a conformance failure, or a harness
// error, which is reported apart (ADR-011 D11 items 5 and 6).
type problem struct {
	check   string
	harness bool
	step    int    // 1-based; 0 for the case as a whole
	where   string // the invocation, when a step ran several
	msg     string
}

func (p problem) String() string {
	kind := "conformance failure"
	if p.harness {
		kind = "harness error"
	}
	at := ""
	if p.step > 0 {
		at = fmt.Sprintf("step %d", p.step)
		if p.where != "" {
			at += " (" + p.where + ")"
		}
		at += ": "
	}
	return fmt.Sprintf("%s [%s]: %s%s", kind, p.check, at, p.msg)
}

// runner runs vectors against one adapter.
type runner struct {
	suite    string
	adapter  adapter
	version  string   // the highest revision claimed
	classes  []string // the classes to run
	tags     []string // run only vectors with one of these; all when empty
	skipTags []string
	short    bool          // environment A only
	wall     time.Duration // the wall-clock bound of every process
	keep     string        // where failing sandboxes are kept, or ""
	base     string        // a directory for the identity's sandbox

	// probe answers whether a requirement is met, and why not; nil is
	// probeRequirement. Only the self-proof sets it, to stand for a host
	// that lacks what a vector requires.
	probe func(dir, req string) (bool, string)

	idOnce sync.Once
	id     toolID
	idErr  error

	rec *recorder // nil unless a record is written

	// slots bounds the environments run at once, -test.parallel of them
	// unless set; nil bounds none. A case holds a slot only while an
	// environment runs, never through the wait between A and B, so the
	// wait costs no other case its turn and the default run is bound by
	// its work rather than by one wait of two seconds per case (ADR-011
	// D11 item 9).
	slots chan struct{}
	// wait sleeps until B may start; nil is time.Sleep. Only the tests
	// that pin the wait set it.
	wait func(until time.Time)
}

// identity is the tool's identity, asked once.
func (r *runner) identity() (toolID, error) {
	r.idOnce.Do(func() {
		r.id, r.idErr = r.adapter.identity(filepath.Join(r.base, "identity"))
	})
	return r.id, r.idErr
}

// selected says why v is not run, or "" when it is.
func (r *runner) selected(v *vector) string {
	switch {
	case !slices.Contains(r.classes, v.Class):
		return fmt.Sprintf("class %s is not selected", v.Class)
	case laterRevision(v.Since, r.version):
		return fmt.Sprintf("since %s is later than the claimed revision %s", v.Since, r.version)
	case len(r.tags) > 0 && !slices.ContainsFunc(v.Tags, func(t string) bool { return slices.Contains(r.tags, t) }):
		return "no selected tag"
	case slices.ContainsFunc(v.Tags, func(t string) bool { return slices.Contains(r.skipTags, t) }):
		return "a skipped tag"
	}
	return ""
}

// stepRun is what one step gave in one environment.
type stepRun struct {
	invs  []*invocation
	trees map[string]*tree // by the declared tree's path
}

// envRun is one environment's run of a case.
type envRun struct {
	sb       *sandbox
	steps    []stepRun
	problems []problem
}

// fatal reports whether the run stopped early: on a harness error, or on
// a process the wall clock killed, whose step has nothing left to compare.
func (e *envRun) fatal() bool {
	return slices.ContainsFunc(e.problems, func(p problem) bool { return p.harness || p.check == checkWall })
}

// caseResult is the outcome of one case.
type caseResult struct {
	skipped  string
	problems []problem
	runs     map[env]*envRun
}

// runCase runs case c of v in fresh sandboxes below base: environment A,
// then, unless short, environment B at least two seconds after A ended,
// and environment A again when a step is repeatable (ADR-011 D10 item 7,
// D12).
//
// B's two seconds are counted from the end of A's last process, not from
// A's start, so every clock reading B's processes make is at
// least two seconds after every one A's made, however long the host took
// to run A: a tool that writes the clock at second resolution differs
// between the two whatever the load.
func (r *runner) runCase(v *vector, c *vcase, base string) *caseResult {
	res := &caseResult{runs: map[env]*envRun{}}
	if c.Setup != nil && len(c.Setup.Unwritable) > 0 && !slices.Contains(v.Requires, "unwritable-dirs") {
		res.problems = append(res.problems, problem{check: checkHarness, harness: true,
			msg: "the setup makes a directory unwritable, and the vector does not require unwritable-dirs"})
		return res
	}
	probe := r.probe
	if probe == nil {
		probe = probeRequirement
	}
	for _, req := range v.Requires {
		if ok, why := probe(base, req); !ok {
			res.skipped = fmt.Sprintf("requires %s: %s", req, why)
			return res
		}
	}
	a := r.runEnv(v, c, base, envA, true)
	aEnded := time.Now()
	res.runs[envA] = a
	res.problems = append(res.problems, a.problems...)
	if a.fatal() {
		return res
	}
	if slices.ContainsFunc(c.Steps, func(s step) bool { return s.Repeatable }) {
		again := r.runEnv(v, c, base, envA, false)
		res.problems = append(res.problems, again.problems...)
		if !again.fatal() {
			res.problems = append(res.problems, differences(a, again, checkRepeat, nil)...)
		}
	}
	if r.short {
		return res
	}
	if until := aEnded.Add(2 * time.Second); r.wait != nil {
		r.wait(until)
	} else {
		time.Sleep(time.Until(until))
	}
	b := r.runEnv(v, c, base, envB, false)
	res.runs[envB] = b
	res.problems = append(res.problems, b.problems...)
	if !b.fatal() {
		res.problems = append(res.problems, differences(a, b, checkAB, differingVars(a.sb.vars, b.sb.vars))...)
	}
	return res
}

// runEnv runs every step of c in one fresh sandbox of environment e. It
// checks the expectations when expectations is set, and in environment A
// scans for the sentinel.
func (r *runner) runEnv(v *vector, c *vcase, base string, e env, expectations bool) *envRun {
	if r.slots != nil {
		r.slots <- struct{}{}
		defer func() { <-r.slots }()
	}
	run := &envRun{}
	fail := func(si int, check, msg string) {
		run.problems = append(run.problems, problem{check: check, harness: true, step: si, msg: msg})
	}
	sb, err := newSandbox(base, e)
	if err != nil {
		fail(0, checkHarness, err.Error())
		return run
	}
	if r.adapter.race() {
		sb.quietRaceExit()
	}
	run.sb = sb
	defer func() {
		if err := sb.unlock(); err != nil {
			fail(0, checkHarness, "giving an unwritable directory its mode back: "+err.Error())
		}
	}()
	if err := sb.applySetup(v, c); err != nil {
		fail(0, checkHarness, err.Error())
		return run
	}
	if e == envA {
		if err := sb.plantDecoys(v, c); err != nil {
			fail(0, checkHarness, err.Error())
			return run
		}
	}
	for si := range c.Steps {
		st := &c.Steps[si]
		before, err := sb.snapshot()
		if err != nil {
			fail(si+1, checkHarness, err.Error())
			return run
		}
		invs, ps := r.adapter.run(r, sb, v, st)
		for i := range ps {
			ps[i].step = si + 1
		}
		run.problems = append(run.problems, ps...)
		sr := stepRun{invs: invs, trees: map[string]*tree{}}
		var treePaths []string
		for _, te := range st.Expect.Trees {
			t, err := readTree(filepath.Join(sb.work, filepath.FromSlash(te.Path)))
			if err != nil {
				fail(si+1, checkHarness, err.Error())
				return run
			}
			sr.trees[te.Path] = t
			treePaths = append(treePaths, te.Path)
		}
		run.steps = append(run.steps, sr)
		if run.fatal() {
			return run
		}
		after, err := sb.snapshot()
		if err != nil {
			fail(si+1, checkHarness, err.Error())
			return run
		}
		var excluded []string
		if x := r.adapter.excluded(st); x != "" {
			excluded = append(excluded, x)
		}
		for _, ch := range changesOutside(before, after, treePaths, excluded) {
			run.problems = append(run.problems, problem{check: checkSnapshot, step: si + 1, msg: ch + " outside a declared tree"})
		}
		if expectations {
			run.problems = append(run.problems, r.expectations(v, si, st, sr)...)
		}
		if e == envA {
			for _, p := range scanWritten(sb, before, after, append(treePaths, excluded...)) {
				p.step = si + 1
				run.problems = append(run.problems, p)
			}
			archive, _, _ := st.args.stream()
			for _, inv := range invs {
				for _, p := range scanInvocation(inv, archive, sr.trees) {
					p.step, p.where = si+1, inv.label
					run.problems = append(run.problems, p)
				}
			}
		}
	}
	return run
}

// expectations compares what step si gave with what it must give (ADR-011
// D10 items 1, 2, 3, 5 and 6).
func (r *runner) expectations(v *vector, si int, st *step, sr stepRun) []problem {
	var ps []problem
	add := func(check, where, msg string) {
		ps = append(ps, problem{check: check, step: si + 1, where: where, msg: msg})
	}
	var wantFindings []any
	if st.Expect.Findings != "" {
		b, err := os.ReadFile(filepath.Join(v.dir, filepath.FromSlash(st.Expect.Findings)))
		if err == nil {
			var parsed any
			if parsed, err = decodeJSON(b); err == nil {
				var isArr bool
				if wantFindings, isArr = parsed.([]any); !isArr {
					err = fmt.Errorf("not an array")
				}
			}
		}
		if err != nil {
			return append(ps, problem{check: checkHarness, harness: true, step: si + 1, msg: "the expected findings: " + err.Error()})
		}
	}
	var wantStream []byte
	if st.Expect.Stream != nil {
		b, err := os.ReadFile(filepath.Join(v.dir, filepath.FromSlash(st.Expect.Stream.Equals)))
		if err != nil {
			return append(ps, problem{check: checkHarness, harness: true, step: si + 1, msg: "the expected stream: " + err.Error()})
		}
		wantStream = b
	}
	var wantListing any
	if l := st.Expect.Listing; l != nil {
		b, err := os.ReadFile(filepath.Join(v.dir, filepath.FromSlash(l.Equals)))
		if err == nil {
			wantListing, err = decodeJSON(b)
		}
		if err != nil {
			return append(ps, problem{check: checkHarness, harness: true, step: si + 1, msg: "the expected listing: " + err.Error()})
		}
	}
	archive, _, _ := st.args.stream()
	for _, inv := range sr.invs {
		if inv.ok != st.Expect.OK {
			add(checkOK, inv.label, fmt.Sprintf("ok is %t, expected %t", inv.ok, st.Expect.OK))
		}
		if d := compareFindings(wantFindings, inv.findings); d != "" {
			add(checkFindings, inv.label, d)
		}
		if st.Expect.Stream != nil {
			var d string
			switch {
			case inv.stream == nil:
				d = "no stream was written"
			case archive:
				d = compareTars(wantStream, inv.stream)
			case st.Expect.Stream.Container != nil:
				methods, all := containerSpec{Methods: st.Expect.Stream.Container.Methods}.methods()
				if d = compareZips(wantStream, inv.stream, methods, all); d != "" {
					d = "compared as a container: " + d
				}
			case !bytes.Equal(wantStream, inv.stream):
				d = "differs\n" + firstDifference(wantStream, inv.stream)
			}
			if d != "" {
				add(checkStream, inv.label, "the stream "+d)
			}
		}
		if l := st.Expect.Listing; l != nil {
			if d := compareListing(wantListing, inv.listing, l.Containers); d != "" {
				add(checkListing, inv.label, d)
			}
		}
	}
	for _, te := range st.Expect.Trees {
		for _, d := range compareTree(sr.trees[te.Path], v, te, r.identity) {
			check := checkTree
			if strings.Contains(d, "compared as a container") {
				check = checkContainer
			}
			add(check, "", fmt.Sprintf("tree %s: %s", te.Path, d))
		}
	}
	return ps
}

// differences compares every observed byte of two runs of one case:
// standard output and error, streams, listings and trees (ADR-011 D12 item
// 4, D10 item 7). vars names the variables that differ between them.
func differences(a, b *envRun, check string, vars []string) []problem {
	var ps []problem
	why := ""
	if len(vars) > 0 {
		why = "; the environments differ in " + strings.Join(vars, ", ")
	}
	for si := range min(len(a.steps), len(b.steps)) {
		sa, sb := a.steps[si], b.steps[si]
		for ii := range min(len(sa.invs), len(sb.invs)) {
			ia, ib := sa.invs[ii], sb.invs[ii]
			for _, art := range []struct {
				name string
				x, y []byte
			}{
				{"standard output", ia.stdout, ib.stdout},
				{"standard error", ia.stderr, ib.stderr},
				{"the stream", ia.stream, ib.stream},
				{"the listing", marshal(ia.listing), marshal(ib.listing)},
			} {
				if !bytes.Equal(art.x, art.y) {
					ps = append(ps, problem{check: check, step: si + 1, where: ia.label,
						msg: fmt.Sprintf("%s differs between the runs%s\n%s", art.name, why, firstDifference(art.x, art.y))})
				}
			}
		}
		for _, p := range sortedKeys(sa.trees) {
			ta, tb := sa.trees[p], sb.trees[p]
			if tb == nil {
				continue
			}
			names := sortedKeys(ta.files)
			for _, n := range sortedKeys(tb.files) {
				if _, ok := ta.files[n]; !ok {
					names = append(names, n)
				}
			}
			for _, n := range names {
				fa, oka := ta.files[n]
				fb, okb := tb.files[n]
				if oka != okb || fa.exec != fb.exec || !bytes.Equal(fa.content, fb.content) {
					ps = append(ps, problem{check: check, step: si + 1,
						msg: fmt.Sprintf("the written file %s/%s differs between the runs%s\n%s", p, n, why, firstDifference(fa.content, fb.content))})
				}
			}
		}
	}
	return ps
}

// marshal is v's JSON, or nil for nil.
func marshal(v any) []byte {
	if v == nil {
		return nil
	}
	b, _ := json.Marshal(v)
	return b
}

// suiteStats counts what a run did.
type suiteStats struct {
	vectors, cases, skipped atomic.Int64
	notSelected             atomic.Int64
}

// summary is the line a run reports: how many vectors and cases ran and
// how many were skipped. With no vector, it says that it ran none.
func (s *suiteStats) summary() string {
	if s.vectors.Load() == 0 {
		return fmt.Sprintf("ran no vector (%d not selected)", s.notSelected.Load())
	}
	return fmt.Sprintf("ran %d vectors, %d cases; %d cases skipped, %d vectors not selected",
		s.vectors.Load(), s.cases.Load(), s.skipped.Load(), s.notSelected.Load())
}

// runSuite runs every selected vector of r's suite as subtests named
// <area>/<slug>/<case>, each vector in parallel, and reports the summary
// when they are done.
//
// A subtest holds a parallel slot for as long as it runs, so a case run
// inside one holds it through the two seconds between A and B as well.
// So each case starts its run before its subtest waits for a slot, and
// the subtest only waits for the result and reports it; and an area and
// a vector start their cases before they wait for a slot themselves,
// since otherwise the cases already waiting for their results would hold
// every slot while the vectors not yet started waited behind them. Every
// selected case has started when runSuite returns, and the runner's own
// slots bound the work.
func runSuite(t *testing.T, r *runner) *suiteStats {
	stats := &suiteStats{}
	dirs, err := vectorDirs(r.suite)
	if err != nil {
		t.Fatalf("the suite: %v", err)
	}
	t.Cleanup(func() {
		t.Log(stats.summary())
		if r.rec != nil && !t.Failed() {
			if err := r.rec.write(r); err != nil {
				t.Errorf("the record: %v", err)
			}
		}
	})
	byArea := map[string][]string{}
	for _, d := range dirs {
		area, _, _ := strings.Cut(vectorPath(r.suite, d), "/")
		byArea[area] = append(byArea[area], d)
	}
	for _, area := range sortedKeys(byArea) {
		t.Run(area, func(t *testing.T) {
			for _, d := range byArea[area] {
				rel := vectorPath(r.suite, d)
				t.Run(path.Base(rel), func(t *testing.T) {
					v, err := readVector(d)
					if err != nil {
						t.Fatalf("%s: %v", rel, err)
					}
					if why := r.selected(v); why != "" {
						stats.notSelected.Add(1)
						t.Skipf("%s not selected: %s", v.ID, why)
					}
					stats.vectors.Add(1)
					for ci := range v.Cases {
						c := &v.Cases[ci]
						t.Run(c.Name, func(t *testing.T) {
							stats.cases.Add(1)
							base := t.TempDir()
							done := make(chan *caseResult, 1)
							go func() { done <- r.runCase(v, c, base) }()
							t.Parallel()
							res := <-done
							if res.skipped != "" {
								stats.skipped.Add(1)
								if r.rec != nil {
									r.rec.skip(v, c, res.skipped)
								}
								t.Skipf("skipped, not passed: %s", res.skipped)
							}
							reportCase(t, r, v, c, res, base)
						})
					}
					t.Parallel()
				})
			}
			t.Parallel()
		})
	}
	return stats
}

// reportCase fails t with each problem of res, naming the vector, the
// case and the statements (ADR-011 D11 item 8), keeps the sandbox of a
// failing case, and records a passing one.
func reportCase(t *testing.T, r *runner, v *vector, c *vcase, res *caseResult, base string) {
	t.Helper()
	for _, p := range res.problems {
		t.Errorf("%s/%s (%s): %s", v.ID, c.Name, strings.Join(v.Statements, ", "), p)
	}
	if len(res.problems) > 0 && r.keep != "" {
		dst := filepath.Join(r.keep, filepath.FromSlash(v.ID), c.Name)
		if err := copyTree(base, dst); err != nil {
			t.Logf("keeping the sandbox: %v", err)
		} else {
			t.Logf("the sandbox is kept at %s", dst)
		}
	}
	if r.rec != nil {
		r.rec.add(v, c, res)
	}
}

// digest is the hexadecimal SHA-256 of b.
func digest(b []byte) string {
	d := sha256.Sum256(b)
	return hex.EncodeToString(d[:])
}
