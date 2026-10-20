// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package markdown

import (
	"bytes"
	"strings"
)

// The block pass reads a document's block structure by CommonMark 0.31.2
// §4 and §5, as far as it decides which lines are headings: container
// blocks (block quotes and list items) with their continuation and
// laziness, and the leaf blocks that can hold a line that would otherwise
// be a heading (code blocks, HTML blocks, paragraphs and the link
// reference definitions at their start). It keeps no content but the raw
// text of each heading and the labels of the definitions; inline content
// is read only for headings, afterwards, by the inline pass.
//
// Every line is read once, from left to right, and every open block is
// matched at most once per line, so the work is linear in the document
// times the depth of the open blocks a line can match, which the line's
// own markers bound.

// rawHeading is a heading the block pass found.
type rawHeading struct {
	level   int
	content []byte // the raw inline content, stripped
	line    int    // the line of the file it begins on
}

type blockKind uint8

const (
	kindDocument blockKind = iota
	kindQuote
	kindItem
	kindParagraph
	kindFence
	kindIndented
	kindHTML
)

// block is an open block.
type block struct {
	kind blockKind

	// hasChild is whether any block has been put in a container. A list
	// item that began with a blank line and has none ends at the next
	// blank line (§5.2, "a list item can begin with at most one blank
	// line").
	hasChild bool

	width int // a list item's content column past the column it began at

	fenceChar byte
	fenceLen  int

	htmlEnd int // an HTML block's start condition, 1 to 7, which names its end

	lines []paraLine // a paragraph's lines, each without its leading spaces and tabs
}

type paraLine struct {
	text []byte
	line int
}

// blockParser is one block pass over a document.
type blockParser struct {
	open     []*block // open[0] is the document; the last is the tip
	headings []rawHeading
	refs     map[string]bool
	// titleOnly keeps no heading after the first of level 1, which is
	// all a title needs; the pass still reads every line, for the link
	// reference definitions the title's text may use.
	titleOnly bool

	// lastText is the offset of the last byte of the line being read that
	// is not a space or a tab, -1 for a blank line, so that whether the
	// rest of a line is blank is known without reading it again.
	lastText int
	// hrFail remembers, for the line being read, a thematic break that was
	// looked for and not found: its character, and the offsets of where the
	// look began and of the byte that ended it. A look with the same
	// character from a later offset before that byte meets the same byte,
	// so a line of many list markers is not read again from each.
	hrFail struct {
		ch       byte
		from, at int
	}
	// blankFrom is the index from which every open block is known to
	// continue a line whose rest is blank, or -1. Whether a block continues
	// a blank rest depends on the block alone, and a line whose rest was
	// blank when it reached some open block leaves open, past that block,
	// only blocks that continued it; so until a block is opened, a line
	// whose rest is blank there continues all of them, and they are not
	// read again. Without this, each line of a long run of blank lines,
	// or of ">" lines, would read every one of thousands of nested items.
	blankFrom int
	pushes    int // blocks opened so far
}

// scanBlocks runs the block pass over src, whose first line is line
// firstLine of the file. A line ends at an LF, a CRLF or a CR (§2.1); the
// line number of the file counts LFs only (ADR-008 D5), so a CR ends a
// CommonMark line without starting a new line of the file.
func scanBlocks(src []byte, firstLine int, titleOnly bool) ([]rawHeading, map[string]bool) {
	p := &blockParser{open: []*block{{kind: kindDocument}}, refs: map[string]bool{}, blankFrom: -1, titleOnly: titleOnly}
	line := firstLine
	for i := 0; i < len(src); {
		j := i
		for j < len(src) && src[j] != '\n' && src[j] != '\r' {
			j++
		}
		p.addLine(src[i:j], line)
		switch {
		case j == len(src):
			i = j
		case src[j] == '\r' && j+1 < len(src) && src[j+1] == '\n':
			i = j + 2
			line++
		case src[j] == '\n':
			i = j + 1
			line++
		default: // a lone CR
			i = j + 1
		}
	}
	p.closeFrom(1)
	return p.headings, p.refs
}

