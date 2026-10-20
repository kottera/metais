// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp/syntax"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/testenv"
)

// The hostile corpus's command cases (ADR-009 D17 item 11) and grep's
// limits at the boundary (D17 item 12): grep is not an operation of
// chapter 15, so no vector can run it, and these run the command line.

// floodPackage is a source directory whose references/flood.txt holds n
// lines, each matching "match".
func floodPackage(t *testing.T, n int) string {
	t.Helper()
	dir := guidePackage(t, []string{"guide"})
	var b strings.Builder
	for i := range n {
		fmt.Fprintf(&b, "match line %d\n", i)
	}
	if err := os.MkdirAll(filepath.Join(dir, "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "references", "flood.txt"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// findingParams returns r's findings as code and params, one line each.
func findingParams(r ran) []string {
	if r.doc == nil {
		return nil
	}
	var out []string
	for _, f := range r.doc.Findings {
		p, _ := json.Marshal(f.Params)
		out = append(out, f.Code+" "+string(p))
	}
	return out
}

// TestHostileGrepMatches is grep-matches at the bound and past it, under
// a lowered limit and at the default (ADR-009 D2 item 6, D17 items 11 and
// 12; RULINGS R2-144): matching lines up to the bound are all reported,
// unmarked; one more is cut at the bound and marked, in the result and on
// standard error, and the exit status is the one the cap does not change.
// A lowered grep-matches bounds what is reported when --max-results is not
// given, and --max-results takes the bound and refuses one more.
func TestHostileGrepMatches(t *testing.T) {
	testenv.Home(t)
	const bound = 5
	lowered := []string{"--limit", "grep-matches=" + strconv.Itoa(bound)}
	for _, c := range []struct {
		name       string
		lines      int
		flags      []string
		reported   int
		truncated  bool
		noticeWant string
	}{
		{"at the bound", bound, lowered, bound, false, ""},
		{"past the bound", bound + 1, lowered, bound, true, "Showing the first 5 of 6 matching lines; 1 more were found and not shown (--max-results).\n"},
		{"past the bound, --max-results at it", bound + 1, append(slices.Clone(lowered), "--max-results", strconv.Itoa(bound)), bound, true, "Showing the first 5 of 6 matching lines; 1 more"},
		{"a flood", 30, lowered, bound, true, "Showing the first 5 of 30 matching lines; 25 more"},
		{"the default, at --max-results", 100, nil, 100, false, ""},
		{"the default, past --max-results", 101, nil, 100, true, "Showing the first 100 of 101 matching lines; 1 more"},
	} {
		t.Run(c.name, func(t *testing.T) {
			pkg := floodPackage(t, c.lines)
			argv := append([]string{"grep", pkg, "match", "references/flood.txt"}, c.flags...)
			r := runTool(t, argv...)
			if r.status != 0 || strings.Count(r.stdout, ":match line ") != c.reported {
				t.Errorf("exit %d, %d lines reported, want %d:\n%s", r.status, strings.Count(r.stdout, ":match line "), c.reported, r.stdout)
			}
			if c.noticeWant == "" && r.stderr != "" || c.noticeWant != "" && !strings.HasPrefix(r.stderr, c.noticeWant) {
				t.Errorf("standard error %q, want %q", r.stderr, c.noticeWant)
			}
			var res grepResult
			j := runTool(t, append([]string{"grep", "--json"}, argv[1:]...)...)
			result(t, j, &res)
			if j.status != 0 || len(res.Matches) != c.reported || res.MatchingLines != c.lines || res.Truncated != c.truncated {
				t.Errorf("as JSON: exit %d, %d matches of %d, truncated %v; want %d of %d, truncated %v",
					j.status, len(res.Matches), res.MatchingLines, res.Truncated, c.reported, c.lines, c.truncated)
			}
		})
	}
	pkg := floodPackage(t, 1)
	for _, c := range []struct {
		flags []string
		ok    bool
	}{
		{append(slices.Clone(lowered), "--max-results", strconv.Itoa(bound)), true},
		{append(slices.Clone(lowered), "--max-results", strconv.Itoa(bound+1)), false},
		{[]string{"--max-results", "10000"}, true},
		{[]string{"--max-results", "10001"}, false},
	} {
		r := runTool(t, append([]string{"grep", "--json", pkg, "match"}, c.flags...)...)
		want := []string{`usage.bad_value {"argument":` + strconv.Itoa(slices.Index(r.args, "--max-results")+1) + `,"flag":"--max-results","reason":"value_not_allowed"}`}
		if c.ok && (r.status != 0 || len(r.doc.Findings) != 0) || !c.ok && (r.status != finding.ExitUsage || !slices.Equal(findingParams(r), want)) {
			t.Errorf("%q: exit %d, findings %q", c.flags, r.status, findingParams(r))
		}
	}
}

// TestHostileGrepContext is grep-context at the bound and past it, at the
// default and lowered (ADR-009 D3, D17 items 11 and 12): -C takes the
// bound and refuses one more, before the package is opened.
func TestHostileGrepContext(t *testing.T) {
	testenv.Home(t)
	pkg := floodPackage(t, 3)
	for _, c := range []struct {
		flags []string
		ok    bool
	}{
		{[]string{"-C", "100"}, true},
		{[]string{"-C", "101"}, false},
		{[]string{"--limit", "grep-context=3", "-C", "3"}, true},
		{[]string{"--limit", "grep-context=3", "-C", "4"}, false},
	} {
		r := runTool(t, append([]string{"grep", "--json", pkg, "match"}, c.flags...)...)
		want := []string{`usage.bad_value {"argument":` + strconv.Itoa(slices.Index(r.args, "-C")+1) + `,"flag":"--context","reason":"value_not_allowed"}`}
		if c.ok && (r.status != 0 || len(r.doc.Findings) != 0) || !c.ok && (r.status != finding.ExitUsage || !slices.Equal(findingParams(r), want)) {
			t.Errorf("%q: exit %d, findings %q", c.flags, r.status, findingParams(r))
		}
	}
}

// programSize is the instruction count regex-program bounds for a grep
// pattern: its regexp/syntax program, as ADR-009 D10 counts it.
func programSize(t *testing.T, pattern string) int {
	t.Helper()
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		t.Fatal(err)
	}
	prog, err := syntax.Compile(re.Simplify())
	if err != nil {
		t.Fatal(err)
	}
	return len(prog.Inst)
}

// TestHostileGrepPattern is pattern-length, regex-program and regex-work
// for a grep pattern, each at the bound and past it (ADR-009 D10, D17
// items 11 and 12): a pattern as long as pattern-length, a program as
// large as regex-program and a search costing exactly regex-work are each
// accepted, and one byte, instruction or unit less refuses the pattern
// with usage.bad_pattern naming the limit. regex-work is the program's
// size times the length of each line it is matched against.
func TestHostileGrepPattern(t *testing.T) {
	testenv.Home(t)
	const lines = 7
	pkg := floodPackage(t, lines)
	pattern := "mat(ch)+ line [0-9]"
	insts := programSize(t, pattern)
	var work int
	for i := range lines {
		work += insts * len("match line "+strconv.Itoa(i))
	}
	for _, c := range []struct {
		limit string
		bound int
	}{
		{"pattern-length", len(pattern)},
		{"regex-program", insts},
		{"regex-work", work},
	} {
		t.Run(c.limit, func(t *testing.T) {
			for _, bound := range []int{c.bound, c.bound - 1} {
				r := runTool(t, "grep", "--json", pkg, pattern, "references/flood.txt", "--limit", c.limit+"="+strconv.Itoa(bound))
				if bound == c.bound {
					var res grepResult
					result(t, r, &res)
					if res.MatchingLines != lines {
						t.Errorf("at the bound: %d matching lines, want %d", res.MatchingLines, lines)
					}
					continue
				}
				got := findingParams(r)
				want := `usage.bad_pattern {"argument":4,"bound":` + strconv.Itoa(bound) + `,"limit":"` + c.limit + `","reason":"` + strings.ReplaceAll(c.limit, "-", "_") + `"}`
				if r.status != finding.ExitUsage || !slices.Equal(got, []string{want}) {
					t.Errorf("past the bound: exit %d, findings %q, want %q", r.status, got, want)
				}
			}
		})
	}
}

// TestHostileGrepPathological is (a|aa)*b, whose backtracking cost is
// exponential, over the largest entry a lowered entry-size admits
// (ADR-009 D17 item 11, D18): the search is linear, finds nothing and
// stays within a CPU time, and at a regex-work below its cost it is
// refused rather than run.
func TestHostileGrepPathological(t *testing.T) {
	testenv.Home(t)
	const size = 1 << 20
	const cpuBound = 10 * time.Second // D18's ceiling; the search takes a fraction of it, under the race detector too
	pkg := guidePackage(t, []string{"guide"})
	if err := os.MkdirAll(filepath.Join(pkg, "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "references", "large.txt"), []byte(strings.Repeat("a", size)), 0o644); err != nil {
		t.Fatal(err)
	}
	const pattern = "(a|aa)*b"
	before, measured := selfCPU()
	r := runTool(t, "grep", "--json", pkg, pattern, "references/large.txt", "--limit", "entry-size="+strconv.Itoa(size))
	if after, _ := selfCPU(); measured {
		t.Logf("grep %s over %d bytes took %v of CPU", pattern, size, after-before)
		if after-before > cpuBound {
			t.Errorf("grep %s over %d bytes took %v of CPU, more than %v", pattern, size, after-before, cpuBound)
		}
	}
	var res grepResult
	result(t, r, &res)
	if res.MatchingLines != 0 {
		t.Errorf("%d matching lines, want none", res.MatchingLines)
	}
	cost := programSize(t, pattern) * size
	r = runTool(t, "grep", "--json", pkg, pattern, "references/large.txt", "--limit", "regex-work="+strconv.Itoa(cost-1))
	want := `usage.bad_pattern {"argument":4,"bound":` + strconv.Itoa(cost-1) + `,"limit":"regex-work","reason":"regex_work"}`
	if got := findingParams(r); r.status != finding.ExitUsage || !slices.Equal(got, []string{want}) {
		t.Errorf("below its cost: exit %d, findings %q, want %q", r.status, got, want)
	}
	// One byte past the lowered entry-size, the package is refused.
	r = runTool(t, "grep", "--json", pkg, pattern, "references/large.txt", "--limit", "entry-size="+strconv.Itoa(size-1))
	if r.status != finding.ExitRefused || !slices.Equal(r.codes(), []string{"limit.package"}) {
		t.Errorf("past entry-size: exit %d, findings %q", r.status, findingParams(r))
	}
}
