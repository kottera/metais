// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package markdown

// cursor is a position in one line, in bytes and in columns. A tab moves
// to the next column that is a multiple of 4 (CommonMark 0.31.2 §2.2), and
// only part of one may be consumed, as a block quote marker's optional
// space or a list item's indentation does; col is then inside the tab at
// s[pos].
type cursor struct {
	s   []byte
	pos int
	col int
}

// tabEnd is the column a tab beginning at, or partly consumed up to, col
// ends at.
func tabEnd(col int) int { return col/4*4 + 4 }

// indent returns the columns of spaces and tabs from the cursor, and the
// byte offset and column of the first byte after them.
func (c *cursor) indent() (cols, pos, col int) {
	pos, col = c.pos, c.col
	for pos < len(c.s) {
		switch c.s[pos] {
		case ' ':
			col++
		case '\t':
			col = tabEnd(col)
		default:
			return col - c.col, pos, col
		}
		pos++
	}
	return col - c.col, pos, col
}

// indentUpTo is indent, stopping once limit columns are counted: a block's
// continuation asks only whether a line is indented by some columns, and
// reading the rest of a long indentation for each open block would cost
// the depth of the blocks times the indentation.
func (c *cursor) indentUpTo(limit int) (cols, pos, col int) {
	pos, col = c.pos, c.col
	for pos < len(c.s) && col-c.col < limit {
		switch c.s[pos] {
		case ' ':
			col++
		case '\t':
			col = tabEnd(col)
		default:
			return col - c.col, pos, col
		}
		pos++
	}
	return col - c.col, pos, col
}

// moveTo moves the cursor to pos and col, which indent returned.
func (c *cursor) moveTo(pos, col int) { c.pos, c.col = pos, col }

// advanceCols consumes n columns of spaces and tabs, taking part of a tab
// when n ends inside one. It stops at any other byte.
func (c *cursor) advanceCols(n int) {
	for n > 0 && c.pos < len(c.s) {
		switch c.s[c.pos] {
		case ' ':
			c.pos++
			c.col++
			n--
		case '\t':
			w := tabEnd(c.col) - c.col
			if n < w {
				c.col += n
				return
			}
			c.pos++
			c.col += w
			n -= w
		default:
			return
		}
	}
}

// advanceBytes consumes n bytes that are not tabs, one column each.
func (c *cursor) advanceBytes(n int) {
	c.pos += n
	c.col += n
}

// rest is the line from the cursor. A partly consumed tab at the cursor is
// still in it: every reader of rest trims its leading whitespace or asks
// whether it is blank.
func (c *cursor) rest() []byte { return c.s[c.pos:] }

// isSpaceOrTab reports whether b is a space or a tab.
func isSpaceOrTab(b byte) bool { return b == ' ' || b == '\t' }

// isBlank reports whether s holds only spaces and tabs.
func isBlank(s []byte) bool {
	for _, b := range s {
		if !isSpaceOrTab(b) {
			return false
		}
	}
	return true
}

// trimSpaceTab removes leading and trailing spaces and tabs.
func trimSpaceTab(s []byte) []byte {
	i, j := 0, len(s)
	for i < j && isSpaceOrTab(s[i]) {
		i++
	}
	for j > i && isSpaceOrTab(s[j-1]) {
		j--
	}
	return s[i:j]
}

// trimLeftSpaceTab removes leading spaces and tabs. A paragraph's line
// keeps its trailing ones until the paragraph's content is formed (§4.8),
// as a hard line break and a code span need them.
func trimLeftSpaceTab(s []byte) []byte {
	i := 0
	for i < len(s) && isSpaceOrTab(s[i]) {
		i++
	}
	return s[i:]
}
