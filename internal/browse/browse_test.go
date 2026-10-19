// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package browse

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/pkgtree"
	"github.com/kottera/metais/internal/pkgtree/pkgtreetest"
	"github.com/kottera/metais/internal/regexprofile"
	"github.com/kottera/metais/internal/testenv"
)

func tree(t *testing.T, files map[string]string) pkgtree.Tree {
	t.Helper()
	m := map[string][]byte{}
	for k, v := range files {
		m[k] = []byte(v)
	}
	tr, err := pkgtreetest.New(m)
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

// TestText is ADR-008 D5's text rule (RULINGS R2-122) and its line count.
func TestText(t *testing.T) {
	for _, c := range []struct {
		b     string
		text  bool
		lines int
	}{
		{"", true, 0},
		{"a", true, 1},
		{"a\n", true, 1},
		{"a\nb", true, 2},
		{"a\r\nb\r\n", true, 2},
		{"\n\n", true, 2},
		{"a\x00b", false, 1},
		{"\xff\n", false, 1},
		{"é\n", true, 1},
	} {
		if got := IsText([]byte(c.b)); got != c.text {
			t.Errorf("IsText(%q) = %v", c.b, got)
		}
		if got := LineCount([]byte(c.b)); got != c.lines {
			t.Errorf("LineCount(%q) = %d, want %d", c.b, got, c.lines)
		}
	}
}

// TestReadLines is ADR-008 D16 item 8's table for read --lines: N, N-M and
// N-, a start beyond the end, a clamped end, CRLF, no final LF and an empty
// entry, and the ranges refused as malformed.
func TestReadLines(t *testing.T) {
	for _, c := range []struct {
		entry, rng  string
		want        string
		first, last int
		total       int
		outOfRange  bool
	}{
		{"a\nb\nc\n", "2", "b\n", 2, 2, 3, false},
		{"a\nb\nc\n", "2-3", "b\nc\n", 2, 3, 3, false},
		{"a\nb\nc\n", "2-", "b\nc\n", 2, 3, 3, false},
		{"a\nb\nc\n", "1-99", "a\nb\nc\n", 1, 3, 3, false},
		{"a\nb\nc\n", "3-3", "c\n", 3, 3, 3, false},
		{"a\nb\nc\n", "4", "", 0, 0, 3, true},
		{"a\nb\nc\n", "4-", "", 0, 0, 3, true},
		{"a\r\nb\r\nc\r\n", "2", "b\r\n", 2, 2, 3, false},
		{"a\nb\nc", "3", "c", 3, 3, 3, false},
		{"a\nb\nc", "2-", "b\nc", 2, 3, 3, false},
		{"", "1", "", 0, 0, 0, true},
		{"\n", "1", "\n", 1, 1, 1, false},
	} {
		r, ok := ParseRange(c.rng)
		if !ok {
			t.Fatalf("ParseRange(%q) refused", c.rng)
		}
		s, err := SelectLines([]byte(c.entry), &r)
		if c.outOfRange {
			if !errors.Is(err, ErrLineOutOfRange) || s.Total != c.total {
				t.Errorf("%q --lines %s: %v, total %d; want out of range, total %d", c.entry, c.rng, err, s.Total, c.total)
			}
			continue
		}
		if err != nil || string(s.Bytes) != c.want || s.First != c.first || s.Last != c.last || s.Total != c.total {
			t.Errorf("%q --lines %s = %q %d-%d of %d, %v; want %q %d-%d of %d", c.entry, c.rng, s.Bytes, s.First, s.Last, s.Total, err, c.want, c.first, c.last, c.total)
		}
	}
	whole, _ := SelectLines(nil, nil)
	if whole.First != 0 || whole.Last != 0 || whole.Total != 0 {
		t.Errorf("an empty entry read whole: %+v", whole)
	}
	for _, bad := range []string{"", "0", "-1", "1-0", "3-2", "a", "1-b", "1--2", "+1", " 1", "1,2", "99999999999999999999", "1-2-3"} {
		if r, ok := ParseRange(bad); ok {
			t.Errorf("ParseRange(%q) = %+v, want refused", bad, r)
		}
	}
}

// TestOutline is ADR-008 D16 item 8's table for outline: ATX, setext,
// multi-line setext, --- after a paragraph, # in a fence, frontmatter, and
// each section's end line.
func TestOutline(t *testing.T) {
	src := strings.Join([]string{
		"---",            // 1
		"title: x",       // 2
		"---",            // 3
		"# One",          // 4
		"text",           // 5
		"## Two",         // 6
		"```",            // 7
		"# not a head",   // 8
		"```",            // 9
		"Setext",         // 10
		"===",            // 11
		"Two lines",      // 12
		"of setext",      // 13
		"---",            // 14
		"para",           // 15
		"",               // 16
		"####### seven",  // 17
		"<div>",          // 18
		"# html",         // 19
		"</div>",         // 20
		"",               // 21
		"### `code` *x*", // 22
		"last",           // 23
	}, "\n") + "\n"
	got := Outline([]byte(src))
	want := []Heading{
		{1, "One", 4, 9},
		{2, "Two", 6, 9},
		{1, "Setext", 10, 23},
		{2, "Two lines of setext", 12, 23},
		{3, "code x", 22, 23},
	}
	if !slices.Equal(got, want) {
		t.Errorf("Outline:\n got %+v\nwant %+v", got, want)
	}
	if got := Outline([]byte("no headings\n")); got == nil || len(got) != 0 {
		t.Errorf("an entry with no headings: %#v", got)
	}
	// CRLF lines count as lines, and a CR is not part of a heading's text.
	if got := Outline([]byte("# A\r\nx\r\n# B\r\n")); !slices.Equal(got, []Heading{{1, "A", 1, 2}, {1, "B", 3, 3}}) {
		t.Errorf("CRLF: %+v", got)
	}
}

// TestList is ls (ADR-008 D5): the root, a directory, a file, recursion,
// the directories implied by entry paths with their file counts, the
// byte order of the paths as written, and a path that names nothing.
func TestList(t *testing.T) {
	tr := tree(t, map[string]string{
		"metais.yaml":           "m",
		"inputs/a.json":         "{}",
		"inputs/x/b.json":       "{}!",
		"inputs-old.txt":        "x",
		"references/wcag/1.md":  "# x\n",
		".compiled/schema.json": "{}",
	})
	paths := func(es []Entry) []string {
		var out []string
		for _, e := range es {
			out = append(out, e.Path)
		}
		return out
	}
	root, err := List(tr, ".", false)
	if err != nil {
		t.Fatal(err)
	}
	if got := paths(root); !slices.Equal(got, []string{".compiled/", "inputs-old.txt", "inputs/", "metais.yaml", "references/"}) {
		t.Errorf("ls of the root: %q", got)
	}
	if root[2] != (Entry{Path: "inputs/", Dir: true, Files: 2}) {
		t.Errorf("inputs/: %+v", root[2])
	}
	for _, p := range []string{"", ".", "./"} {
		if again, _ := List(tr, p, false); !slices.Equal(again, root) {
			t.Errorf("ls %q is not the root's listing", p)
		}
	}
	sub, _ := List(tr, "inputs/", false)
	if got := paths(sub); !slices.Equal(got, []string{"inputs/a.json", "inputs/x/"}) {
		t.Errorf("ls inputs/: %q", got)
	}
	rec, _ := List(tr, "inputs", true)
	if !slices.Equal(rec, []Entry{{Path: "inputs/a.json", Size: 2}, {Path: "inputs/x/", Dir: true, Files: 1}, {Path: "inputs/x/b.json", Size: 3}}) {
		t.Errorf("ls -r inputs: %+v", rec)
	}
	file, _ := List(tr, "metais.yaml", false)
	if !slices.Equal(file, []Entry{{Path: "metais.yaml", Size: 1}}) {
		t.Errorf("ls of a file: %+v", file)
	}
	all, _ := List(tr, ".", true)
	if len(all) != 11 {
		t.Errorf("ls -r of the root: %q", paths(all))
	}
	for p, near := range map[string]string{"nope": ".", "inputs/nope/x": "inputs/", "inputs/x/c": "inputs/x/", "Inputs": ".", "inputs/a.json/x": "inputs/"} {
		_, err := List(tr, p, false)
		var nf *NotFoundError
		if !errors.As(err, &nf) || nf.Nearest != near {
			t.Errorf("ls %q: %v, want not found with nearest %q", p, err, near)
		}
	}
	if _, err := Lookup(tr, "inputs/"); !errors.Is(err, ErrIsDirectory) {
		t.Errorf("Lookup of a directory: %v", err)
	}
	if _, err := Lookup(tr, "."); !errors.Is(err, ErrIsDirectory) {
		t.Errorf("Lookup of the root: %v", err)
	}
	if info, err := Lookup(tr, "inputs/a.json"); err != nil || info.Size != 2 {
		t.Errorf("Lookup of a file: %+v %v", info, err)
	}
}

func grepOf(t *testing.T, tr pkgtree.Tree, pattern string, globs []string, context, max int) GrepResult {
	t.Helper()
	re, err := regexprofile.CompileRE2(pattern, false, limits.Defaults(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if globs == nil {
		globs = []string{"**"}
	}
	var entries []string
	for _, e := range tr.List() {
		for _, gs := range globs {
			g, err := CompileGlob(gs)
			if err != nil {
				t.Fatal(err)
			}
			if g.Match(e) {
				entries = append(entries, e)
				break
			}
		}
	}
	res, err := Grep(tr, entries, re, nil, GrepOptions{Context: context, MaxResults: max}, 0)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// TestGrep is ADR-008 D16 item 8's table for grep: anchors per line
// without the line ending, context, binary entries skipped, truncation
// past --max-results, the long-line cut and the dot-segment rule.
func TestGrep(t *testing.T) {
	long := strings.Repeat("x", 1020) + "ééé" + "tail"
	tr := tree(t, map[string]string{
		"a.md":         "alpha\r\nbeta\r\ngamma\n",
		"b.md":         "one\ntwo\nthree\nfour\nfive\nsix\n",
		"bin.dat":      "beta\x00",
		"long.txt":     long + "\n",
		".compiled/x":  "beta\n",
		"references/y": "betamax\nbeta beta\n",
	})
	res := grepOf(t, tr, "^beta$", nil, 0, 100)
	if res.MatchingLines != 1 || len(res.Matches) != 1 || res.Matches[0].Path != "a.md" || res.Matches[0].Line != 2 || res.Matches[0].Text != "beta" {
		t.Errorf("^beta$: %+v", res)
	}
	if !slices.Equal(res.Skipped, []string{"bin.dat"}) {
		t.Errorf("skipped: %q", res.Skipped)
	}
	res = grepOf(t, tr, "beta", nil, 0, 100)
	if res.MatchingLines != 3 {
		t.Errorf("beta matched %d lines, want 3 (a line with two matches counts once, .compiled/ is not searched)", res.MatchingLines)
	}
	res = grepOf(t, tr, "beta", []string{".compiled/**"}, 0, 100)
	if res.MatchingLines != 1 || res.Matches[0].Path != ".compiled/x" {
		t.Errorf("a glob naming .compiled: %+v", res)
	}
	res = grepOf(t, tr, "three|five", []string{"b.md"}, 1, 100)
	m := res.Matches
	if len(m) != 2 || !slices.Equal(m[0].Before, []Shown{{Line: 2, Text: "two"}}) || !slices.Equal(m[0].After, []Shown{{Line: 4, Text: "four"}}) ||
		!slices.Equal(m[1].Before, []Shown{{Line: 4, Text: "four"}}) || !slices.Equal(m[1].After, []Shown{{Line: 6, Text: "six"}}) {
		t.Errorf("context 1: %+v", m)
	}
	res = grepOf(t, tr, "one", []string{"b.md"}, 3, 100)
	if len(res.Matches[0].Before) != 0 || len(res.Matches[0].After) != 3 {
		t.Errorf("context at the start: %+v", res.Matches[0])
	}
	res = grepOf(t, tr, "e", []string{"b.md"}, 0, 2)
	if res.MatchingLines != 3 || len(res.Matches) != 2 || !res.Truncated {
		t.Errorf("truncation: %d of %d, truncated %v", len(res.Matches), res.MatchingLines, res.Truncated)
	}
	res = grepOf(t, tr, "tail", nil, 0, 100)
	if s := res.Matches[0].Shown; len(s.Text) != 1020+2*2 || s.Omitted != len(long)-len(s.Text) || !strings.HasSuffix(s.Text, "éé") {
		t.Errorf("the long line: %d bytes shown, %d omitted", len(s.Text), s.Omitted)
	}
}

// TestGrepRegexWork is regex-work over grep (ADR-009 D10 item 7): each
// line is charged the program's size times its length, and a search the
// meter refuses gives no result.
func TestGrepRegexWork(t *testing.T) {
	tr := tree(t, map[string]string{"a.txt": strings.Repeat("abcdefgh\n", 100)})
	re, _ := regexprofile.CompileRE2("(a|aa)*b", false, limits.Defaults(), nil)
	need := int64(re.Instructions() * 8 * 100)
	for _, c := range []struct {
		budget int64
		err    error
	}{{need, nil}, {need - 1, ErrRegexWork}} {
		m := regexprofile.NewMeter(c.budget)
		re, _ := regexprofile.CompileRE2("(a|aa)*b", false, limits.Defaults(), m)
		_, err := Grep(tr, []string{"a.txt"}, re, m, GrepOptions{MaxResults: 1}, 0)
		if !errors.Is(err, c.err) {
			t.Errorf("a budget of %d for %d units of work: %v, want %v", c.budget, need, err, c.err)
		}
	}
}

// TestCompileRE2 is grep's pattern bounds (ADR-009 D10 item 5): the
// length of the pattern as written, the program's size, and RE2 syntax.
func TestCompileRE2(t *testing.T) {
	lim := limits.Defaults()
	if _, err := regexprofile.CompileRE2(strings.Repeat("a", 4097), false, lim, nil); !isLimit(err, limits.PatternLength) {
		t.Errorf("a pattern over pattern-length: %v", err)
	}
	if _, err := regexprofile.CompileRE2(strings.Repeat("a", 4096), false, lim, nil); !isLimit(err, limits.RegexProgram) {
		t.Errorf("a program over regex-program: %v", err)
	}
	if _, err := regexprofile.CompileRE2("(", false, lim, nil); !errors.Is(err, regexprofile.ErrRE2Syntax) {
		t.Errorf("a pattern that does not parse: %v", err)
	}
	re, err := regexprofile.CompileRE2("BETA", true, lim, nil)
	if err != nil || !re.MatchString("beta") {
		t.Errorf("--ignore-case: %v", err)
	}
	if _, err := regexprofile.CompileRE2(`(?P<x>a)\Aa`, false, lim, nil); err != nil {
		t.Errorf("RE2 syntax is grep's: %v", err)
	}
}

func isLimit(err error, id limits.ID) bool {
	var le *regexprofile.LimitError
	return errors.As(err, &le) && le.Limit == id
}

// TestExtract is extract's write (ADR-008 D5): every entry selected, at
// its path beneath a directory that did not exist, created with its
// missing parents; and an existing destination refused with nothing
// written, listing what it holds.
func TestExtract(t *testing.T) {
	tr := tree(t, map[string]string{"metais.yaml": "m", "inputs/a.json": "{}", "inputs/x/b.json": "{}!"})
	base := testenv.Dir(t)
	to := filepath.Join(base, "a", "b", "out")
	files, err := Extract(tr, []string{"inputs/a.json", "inputs/x/b.json"}, to, ExtractOptions{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(files, []File{{"inputs/a.json", 2}, {"inputs/x/b.json", 3}}) {
		t.Errorf("files: %+v", files)
	}
	for name, want := range map[string]string{"inputs/a.json": "{}", "inputs/x/b.json": "{}!"} {
		b, err := os.ReadFile(filepath.Join(to, filepath.FromSlash(name)))
		if err != nil || string(b) != want {
			t.Errorf("%s: %q %v", name, b, err)
		}
		info, _ := os.Stat(filepath.Join(to, filepath.FromSlash(name)))
		if info.Mode().Perm()&0o600 != 0o600 || info.Mode().Perm()&0o111 != 0 {
			t.Errorf("%s has mode %v", name, info.Mode())
		}
	}
	if _, err := os.Stat(filepath.Join(to, "metais.yaml")); err == nil {
		t.Error("an entry not selected was written")
	}
	_, err = Extract(tr, []string{"metais.yaml"}, to, ExtractOptions{}, 0)
	var de *DestinationError
	if !errors.As(err, &de) || de.NotDirectory || !slices.Equal(de.Entries, []string{"inputs"}) || de.More {
		t.Errorf("an existing destination: %#v", err)
	}
	empty := filepath.Join(base, "empty")
	os.Mkdir(empty, 0o755)
	if _, err := Extract(tr, []string{"metais.yaml"}, empty, ExtractOptions{}, 0); !errors.As(err, &de) || len(de.Entries) != 0 {
		t.Errorf("an existing empty destination: %#v", err)
	}
	if _, err := os.Stat(filepath.Join(empty, "metais.yaml")); err == nil {
		t.Error("extract wrote into an existing directory")
	}
	file := filepath.Join(base, "file")
	os.WriteFile(file, nil, 0o644)
	for _, p := range []string{file, filepath.Join(file, "under")} {
		if _, err := Extract(tr, []string{"metais.yaml"}, p, ExtractOptions{}, 0); !errors.As(err, &de) || !de.NotDirectory {
			t.Errorf("a destination that is or lies beneath a file: %#v", err)
		}
	}
	dry := filepath.Join(base, "dry")
	files, err = Extract(tr, []string{"metais.yaml"}, dry, ExtractOptions{DryRun: true}, 0)
	if err != nil || !slices.Equal(files, []File{{"metais.yaml", 1}}) {
		t.Errorf("dry run: %+v %v", files, err)
	}
	if _, err := os.Stat(dry); err == nil {
		t.Error("a dry run wrote")
	}
	if _, err := Extract(tr, []string{"metais.yaml"}, to, ExtractOptions{DryRun: true}, 0); !errors.As(err, &de) {
		t.Errorf("a dry run into an existing destination: %v", err)
	}
}

// TestExtractStopsAndRemovesNothing is a write that fails part way: the
// files already written are reported and left, and nothing else is
// touched.
func TestExtractStopsAndRemovesNothing(t *testing.T) {
	tr := tree(t, map[string]string{"a.txt": "a", "b/c.txt": "c"})
	to := filepath.Join(testenv.Dir(t), "out")
	// The tree changes beneath Extract: its second entry cannot be read.
	_, err := Extract(failing{tr, "b/c.txt"}, []string{"a.txt", "b/c.txt"}, to, ExtractOptions{}, 0)
	var we *WriteError
	if !errors.As(err, &we) || !slices.Equal(we.Written, []string{"a.txt"}) || we.Reason != string(hostfs.IOError) {
		t.Fatalf("a failed read: %#v", err)
	}
	if b, err := os.ReadFile(filepath.Join(to, "a.txt")); err != nil || string(b) != "a" {
		t.Errorf("the file written before the failure was not left: %v", err)
	}
}

type failing struct {
	pkgtree.Tree
	bad string
}

func (f failing) Open(p string) (io.ReadCloser, error) {
	if p == f.bad {
		return nil, errors.New("unreadable")
	}
	return f.Tree.Open(p)
}

// TestContentReadsInPlace is Content's seam: without a fault it writes
// nothing; under fault.UnpackToTemp it leaves an extracted copy beneath
// the working directory and returns the same bytes.
func TestContentReadsInPlace(t *testing.T) {
	tr := tree(t, map[string]string{"a.txt": "hello"})
	b, err := Content(tr, "a.txt", 0)
	if err != nil || !bytes.Equal(b, []byte("hello")) {
		t.Errorf("Content: %q %v", b, err)
	}
}

// TestUnpackToTemp is fault.UnpackToTemp at its seam: the entry is read
// back from a file extracted beneath the working directory, which is left
// for the read-in-place detector to find.
func TestUnpackToTemp(t *testing.T) {
	tr := tree(t, map[string]string{"a/b.txt": "hello"})
	dir := testenv.Dir(t)
	t.Chdir(dir)
	b, err := Content(tr, "a/b.txt", fault.UnpackToTemp)
	if err != nil || string(b) != "hello" {
		t.Fatalf("Content under UnpackToTemp: %q %v", b, err)
	}
	found, _ := filepath.Glob(filepath.Join(dir, ".metais-unpack-*", "a", "b.txt"))
	if len(found) != 1 {
		t.Errorf("the extracted file is not left beneath the working directory: %q", found)
	}
}