// addLine reads one line (§5, "Appendix: A parsing strategy", phases 1
// and 2).
func (p *blockParser) addLine(s []byte, line int) {
	p.lastText = len(s) - 1
	for p.lastText >= 0 && isSpaceOrTab(s[p.lastText]) {
		p.lastText--
	}
	p.hrFail.ch = 0
	c := &cursor{s: s}

	known, pushes, blankAt := p.blankFrom, p.pushes, -1
	p.blankFrom = -1
	defer func() {
		if blankAt >= 0 && p.pushes == pushes {
			p.blankFrom = min(blankAt, len(p.open))
		}
	}()

	// The open blocks this line continues, outermost first.
	matched := 0
	for i := 1; i < len(p.open); i++ {
		if c.pos > p.lastText {
			if blankAt < 0 {
				blankAt = i
			}
			if known >= 1 && i >= known {
				matched = len(p.open) - 1
				break
			}
		}
		ok, done := p.continues(p.open[i], c, i)
		if done {
			return
		}
		if !ok {
			break
		}
		matched = i
	}
	if blankAt < 0 && matched == len(p.open)-1 && c.pos > p.lastText {
		blankAt = len(p.open)
	}
	tip := p.open[len(p.open)-1]
	tipIsPara := tip.kind == kindParagraph
	container := p.open[matched]
	opened := false // whether this line opened a container
	closed := false // whether the unmatched blocks are closed

	closeUnmatched := func() {
		if !closed {
			p.closeFrom(matched + 1)
			closed = true
		}
	}
	// closeForLeaf closes what a new block ends: the unmatched blocks, and
	// the container itself when it is the paragraph the block interrupts.
	closeForLeaf := func() {
		closeUnmatched()
		if container.kind == kindParagraph {
			p.closeFrom(len(p.open) - 1)
			container = p.open[len(p.open)-1]
		}
		container.hasChild = true
	}

starts:
	for container.kind != kindFence && container.kind != kindIndented && container.kind != kindHTML {
		ind, ns, nsCol := c.indent()
		if ns == len(s) {
			break // a blank line starts nothing
		}
		// A paragraph would take this line as its continuation, matched
		// or lazy, so the line cannot begin indented code (§4.4).
		paraTakes := tipIsPara && !opened
		if ind >= 4 {
			if paraTakes {
				break
			}
			closeForLeaf()
			c.advanceCols(4)
			p.push(&block{kind: kindIndented})
			return
		}
		rest := s[ns:]
		switch rest[0] {
		case '>':
			closeUnmatched()
			if container.kind == kindParagraph {
				p.closeFrom(len(p.open) - 1)
				container = p.open[len(p.open)-1]
			}
			c.moveTo(ns, nsCol)
			c.advanceBytes(1)
			if c.pos < len(s) && isSpaceOrTab(s[c.pos]) {
				c.advanceCols(1)
			}
			container.hasChild = true
			container = p.push(&block{kind: kindQuote})
			opened = true
			continue
		case '#':
			if level, content, ok := atxHeading(rest); ok {
				closeForLeaf()
				p.heading(rawHeading{level: level, content: content, line: line})
				return
			}
		case '`', '~':
			if ch, n, ok := fenceOpen(rest); ok {
				closeForLeaf()
				p.push(&block{kind: kindFence, fenceChar: ch, fenceLen: n, width: ind})
				return
			}
		case '<':
			if kind := htmlBlockStart(rest, !paraTakes); kind != 0 {
				closeForLeaf()
				p.push(&block{kind: kindHTML, htmlEnd: kind})
				if htmlBlockEnds(kind, rest) {
					p.closeFrom(len(p.open) - 1)
				}
				return
			}
		}
		if container.kind == kindParagraph && (rest[0] == '=' || rest[0] == '-') && setextUnderline(rest) {
			if p.setextHeading(container, rest[0]) {
				return
			}
			// The paragraph was only link reference definitions: it is
			// gone, and the line is read again as the start of a block.
			container = p.open[len(p.open)-1]
			matched = len(p.open) - 1
			tipIsPara = false
			continue
		}
		if p.thematicBreak(s, ns) {
			closeForLeaf()
			return
		}
		if m, ok := listMarker(rest, container.kind == kindParagraph); ok {
			closeUnmatched()
			if container.kind == kindParagraph {
				p.closeFrom(len(p.open) - 1)
				container = p.open[len(p.open)-1]
			}
			startCol := c.col
			c.moveTo(ns, nsCol)
			c.advanceBytes(m.markerLen)
			markerEnd := c.col
			pad, _, _ := c.indent()
			switch {
			case m.blank:
				pad = 1
			case pad > 4:
				pad = 1 // the content is indented code; one space belongs to the marker
			}
			c.advanceCols(pad)
			container.hasChild = true
			container = p.push(&block{kind: kindItem, width: markerEnd + pad - startCol})
			opened = true
			continue
		}
		break starts
	}

	rest := c.rest()
	blank := isBlank(rest)
	if !opened && tipIsPara && container == p.open[matched] && matched < len(p.open)-1 && !blank {
		// A lazy continuation line (§5.1).
		tip.lines = append(tip.lines, paraLine{text: trimLeftSpaceTab(rest), line: line})
		return
	}
	closeUnmatched()
	container = p.open[len(p.open)-1]
	switch container.kind {
	case kindParagraph:
		container.lines = append(container.lines, paraLine{text: trimLeftSpaceTab(rest), line: line})
	case kindHTML:
		if htmlBlockEnds(container.htmlEnd, rest) {
			p.closeFrom(len(p.open) - 1)
		}
	case kindFence, kindIndented:
	default:
		if !blank {
			container.hasChild = true
			p.push(&block{kind: kindParagraph, lines: []paraLine{{text: trimLeftSpaceTab(rest), line: line}}})
		}
	}
}

