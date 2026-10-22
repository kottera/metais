// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/testenv"
)

// The hostile corpus's Markdown cases (ADR-009 D17 item 10): every case
// cmark's pathological suite names, each input built here by a program of
// this repository, in its own words, and run through the outline command
// and through a Markdown input's frontmatter splitting in validate. Each
// input is shaped to make a reading that rescans what it has read
// quadratic, or one that recurses as deep as it nests exhaust its stack;
// at these sizes a linear reading costs a fraction of a second.

// repeat is s, n times.
func repeat(s string, n int) string { return strings.Repeat(s, n) }

// markdownCases builds the input of each case, by the name cmark's suite
// gives it (internal/markdown's TestPathologicalCasesRead pins the same
// names against cmark's own file).
var markdownCases = map[string]func() string{
	// Strong emphasis opened inside emphasis inside strong emphasis, deep,
	// and closed in the mirror order.
	"nested strong emph": func() string { return repeat("*a **a ", 30000) + "b" + repeat(" a** a*", 30000) },
	// Underscores that can only close, and none ever opened.
	"many emph closers with no openers": func() string { return repeat("x_ ", 50000) },
	// Underscores that can only open, and none ever closed.
	"many emph openers with no closers": func() string { return repeat("_x ", 50000) },
	// Closing brackets with no bracket open.
	"many link closers with no openers": func() string { return repeat("x]", 50000) },
	// Opening brackets never closed.
	"many link openers with no closers": func() string { return repeat("[x", 50000) },
	// An asterisk opener and an underscore closer, which never pair.
	"mismatched openers and closers": func() string { return repeat("*x_ ", 50000) },
	// Asterisk openers left on the stack while underscore runs that both
	// open and close arrive after them.
	"issue #389": func() string { return repeat("*x ", 25000) + repeat("_x*_ ", 25000) },
	// Runs whose lengths sum to a multiple of three, which the rule of
	// three forbids from pairing, after one double opener.
	"openers and closers multiple of 3": func() string { return "x**y" + repeat("z* ", 50000) },
	// A bracket opener and an emphasis closer, alternating.
	"link openers and emph closers": func() string { return repeat("[ x_", 50000) },
	// A bracket, a parenthesis, then what looks like the end of a link
	// text and the start of its destination, never finished.
	"pattern [ (]( repeated": func() string { return repeat("[ (](", 50000) },
	// An image opener holding a bracket pair and an empty destination.
	"pattern ![[]() repeated": func() string { return repeat("![[]()", 50000) },
	// Strong emphasis around a link whose text holds emphasis that
	// crosses it.
	"hard link/emph case": func() string { return "**y [x*z**w*](v)" },
	// Brackets nested deep around one word.
	"nested brackets": func() string { return repeat("[", 50000) + "x" + repeat("]", 50000) },
	// Block quotes nested deep on one line.
	"nested block quotes": func() string { return repeat("> ", 50000) + "x\n" },
	// List items each indented under the one before.
	"deeply nested lists": func() string {
		var b strings.Builder
		for i := range 1000 {
			b.WriteString(repeat("  ", i) + "* x\n")
		}
		return b.String()
	},
	// NUL characters amid text.
	"U+0000 in input": func() string { return "xyz\x00uv\x00" },
	// Backtick runs of every length up to a bound, none closed by a run
	// of its own length.
	"backticks": func() string {
		var b strings.Builder
		for n := 1; n < 2000; n++ {
			b.WriteString("x" + repeat("`", n))
		}
		return b.String()
	},
	// A link whose destination opens with < and never closes.
	"unclosed links A": func() string { return repeat("[x](<y", 50000) },
	// A link whose destination never reaches its closing parenthesis.
	"unclosed links B": func() string { return repeat("[x](y", 50000) },
	// HTML comment openers never closed, after a closing-tag start.
	"unclosed <!--": func() string { return "</" + repeat("<!--", 100000) },
	// List markers nested deep on one line, then many blank lines.
	"empty lines in deeply nested lists": func() string { return repeat("- ", 20000) + "x" + repeat("\n", 20000) },
	// The same inside a block quote, each blank line quoted.
	"empty lines in deeply nested lists in blockquote": func() string {
		return "> " + repeat("- ", 20000) + "x\n" + repeat(">\n", 20000)
	},
	// Block quotes nested deep, then emphasis openers on the same line.
	"emph in deep blockquote": func() string { return repeat(">", 50000) + repeat("x*", 50000) },
	// Link reference definitions by the thousand, each followed by a
	// reference to the first, as a map with a weak hash would collide.
	"reference collisions": func() string {
		var b strings.Builder
		for i := 1; i < 20000; i++ {
			b.WriteString("[r" + strconv.Itoa(i) + "]: /u\n\n[r0]\n\n")
		}
		return b.String()
	},
	// Asterisks nested deep around one word.
	"nested inlines": func() string { return repeat("*", 20000) + "x" + repeat("*", 20000) },
}

