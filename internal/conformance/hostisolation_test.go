// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// canary is one of ADR-009 D16 item 2's canary builds: the leak its build
// tag inserts, the vector that probes it, and how the harness sees it: by
// the sentinel scan, a hit naming one of sources, or, with no sources, by
// the differential of check, a host fact no sentinel can be planted in.
type canary struct {
	channel string
	probe   string
	sources []string
	check   string
	also    string // another check the leak must turn red, or ""
}

// canaries are the canaries built so far (PLAN-2026-10-metais-v1 Waves 4
// and 5): env binds the environment as env, cwd reads a partial the
// package lacks from the working directory, path (W4-1's) prints the data
// directory's absolute path in every finding, pid (W5-3's) records the
// process id in every provenance line, home (W5-5's) reads an asset the
// package lacks from the home directory, tz formats date in the host's
// zone, and clock reads "now" in date from the host's clock. The probes
// but pid's and tz's are refused by the release build: tpl/no-env reads
// env.HOME, tpl/partial-not-from-host renders a partial named for the
// working directory's decoy, dat/required-input-missing lacks a required
// input, tag/asset-not-from-host writes the contents of an asset named for
// the home directory's decoy, and flt/date-now-refused refuses "now",
// which the clock canary renders as a time that differs between
// environment A and environment B, started at least two seconds after it;
// tag/provenance renders provenance lines, the same bytes in both
// environments, and flt/date-own-offset renders
// {{ 0 | date: "%H:%M %z" }} as 00:00 +0000, which environment A's
// TZ=Pacific/Chatham moves under the tz canary.
var canaries = []canary{
	{"env", "tpl/no-env", []string{"HOME", "USER", "HOSTNAME", "TERM", "NO_COLOR", "METAIS_SENTINEL", "TMPDIR", "PATH"}, checkSentinel, ""},
	{"cwd", "tpl/partial-not-from-host", []string{"a working-directory decoy"}, checkSentinel, ""},
	{"path", "dat/required-input-missing", []string{"the sandbox path"}, checkSentinel, checkHost},
	{"pid", "tag/provenance", nil, checkAB, ""},
	{"home", "tag/asset-not-from-host", []string{"HOME"}, checkSentinel, ""},
	{"tz", "flt/date-own-offset", nil, checkAB, ""},
	{"clock", "flt/date-now-refused", nil, checkAB, ""},
}

