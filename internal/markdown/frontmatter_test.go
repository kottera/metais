// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/testenv"
)

var testSource = Source{
	Input: finding.Subject{Kind: finding.SubjectInput, Name: "articles"},
	At:    finding.Location{Origin: finding.OriginDataDir, File: "articles/001.md"},
}

// TestSplitFrontmatter is a table of ADR-003 D13 item 2's rules, each case
// a file the rule decides.
func TestSplitFrontmatter(t *testing.T) {
	testenv.Home(t)
	absent := -1 // frontLine 0, front nil, body the whole file normalized
	for _, tc := range []struct {
		name, src string
		front     string
		frontLine int
		body      string
	}{
		// The first line is exactly "---": frontmatter is present.
		{"present", "---\ntitle: A\n---\n# A\n", "title: A\n", 2, "# A\n"},
		{"several lines", "---\na: 1\nb: 2\n---\nbody\n", "a: 1\nb: 2\n", 2, "body\n"},
		{"empty", "---\n---\nbody\n", "", 2, "body\n"},
		{"blank lines inside", "---\n\na: 1\n\n---\nx", "\na: 1\n\n", 2, "x"},

		// It ends at the next line that is exactly "---".
		{"first closing line wins", "---\na: 1\n---\nb\n---\n", "a: 1\n", 2, "b\n---\n"},
		{"closing at the end of the file", "---\na: 1\n---", "a: 1\n", 2, ""},
		{"closing line then end of file", "---\na: 1\n---\n", "a: 1\n", 2, ""},
		{"a longer rule does not close", "---\na\n----\n---\nb", "a\n----\n", 2, "b"},
		{"a rule with a space does not close", "---\na\n--- \n---\nb", "a\n--- \n", 2, "b"},
		{"an indented rule does not close", "---\na\n ---\n---\nb", "a\n ---\n", 2, "b"},
		{"a rule with text does not close", "---\na\n---b\n---\nb", "a\n---b\n", 2, "b"},

		// The body is the text after the closing line, nothing else changed.
		{"a blank line after stays in the body", "---\n---\n\npara\n", "", 2, "\npara\n"},
		{"the final LF is kept", "---\n---\nx\n\n", "", 2, "x\n\n"},
		{"a later rule is body text", "---\na: 1\n---\n---\n", "a: 1\n", 2, "---\n"},

		// After line-break normalization: CRLF and CR are LF.
		{"CRLF", "---\r\na: 1\r\n---\r\nbody\r\n", "a: 1\n", 2, "body\n"},
		{"CR", "---\ra: 1\r---\rbody\r", "a: 1\n", 2, "body\n"},
		{"mixed", "---\r\na\rb\n---\rc\r\nd", "a\nb\n", 2, "c\nd"},
		{"CR CR is two breaks", "---\r\r---\rx", "\n", 2, "x"},

		// A first line that is anything else: no frontmatter.
		{"empty file", "", "", absent, ""},
		{"no frontmatter", "# Title\n\ntext\n", "", absent, ""},
		{"thematic break with stars", "***\na\n---\n", "", absent, ""},
		{"four dashes", "----\na\n---\n", "", absent, ""},
		{"two dashes", "--\na\n---\n", "", absent, ""},
		{"trailing space", "--- \na\n---\n", "", absent, ""},
		{"leading space", " ---\na\n---\n", "", absent, ""},
		{"text after the dashes", "---yaml\na\n---\n", "", absent, ""},
		{"a blank first line", "\n---\na\n---\n", "", absent, ""},
		{"absent with CRLF", "a\r\nb\r", "", absent, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			front, frontLine, body, fs := SplitFrontmatter([]byte(tc.src), testSource, limits.Defaults())
			if len(fs) != 0 {
				t.Fatalf("SplitFrontmatter(%q) reported %v", tc.src, codes(fs))
			}
			if tc.frontLine == absent {
				want := strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(tc.src)
				if front != nil || frontLine != 0 || string(body) != want {
					t.Fatalf("SplitFrontmatter(%q) = %q, %d, %q; want no frontmatter and body %q", tc.src, front, frontLine, body, want)
				}
				return
			}
			if front == nil || string(front) != tc.front || frontLine != tc.frontLine || string(body) != tc.body {
				t.Fatalf("SplitFrontmatter(%q) = %q, %d, %q; want %q, %d, %q", tc.src, front, frontLine, body, tc.front, tc.frontLine, tc.body)
			}
		})
	}
}

// TestSplitFrontmatterUnclosed is the refusal of a frontmatter block with
// no closing line: markdown.frontmatter_unclosed, at the opening line.
func TestSplitFrontmatterUnclosed(t *testing.T) {
	testenv.Home(t)
	for _, src := range []string{
		"---",
		"---\n",
		"---\r\n",
		"---\na: 1\n",
		"---\na: 1\n--- \n",
		"---\na: 1\n----\nbody\n",
		"---\na: 1\n ---\nbody",
		"---\na: 1\n...\nbody",
	} {
		front, frontLine, body, fs := SplitFrontmatter([]byte(src), testSource, limits.Defaults())
		if front != nil || frontLine != 0 || body != nil {
			t.Errorf("SplitFrontmatter(%q) = %q, %d, %q with a finding; want nothing", src, front, frontLine, body)
		}
		if len(fs) != 1 || fs[0].Code() != CodeFrontmatterUnclosed {
			t.Errorf("SplitFrontmatter(%q) reported %v, want [%s]", src, codes(fs), CodeFrontmatterUnclosed)
			continue
		}
		f := fs[0]
		if f.Class() != finding.ClassContent || f.Subject() != testSource.Input {
			t.Errorf("SplitFrontmatter(%q): class %s, subject %+v", src, f.Class(), f.Subject())
		}
		if l, ok := f.Location(); !ok || l.File != testSource.At.File || l.Origin != testSource.At.Origin || l.Line != 1 || l.Column != 1 {
			t.Errorf("SplitFrontmatter(%q): location %+v, want %s at 1:1", src, l, testSource.At.File)
		}
	}
}

