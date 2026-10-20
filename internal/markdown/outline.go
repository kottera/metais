// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package markdown

import "bytes"

// Outline lists src's ATX and setext headings in document order (ADR-008
// D5, outline items 1 and 2; ADR-010 D9 item 2). src is a text entry's
// bytes, as stored.
//
// A leading frontmatter block, delimited by SplitFrontmatter's rule, is
// split off first and the rest read as CommonMark 0.31.2; each line number
// is a line of src, counted at LFs, so a block's lines still count. A
// first line "---" with no closing line opens no block, and the whole of
// src is read. What CommonMark does not make a heading, such as a "#" line
// in a code block or an HTML block, is not listed. A file with no headings
// gives an empty list.
//
// The reading is METAIS's own (ADR-010 D9, A2): a block pass that keeps
// only what decides a heading, then an inline pass over each heading's
// content alone. Both are linear in src, cmark's pathological cases
// included (ADR-009 D10 item 4).
func Outline(src []byte) []Heading { return outline(src, false) }

// Title is the plain text of src's first level-1 heading, as Outline
// gives it, and false when src has none. It keeps no other heading, so
// what it holds does not grow with the headings src has (ADR-009
// Amendment A6 item 2); it still reads all of src, since a link
// reference definition after the heading decides its text.
func Title(src []byte) (string, bool) {
	if hs := outline(src, true); len(hs) > 0 {
		return hs[0].Text, true
	}
	return "", false
}

func outline(src []byte, titleOnly bool) []Heading {
	start := 0
	if b := delimit(src, -1); b.present && b.closed {
		start = b.body
	}
	firstLine := 1 + bytes.Count(src[:start], []byte{'\n'})
	raw, refs := scanBlocks(src[start:], firstLine, titleOnly)
	out := make([]Heading, 0, len(raw))
	for _, h := range raw {
		out = append(out, Heading{Level: h.level, Text: flatten(h.content, refs), Line: h.line})
	}
	return out
}
