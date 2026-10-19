// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package markdown

import (
	"bytes"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
)

// delimiter is the whole of the line that opens a frontmatter block and
// the whole of the line that closes it (ADR-003 D13 item 2).
const delimiter = "---"

// SplitFrontmatter splits src, one Markdown file's text, into its
// frontmatter and its body by ADR-003 D13 item 2, under lim's
// frontmatter-size (ADR-009 D10 item 4).
//
// src is the file after ADR-003 D8 item 1's UTF-8 check and byte order mark
// removal. Its line breaks may be CRLF, CR or LF: the rule is applied after
// ADR-003 D8 item 2's normalization, and front and body are returned with
// every line break an LF. Lines are counted after that normalization.
//
// Frontmatter is present when the first line is exactly "---", and it ends
// at the next line that is exactly "---". Then front is the text between
// the two lines, each of its lines ending in LF, and empty when they are
// adjacent; frontLine is the line of the file on which front begins, which
// is 2, so a position in front at line L is at line L+frontLine-1 of the
// file; and body is the text after the closing line. When the first line is
// anything else there is no frontmatter: front is nil, frontLine is 0 and
// body is the whole file. Nothing else of the body is changed; removing
// its final LF is ADR-003 D13 item 3's, and the caller's.
//
// A frontmatter block with no closing line is markdown.frontmatter_unclosed,
// at the opening line. A block whose text passes frontmatter-size is
// limit.exceeded, at its first line: the reading stops at the bound, so
// the work is bounded by the limit and not by the file. With a finding the
// file contributes no value (ADR-003 D22 item 3), and front and body are
// nil.
func SplitFrontmatter(src []byte, from Source, lim limits.Set) (front []byte, frontLine int, body []byte, findings []finding.Finding) {
	b := delimit(src, lim.Value(limits.FrontmatterSize))
	switch {
	case !b.present:
		return nil, 0, normalize(src), nil
	case b.exceeded:
		l := from.At.At(finding.Pos{Line: 2, Column: 1})
		l.Pointer, l.HasPointer = "", false
		return nil, 0, nil, []finding.Finding{finding.New(CodeLimitExceeded, from.Input, &l, finding.Params{
			"limit":   finding.String(limits.Get(limits.FrontmatterSize).Name),
			"bound":   finding.Int(lim.Value(limits.FrontmatterSize)),
			"lowered": finding.Bool(lim.Lowered(limits.FrontmatterSize)),
		})}
	case !b.closed:
		l := from.At.At(finding.Pos{Line: 1, Column: 1})
		l.Pointer, l.HasPointer = "", false
		return nil, 0, nil, []finding.Finding{finding.New(CodeFrontmatterUnclosed, from.Input, &l, finding.Params{})}
	}
	return normalize(src[b.front:b.frontEnd]), 2, normalize(src[b.body:]), nil
}

// frontBlock is where a frontmatter block lies in a file's bytes.
type frontBlock struct {
	present  bool // the first line is exactly "---"
	closed   bool // a later line is exactly "---"
	exceeded bool // the block's text passed the bound before it closed

	front, frontEnd int // the block's text is src[front:frontEnd]
	body            int // the first byte after the closing line's break

	scanned int // how many bytes of src were examined
}

// delimit finds src's frontmatter block by ADR-003 D13 item 2, a line
// ending at CRLF, CR or LF (ADR-003 D8 item 2). It reads at most bound
// bytes of the block's text, measured after normalization, before it stops
// with exceeded set; a negative bound is none. Every byte it examines is
// examined once, so its work is linear in what it reads.
func delimit(src []byte, bound int64) frontBlock {
	var b frontBlock
	if !isDelimiterLine(src, 0) {
		b.scanned = min(len(src), len(delimiter)+1)
		return b
	}
	b.present = true
	s := afterBreak(src, len(delimiter))
	b.front = s
	var size int64 // the normalized length of the block's text so far
	for s < len(src) {
		if isDelimiterLine(src, s) {
			b.closed, b.frontEnd = true, s
			b.body = afterBreak(src, s+len(delimiter))
			b.scanned = b.body
			return b
		}
		e := s
		for e < len(src) && src[e] != '\n' && src[e] != '\r' {
			e++
			if bound >= 0 && size+int64(e-s) > bound {
				b.exceeded, b.scanned = true, e
				return b
			}
		}
		size += int64(e-s) + 1 // the line and its LF
		if e == len(src) {
			size-- // the last line has no break
		}
		if bound >= 0 && size > bound {
			b.exceeded, b.scanned = true, e
			return b
		}
		s = afterBreak(src, e)
	}
	b.frontEnd, b.scanned = len(src), len(src)
	return b
}

// isDelimiterLine reports whether the line beginning at src[i] is exactly
// "---".
func isDelimiterLine(src []byte, i int) bool {
	if !bytes.HasPrefix(src[i:], []byte(delimiter)) {
		return false
	}
	e := i + len(delimiter)
	return e == len(src) || src[e] == '\n' || src[e] == '\r'
}

// afterBreak returns the index after the line break at src[i], which is i
// itself at the end of src. A CRLF is one break.
func afterBreak(src []byte, i int) int {
	switch {
	case i >= len(src):
		return len(src)
	case src[i] == '\r' && i+1 < len(src) && src[i+1] == '\n':
		return i + 2
	default:
		return i + 1
	}
}

// normalize returns b with every CRLF and every lone CR replaced by an LF
// (ADR-003 D8 item 2). b is returned as it is when it holds no CR.
func normalize(b []byte) []byte {
	if bytes.IndexByte(b, '\r') < 0 {
		return b
	}
	out := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		if b[i] != '\r' {
			out = append(out, b[i])
			continue
		}
		out = append(out, '\n')
		if i+1 < len(b) && b[i+1] == '\n' {
			i++
		}
	}
	return out
}