// TestSplitFrontmatterSize is frontmatter-size (ADR-009 D10 item 4): a
// block whose text passes the bound is limit.exceeded, at its first line,
// and one at the bound is read. The bound is on the text after
// normalization, so a CRLF counts one byte.
func TestSplitFrontmatterSize(t *testing.T) {
	testenv.Home(t)
	lowered := limits.Defaults()
	if f, ok := lowered.Lower("frontmatter-size=8"); !ok {
		t.Fatalf("Lower: %v", f.Message())
	}
	for _, tc := range []struct {
		name, src string
		exceeded  bool
	}{
		{"at the bound", "---\nabc\nefg\n---\n", false}, // 8 bytes
		{"at the bound with CRLF", "---\r\nabc\r\nefg\r\n---\r\n", false},
		{"one past the bound", "---\nabc\nefgh\n---\n", true}, // 9 bytes
		{"one long line", "---\n" + strings.Repeat("a", 100) + "\n---\n", true},
		{"unclosed past the bound", "---\n" + strings.Repeat("a\n", 100), true},
		{"unclosed within the bound", "---\nabc\n", false},
		{"no frontmatter", strings.Repeat("a\n", 100), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, fs := SplitFrontmatter([]byte(tc.src), testSource, lowered)
			if !tc.exceeded {
				for _, f := range fs {
					if f.Code() == CodeLimitExceeded {
						t.Fatalf("SplitFrontmatter(%q) reported limit.exceeded within the bound", tc.src)
					}
				}
				return
			}
			if len(fs) != 1 || fs[0].Code() != CodeLimitExceeded {
				t.Fatalf("SplitFrontmatter(%q) reported %v, want [limit.exceeded]", tc.src, codes(fs))
			}
			f := fs[0]
			limit, _ := f.Param("limit")
			bound, _ := f.Param("bound")
			lowered, _ := f.Param("lowered")
			if s, _ := limit.Str(); s != "frontmatter-size" {
				t.Errorf("limit = %q, want frontmatter-size", s)
			}
			if n, _ := bound.Int(); n != 8 {
				t.Errorf("bound = %d, want 8", n)
			}
			if b, ok := lowered.Bool(); !ok || !b {
				t.Errorf("lowered = %t, %t; want true", b, ok)
			}
			if _, ok := f.Param("observed"); ok {
				t.Error("the reading stopped at the bound, but the finding states an observed size")
			}
			if l, ok := f.Location(); !ok || l.Line != 2 || l.Column != 1 {
				t.Errorf("location %+v, want line 2, column 1", l)
			}
		})
	}

	// The default bound is the specification's: 1 MiB, not lowered.
	big := "---\n" + strings.Repeat("a", 1<<20) + "\n---\n"
	_, _, _, fs := SplitFrontmatter([]byte(big), testSource, limits.Defaults())
	if len(fs) != 1 || fs[0].Code() != CodeLimitExceeded {
		t.Fatalf("a 1 MiB + 1 frontmatter reported %v, want [limit.exceeded]", codes(fs))
	}
	atDefault, _ := fs[0].Param("lowered")
	if b, ok := atDefault.Bool(); !ok || b {
		t.Errorf("lowered = %t, %t at the default bound; want false", b, ok)
	}
}

// TestSplitFrontmatterStopsAtTheBound pins that the reading of a block
// stops at frontmatter-size: the bytes examined are counted, and are the
// bound's worth whatever follows. A splitter that read on to find the
// closing line would examine the whole file.
func TestSplitFrontmatterStopsAtTheBound(t *testing.T) {
	const bound = 1024
	for _, rest := range [][]byte{
		bytes.Repeat([]byte("a"), 1<<20),
		bytes.Repeat([]byte("a\n"), 1<<19),
		bytes.Repeat([]byte("\r\n"), 1<<19),
	} {
		src := append([]byte("---\n"), rest...)
		b := delimit(src, bound)
		if !b.exceeded {
			t.Fatal("a block of 1 MiB passed a bound of 1 KiB")
		}
		// The opening line, then the bound and one byte of text, each
		// byte of it at most two of the file's: a CRLF is one byte of
		// the text.
		if max := len("---\n") + 2*(bound+1); b.scanned > max {
			t.Errorf("examined %d bytes of %d at a bound of %d, want at most %d", b.scanned, len(src), bound, max)
		}
	}
	// The control: with no bound the same file is read to its end.
	src := append([]byte("---\n"), bytes.Repeat([]byte("a\n"), 1<<19)...)
	if b := delimit(src, -1); b.exceeded || b.closed || b.scanned != len(src) {
		t.Errorf("delimit with no bound: %+v, want the whole of %d bytes read", b, len(src))
	}
}

// TestNormalize pins ADR-003 D8 item 2's line breaks.
func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{
		"":             "",
		"a\nb":         "a\nb",
		"a\r\nb":       "a\nb",
		"a\rb":         "a\nb",
		"\r\r\n\n\r":   "\n\n\n\n",
		"a\r\n\r\nb\r": "a\n\nb\n",
	} {
		if got := string(normalize([]byte(in))); got != want {
			t.Errorf("normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func codes(fs []finding.Finding) []finding.Code {
	var cs []finding.Code
	for _, f := range fs {
		cs = append(cs, f.Code())
	}
	return cs
}
