// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"sort"
	"strings"
)

// The template lexer finds the boundaries of text, "{{ … }}" and "{% … %}"
// tokens exactly where Shopify's tokenizer finds them, so that a template
// splits into the same tokens in METAIS as in the reference:
//
//   - text runs to the next "{{" or "{%";
//   - a tag runs from "{%" to the first "%}" after it;
//   - an output runs from "{{" to the first "}" after it, and is well
//     formed only when that "}" is followed by another. A "{%" met before
//     the first "}" makes the token run on to the next "%}", as a tag
//     would; such a token is never well formed.
//
// A token that is not well formed carries the reason in bad. Where the
// grammar reads a token, a bad one is a syntax error; inside a comment,
// whose body is skipped, it is skipped too, as Shopify skips it.

type tokKind uint8

const (
	tokEOF tokKind = iota
	tokText
	tokOutput
	tokTag
)

// tok is one token. start and end are the whole token's bytes. For a tag,
// name is the tag name and markup what follows it, without the
// whitespace-control markers; for an output, markup is what lies between
// the delimiters. markupAt is markup's offset in the source.
type tok struct {
	kind         tokKind
	start, end   int
	trimL, trimR bool
	name         string
	markup       string
	markupAt     int
	bad          string
}

// scanner yields a template's tokens in order.
type scanner struct {
	src string
	off int
}

func (s *scanner) next() tok {
	src := s.src
	start := s.off
	if start >= len(src) {
		return tok{kind: tokEOF, start: start, end: start}
	}
	if i := openDelim(src, start); i != start {
		if i < 0 {
			i = len(src)
		}
		s.off = i
		return tok{kind: tokText, start: start, end: i}
	}
	if src[start+1] == '%' {
		end := strings.Index(src[start+2:], "%}")
		if end < 0 {
			s.off = len(src)
			return tok{kind: tokTag, start: start, end: len(src), bad: "tag not closed by %}"}
		}
		s.off = start + 2 + end + 2
		return tagToken(src, start, s.off)
	}
	// An output: scan to the first "}", or to a "{%", which turns the token
	// into one running to the next "%}".
	for i := start + 2; i < len(src); i++ {
		switch {
		case src[i] == '}':
			if i+1 < len(src) && src[i+1] == '}' {
				s.off = i + 2
				return outputToken(src, start, s.off)
			}
			s.off = i + 1
			return tok{kind: tokOutput, start: start, end: s.off, bad: "output not closed by }}"}
		case src[i] == '{' && i+1 < len(src) && src[i+1] == '%':
			end := strings.Index(src[i+2:], "%}")
			if end < 0 {
				s.off = len(src)
			} else {
				s.off = i + 2 + end + 2
			}
			return tok{kind: tokOutput, start: start, end: s.off, bad: "output not closed by }}"}
		}
	}
	s.off = len(src)
	return tok{kind: tokOutput, start: start, end: len(src), bad: "output not closed by }}"}
}

// openDelim returns the offset of the first "{{" or "{%" at or after i, or
// -1.
func openDelim(src string, i int) int {
	for {
		j := strings.IndexByte(src[i:], '{')
		if j < 0 {
			return -1
		}
		i += j
		if i+1 < len(src) && (src[i+1] == '{' || src[i+1] == '%') {
			return i
		}
		i++
	}
}

// outputToken splits the well-formed output src[start:end], which begins
// "{{" and ends "}}".
func outputToken(src string, start, end int) tok {
	t := tok{kind: tokOutput, start: start, end: end}
	in, inEnd := start+2, end-2
	if in < inEnd && src[in] == '-' {
		t.trimL = true
		in++
	}
	if in < inEnd && src[inEnd-1] == '-' {
		t.trimR = true
		inEnd--
	}
	t.markup, t.markupAt = src[in:inEnd], in
	return t
}

