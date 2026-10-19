// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package browse

import "github.com/kottera/metais/internal/markdown"

// Heading is one heading of an outline (ADR-008 D5, outline item 2).
type Heading struct {
	Level int
	// Text is the heading's inline content as plain text.
	Text string
	// Line is the heading's first line, and EndLine the last line of its
	// section: the line before the next heading of the same or a higher
	// level, or the entry's last line. read --lines Line-EndLine reads
	// exactly the section.
	Line, EndLine int
}

// Outline lists the headings of b, a text entry read as CommonMark 0.31.2
// by METAIS's own reading (ADR-010 D9, A3), a leading frontmatter block
// skipped and its lines still counted. An entry with no headings gives an
// empty list.
func Outline(b []byte) []Heading {
	hs := markdown.Outline(b)
	total := LineCount(b)
	out := make([]Heading, len(hs))
	for i, h := range hs {
		end := total
		for _, next := range hs[i+1:] {
			if next.Level <= h.Level {
				end = next.Line - 1
				break
			}
		}
		out[i] = Heading{Level: h.Level, Text: h.Text, Line: h.Line, EndLine: end}
	}
	return out
}
