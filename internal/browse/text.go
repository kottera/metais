// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package browse

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"
)

// IsText reports whether b is text by ADR-008 D5's rule (RULINGS R2-122):
// valid UTF-8 with no NUL byte. Anything else is binary.
func IsText(b []byte) bool {
	return utf8.Valid(b) && bytes.IndexByte(b, 0) < 0
}

// LineCount is the number of lines of b by ADR-008 D5's rule: the number
// of LFs, plus one when b is non-empty and does not end with an LF. An
// empty b has no lines.
func LineCount(b []byte) int {
	n := bytes.Count(b, []byte{'\n'})
	if len(b) > 0 && b[len(b)-1] != '\n' {
		n++
	}
	return n
}

// Line is one line of an entry: its 1-based number, and its bytes with
// and without its line ending.
type Line struct {
	Number int
	// Bytes is the line with its own line ending, as stored.
	Bytes []byte
	// Text is the line without its ending: without the LF, and without a
	// CR immediately before it, which is part of the ending for matching
	// and display (ADR-008 D5).
	Text []byte
}

// Lines splits b into its lines.
func Lines(b []byte) []Line {
	out := make([]Line, 0, LineCount(b))
	for n := 1; len(b) > 0; n++ {
		end := bytes.IndexByte(b, '\n')
		var whole []byte
		if end < 0 {
			whole, b = b, nil
		} else {
			whole, b = b[:end+1], b[end+1:]
		}
		text := bytes.TrimSuffix(whole, []byte{'\n'})
		if len(text) < len(whole) {
			text = bytes.TrimSuffix(text, []byte{'\r'})
		}
		out = append(out, Line{Number: n, Bytes: whole, Text: text})
	}
	return out
}

// Range is read --lines' range (ADR-008 D5, read item 2): First to Last,
// 1-based and inclusive, or First to the end when Open is set.
type Range struct {
	First, Last int
	Open        bool
}

// ParseRange reads N, N-M or N-, with N at least 1 and M at least N, each
// a decimal integer of ASCII digits. Anything else is refused.
func ParseRange(s string) (Range, bool) {
	num := func(t string) (int, bool) {
		if t == "" || len(t) > 18 || strings.TrimLeft(t, "0123456789") != "" {
			return 0, false
		}
		n, err := strconv.Atoi(t)
		return n, err == nil && n >= 1
	}
	a, b, dash := strings.Cut(s, "-")
	first, ok := num(a)
	if !ok {
		return Range{}, false
	}
	switch {
	case !dash:
		return Range{First: first, Last: first}, true
	case b == "":
		return Range{First: first, Open: true}, true
	}
	last, ok := num(b)
	if !ok || last < first {
		return Range{}, false
	}
	return Range{First: first, Last: last}, true
}

// ErrLineOutOfRange is the refusal of a range whose first line is beyond
// the entry's last line.
var ErrLineOutOfRange = errors.New("browse: the range starts beyond the last line")

// Selection is the lines of an entry a range selects.
type Selection struct {
	// Bytes are the selected lines with their own line endings, as stored.
	Bytes []byte
	// First and Last are the first and last lines selected, Last clamped
	// to the entry's last line; Total is the entry's line count. An empty
	// entry, read whole, has First and Last 0.
	First, Last, Total int
	// Lines are the selected lines.
	Lines []Line
}

// SelectLines returns the lines of b that r selects, the whole of b when r
// is nil. An end beyond the last line is clamped; a start beyond it is
// ErrLineOutOfRange.
func SelectLines(b []byte, r *Range) (Selection, error) {
	lines := Lines(b)
	total := len(lines)
	if r == nil {
		s := Selection{Bytes: b, Total: total, Lines: lines}
		if total > 0 {
			s.First, s.Last = 1, total
		}
		return s, nil
	}
	if r.First > total {
		return Selection{Total: total}, ErrLineOutOfRange
	}
	last := total
	if !r.Open {
		last = min(r.Last, total)
	}
	sel := lines[r.First-1 : last]
	start := 0
	for _, l := range lines[:r.First-1] {
		start += len(l.Bytes)
	}
	end := start
	for _, l := range sel {
		end += len(l.Bytes)
	}
	return Selection{Bytes: b[start:end], First: r.First, Last: last, Total: total, Lines: sel}, nil
}

// Neutralise writes s for a person to read (ADR-009 D15 item 3, RULINGS
// R2-145): C0 and C1 controls, DEL and the bidirectional formatting
// characters U+202A–U+202E and U+2066–U+2069 as \u and four lowercase hex
// digits, so package text cannot drive a terminal or reorder what a
// reviewer sees. A tab is kept, since a listing, outline or grep line is
// content. s is valid UTF-8.
func Neutralise(s string) string {
	if !strings.ContainsFunc(s, neutralised) {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		if neutralised(r) {
			const hex = "0123456789abcdef"
			b.WriteString(`\u`)
			for shift := 12; shift >= 0; shift -= 4 {
				b.WriteByte(hex[r>>shift&0xf])
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func neutralised(r rune) bool {
	switch {
	case r == '\t':
		return false
	case r < 0x20, r == 0x7f, r >= 0x80 && r <= 0x9f:
		return true
	case r >= 0x202a && r <= 0x202e, r >= 0x2066 && r <= 0x2069:
		return true
	}
	return false
}