// continues reports whether the line at c continues the open block b,
// which is open[i], moving c past b's marker when it does. done is set
// when the line was the closing fence of b and nothing more is read.
func (p *blockParser) continues(b *block, c *cursor, i int) (ok, done bool) {
	blank := c.pos > p.lastText
	switch b.kind {
	case kindQuote:
		ind, ns, nsCol := c.indentUpTo(4)
		if blank || ind >= 4 || c.s[ns] != '>' {
			return false, false
		}
		c.moveTo(ns, nsCol)
		c.advanceBytes(1)
		if c.pos < len(c.s) && isSpaceOrTab(c.s[c.pos]) {
			c.advanceCols(1)
		}
		return true, false
	case kindItem:
		if blank {
			return b.hasChild, false
		}
		if ind, _, _ := c.indentUpTo(b.width); ind >= b.width {
			c.advanceCols(b.width)
			return true, false
		}
		return false, false
	case kindFence:
		ind, ns, _ := c.indentUpTo(4)
		if !blank && ind < 4 && fenceCloses(c.s[ns:], b.fenceChar, b.fenceLen) {
			p.closeFrom(i)
			return false, true
		}
		c.advanceCols(min(ind, b.width))
		return true, false
	case kindIndented:
		if blank {
			return true, false
		}
		if ind, _, _ := c.indentUpTo(4); ind >= 4 {
			c.advanceCols(4)
			return true, false
		}
		return false, false
	case kindHTML:
		return !(blank && b.htmlEnd >= 6), false
	case kindParagraph:
		return !blank, false
	}
	return false, false
}

// push opens b as the new tip and returns it.
func (p *blockParser) push(b *block) *block {
	p.pushes++
	p.open = append(p.open, b)
	return b
}

// closeFrom closes open[i:], innermost first. A paragraph that closes
// gives up the link reference definitions at its start (§4.7).
func (p *blockParser) closeFrom(i int) {
	for j := len(p.open) - 1; j >= i; j-- {
		if b := p.open[j]; b.kind == kindParagraph {
			p.definitions(b)
		}
	}
	p.open = p.open[:i]
}

// definitions takes the link reference definitions at the start of the
// paragraph b and returns its first line that is not one, len(b.lines)
// when every line is.
func (p *blockParser) definitions(b *block) int {
	text, starts := joinLines(b.lines)
	at := 0
	for at < len(text) {
		end, label, ok := refDefinition(text, at)
		if !ok {
			break
		}
		if !p.refs[label] {
			p.refs[label] = true
		}
		at = end
	}
	// at is at a line start, or the end.
	n := 0
	for n < len(starts) && starts[n] < at {
		n++
	}
	b.lines = b.lines[n:]
	return n
}

// heading keeps h, unless the pass keeps only a title and has it.
func (p *blockParser) heading(h rawHeading) {
	if p.titleOnly && (h.level != 1 || len(p.headings) > 0) {
		return
	}
	p.headings = append(p.headings, h)
}

// setextHeading reads a setext heading underline after the paragraph para,
// the tip, at level 1 for '=' and 2 for '-' (§4.3). It reports false,
// having closed the paragraph, when the paragraph was only link reference
// definitions and so is no heading.
func (p *blockParser) setextHeading(para *block, ch byte) bool {
	p.definitions(para)
	p.open = p.open[:len(p.open)-1]
	if len(para.lines) == 0 {
		return false
	}
	text, _ := joinLines(para.lines)
	level := 1
	if ch == '-' {
		level = 2
	}
	p.heading(rawHeading{level: level, content: trimSpaceTab(text), line: para.lines[0].line})
	p.open[len(p.open)-1].hasChild = true
	return true
}