// tagToken splits the tag src[start:end], which begins "{%" and ends
// "%}", into its whitespace control, name and markup. The name is "#" or a
// run of ASCII letters, digits and underscores, as Shopify's \w is; a tag
// whose name is neither is bad.
func tagToken(src string, start, end int) tok {
	t := tok{kind: tokTag, start: start, end: end}
	in, inEnd := start+2, end-2
	if in < inEnd && src[in] == '-' {
		t.trimL = true
		in++
	}
	if in < inEnd && src[inEnd-1] == '-' {
		t.trimR = true
		inEnd--
	}
	for in < inEnd && isSpace(src[in]) {
		in++
	}
	n := in
	if n < inEnd && src[n] == '#' {
		n++
	} else {
		for n < inEnd && isWordByte(src[n]) {
			n++
		}
	}
	if n == in {
		t.bad = "tag has no name"
		return t
	}
	t.name = src[in:n]
	for n < inEnd && isSpace(src[n]) && t.name != "#" {
		n++
	}
	t.markup, t.markupAt = src[n:inEnd], n
	return t
}

// isSpace reports whether b is whitespace as Ruby's \s is: space, tab,
// line feed, vertical tab, form feed or carriage return, the set ADR-004
// D3 item 1 names.
func isSpace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\v', '\f', '\r':
		return true
	}
	return false
}

// isWordByte reports whether b is an ASCII letter, digit or underscore.
func isWordByte(b byte) bool {
	return b == '_' || 'a' <= b && b <= 'z' || 'A' <= b && b <= 'Z' || '0' <= b && b <= '9'
}

// lineIndex turns byte offsets into lines and columns as ADR-006 D5 item 3
// counts them: lines end at LF, CR LF or a lone CR, and a column counts
// code points from the start of its line. A code point is counted at its
// first byte, so an offset is always counted as the start of the code
// point it falls in.
//
// Every lookup is a binary search and at most one chunk's scan, whatever
// the line's length: the code points before each chunkSize-byte boundary
// are counted once, when the index is built.
type lineIndex struct {
	src    string
	starts []int // the offset at which each line begins
	chunks []int // chunks[k] is the code points in src[:k*chunkSize]
	// scanned counts the bytes lookups have scanned, so a test holds the
	// work of a lookup to its bound by a count (ADR-010 D14 item 7).
	scanned int
}

const chunkSize = 64

func newLineIndex(src string) *lineIndex {
	x := &lineIndex{src: src, starts: []int{0}, chunks: make([]int, 0, len(src)/chunkSize+1)}
	n := 0
	for i := 0; i < len(src); i++ {
		if i%chunkSize == 0 {
			x.chunks = append(x.chunks, n)
		}
		if !isContinuation(src[i]) {
			n++
		}
		switch src[i] {
		case '\n':
			x.starts = append(x.starts, i+1)
		case '\r':
			if i+1 >= len(src) || src[i+1] != '\n' {
				x.starts = append(x.starts, i+1)
			}
		}
	}
	return x
}

// isContinuation reports whether b continues a UTF-8 sequence.
func isContinuation(b byte) bool { return b&0xC0 == 0x80 }

// codePoints returns the number of code points that begin in src[:off].
func (x *lineIndex) codePoints(off int) int {
	k := off / chunkSize
	if k >= len(x.chunks) {
		k = len(x.chunks) - 1
	}
	if k < 0 {
		return 0
	}
	n := x.chunks[k]
	x.scanned += off - k*chunkSize
	for i := k * chunkSize; i < off; i++ {
		if !isContinuation(x.src[i]) {
			n++
		}
	}
	return n
}

// at returns the 1-based line and column of the byte at off, which may be
// len(src).
func (x *lineIndex) at(off int) (line, col int) {
	off = max(0, min(off, len(x.src)))
	l := sort.Search(len(x.starts), func(i int) bool { return x.starts[i] > off }) - 1
	return l + 1, x.codePoints(off) - x.codePoints(x.starts[l]) + 1
}
