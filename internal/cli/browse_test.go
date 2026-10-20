// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/kottera/metais/internal/browse"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostenv"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/testenv"
)

// readingCommands are ADR-008 D5's commands, in the table's order.
var readingCommands = []string{"ls", "read", "outline", "grep", "schema", "extract"}

// browsePackage writes a source directory with the entries the reading
// commands' tests read: Markdown with sections, CRLF and no final LF, a
// binary entry, an empty one, a long line, and package text holding a
// terminal escape and a bidirectional override.
func browsePackage(t *testing.T) string {
	t.Helper()
	dir := guidePackage(t, []string{"guide"})
	files := map[string]string{
		"references/guide.md":  "# One\ntext\n## Two\n```\n# not\n```\nSetext\n===\nend",
		"references/crlf.txt":  "a\r\nb\r\nc\r\n",
		"references/nolf.txt":  "a\nb\nc",
		"references/empty.txt": "",
		"references/bin.dat":   "\x00\x01\x02",
		"references/long.txt":  strings.Repeat("x", 1030) + "\n",
		"references/evil.md":   "# Head \x1b[31mred\u202e\u2069\n\tline \x1b[2J gone\u2066\u2067\u2068\u2069\n",
		"references/ctx.txt":   "1\n2\nhit\n4\nhit\n6\n7\n8\n9\nhit\n",
	}
	if err := os.MkdirAll(filepath.Join(dir, "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// result unmarshals r's result into v.
func result(t *testing.T, r ran, v any) {
	t.Helper()
	if r.doc == nil || !r.doc.OK {
		t.Fatalf("metais %q: %s", r.args, r.stdout)
	}
	if err := json.Unmarshal(r.doc.Result, v); err != nil {
		t.Fatal(err)
	}
}

// TestReadingRefusalsTriggered is ADR-008 D16 item 2 for the reading
// commands (W4-2.3): every refusal each declares is triggered, by the
// command over some package of the conformance suite or by a line below,
// and runTool holds every run to its declaration, to a non-empty next
// step and to next steps that parse. The codes the unwired list says no
// vector produces yet, and the line codes lineCases proves for every
// command, are excepted, as inspect's test excepts them.
func TestReadingRefusalsTriggered(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]map[string]bool{}
	note := func(cmd string, codes []string) {
		mu.Lock()
		defer mu.Unlock()
		if seen[cmd] == nil {
			seen[cmd] = map[string]bool{}
		}
		for _, c := range codes {
			seen[cmd][c] = true
		}
	}
	run := func(argv ...string) ran {
		t.Helper()
		r := runTool(t, argv...)
		note(argv[0], r.codes())
		return r
	}
	lines := map[string][]string{
		"ls": {}, "read": {"metais.yaml"}, "outline": {"metais.yaml"}, "grep": {"x"}, "schema": {},
	}
	// The suite's packages are walked first, each package a parallel
	// subtest within the gate's -parallel share, so the walk's wall time
	// does not grow with the corpus one package at a time (R-W6-9a). It
	// sets no environment, so it runs before testenv.Home, whose t.Setenv
	// no parallel test may share; Run reads no environment, as ADR-009
	// D16 item 3's symbol rule holds every non-test file outside hostenv
	// to. Each package extracts to a destination of its own.
	pkgs, _ := filepath.Glob(filepath.Join(vectorsDir, "*", "*", "package*"))
	t.Run("suite", func(t *testing.T) {
		for _, p := range pkgs {
			name, _ := filepath.Rel(vectorsDir, p)
			t.Run(filepath.ToSlash(name), func(t *testing.T) {
				t.Parallel()
				readSuitePackage(t, p, lines, note)
			})
		}
	})
	testenv.Home(t)
	pkg := browsePackage(t)
	dest := testenv.Dir(t)
	ro := filepath.Join(dest, "ro")
	os.Mkdir(ro, 0o555)
	t.Cleanup(func() { os.Chmod(ro, 0o755) })
	file := filepath.Join(dest, "file")
	os.WriteFile(file, nil, 0o644)
	for _, argv := range [][]string{
		{"ls", "--json", "-"},
		{"ls", "--json"},
		{"ls", "--json", pkg, "nope"},
		{"ls", "--json", pkg, "--limit", "nope=1"},
		{"read", "--json", pkg, "references"},
		{"read", "--json", pkg, "references/nope.md"},
		{"read", "--json", pkg, "references/bin.dat", "--lines", "1"},
		{"read", "--json", pkg, "references/bin.dat", "--number"},
		{"read", "--json", pkg, "references/nolf.txt", "--lines", "4"},
		{"read", "--json", pkg, "references/empty.txt", "--lines", "1"},
		{"read", "--json", pkg, "references/nolf.txt", "--lines", "2-1"},
		{"read", "--json", "-", "x"},
		{"read", "--json", pkg},
		{"read", "--json", pkg, "x", "--limit", "nope=1"},
		{"outline", "--json", pkg, "references/bin.dat"},
		{"outline", "--json", pkg, "."},
		{"outline", "--json", pkg, "Metais.yaml"},
		{"outline", "--json", "-", "x"},
		{"outline", "--json", pkg},
		{"outline", "--json", pkg, "x", "--limit", "nope=1"},
		{"grep", "--json", pkg, "("},
		{"grep", "--json", pkg, "x", "a**"},
		{"grep", "--json", pkg, "abc", "--limit", "pattern-length=2"},
		{"grep", "--json", pkg, strings.Repeat("a", 200), "--limit", "regex-program=100"},
		{"grep", "--json", pkg, "x", "--limit", "regex-work=10"},
		{"grep", "--json", pkg, "x", "-C", "101"},
		{"grep", "--json", pkg, "x", "--max-results", "0"},
		{"grep", "--json", "-", "x"},
		{"grep", "--json", pkg},
		{"grep", "--json", pkg, "x", "--limit", "nope=1"},
		{"schema", "--json", pkg, "nope"},
		{"schema", "--json", "-"},
		{"schema", "--json"},
		{"schema", "--json", pkg, "--limit", "nope=1"},
		{"extract", "--json", pkg, "[x", "--to", filepath.Join(dest, "a")},
		{"extract", "--json", pkg, "nope/**", "--to", filepath.Join(dest, "a")},
		{"extract", "--json", pkg, "**", "--to", dest},
		{"extract", "--json", pkg, "**", "--to", file},
		{"extract", "--json", pkg, "**", "--to", filepath.Join(ro, "out")},
		{"extract", "--json", "-", "x", "--to", filepath.Join(dest, "a")},
		{"extract", "--json", pkg, "**"},
		{"extract", "--json", pkg},
		{"extract", "--json", pkg, "**", "--to", filepath.Join(dest, "a"), "--limit", "nope=1"},
	} {
		if r := run(argv...); r.status == 0 {
			t.Errorf("metais %q was not refused", argv)
		}
	}
	if os.Geteuid() == 0 || runtime.GOOS == "windows" {
		// A read-only directory refuses no one here.
		seen["extract"][string(finding.CodeDestinationNothingWritten)] = true
	}
	// A write into a new <dir> that fails after creating it needs a tree
	// that fails beneath Extract, which no command line makes:
	// TestExtractFailedFindings and internal/browse's
	// TestExtractStopsAndRemovesNothing prove it.
	seen["extract"][string(finding.CodeDestinationWriteFailed)] = true
	pkg2 := browsePackage(t)
	for _, c := range extractDestinationCases() {
		if c.runsHere() {
			to := testenv.Dir(t)
			c.setup(t, to)
			run("extract", "--json", pkg2, "metais.yaml", "references/*.txt", "--to", to, "--overwrite")
		} else {
			// Proved where it can be set up: a link, or a directory the
			// user cannot write.
			seen["extract"][strings.Fields(c.want[0])[0]] = true
		}
	}
	// A binary entry to a terminal is a refusal of human mode only.
	saved := isTerminal
	t.Cleanup(func() { isTerminal = saved })
	isTerminal = func(any) bool { return true }
	r := runWith(t, hostenv.Display{NoColor: true}, "read", pkg, "references/bin.dat")
	isTerminal = saved
	if r.stdout != "" || r.status != finding.ExitUsage || !strings.Contains(r.stderr, "[usage.binary_to_terminal]") {
		t.Errorf("a binary entry to a terminal: %d %q %q", r.status, r.stdout, r.stderr)
	} else {
		seen["read"][string(finding.CodeUsageBinaryToTerminal)] = true
		_, next, _ := strings.Cut(r.stderr, "\nNext:\n")
		run, _, _ := strings.Cut(strings.TrimSpace(next), "  — ")
		if err := parsesAsLine(run); err != nil || !strings.Contains(run, "extract") {
			t.Errorf("a binary entry to a terminal names %q next: %v", run, err)
		}
	}
	// What git cannot commit: an empty directory and a link, made here.
	for _, mk := range []func(dir string) error{
		func(dir string) error { return os.Mkdir(filepath.Join(dir, "assets"), 0o755) },
		func(dir string) error {
			return os.Symlink("out.txt.liquid", filepath.Join(dir, "templates", "link.liquid"))
		},
	} {
		dir := guidePackage(t, []string{"guide"})
		if err := mk(dir); err != nil {
			t.Fatal(err)
		}
		for _, name := range readingCommands {
			run(append(append([]string{name, "--json", dir}, lines[name]...), extractTo(t, name)...)...)
		}
	}
	unwired := map[string]bool{}
	b, err := os.ReadFile("../finding/testdata/unwired.txt")
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range strings.Split(string(b), "\n") {
		if l != "" && !strings.HasPrefix(l, "#") {
			unwired[strings.Fields(l)[0]] = true
		}
	}
	lineCodes := []finding.Code{finding.CodeUsageBadValue, finding.CodeUsageRepeatedFlag, finding.CodeUsageUnexpectedOperand, finding.CodeUsageUnknownFlag}
	for _, name := range readingCommands {
		for _, rf := range lookup(table, name).Refusals {
			code := string(rf.Code)
			if !seen[name][code] && (!unwired[code] || rf.Code == finding.CodeDestinationNotEmpty ||
				rf.Code == finding.CodeDestinationNotDirectory || rf.Code == finding.CodeDestinationWriteFailed ||
				rf.Code == finding.CodeDestinationNothingWritten || rf.Code == finding.CodeDestinationStagingExists) &&
				!slices.Contains(lineCodes, rf.Code) && code != "package.not_found" && code != "package.not_readable" {
				t.Errorf("%s declares %s, which nothing triggers", name, code)
			}
		}
	}
	for _, name := range readingCommands {
		r := runTool(t, append(append([]string{name, "--json", filepath.Join(testenv.Dir(t), "absent")}, lines[name]...), extractTo(t, name)...)...)
		if !slices.Equal(r.codes(), []string{"package.not_found"}) || r.status != finding.ExitRefused {
			t.Errorf("metais %s of an absent package: %s", name, r.stdout)
		}
	}
}

// readSuitePackage runs inspect and every reading command over the
// suite's package p, under the limits its vector's first step lowers, and
// notes each command's codes. A package inspect refuses refuses every
// reading command with inspect's findings and the refused exit (ADR-011
// D16 item 3).
func readSuitePackage(t *testing.T, p string, lines map[string][]string, note func(cmd string, codes []string)) {
	var limitArgs []string
	if b, err := os.ReadFile(filepath.Join(filepath.Dir(p), "vector.json")); err == nil && bytes.Contains(b, []byte(`"limits"`)) {
		var v struct {
			Cases []struct {
				Steps []struct {
					Args struct {
						Limits map[string]json.Number `json:"limits"`
					} `json:"args"`
				} `json:"steps"`
			} `json:"cases"`
		}
		json.Unmarshal(b, &v)
		for name, val := range v.Cases[0].Steps[0].Args.Limits {
			limitArgs = append(limitArgs, "--limit", name+"="+val.String())
		}
	}
	var want []string
	for _, cmd := range append([]string{"inspect"}, readingCommands...) {
		args := append(append(append([]string{cmd, "--json", p}, lines[cmd]...), extractTo(t, cmd)...), limitArgs...)
		r := runTool(t, args...)
		note(cmd, r.codes())
		if cmd == "inspect" {
			want = r.codes()
			continue
		}
		if len(want) > 0 && (!slices.Equal(r.codes(), want) || r.status != finding.ExitRefused) {
			t.Errorf("metais %s over %s: %q and exit %d; inspect: %q", cmd, p, r.codes(), r.status, want)
		}
	}
}

// extractTo is extract's destination operands, a directory of t's own
// that does not exist yet, and nothing for any other command.
func extractTo(t *testing.T, cmd string) []string {
	if cmd != "extract" {
		return nil
	}
	return []string{"**", "--to", filepath.Join(testenv.Dir(t), "never")}
}

// TestReadingRefusalExits is ADR-008 D13 for the reading commands: a
// usage refusal exits 3, a refused package 4, and a refused destination 5.
func TestReadingRefusalExits(t *testing.T) {
	testenv.Home(t)
	pkg := browsePackage(t)
	dest := testenv.Dir(t)
	if err := os.WriteFile(filepath.Join(dest, "notes.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		argv []string
		code string
		exit int
	}{
		{[]string{"read", "--json", pkg, "references/nope"}, "usage.entry_not_found", finding.ExitUsage},
		{[]string{"read", "--json", pkg, "references/nolf.txt", "--lines", "4"}, "usage.line_out_of_range", finding.ExitUsage},
		{[]string{"extract", "--json", pkg, "nope", "--to", filepath.Join(dest, "x")}, "usage.no_match", finding.ExitUsage},
		{[]string{"extract", "--json", pkg, "**", "--to", dest}, "output.destination.not_empty", finding.ExitDestination},
		{[]string{"grep", "--json", pkg, "("}, "usage.bad_pattern", finding.ExitUsage},
	} {
		r := runTool(t, c.argv...)
		if !slices.Equal(r.codes(), []string{c.code}) || r.status != c.exit {
			t.Errorf("metais %q: %q and exit %d, want %s and %d", c.argv, r.codes(), r.status, c.code, c.exit)
		}
	}
}

// TestReadingNextSteps is ADR-008 D14 for the reading commands' refusals:
// the next step names what the refusal knows, the nearest directory, the
// entry or its line count, and never a path the caller gave.
func TestReadingNextSteps(t *testing.T) {
	testenv.Home(t)
	pkg := browsePackage(t)
	for _, c := range []struct {
		argv []string
		run  string
	}{
		{[]string{"ls", "--json", pkg, "references/x/y"}, "metais ls <package> references/"},
		{[]string{"ls", "--json", pkg, "nope"}, "metais ls <package> ."},
		{[]string{"read", "--json", pkg, "references/"}, "metais ls <package> references/"},
		{[]string{"outline", "--json", pkg, "."}, "metais ls <package> ."},
		{[]string{"read", "--json", pkg, "references/nolf.txt", "--lines", "9-"}, "metais read <package> references/nolf.txt --lines 1-3"},
		{[]string{"read", "--json", pkg, "references/empty.txt", "--lines", "1"}, "metais read <package> references/empty.txt"},
		{[]string{"outline", "--json", pkg, "references/bin.dat"}, "metais extract <package> references/bin.dat --to <dir>"},
	} {
		r := runTool(t, c.argv...)
		if len(r.doc.Next) == 0 || r.doc.Next[0].Run != c.run {
			t.Errorf("metais %q: next %+v, want %q", c.argv, r.doc.Next, c.run)
		}
		if strings.Contains(r.stdout, pkg) {
			t.Errorf("metais %q echoes the package operand", c.argv)
		}
	}
}

// TestReadCommand is read (ADR-008 D5, read; D16 item 8): bytes unchanged
// (RULINGS R2-145), --lines with its clamped end reported, CRLF and no
// final LF kept, --number, an empty entry, and a binary entry in base64.
func TestReadCommand(t *testing.T) {
	testenv.Home(t)
	pkg := browsePackage(t)
	for _, c := range []struct {
		args  []string
		human string
		lines readLines
	}{
		{[]string{"references/crlf.txt"}, "a\r\nb\r\nc\r\n", readLines{1, 3, 3}},
		{[]string{"references/crlf.txt", "--lines", "2"}, "b\r\n", readLines{2, 2, 3}},
		{[]string{"references/nolf.txt", "--lines", "2-99"}, "b\nc", readLines{2, 3, 3}},
		{[]string{"references/nolf.txt", "--lines", "3-"}, "c", readLines{3, 3, 3}},
		{[]string{"references/nolf.txt", "--number"}, "1\ta\n2\tb\n3\tc", readLines{1, 3, 3}},
		{[]string{"references/crlf.txt", "--number", "--lines", "2-"}, "2\tb\r\n3\tc\r\n", readLines{2, 3, 3}},
		{[]string{"references/empty.txt"}, "", readLines{0, 0, 0}},
		{[]string{"references/evil.md"}, "# Head \x1b[31mred\u202e\u2069\n\tline \x1b[2J gone\u2066\u2067\u2068\u2069\n", readLines{1, 2, 2}},
	} {
		argv := append([]string{"read", pkg}, c.args...)
		h := runTool(t, argv...)
		if h.status != 0 || h.stdout != c.human || h.stderr != "" {
			t.Errorf("metais read %q: %d %q %q, want %q", c.args, h.status, h.stdout, h.stderr, c.human)
		}
		var res readResult
		result(t, runTool(t, append(argv, "--json")...), &res)
		if res.Binary || res.Lines == nil || *res.Lines != c.lines {
			t.Errorf("metais read %q --json: %+v", c.args, res)
		}
		if !slices.Contains(c.args, "--number") && (res.Content == nil || *res.Content != c.human) {
			t.Errorf("metais read %q --json: content %v", c.args, res.Content)
		}
	}
	h := runTool(t, "read", pkg, "references/bin.dat")
	if h.stdout != "\x00\x01\x02" || h.status != 0 {
		t.Errorf("a binary entry to a pipe: %q", h.stdout)
	}
	var res struct {
		Binary        bool    `json:"binary"`
		ContentBase64 string  `json:"content_base64"`
		Content       *string `json:"content"`
	}
	result(t, runTool(t, "read", "--json", pkg, "references/bin.dat"), &res)
	if b, _ := base64.StdEncoding.DecodeString(res.ContentBase64); !res.Binary || !bytes.Equal(b, []byte("\x00\x01\x02")) || res.Content != nil {
		t.Errorf("a binary entry as JSON: %+v", res)
	}
	// A binary entry is written to standard output that is not a terminal,
	// and with --json to a terminal as well.
	saved := isTerminal
	t.Cleanup(func() { isTerminal = saved })
	isTerminal = func(any) bool { return true }
	if r := runTool(t, "read", "--json", pkg, "references/bin.dat"); !r.doc.OK {
		t.Errorf("read --json of a binary entry to a terminal: %s", r.stdout)
	}
	if r := runTool(t, "read", pkg, "references/nolf.txt"); r.stdout != "a\nb\nc" {
		t.Errorf("a text entry to a terminal: %q", r.stdout)
	}
}

// TestOutlineCommand is outline's human form (ADR-008 D5, outline item 3):
// the section's lines, a tab, two spaces per level below 1, the #s and the
// text, package text neutralised (RULINGS R2-145) and exact in JSON; and
// an entry with no headings, said on standard error.
func TestOutlineCommand(t *testing.T) {
	testenv.Home(t)
	pkg := browsePackage(t)
	r := runTool(t, "outline", pkg, "references/guide.md")
	if want := "1-6\t# One\n3-6\t  ## Two\n7-9\t# Setext\n"; r.stdout != want || r.status != 0 {
		t.Errorf("outline:\n%q\nwant\n%q", r.stdout, want)
	}
	r = runTool(t, "outline", pkg, "references/evil.md")
	if want := "1-2\t# Head \\u001b[31mred\\u202e\\u2069\n"; r.stdout != want {
		t.Errorf("a heading with controls: %q, want %q", r.stdout, want)
	}
	var res outlineResult
	result(t, runTool(t, "outline", "--json", pkg, "references/evil.md"), &res)
	if len(res.Headings) != 1 || res.Headings[0].Text != "Head \x1b[31mred\u202e\u2069" {
		t.Errorf("JSON carries the exact text: %+v", res)
	}
	r = runTool(t, "outline", pkg, "references/nolf.txt")
	if r.stdout != "" || r.status != 0 || !strings.Contains(r.stderr, "no headings") {
		t.Errorf("no headings: %q %q", r.stdout, r.stderr)
	}
	result(t, runTool(t, "outline", "--json", pkg, "references/nolf.txt"), &res)
	if res.Headings == nil || len(res.Headings) != 0 {
		t.Errorf("no headings as JSON: %+v", res)
	}
}

// TestGrepCommand is grep's human form and result (ADR-008 D5, grep):
// GNU grep's path:line:text and path-line-text, groups merged or
// separated by --, a long line cut and marked, the truncation notice and
// the skipped list on standard error, package text neutralised with its
// tab kept, and no match an empty result with exit 0 (RULINGS R2-120).
func TestGrepCommand(t *testing.T) {
	testenv.Home(t)
	pkg := browsePackage(t)
	r := runTool(t, "grep", pkg, "hit", "references/ctx.txt", "-C", "1")
	want := "references/ctx.txt-2-2\nreferences/ctx.txt:3:hit\nreferences/ctx.txt-4-4\nreferences/ctx.txt:5:hit\nreferences/ctx.txt-6-6\n--\nreferences/ctx.txt-9-9\nreferences/ctx.txt:10:hit\n"
	if r.stdout != want || r.status != 0 {
		t.Errorf("grep -C 1:\n%s\nwant\n%s", r.stdout, want)
	}
	if !strings.Contains(r.stderr, "Binary entries skipped") && r.stderr != "" {
		t.Errorf("stderr: %q", r.stderr)
	}
	r = runTool(t, "grep", pkg, "hit", "--max-results", "2")
	if !strings.Contains(r.stderr, "Showing the first 2 of 3 matching lines; 1 more") || strings.Count(r.stdout, ":hit") != 2 {
		t.Errorf("truncation: %q %q", r.stdout, r.stderr)
	}
	var res grepResult
	result(t, runTool(t, "grep", "--json", pkg, "hit", "--max-results", "2"), &res)
	if !res.Truncated || res.MatchingLines != 3 || len(res.Matches) != 2 {
		t.Errorf("truncation as JSON: %+v", res)
	}
	r = runTool(t, "grep", pkg, "^x+$")
	if want := "references/long.txt:1:" + strings.Repeat("x", 1024) + "[…+6 bytes]\n"; r.stdout != want {
		t.Errorf("a long line: %q", r.stdout)
	}
	if !strings.Contains(r.stderr, "Binary entries skipped:\n  references/bin.dat\n") {
		t.Errorf("the skipped list: %q", r.stderr)
	}
	r = runTool(t, "grep", pkg, "gone")
	if want := "references/evil.md:2:\tline \\u001b[2J gone\\u2066\\u2067\\u2068\\u2069\n"; r.stdout != want {
		t.Errorf("a line with controls: %q, want %q", r.stdout, want)
	}
	result(t, runTool(t, "grep", "--json", pkg, "gone"), &res)
	if res.Matches[0].Text != "\tline \x1b[2J gone\u2066\u2067\u2068\u2069" {
		t.Errorf("JSON carries the exact text: %q", res.Matches[0].Text)
	}
	r = runTool(t, "grep", "--json", pkg, "no such text")
	result(t, r, &res)
	if r.status != 0 || res.MatchingLines != 0 || len(res.Matches) != 0 || res.Truncated {
		t.Errorf("no match: %d %+v", r.status, res)
	}
	if r := runTool(t, "grep", pkg, "no such text"); r.status != 0 || r.stdout != "" {
		t.Errorf("no match, human: %d %q", r.status, r.stdout)
	}
	result(t, runTool(t, "grep", "--json", pkg, "-i", "ONE", "--", "references/guide.md"), &res)
	if res.MatchingLines != 1 || !slices.Equal(res.Globs, []string{"references/guide.md"}) {
		t.Errorf("-i and a glob after --: %+v", res)
	}
	result(t, runTool(t, "grep", "--json", pkg, "--", "-x"), &res)
	if res.Pattern != "-x" {
		t.Errorf("a pattern after --: %+v", res)
	}
}

// TestLsCommand is ls's human form (ADR-008 D5, ls item 4).
func TestLsCommand(t *testing.T) {
	testenv.Home(t)
	pkg := browsePackage(t)
	r := runTool(t, "ls", pkg)
	if want := "file\t160\tmetais.yaml\ndir\t8\treferences/\ndir\t1\ttemplates/\n"; r.stdout != want || r.status != 0 || r.stderr != "" {
		t.Errorf("ls:\n%q\nwant\n%q", r.stdout, want)
	}
	r = runTool(t, "ls", pkg, "references/bin.dat")
	if r.stdout != "file\t3\treferences/bin.dat\n" {
		t.Errorf("ls of a file: %q", r.stdout)
	}
}

// TestSchemaCommand is schema (ADR-008 D5, schema): the combined schema,
// one input's self-contained document, both in canonical key order and
// indented by two spaces, and schema's next step.
func TestSchemaCommand(t *testing.T) {
	testenv.Home(t)
	pkg := filepath.Join(vectorsDir, "pkg/opens-source/package")
	r := runTool(t, "schema", pkg)
	if !strings.HasPrefix(r.stdout, "{\n  \"$defs\": {\n    \"inputs/items.schema.yaml\": {") || !strings.HasSuffix(r.stdout, "\n}\n") {
		t.Errorf("schema:\n%s", r.stdout)
	}
	var res schemaResult
	result(t, runTool(t, "schema", "--json", pkg, "items"), &res)
	var doc map[string]any
	json.Unmarshal(res.Schema, &doc)
	defs, _ := doc["$defs"].(map[string]any)
	if *res.Input != "items" || doc["$ref"] != "metais:///inputs/items.schema.yaml" || doc["title"] != "Items" || len(defs) != 1 || defs["inputs/items.schema.yaml"] == nil {
		t.Errorf("the schema of items: %s", res.Schema)
	}
	if doc["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
		t.Errorf("the input's document has no $schema: %s", res.Schema)
	}
	var combined schemaResult
	result(t, runTool(t, "schema", "--json", pkg), &combined)
	if combined.Input != nil {
		t.Errorf("the combined schema names an input")
	}
	human := runTool(t, "schema", pkg, "items").stdout
	var a, b any
	json.Unmarshal([]byte(human), &a)
	json.Unmarshal(res.Schema, &b)
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if !bytes.Equal(ja, jb) {
		t.Errorf("the human schema is not the JSON result's")
	}
}

// TestExtractCommand is extract (ADR-008 D5, extract; W4-2.7): the files
// written beneath a new directory, a dry run that writes nothing, an
// existing destination refused with nothing written in it, and no
// --overwrite, which this wave does not offer.
func TestExtractCommand(t *testing.T) {
	testenv.Home(t)
	pkg := browsePackage(t)
	base := testenv.Dir(t)
	to := filepath.Join(base, "new", "dir")
	var res extractResult
	result(t, runTool(t, "extract", "--json", pkg, "references/*.txt", "metais.yaml", "--to", to), &res)
	var got []string
	for _, f := range res.Files {
		got = append(got, f.Path)
	}
	if res.DryRun || !slices.Equal(got, []string{"metais.yaml", "references/crlf.txt", "references/ctx.txt", "references/empty.txt", "references/long.txt", "references/nolf.txt"}) {
		t.Errorf("extract: %+v", res)
	}
	if b, err := os.ReadFile(filepath.Join(to, "references", "crlf.txt")); err != nil || string(b) != "a\r\nb\r\nc\r\n" {
		t.Errorf("the file written: %q %v", b, err)
	}
	r := runTool(t, "extract", pkg, "references/bin.dat", "--to", to)
	if r.status != finding.ExitDestination || !strings.Contains(r.stderr, "Found in the directory:\n  metais.yaml\n  references\n") {
		t.Errorf("an existing destination: %d %q", r.status, r.stderr)
	}
	if _, err := os.Stat(filepath.Join(to, "references", "bin.dat")); err == nil {
		t.Error("extract wrote into an existing directory without --overwrite")
	}
	if r := runTool(t, "extract", "--json", pkg, "references/bin.dat", "--to", to); !slices.Contains(r.doc.Next, nextExtractOverwrite) {
		t.Errorf("the refusal does not name --overwrite as its next step: %v", r.doc.Next)
	}
	empty := filepath.Join(base, "empty")
	os.Mkdir(empty, 0o755)
	result(t, runTool(t, "extract", "--json", pkg, "metais.yaml", "--to", empty), &res)
	if res.DryRun || len(res.Files) != 1 {
		t.Errorf("an existing empty destination: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(empty, "metais.yaml")); err != nil {
		t.Errorf("extract did not write into an existing empty directory: %v", err)
	}
	dry := filepath.Join(base, "dry")
	result(t, runTool(t, "extract", "--json", pkg, "references", "--to", dry, "--dry-run"), &res)
	if !res.DryRun || len(res.Files) != 8 {
		t.Errorf("a dry run: %+v", res)
	}
	if _, err := os.Stat(dry); err == nil {
		t.Error("a dry run wrote")
	}
	if r := runTool(t, "extract", pkg, "metais.yaml", "--to", filepath.Join(base, "h")); !strings.HasPrefix(r.stdout, "Wrote 1 file:\n  metais.yaml\n") || strings.Contains(r.stdout+r.stderr, base) {
		t.Errorf("extract's human result: %q %q", r.stdout, r.stderr)
	}
}

// TestReadingPackedAndSourceAgree is ADR-008 D16 item 4 (W4-2.2): for a
// fixture package and its source directory, ls, read, outline, grep,
// schema and inspect give the same results apart from source. A packed
// package's entries include .compiled/, which a source directory's do
// not (ADR-008 D2 items 2 and 3), so ls compares the entries outside it.
func TestReadingPackedAndSourceAgree(t *testing.T) {
	testenv.Home(t)
	for _, pair := range [][2]string{
		{"pkg/opens-source/package", "pkg/opens-packed/package.metais"},
		{"dat/combined-schema-source/package", "dat/combined-schema-packed/package.metais"},
		{"man/canonical-form-source/package", "man/canonical-form-packed/package.metais"},
	} {
		src, packed := filepath.Join(vectorsDir, pair[0]), filepath.Join(vectorsDir, pair[1])
		var listing lsResult
		result(t, runTool(t, "ls", "--json", src, "-r"), &listing)
		lines := [][]string{{"ls", "-r"}, {"ls"}, {"grep", "."}, {"grep", "-i", "-C", "2", "e"}, {"schema"}}
		var md []string
		for _, e := range listing.Entries {
			if e.Type == "file" {
				lines = append(lines, []string{"read", e.Path}, []string{"read", e.Path, "--lines", "1-2"})
				if strings.HasSuffix(e.Path, ".md") {
					md = append(md, e.Path)
					lines = append(lines, []string{"outline", e.Path})
				}
			}
		}
		var inspect inspectResult
		result(t, runTool(t, "inspect", "--json", src), &inspect)
		for _, in := range *inspect.Inputs {
			lines = append(lines, []string{"schema", in.Name})
		}
		if len(md) == 0 && strings.HasPrefix(pair[0], "pkg/") {
			t.Errorf("%s has no Markdown entry to outline", pair[0])
		}
		for _, l := range lines {
			a := runTool(t, append([]string{l[0], "--json", src}, l[1:]...)...)
			b := runTool(t, append([]string{l[0], "--json", packed}, l[1:]...)...)
			ra, rb := a.doc.Result, b.doc.Result
			if l[0] == "ls" {
				ra, rb = withoutCompiled(t, ra), withoutCompiled(t, rb)
			}
			if !bytes.Equal(ra, rb) || a.status != b.status || !slices.Equal(a.codes(), b.codes()) {
				t.Errorf("metais %q of %s and of %s differ:\n%s\n%s", l, pair[0], pair[1], ra, rb)
			}
		}
	}
}

// withoutCompiled is an ls result without the entries beneath .compiled/.
func withoutCompiled(t *testing.T, b json.RawMessage) json.RawMessage {
	var res lsResult
	if err := json.Unmarshal(b, &res); err != nil {
		t.Fatal(err)
	}
	res.Entries = slices.DeleteFunc(res.Entries, func(e lsEntry) bool { return strings.HasPrefix(e.Path, ".compiled/") })
	out, _ := json.Marshal(res)
	return out
}

// TestReadLinesDecompressesOnceMore is ADR-010 D15 item 2's budget
// (W4-2.4), counted rather than timed: read --lines a-b of a packed
// package decodes the entry it reads at most once more after the open
// has verified every entry, decoding each once.
func TestReadLinesDecompressesOnceMore(t *testing.T) {
	testenv.Home(t)
	type decoder interface{ Decodes() int }
	var pkg *loader.Package
	var atOpen int
	opened = func(p *loader.Package) { pkg, atOpen = p, p.Tree.(decoder).Decodes() }
	t.Cleanup(func() { opened = nil })
	packed := filepath.Join(vectorsDir, "pkg/opens-packed/package.metais")
	for _, argv := range [][]string{
		{"read", packed, "references/standards/practice.md", "--lines", "2-3"},
		{"read", "--json", packed, "references/standards/practice.md", "--lines", "1-"},
		{"read", packed, "inputs/project.md", "--number", "--lines", "2"},
		{"outline", packed, "references/standards/practice.md"},
	} {
		pkg = nil
		if r := runTool(t, argv...); r.status != 0 {
			t.Fatalf("metais %q: %s", argv, r.stderr)
		}
		d, ok := pkg.Tree.(decoder)
		if !ok {
			t.Fatalf("a packed package's tree counts no decodes")
		}
		if entries := len(pkg.Tree.List()); atOpen < entries || d.Decodes()-atOpen != 1 {
			t.Errorf("metais %q decoded %d entries at open, of %d, and %d after; the budget is one", argv, atOpen, entries, d.Decodes()-atOpen)
		}
	}
}

// TestExtractOverwrite is W5-12.1 and W5-12.2 through the command line:
// with --overwrite, extract writes into a directory that holds entries,
// replacing the file at a path it writes and leaving every other entry,
// a hidden one included, as it was; with --dry-run as well it lists the
// same files and writes nothing.
func TestExtractOverwrite(t *testing.T) {
	testenv.Home(t)
	pkg := browsePackage(t)
	host := func(t *testing.T) string {
		t.Helper()
		to := testenv.Dir(t)
		for _, f := range []string{".hidden", "metais.yaml", "references/theirs.txt"} {
			p := filepath.Join(to, filepath.FromSlash(f))
			os.MkdirAll(filepath.Dir(p), 0o755)
			if err := os.WriteFile(p, []byte("theirs\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return to
	}
	want, _ := os.ReadFile(filepath.Join(pkg, "metais.yaml"))
	if len(want) == 0 {
		t.Fatal("the fixture's manifest is not readable as a file")
	}

	dry := host(t)
	before := snapshot(t, dry)
	var res extractResult
	result(t, runTool(t, "extract", "--json", pkg, "metais.yaml", "references/*.txt", "--to", dry, "--overwrite", "--dry-run"), &res)
	if !res.DryRun || len(res.Files) != 6 {
		t.Errorf("a dry run with --overwrite: %+v", res)
	}
	if after := snapshot(t, dry); !slices.Equal(after, before) {
		t.Errorf("the dry run wrote: %q, was %q", after, before)
	}

	to := host(t)
	r := runTool(t, "extract", pkg, "metais.yaml", "references/*.txt", "--to", to, "--overwrite")
	if r.status != 0 || !strings.HasPrefix(r.stdout, "Wrote 6 files:\n  metais.yaml\n") {
		t.Fatalf("extract --overwrite: %d %q %q", r.status, r.stdout, r.stderr)
	}
	if b, _ := os.ReadFile(filepath.Join(to, "metais.yaml")); !bytes.Equal(b, want) {
		t.Errorf("metais.yaml holds %q, not the entry's bytes", b)
	}
	for _, f := range []string{".hidden", "references/theirs.txt"} {
		if b, err := os.ReadFile(filepath.Join(to, filepath.FromSlash(f))); err != nil || string(b) != "theirs\n" {
			t.Errorf("%s, which extract does not write: %q %v", f, b, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(to, ".metais-staging")); err == nil {
		t.Error("the staging directory was left")
	}
}

// TestExtractFailedFindings is each failed write of extract as its
// finding: a staging directory left in an existing <dir> is
// staging_exists; a failure that left nothing, nothing_written; and one
// that left what it created, write_failed, listing the files written, an
// empty list included when only <dir> itself was created.
func TestExtractFailedFindings(t *testing.T) {
	s := &session{table: table, line: line{cmd: lookup(table, "extract")}}
	for _, c := range []struct {
		err  *browse.WriteError
		want string
	}{
		{&browse.WriteError{Reason: "exists", Staging: true}, `output.destination.staging_exists {}`},
		{&browse.WriteError{Reason: "permission_denied"}, `output.destination.nothing_written {"reason":"permission_denied"}`},
		{&browse.WriteError{Reason: "io_error", Left: true}, `output.destination.write_failed {"reason":"io_error","written":[]}`},
		{&browse.WriteError{Reason: "io_error", Written: []string{"a.txt"}, Left: true}, `output.destination.write_failed {"reason":"io_error","written":["a.txt"]}`},
	} {
		o := s.extractFailed(c.err)
		if len(o.findings) != 1 || len(o.next) == 0 {
			t.Fatalf("%+v: findings %v, next %v", c.err, o.findings, o.next)
		}
		var doc []struct {
			Code   string
			Params json.RawMessage
		}
		if err := json.Unmarshal(finding.AppendJSON(nil, o.findings[0]), &doc); err != nil {
			t.Fatal(err)
		}
		params := string(doc[0].Params)
		if params == "" {
			params = "{}"
		}
		if got := doc[0].Code + " " + params; got != c.want {
			t.Errorf("%+v: %s, want %s", c.err, got, c.want)
		}
	}
}

// extractDestinationCase is one existing --to that extract --overwrite
// refuses: the host beneath it, made by setup, and the findings it
// reports, in order, each as its code and parameters.
type extractDestinationCase struct {
	name  string
	posix bool // needs a symbolic link, or permissions the superuser ignores
	setup func(t *testing.T, to string)
	want  []string
}

// extractDestinationCases are W5-12.1's per-file checks and commit, through
// the command line, each refusing extract of metais.yaml and
// references/*.txt with --overwrite.
func extractDestinationCases() []extractDestinationCase {
	mk := func(t *testing.T, to string, files ...string) {
		t.Helper()
		for _, f := range files {
			p := filepath.Join(to, filepath.FromSlash(f))
			if strings.HasSuffix(f, "/") {
				if err := os.MkdirAll(p, 0o755); err != nil {
					t.Fatal(err)
				}
				continue
			}
			os.MkdirAll(filepath.Dir(p), 0o755)
			if err := os.WriteFile(p, []byte("theirs\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	return []extractDestinationCase{
		{"a directory and a file in the way, together", false, func(t *testing.T, to string) {
			mk(t, to, "metais.yaml/", "references")
		}, []string{
			`output.destination.not_directory {"path":"references/crlf.txt"}`,
			`output.destination.not_directory {"path":"references/ctx.txt"}`,
			`output.destination.not_directory {"path":"references/empty.txt"}`,
			`output.destination.not_directory {"path":"references/long.txt"}`,
			`output.destination.not_directory {"path":"references/nolf.txt"}`,
			`output.destination.not_regular_file {"path":"metais.yaml","reason":"is_directory"}`,
		}},
		{"a case variant", false, func(t *testing.T, to string) {
			mk(t, to, "References/")
		}, []string{
			`output.destination.case_variant {"path":"references/crlf.txt"}`,
			`output.destination.case_variant {"path":"references/ctx.txt"}`,
			`output.destination.case_variant {"path":"references/empty.txt"}`,
			`output.destination.case_variant {"path":"references/long.txt"}`,
			`output.destination.case_variant {"path":"references/nolf.txt"}`,
		}},
		{"a staging directory left", false, func(t *testing.T, to string) {
			mk(t, to, ".metais-staging/")
		}, []string{`output.destination.staging_exists {}`}},
		{"a link", true, func(t *testing.T, to string) {
			mk(t, to, "target.txt")
			if err := os.Symlink("target.txt", filepath.Join(to, "metais.yaml")); err != nil {
				t.Fatal(err)
			}
		}, []string{`output.destination.link {"path":"metais.yaml"}`}},
		{"a commit that fails part way", true, func(t *testing.T, to string) {
			mk(t, to, "references/")
			locked := filepath.Join(to, "references")
			if err := os.Chmod(locked, 0o555); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.Chmod(locked, 0o755) })
		}, []string{`output.destination.commit_failed {"committed":["metais.yaml"],"reason":"permission_denied"}`}},
	}
}

// runsHere reports whether c can be set up on this host.
func (c extractDestinationCase) runsHere() bool {
	return !c.posix || runtime.GOOS != "windows" && os.Geteuid() != 0
}

// TestExtractDestinationRefusals is W5-12.1's every per-file check a
// finding before anything is written, and A1 item 3's commit failure,
// through the command line: each case reports its findings, all of them
// together, with the exit status of a refused destination, names a next
// step, and changes nothing in --to but what a commit already moved.
func TestExtractDestinationRefusals(t *testing.T) {
	testenv.Home(t)
	pkg := browsePackage(t)
	for _, c := range extractDestinationCases() {
		if !c.runsHere() {
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			for _, dry := range []bool{false, true} {
				to := testenv.Dir(t)
				c.setup(t, to)
				before := snapshot(t, to)
				argv := []string{"extract", "--json", pkg, "metais.yaml", "references/*.txt", "--to", to, "--overwrite"}
				if dry {
					argv = append(argv, "--dry-run")
				}
				r := runTool(t, argv...)
				var got []string
				for _, f := range r.doc.Findings {
					p, _ := json.Marshal(f.Params)
					got = append(got, f.Code+" "+string(p))
				}
				want := c.want
				if dry && strings.HasPrefix(c.want[0], "output.destination.commit_failed") {
					// A dry run makes the checks, which this host passes;
					// only the commit finds the directory unwritable.
					if r.status != 0 {
						t.Errorf("dry run: %s", r.stdout)
					}
				} else {
					if r.status != finding.ExitDestination || !slices.Equal(got, want) || len(r.doc.Next) == 0 {
						t.Errorf("dry run %v: status %d, findings\n%s\nwant\n%s\nnext %v", dry, r.status, strings.Join(got, "\n"), strings.Join(want, "\n"), r.doc.Next)
					}
				}
				after := snapshot(t, to)
				if !dry && strings.HasPrefix(c.want[0], "output.destination.commit_failed") {
					if b, err := os.ReadFile(filepath.Join(to, "metais.yaml")); err != nil || len(b) == 0 || string(b) == "theirs\n" {
						t.Errorf("the file committed: %q %v", b, err)
					}
					if _, err := os.Lstat(filepath.Join(to, ".metais-staging")); err == nil {
						t.Error("the staging directory was left")
					}
					continue
				}
				if !slices.Equal(after, before) {
					t.Errorf("dry run %v: the refusal wrote: %q, was %q", dry, after, before)
				}
			}
		})
	}
}
