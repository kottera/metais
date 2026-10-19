// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package markdown

import (
	"bufio"
	"fmt"
	"html"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// specFile is CommonMark 0.31.2, vendored from its source
// (testdata/commonmark-spec/SOURCE).
const specFile = "testdata/commonmark-spec/spec.txt"

// specExclusions lists the examples TestSpecHeadings does not compare,
// each with its reason.
const specExclusions = "testdata/commonmark-spec/exclusions.txt"

// specExample is one example of the specification.
type specExample struct {
	number   int
	section  string
	markdown string
	html     string
}

// specExamples reads every example of the vendored specification. An
// example is the lines between a fence of 32 backticks with the info
// string "example" and its closing fence: the Markdown, a line ".", and
// the HTML. "→" stands for a tab.
func specExamples(t *testing.T) []specExample {
	t.Helper()
	f, err := os.Open(specFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fence := strings.Repeat("`", 32)
	var (
		out     []specExample
		section string
		in      int // 0 outside an example, 1 in its Markdown, 2 in its HTML
		md, ht  strings.Builder
	)
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case in == 0 && line == fence+" example":
			in = 1
			md.Reset()
			ht.Reset()
		case in == 0 && strings.HasPrefix(line, "#"):
			section = strings.TrimLeft(line, "# ")
		case in == 1 && line == ".":
			in = 2
		case in == 1:
			md.WriteString(line + "\n")
		case in == 2 && line == fence:
			in = 0
			out = append(out, specExample{
				number:   len(out) + 1,
				section:  section,
				markdown: strings.ReplaceAll(md.String(), "→", "\t"),
				html:     strings.ReplaceAll(ht.String(), "→", "\t"),
			})
		case in == 2:
			ht.WriteString(line + "\n")
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

var headingHTML = regexp.MustCompile(`(?s)<h([1-6])>(.*?)</h[1-6]>`)

// expectedHeadings reads the headings of an example's HTML: the level of
// each <hN> element and the plain text of what it holds, which is what
// Outline gives for it (ADR-008 D5): the text of every element, an
// image's alt text, and a line break a space.
func expectedHeadings(doc string) []Heading {
	var out []Heading
	for _, m := range headingHTML.FindAllStringSubmatch(doc, -1) {
		level, _ := strconv.Atoi(m[1])
		inner := strings.ReplaceAll(m[2], "<br />\n", " ")
		inner = strings.ReplaceAll(inner, "\n", " ")
		out = append(out, Heading{Level: level, Text: html.UnescapeString(stripTags(inner))})
	}
	return out
}

var altAttr = regexp.MustCompile(`\salt="([^"]*)"`)

// stripTags removes the tags of an HTML fragment, keeping an <img>'s alt
// text in its place.
func stripTags(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '<' {
			b.WriteByte(s[i])
			i++
			continue
		}
		// A comment and a CDATA section end at their own terminators.
		for _, c := range [][2]string{{"<!--", "-->"}, {"<![CDATA[", "]]>"}} {
			if strings.HasPrefix(s[i:], c[0]) {
				if k := strings.Index(s[i+len(c[0]):], c[1]); k >= 0 {
					i += len(c[0]) + k + len(c[1])
					goto next
				}
			}
		}
		{
			j, q := i+1, byte(0)
			for j < len(s) && (q != 0 || s[j] != '>') {
				switch {
				case q != 0 && s[j] == q:
					q = 0
				case q == 0 && (s[j] == '"' || s[j] == '\''):
					q = s[j]
				}
				j++
			}
			tag := s[i:min(j+1, len(s))]
			if strings.HasPrefix(tag, "<img ") {
				if m := altAttr.FindStringSubmatch(tag); m != nil {
					b.WriteString(m[1])
				}
			}
			i = j + 1
		}
	next:
	}
	return b.String()
}

// TestSpecHeadings holds Outline to CommonMark 0.31.2's own examples: for
// every example not excluded, the headings Outline finds in its Markdown
// are, in order, the levels and the plain text of the headings of its
// expected HTML. An example with no heading is held to giving none, so
// every example that decides whether a line is a heading counts.
func TestSpecHeadings(t *testing.T) {
	examples := specExamples(t)
	if len(examples) != 652 {
		t.Fatalf("read %d examples from %s, want 652", len(examples), specFile)
	}
	excluded := readSpecExclusions(t)
	withHeadings := 0
	for _, ex := range examples {
		want := expectedHeadings(ex.html)
		if len(want) > 0 {
			withHeadings++
		}
		var got []Heading
		for _, h := range Outline([]byte(ex.markdown)) {
			got = append(got, Heading{Level: h.Level, Text: h.Text})
		}
		same := fmt.Sprint(got) == fmt.Sprint(want)
		reason, isExcluded := excluded[ex.number]
		switch {
		case isExcluded && same:
			t.Errorf("example %d (%s) is excluded (%s), and passes: remove the exclusion", ex.number, ex.section, reason)
		case !isExcluded && !same:
			t.Errorf("example %d (%s):\n%q\nOutline = %+v\nwant      %+v", ex.number, ex.section, ex.markdown, got, want)
		}
	}
	if withHeadings < 30 {
		t.Errorf("only %d examples have a heading; the HTML is misread", withHeadings)
	}
}

// readSpecExclusions reads exclusions.txt: "<example number> <reason>" per
// line, with "#" comments. A number written negative excludes the example
// from TestSpecInlineAsHeading, and a positive one from TestSpecHeadings.
func readSpecExclusions(t *testing.T) map[int]string {
	t.Helper()
	b, err := os.ReadFile(specExclusions)
	if err != nil {
		t.Fatal(err)
	}
	out := map[int]string{}
	for _, line := range strings.Split(string(b), "\n") {
		if line = strings.TrimSpace(line); line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		num, reason, _ := strings.Cut(line, " ")
		n, err := strconv.Atoi(num)
		if err != nil || strings.TrimSpace(reason) == "" {
			t.Fatalf("%s: %q is not \"<number> <reason>\"", specExclusions, line)
		}
		out[n] = reason
	}
	return out
}

var paragraphHTML = regexp.MustCompile(`(?s)\A<p>(.*)</p>\n\z`)

// TestSpecInlineAsHeading holds the inline pass to the specification's
// inline examples. An example whose Markdown has no blank line and whose
// HTML is a single paragraph becomes a setext heading when "===" is
// written under it, and that heading's text must be the paragraph's plain
// text. Link reference definitions at the paragraph's start are taken
// before the heading is, as they are from the paragraph.
func TestSpecInlineAsHeading(t *testing.T) {
	excluded := readSpecExclusions(t)
	compared := 0
	for _, ex := range specExamples(t) {
		m := paragraphHTML.FindStringSubmatch(ex.html)
		if m == nil || strings.Contains(m[1], "<p>") || strings.Contains(ex.markdown, "\n\n") {
			continue
		}
		compared++
		inner := strings.ReplaceAll(m[1], "<br />\n", " ")
		inner = strings.ReplaceAll(inner, "\n", " ")
		want := []Heading{{Level: 1, Text: html.UnescapeString(stripTags(inner))}}
		var got []Heading
		for _, h := range Outline([]byte(ex.markdown + "===\n")) {
			got = append(got, Heading{Level: h.Level, Text: h.Text})
		}
		same := fmt.Sprint(got) == fmt.Sprint(want)
		reason, isExcluded := excluded[-ex.number]
		switch {
		case isExcluded && same:
			t.Errorf("example %d (%s) as a heading is excluded (%s), and passes: remove the exclusion", ex.number, ex.section, reason)
		case !isExcluded && !same:
			t.Errorf("example %d (%s) as a heading:\n%q\nOutline = %+v\nwant      %+v", ex.number, ex.section, ex.markdown+"===\n", got, want)
		}
	}
	if compared < 200 {
		t.Errorf("only %d examples compared as headings", compared)
	}
	t.Logf("%d examples compared as headings", compared)
}