// hostRunner is a runner over the suite in this module for the tool at
// program, with both environments and the whole SC7 harness: check 2's
// host facts, and the harness's environment, decoys, path modes and
// commands (ADR-009 D16 item 1).
func hostRunner(t *testing.T, program string) (*runner, string) {
	t.Helper()
	suite := defaultSuite()
	a, err := newAdapter("cli", program, suite)
	if err != nil {
		t.Fatal(err)
	}
	r := newRunner(t, suite, a)
	r.short = false
	r.sc7 = true
	vocab, err := toolVocabulary(program, testenv.Dir(t))
	if err != nil {
		t.Fatal(err)
	}
	reg, err := os.ReadFile(filepath.Join(suite, "..", "findings", "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	r.facts = newHostFacts(append(vocab, reg)...)
	return r, suite
}

// TestHostIsolation is ADR-009 D16 item 1's harness: the release build
// runs every vector of the suite under the SC7 harness, in the sentinel
// and the plain environment and in both path modes, every case followed
// by every command over each package it names, and must give no hit on
// any channel — check 1, the sentinel scan of every stream and of every
// file the step wrote anywhere in its sandbox; check 2, the host's real
// facts; check 3, the differential; and check 4, nothing written outside —
// and no crash (D18 item 2). Each vector's own expectations hold in both
// path modes too.
//
// TestConformance runs ADR-011 D12's environments alone, so its default
// run stays within ADR-011 D11 item 9's budget; this test carries the
// harness's reach.
func TestHostIsolation(t *testing.T) {
	if builtTool == "" {
		t.Skip("runs the release build of ./cmd/metais, which -conformance.tool replaces")
	}
	r, _ := hostRunner(t, releaseBuild(t))
	before := procStarted.Load()
	t.Cleanup(func() { t.Logf("the harness ran %d processes", procStarted.Load()-before) })
	runSuite(t, r)
}

// TestHostIsolationCanaries is ADR-009 D16 item 2: each canary build, made
// with its metais_canary_<channel> tag, must be caught by the harness on
// its own channel, by the check its row names, so a harness gone blind to
// a channel fails here. The
// release build's half is TestHostIsolation.
func TestHostIsolationCanaries(t *testing.T) {
	if builtTool == "" {
		t.Skip("builds the canaries beside the binary built from ./cmd/metais")
	}
	for _, c := range canaries {
		t.Run(c.channel, func(t *testing.T) {
			t.Parallel()
			bin := buildCanary(t, c.channel)
			r, suite := hostRunner(t, bin)
			ps := runVector(r, suite, c.probe, testenv.Dir(t))
			caught := slices.ContainsFunc(ps, func(p problem) bool {
				if p.check != c.check {
					return false
				}
				return c.sources == nil || slices.ContainsFunc(c.sources, func(s string) bool {
					return strings.HasSuffix(p.msg, "holds the sentinel from "+s)
				})
			})
			for _, p := range ps {
				t.Logf("the %s canary: %s", c.channel, p)
			}
			if !caught {
				t.Errorf("the %s canary is not caught on its channel by %s; the harness reported %q", c.channel, c.probe, ps)
			}
			if c.also != "" && !slices.ContainsFunc(ps, func(p problem) bool { return p.check == c.also }) {
				t.Errorf("the %s canary is not caught by the %s check too; the harness reported %q", c.channel, c.also, ps)
			}
		})
	}
}

// buildCanary builds ./cmd/metais with the canary tag of channel, as the
// release build is built otherwise, and returns its path.
func buildCanary(t *testing.T, channel string) string {
	t.Helper()
	return build(t, "canary-"+channel, "-tags", "metais_canary_"+channel)
}

// releaseBuild builds ./cmd/metais as a release is built, with -trimpath
// and release tags and without the race detector, whatever this test
// binary is built with, and returns its path.
func releaseBuild(t *testing.T) string {
	t.Helper()
	return build(t, "release")
}

// build builds ./cmd/metais with -trimpath and flags into a directory
// named name, and returns the binary's path.
func build(t *testing.T, name string, flags ...string) string {
	t.Helper()
	dir := filepath.Join(testenv.Dir(t), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, toolName)
	args := append(append([]string{"build", "-trimpath"}, flags...), "-o", bin, "./cmd/metais")
	cmd := exec.Command("go", args...)
	cmd.Dir = moduleRoot
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building %s: %v\n%s", name, err, b)
	}
	return bin
}

// TestHarnessReach pins the harness's reach (ADR-009 D16 item 1): a case
// run under it in the absolute-path mode ends with one more step that
// runs every command over the package in both forms, in both
// environments, giving the implementation only absolute paths into the
// work directory from a working directory outside it; the relative mode
// runs the case's own steps alone. A second case over the same package
// and inputs runs none of it again.
func TestHarnessReach(t *testing.T) {
	if builtTool == "" {
		t.Skip("runs the binary built from ./cmd/metais")
	}
	t.Parallel()
	r, suite := hostRunner(t, builtTool)
	v, err := readVector(filepath.Join(suite, "vectors", "rnd", "archive"))
	if err != nil {
		t.Fatal(err)
	}
	entries := packageEntries(filepath.Join(v.dir, "package"))
	if len(entries) < 2 {
		t.Fatalf("rnd/archive's package has %d entries", len(entries))
	}
	want := map[string]int{"ls": 2, "read": 2 * len(entries), "grep": 2, "schema": 2, "extract": 2, "render": 4}
	again := v.Cases[0]
	again.Name = "again"
	for _, c := range []*vcase{&v.Cases[0], &again} {
		rel := r.runCaseIn(v, c, testenv.Dir(t), relPaths)
		for _, run := range rel.runs {
			if len(run.steps) != len(c.Steps) {
				t.Errorf("%s: the relative mode ran %d steps, want the case's %d", c.Name, len(run.steps), len(c.Steps))
			}
		}
		res := r.runCaseIn(v, c, testenv.Dir(t), absPaths)
		for _, p := range append(rel.problems, res.problems...) {
			t.Errorf("%s: %s", c.Name, p)
		}
		for _, e := range []env{envA, envB} {
			run := res.runs[e]
			if run == nil {
				t.Fatalf("%s: environment %s did not run", c.Name, e)
			}
			if run.sb.cwd == run.sb.work {
				t.Errorf("%s: the working directory is the work directory", c.Name)
			}
			if c == &again {
				if len(run.steps) != len(c.Steps) {
					t.Errorf("%s, environment %s: a second case over the same package and inputs ran the reach again", c.Name, e)
				}
				continue
			}
			if len(run.steps) != len(c.Steps)+1 {
				t.Fatalf("%s, environment %s: no reach step", c.Name, e)
			}
			reach := run.steps[len(c.Steps)]
			counts := map[string]int{}
			for _, inv := range reach.invs {
				cmd, _, _ := strings.Cut(strings.TrimPrefix(inv.label, "reach: "), " ")
				counts[cmd]++
				for _, a := range inv.args {
					if a == "package" || strings.HasPrefix(a, reachPrefix) {
						t.Errorf("%s: %s gives a relative path %q", c.Name, inv.label, a)
					}
				}
			}
			if !maps.Equal(counts, want) {
				t.Errorf("%s, environment %s: the reach ran %v, want %v", c.Name, e, counts, want)
			}
			if len(reach.trees) != 2 {
				t.Errorf("%s, environment %s: the reach read %d extract trees, want 2", c.Name, e, len(reach.trees))
			}
		}
	}
}
