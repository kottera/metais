// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package markdown

import (
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// TestOutline is ADR-008 D16 item 8's outline rows (ATX, setext,
// multi-line setext, "---" after a paragraph, "#" in a fence,
// frontmatter), with CommonMark 0.31.2's heading rules for ATX and setext
// headings, fences and containers (ADR-010 D9 item 4).
func TestOutline(t *testing.T) {
	testenv.Home(t)
	for _, tc := range []struct {
		name, src string
		want      []Heading
	}{
		// ATX headings.
		{"ATX levels", "# 1\n## 2\n### 3\n#### 4\n##### 5\n###### 6\n",
			[]Heading{{1, "1", 1}, {2, "2", 2}, {3, "3", 3}, {4, "4", 4}, {5, "5", 5}, {6, "6", 6}}},
		{"ATX seven is not a heading", "####### 7\n", nil},
		{"ATX needs a space", "#5 bolt\n#hashtag\n", nil},
		{"ATX escaped", "\\## foo\n", nil},
		{"ATX closing sequence", "## foo ##\n### bar    ###\n", []Heading{{2, "foo", 1}, {3, "bar", 2}}},
		{"ATX closing needs a space", "# foo#\n", []Heading{{1, "foo#", 1}}},
		{"ATX indented by three", "   # foo\n", []Heading{{1, "foo", 1}}},
		{"ATX indented by four is code", "    # foo\n", nil},
		{"ATX empty", "#\n## \n### ###\n", []Heading{{1, "", 1}, {2, "", 2}, {3, "", 3}}},
		{"ATX interrupts a paragraph", "text\n# H\nmore\n", []Heading{{1, "H", 2}}},
		{"ATX inline content", "# a *b* **c** `d` [e](f) ![g](h) <http://i.j> <b>k</b> &amp; \\*\n",
			[]Heading{{1, "a b c d e g http://i.j k & *", 1}}},
		{"ATX code span spaces kept", "# `a  b`\n", []Heading{{1, "a  b", 1}}},

		// Setext headings.
		{"setext", "Title\n=====\n\nSub\n---\n", []Heading{{1, "Title", 1}, {2, "Sub", 4}}},
		{"setext one character underline", "A\n=\nB\n-\n", []Heading{{1, "A", 1}, {2, "B", 3}}},
		{"setext multi-line", "Foo\nbar\nbaz\n===\n", []Heading{{1, "Foo bar baz", 1}}},
		{"setext multi-line indented", "Foo\n  bar \n===\n", []Heading{{1, "Foo bar", 1}}},
		{"setext hard break", "Foo  \nbar\n---\n", []Heading{{2, "Foo bar", 1}}},
		{"setext backslash break", "Foo\\\nbar\n===\n", []Heading{{1, "Foo bar", 1}}},
		{"--- after a paragraph", "para\n---\n", []Heading{{2, "para", 1}}},
		{"--- after a blank line", "para\n\n---\n", nil},
		{"setext underline with spaces inside is a break", "Foo\n= =\n", nil},
		{"setext underline indented four is text", "Foo\n    ---\n", nil},

		// Fences, code and HTML.
		{"# in a fence", "```\n# no\n```\n# yes\n", []Heading{{1, "yes", 4}}},
		{"# in a tilde fence", "~~~md\n# no\n~~~\n", nil},
		{"# in an unclosed fence", "```\n# no\n", nil},
		{"setext after a fence", "```\nx\n```\n---\n", nil},
		{"# in an HTML block", "<div>\n# no\n</div>\n", nil},
		{"# in an indented block", "    # no\n\n# yes\n", []Heading{{1, "yes", 3}}},

		// Containers.
		{"in a block quote", "> # A\n> B\n> ---\n", []Heading{{1, "A", 1}, {2, "B", 2}}},
		{"in a list item", "- # A\n-  B\n   ===\n", []Heading{{1, "A", 1}, {1, "B", 2}}},
		{"--- after a list is a break", "- item\n---\n", nil},
		{"lazy line is not a setext underline", "> Foo\n---\n", nil},

		// Frontmatter: split off, its lines counted.
		{"frontmatter", "---\ntitle: A\n---\n# A\n", []Heading{{1, "A", 4}}},
		{"frontmatter is not a setext heading", "---\ntitle: x\n---\n", nil},
		{"frontmatter empty", "---\n---\nA\n=\n", []Heading{{1, "A", 3}}},
		{"frontmatter CRLF", "---\r\na: 1\r\n---\r\n\r\n# A\r\n", []Heading{{1, "A", 5}}},
		{"frontmatter blank line kept", "---\na: 1\n---\n\n\n## B\n", []Heading{{2, "B", 6}}},
		{"frontmatter unclosed opens no block", "---\n# A\n", []Heading{{1, "A", 2}}},
		{"frontmatter unclosed setext", "---\nFoo\n--- \nbar\n", []Heading{{2, "Foo", 2}}},
		{"no frontmatter after a blank line", "\n---\ntitle\n---\n", []Heading{{2, "title", 3}}},

		// Lines: an LF ends a line; a CR is a CommonMark line ending.
		{"CRLF", "a\r\n\r\n# b\r\n", []Heading{{1, "b", 3}}},
		{"lone CR is a line ending, not a line", "a\r# b\r", []Heading{{1, "b", 1}}},
		{"no final LF", "x\n\n# end", []Heading{{1, "end", 3}}},
		{"empty", "", nil},
		{"no headings", "just text\n\nmore text\n", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Outline([]byte(tc.src))
			if got == nil {
				t.Fatalf("Outline(%q) = nil, want an empty list", tc.src)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("Outline(%q) =\n %+v, want\n %+v", tc.src, got, tc.want)
			}
		})
	}
}

// TestOutlineLinesOffsetPastFrontmatter pins that a heading's line is its
// line in the file: a frontmatter block of n lines moves every heading
// down by n. An Outline that parsed the body alone and counted from it
// would give the same lines for every n.
func TestOutlineLinesOffsetPastFrontmatter(t *testing.T) {
	const body = "# A\n\ntext\n\n## B\n"
	for n := range 5 {
		src := "---\n" + strings.Repeat("k: v\n", n) + "---\n" + body
		got := Outline([]byte(src))
		want := []Heading{{1, "A", n + 3}, {2, "B", n + 7}}
		if !slices.Equal(got, want) {
			t.Errorf("with %d frontmatter lines: Outline = %+v, want %+v", n, got, want)
		}
	}
}