// joinLines joins a paragraph's lines with LFs, and gives the offset each
// line begins at.
func joinLines(lines []paraLine) ([]byte, []int) {
	n := 0
	for _, l := range lines {
		n += len(l.text) + 1
	}
	text := make([]byte, 0, n)
	starts := make([]int, len(lines))
	for i, l := range lines {
		if i > 0 {
			text = append(text, '\n')
		}
		starts[i] = len(text)
		text = append(text, l.text...)
	}
	return text, starts
}

// atxHeading reads an ATX heading (§4.2) from s, which begins at the
// line's first non-space byte: its level, and its content with the
// optional closing sequence and the surrounding spaces and tabs removed.
func atxHeading(s []byte) (level int, content []byte, ok bool) {
	for level < len(s) && s[level] == '#' {
		level++
	}
	if level > 6 || (level < len(s) && !isSpaceOrTab(s[level])) {
		return 0, nil, false
	}
	content = trimSpaceTab(s[level:])
	// The closing sequence: a run of '#' that is the whole content or is
	// preceded by a space or a tab.
	j := len(content)
	for j > 0 && content[j-1] == '#' {
		j--
	}
	if j == 0 {
		content = content[:0]
	} else if j < len(content) && isSpaceOrTab(content[j-1]) {
		content = trimSpaceTab(content[:j])
	}
	return level, content, true
}

// setextUnderline reports whether s is a setext heading underline: a run of
// '=' or of '-', then only spaces and tabs (§4.3).
func setextUnderline(s []byte) bool {
	i := 0
	for i < len(s) && s[i] == s[0] {
		i++
	}
	return isBlank(s[i:])
}

// thematicBreak reports whether the line s from offset i is a thematic
// break: three or more of one of '*', '-' and '_', with only spaces and
// tabs among them (§4.1).
func (p *blockParser) thematicBreak(s []byte, i int) bool {
	ch := s[i]
	if ch != '*' && ch != '-' && ch != '_' {
		return false
	}
	if f := p.hrFail; f.ch == ch && i >= f.from && i < f.at {
		return false
	}
	n := 0
	for j := i; j < len(s); j++ {
		switch b := s[j]; {
		case b == ch:
			n++
		case isSpaceOrTab(b):
		default:
			p.hrFail.ch, p.hrFail.from, p.hrFail.at = ch, i, j
			return false
		}
	}
	return n >= 3
}

// fenceOpen reads a code fence opening (§4.5): its character and length.
// A backtick fence's info string has no backtick.
func fenceOpen(s []byte) (ch byte, n int, ok bool) {
	ch = s[0]
	for n < len(s) && s[n] == ch {
		n++
	}
	if n < 3 || (ch == '`' && bytes.IndexByte(s[n:], '`') >= 0) {
		return 0, 0, false
	}
	return ch, n, true
}

// fenceCloses reports whether s closes a fence of n ch's: n or more of
// them, then only spaces and tabs.
func fenceCloses(s []byte, ch byte, n int) bool {
	i := 0
	for i < len(s) && s[i] == ch {
		i++
	}
	return i >= n && isBlank(s[i:])
}

// marker is a list item's marker (§5.2).
type marker struct {
	markerLen int  // bytes of the marker
	blank     bool // nothing follows it on the line
}

// listMarker reads a list marker at the start of s: a bullet, or one to
// nine digits and '.' or ')', followed by a space, a tab or the end of the
// line. When the item would interrupt a paragraph, it must not be empty
// and an ordered one must start at 1.
func listMarker(s []byte, interrupts bool) (marker, bool) {
	var m marker
	switch s[0] {
	case '-', '+', '*':
		m.markerLen = 1
	default:
		d := 0
		for d < len(s) && d < 10 && s[d] >= '0' && s[d] <= '9' {
			d++
		}
		if d == 0 || d > 9 || d == len(s) || (s[d] != '.' && s[d] != ')') {
			return m, false
		}
		if interrupts && strings.TrimLeft(string(s[:d]), "0") != "1" {
			return m, false
		}
		m.markerLen = d + 1
	}
	after := s[m.markerLen:]
	if len(after) > 0 && !isSpaceOrTab(after[0]) {
		return m, false
	}
	m.blank = isBlank(after)
	if interrupts && m.blank {
		return m, false
	}
	return m, true
}
