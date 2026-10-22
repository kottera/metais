// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/testenv"
)

// TestDirectoryFlagAgrees is ADR-008 Amendment A4: every command that
// writes a directory or a file names where with --out, so a session that
// guesses --out for one of them is accepted: no usage refusal, whatever
// the package itself is found to hold. Each is run with --dry-run,
// which parses its whole command line and writes nothing.
func TestDirectoryFlagAgrees(t *testing.T) {
	testenv.Home(t)
	pkg := filepath.Join(vectorsDir, "pkg", "opens-packed", "package.metais")
	work := testenv.Dir(t)
	src := filepath.Join(work, "src")
	if r := runTool(t, "unpack", "--json", pkg, "--out", src); r.status != 0 {
		t.Fatalf("metais unpack --out: exit %d: %s", r.status, r.stdout)
	}
	for _, argv := range [][]string{
		{"render", pkg, "--sample", "--out", filepath.Join(work, "rendered"), "--dry-run"},
		{"pack", src, "--out", filepath.Join(work, "p.metais"), "--dry-run"},
		{"unpack", pkg, "--out", filepath.Join(work, "unpacked"), "--dry-run"},
		{"extract", pkg, "**", "--out", filepath.Join(work, "extracted"), "--dry-run"},
		{"sample", pkg, "--out", filepath.Join(work, "sampled"), "--dry-run"},
	} {
		c := lookup(table, argv[0])
		names := []string{}
		for _, f := range c.spec().Flags {
			names = append(names, f.Name)
		}
		if !slices.Contains(names, "out") || slices.Contains(names, "to") {
			t.Errorf("metais %s takes the flags %q, want --out and no --to", argv[0], names)
		}
		r := runTool(t, append(argv, "--json")...)
		if r.status == finding.ExitUsage || slices.ContainsFunc(r.codes(), func(c string) bool { return strings.HasPrefix(c, "usage.") }) {
			t.Errorf("metais %q: exit %d, want it accepted: %s", argv, r.status, r.stdout)
		}
	}
}
