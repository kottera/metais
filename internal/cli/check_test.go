// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/testenv"
)

// TestCheckCommand is metais check (ADR-008 D9): the operand's forms, the
// exit status from severity alone, counts as the result, its findings
// and Next: block on standard output (D11 item 3), and the next step.
func TestCheckCommand(t *testing.T) {
	aut := filepath.Join(vectorsDir, "aut")
	for _, c := range []struct {
		name   string
		args   []string
		status int
		counts checkCounts
		next   string
	}{
		{"clean", []string{"check", "--json", filepath.Join(aut, "check-source/package")}, finding.ExitOK, checkCounts{}, ""},
		{"packed", []string{"check", "--json", filepath.Join(aut, "check-packed/package.metais")}, finding.ExitOK, checkCounts{}, ""},
		{"warnings pass", []string{"check", "--json", filepath.Join(aut, "check-source-tree/package")}, finding.ExitOK, checkCounts{Warning: 5}, ""},
		{"an error fails", []string{"check", "--json", filepath.Join(aut, "check-step-skipped/package")}, finding.ExitFindings, checkCounts{}, "metais check <package>"},
		{"refused", []string{"check", "--json", filepath.Join(aut, "check-refused/package.metais")}, finding.ExitRefused, checkCounts{}, "metais check <package>"},
		{"a lowered limit", []string{"check", "--json", "--limit", "entries=4", filepath.Join(aut, "check-limits/package")}, finding.ExitRefused, checkCounts{}, "metais check <package>"},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := runTool(t, c.args...)
			if r.status != c.status {
				t.Fatalf("status %d, want %d: %s", r.status, c.status, r.stdout)
			}
			if c.status == finding.ExitOK {
				var res checkResult
				if err := jsonUnmarshal(r.doc.Result, &res); err != nil || res.Counts != c.counts {
					t.Errorf("result %s, want counts %+v", r.doc.Result, c.counts)
				}
				if len(r.doc.Next) != 1 || r.doc.Next[0] != nextTest {
					t.Errorf("next %+v, want metais test", r.doc.Next)
				}
			} else if string(r.doc.Result) != "null" || len(r.doc.Next) != 1 || r.doc.Next[0].Run != c.next {
				t.Errorf("result %s, next %+v", r.doc.Result, r.doc.Next)
			}
		})
	}

	// Human mode: findings, the counts and Next: on standard output; a
	// refused package's on standard error.
	r := runTool(t, "check", filepath.Join(aut, "check-step-skipped/package"))
	if r.stderr != "" || !strings.Contains(r.stdout, "[template.syntax]") || !strings.Contains(r.stdout, "check: 2 errors, 0 warnings, 1 note\n") ||
		!strings.Contains(r.stdout, "Next:\n  metais check <package>") {
		t.Errorf("human: stdout %q, stderr %q", r.stdout, r.stderr)
	}
	r = runTool(t, "check", filepath.Join(aut, "check-refused/package.metais"))
	if r.stdout != "" || !strings.Contains(r.stderr, "[package.container.not_zip]") {
		t.Errorf("refused, human: stdout %q, stderr %q", r.stdout, r.stderr)
	}

	// The operand defaults to the working directory, and - is refused.
	t.Chdir(filepath.Join(aut, "check-source/package"))
	if r := runTool(t, "check", "--json"); r.status != finding.ExitOK || !r.doc.OK {
		t.Errorf("check with no operand: status %d, %s", r.status, r.stdout)
	}
	t.Chdir(testenv.Dir(t))
	if r := runTool(t, "check", "--json", "-"); !slices.Equal(r.codes(), []string{string(finding.CodeUsagePackageFromStdin)}) || r.status != finding.ExitUsage {
		t.Errorf("check -: status %d, %s", r.status, r.stdout)
	}
	if r := runTool(t, "check", "--json"); !slices.Equal(r.codes(), []string{"package.layout.missing_manifest"}) || r.status != finding.ExitRefused {
		t.Errorf("check in an empty directory: status %d, %s", r.status, r.stdout)
	}
}
