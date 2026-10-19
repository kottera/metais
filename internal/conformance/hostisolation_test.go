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
	{"env", "tpl/no-env", []string{"HOME", "USER", "HOSTNAME", "TERM", "NO_COLOR", "METAIS_SENTINEL", "TMPDIR", "PATH"}, checkSentinel},
	{"cwd", "tpl/partial-not-from-host", []string{"a working-directory decoy"}, checkSentinel},
	{"path", "dat/required-input-missing", []string{"the sandbox path"}, checkSentinel},
	{"pid", "tag/provenance", nil, checkAB},
	{"home", "tag/asset-not-from-host", []string{"HOME"}, checkSentinel},
	{"tz", "flt/date-own-offset", nil, checkAB},
	{"clock", "flt/date-now-refused", nil, checkAB},
}

// hostRunner is a runner over the suite in this module for the tool at
// program, with both environments.
func hostRunner(t *testing.T, program string) (*runner, string) {
	t.Helper()
	suite := defaultSuite()
	a, err := newAdapter("cli", program, suite)
	if err != nil {
		t.Fatal(err)
	}
	r := newRunner(t, suite, a)
	r.short = false
	return r, suite
}

// TestHostIsolation is ADR-009 D16 item 1's harness, to the reach this
// wave's canaries need: the release build runs each canary's probe vector
// in the sentinel and the plain environment, and must give no hit on any
// channel — check 1, the sentinel scan of every stream and of every file
// the step wrote anywhere in its sandbox; check 3, the differential; and
// check 4, nothing written outside. Every vector of the suite gets the same
// checks from TestConformance; the harness's every-command reach, its
// further environment and decoys, and check 2 are Wave 6's.
func TestHostIsolation(t *testing.T) {
	if builtTool == "" {
		t.Skip("runs the binary built from ./cmd/metais")
	}
	r, suite := hostRunner(t, builtTool)
	for _, c := range canaries {
		for _, p := range runVector(r, suite, c.probe, testenv.Dir(t)) {
			t.Errorf("%s: %s", c.probe, p)
		}
	}
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
		})
	}
}

// buildCanary builds ./cmd/metais with the canary tag of channel, as the
// release build is built otherwise, and returns its path.
func buildCanary(t *testing.T, channel string) string {
	t.Helper()
	dir := filepath.Join(testenv.Dir(t), "canary-"+channel)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, toolName)
	cmd := exec.Command("go", "build", "-trimpath", "-tags", "metais_canary_"+channel, "-o", bin, "./cmd/metais")
	cmd.Dir = moduleRoot
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building the %s canary: %v\n%s", channel, err, b)
	}
	return bin
}
