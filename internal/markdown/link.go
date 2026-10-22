// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package markdown

import (
	"strings"
	"unicode/utf8"

	"metais.org/metais/internal/unitext"
)

// maxLabel is the most characters a link label holds (§4.7).
const maxLabel = 999

// maxParenDepth is how deep the balanced parentheses of a link destination
// may nest (§6.3 allows a limit of at least three). It bounds the reading
// of a destination that never closes.
const maxParenDepth = 32

// isASCIIPunct reports whether b is ASCII punctuation (§2.1).
func isASCIIPunct(b byte) bool {
	return b >= '!' && b <= '/' || b >= ':' && b <= '@' || b >= '[' && b <= '`' || b >= '{' && b <= '~'
}

// scanLabel reads a link label at s[i], which is '[' (§4.7): at most 999
// characters, at least one not whitespace, and no unescaped bracket. It
// returns the offset after the closing ']' and the label's text.
func scanLabel(s []byte, i int) (end int, label []byte, ok bool) {
	chars, content := 0, false
	for j := i + 1; j < len(s); {
		switch c := s[j]; {
		case c == '\\' && j+1 < len(s) && isASCIIPunct(s[j+1]):
			j += 2
			chars += 2
			content = true
		case c == '[':
			return 0, nil, false
		case c == ']':
			if !content {
				return 0, nil, false
			}
			return j + 1, s[i+1 : j], true
		default:
			if c != ' ' && c != '\t' && c != '\n' {
				content = true
			}
			if c&0xC0 != 0x80 {
				chars++
			}
			j++
		}
		if chars > maxLabel {
			return 0, nil, false
		}
	}
	return 0, nil, false
}

// labelText reports whether text, which holds no unescaped bracket, is
// the inside of a link label: at most 999 characters, and at least one of
// them not whitespace (§4.7).
func labelText(text []byte) bool {
	if len(text) > maxLabel*utf8.UTFMax {
		return false
	}
	content := false
	for _, b := range text {
		if b != ' ' && b != '\t' && b != '\n' {
			content = true
			break
		}
	}
	return content && utf8.RuneCount(text) <= maxLabel
}

// normalizeLabel is the form two labels match in (§4.7): Unicode case
// folded, leading and trailing whitespace removed, and each run of inner
// whitespace one space.
func normalizeLabel(label []byte) string {
	return unitext.Fold(strings.Join(strings.FieldsFunc(string(label), func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n'
	}), " "))
}

// skipLinkSpace skips spaces and tabs and at most one line ending.
func skipLinkSpace(s []byte, i int) int {
	nl := false
	for i < len(s) {
		switch s[i] {
		case ' ', '\t':
		case '\n':
			if nl {
				return i
			}
			nl = true
		default:
			return i
		}
		i++
	}
	return i
}

// skipSpaceTab skips spaces and tabs.
func skipSpaceTab(s []byte, i int) int {
	for i < len(s) && isSpaceOrTab(s[i]) {
		i++
	}
	return i
}

// scanDestination reads a link destination at s[i] (§6.3): between '<' and
// '>' with no line ending and no unescaped angle bracket, or a non-empty run
// with no space or control character and balanced parentheses.
func scanDestination(s []byte, i int) (end int, ok bool) {
	if i < len(s) && s[i] == '<' {
		for j := i + 1; j < len(s); j++ {
			switch c := s[j]; {
			case c == '\\' && j+1 < len(s) && isASCIIPunct(s[j+1]):
				j++
			case c == '>':
				return j + 1, true
			case c == '\n' || c == '<':
				return 0, false
			}
		}
		return 0, false
	}
	depth := 0
	j := i
scan:
	for j < len(s) {
		switch c := s[j]; {
		case c == '\\' && j+1 < len(s) && isASCIIPunct(s[j+1]):
			j += 2
			continue
		case c == '(':
			if depth++; depth > maxParenDepth {
				return 0, false
			}
		case c == ')':
			if depth == 0 {
				break scan
			}
			depth--
		case c <= ' ' || c == 0x7f:
			break scan
		}
		j++
	}
	if j == i || depth != 0 {
		return 0, false
	}
	return j, true
}

// titleMemo remembers where a title scan reached the end unclosed, so that
// no later one is read again (a title is read from left to right, and one
// that finds no closing quote from some offset finds none from a later
// one).
type titleMemo struct {
	unclosed [3]int // by quote: '"', '\'', '('; the offset from which none closes, or -1
}

func newTitleMemo() *titleMemo { return &titleMemo{unclosed: [3]int{-1, -1, -1}} }

// scanTitle reads a link title at s[i] (§6.3): between double quotes,
// single quotes or parentheses, with backslash escapes, and in parentheses
// no unescaped parenthesis.
func scanTitle(s []byte, i int, memo *titleMemo) (end int, ok bool) {
	if i >= len(s) {
		return 0, false
	}
	q, k := byte(0), 0
	switch s[i] {
	case '"':
		q, k = '"', 0
	case '\'':
		q, k = '\'', 1
	case '(':
		q, k = ')', 2
	default:
		return 0, false
	}
	if memo != nil && memo.unclosed[k] >= 0 && i >= memo.unclosed[k] {
		return 0, false
	}
	for j := i + 1; j < len(s); j++ {
		switch c := s[j]; {
		case c == '\\' && j+1 < len(s) && isASCIIPunct(s[j+1]):
			j++
		case c == q:
			return j + 1, true
		case q == ')' && c == '(':
			return 0, false
		}
	}
	if memo != nil {
		memo.unclosed[k] = i
	}
	return 0, false
}

// refDefinition reads a link reference definition at s[at], the start of
// a line of a paragraph's text (§4.7): a label, ':', a destination and an
// optional title, then nothing but spaces and tabs on the line. It returns
// the offset of the next line and the label, normalized.
func refDefinition(s []byte, at int) (end int, label string, ok bool) {
	if at >= len(s) || s[at] != '[' {
		return 0, "", false
	}
	lend, raw, ok := scanLabel(s, at)
	if !ok || lend >= len(s) || s[lend] != ':' {
		return 0, "", false
	}
	d := skipLinkSpace(s, lend+1)
	dend, ok := scanDestination(s, d)
	if !ok {
		return 0, "", false
	}
	// Without a title, the line must end after the destination.
	noTitle := -1
	if k := skipSpaceTab(s, dend); k == len(s) || s[k] == '\n' {
		noTitle = min(k+1, len(s))
	}
	if t := skipLinkSpace(s, dend); t > dend {
		if tend, ok := scanTitle(s, t, nil); ok {
			if k := skipSpaceTab(s, tend); k == len(s) || s[k] == '\n' {
				return min(k+1, len(s)), normalizeLabel(raw), true
			}
		}
	}
	if noTitle < 0 {
		return 0, "", false
	}
	return noTitle, normalizeLabel(raw), true
}