// cmarkCases are the names of the cases of cmark's pathological suite, in
// its order: every one has an input above.
var cmarkCases = []string{
	"nested strong emph", "many emph closers with no openers", "many emph openers with no closers",
	"many link closers with no openers", "many link openers with no closers",
	"mismatched openers and closers", "issue #389", "openers and closers multiple of 3",
	"link openers and emph closers", "pattern [ (]( repeated", "pattern ![[]() repeated",
	"hard link/emph case", "nested brackets", "nested block quotes", "deeply nested lists",
	"U+0000 in input", "backticks", "unclosed links A", "unclosed links B", "unclosed <!--",
	"empty lines in deeply nested lists", "empty lines in deeply nested lists in blockquote",
	"emph in deep blockquote", "reference collisions", "nested inlines",
}

// markdownPackage is a source directory whose page input is Markdown.
func markdownPackage(t *testing.T) string {
	return writeTree(t, map[string]string{
		"metais.yaml": `metais: "1.0"
name: org.example.markdown
version: 1.0.0
title: Markdown
description: A package whose input is Markdown.
guide: Supply the page.
inputs:
  - name: page
    title: Page
    schema: inputs/page.schema.json
    forms: [markdown]
outputs:
  - id: report
    template: report.md.liquid
`,
		"inputs/page.schema.json":    `{"type": "object"}`,
		"templates/report.md.liquid": "Report\n",
	})
}

// TestHostileMarkdown runs every case through outline, as a package entry
// with and without a closed frontmatter block before it, and through
// validate as a Markdown input three ways: as the whole file, as the text
// of a closed frontmatter block, and after an opening line that is never
// closed. Every run ends with a result or with findings, never an internal
// error, within D18's ceiling of CPU time, and outline lists only headings
// at lines of the entry, in order.
func TestHostileMarkdown(t *testing.T) {
	testenv.Home(t)
	const cpuBound = 10 * time.Second // ADR-009 D18; each run takes a fraction of it, under the race detector too
	if got := slices.Sorted(func(yield func(string) bool) {
		for name := range markdownCases {
			if !yield(name) {
				return
			}
		}
	}); !slices.Equal(got, slices.Sorted(slices.Values(cmarkCases))) {
		t.Fatalf("the cases built are %q, and cmark's suite names %q", got, cmarkCases)
	}
	pkg := markdownPackage(t)
	if err := os.MkdirAll(filepath.Join(pkg, "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	measure := func(t *testing.T, what string, argv ...string) ran {
		t.Helper()
		before, measured := selfCPU()
		r := runTool(t, argv...)
		if after, _ := selfCPU(); measured && after-before > cpuBound {
			t.Errorf("%s took %v of CPU, more than %v", what, after-before, cpuBound)
		}
		return r
	}
	for _, name := range cmarkCases {
		t.Run(name, func(t *testing.T) {
			in := markdownCases[name]()
			for i, src := range []string{in, "---\ntitle: T\n---\n" + in} {
				entry := "references/case" + strconv.Itoa(i) + ".md"
				if err := os.WriteFile(filepath.Join(pkg, filepath.FromSlash(entry)), []byte(src), 0o644); err != nil {
					t.Fatal(err)
				}
				r := measure(t, "outline "+entry, "outline", "--json", pkg, entry)
				if strings.ContainsRune(src, 0) {
					// A NUL makes the entry binary, which outline refuses
					// to read as lines (ADR-008 D5).
					if r.status != finding.ExitUsage || !slices.Equal(r.codes(), []string{"usage.not_text"}) {
						t.Errorf("outline %s: exit %d, findings %q, want usage.not_text", entry, r.status, findingParams(r))
					}
					continue
				}
				var res outlineResult
				result(t, r, &res)
				lines := strings.Count(src, "\n")
				if !strings.HasSuffix(src, "\n") {
					lines++
				}
				last := 1
				for _, hd := range res.Headings {
					if hd.Level < 1 || hd.Level > 6 || hd.Line < last || hd.Line > lines || hd.EndLine < hd.Line || hd.EndLine > lines {
						t.Fatalf("outline %s: heading %+v of a %d-line entry, after line %d", entry, hd, lines, last)
					}
					last = hd.Line
				}
			}
			for _, src := range []string{in, "---\n" + in + "\n---\n", "---\n" + in} {
				data := writeTree(t, map[string]string{"page.md": src})
				if r := measure(t, "validate", "validate", "--json", pkg, "--data", data); r.status != finding.ExitOK && r.status != finding.ExitFindings {
					t.Errorf("validate: exit %d: %s", r.status, r.stdout[:min(len(r.stdout), 2048)])
				}
			}
		})
	}
}
