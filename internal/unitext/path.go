// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package unitext

import (
	"strings"
	"unicode/utf8"

	"metais.org/metais/internal/limits"
)

// Kind is which rule of ADR-005 D12 a path breaks.
type Kind uint8

const (
	// Encoding: the path is not valid UTF-8, or begins with a byte order
	// mark (ADR-001 D2 item 1).
	Encoding Kind = iota + 1
	// NotNFC: the path is not in Normalization Form C (item 1).
	NotNFC
	// Shape: the path is empty, begins or ends with "/", has an empty,
	// "." or ".." segment, or holds "\" or ":" (item 1, ADR-001 D2 item 2).
	Shape
	// Character: a code point is not a path character, is one of Windows'
	// reserved characters, or is in U+FFF0–U+FFFF (item 2).
	Character
	// NotPortable: a segment ends in "." or U+0020 (item 3), is a reserved
	// device name or a spelling of ".git" (item 4), or is longer than 255
	// bytes (item 5).
	NotPortable
	// Length: the path is longer than the path-length limit (item 5).
	Length
	// Depth: the path has more segments than the path-depth limit (item 5).
	Depth
)

// The reasons a Problem gives, each a fixed snake_case token a finding can
// carry as a parameter.
const (
	ReasonInvalidUTF8   = "invalid_utf8"
	ReasonByteOrderMark = "byte_order_mark"

	ReasonEmpty         = "empty"
	ReasonLeadingSlash  = "leading_slash"
	ReasonTrailingSlash = "trailing_slash"
	ReasonEmptySegment  = "empty_segment"
	ReasonDotSegment    = "dot_segment"
	ReasonBackslash     = "backslash"
	ReasonColon         = "colon"

	ReasonReservedName  = "reserved_name"
	ReasonGit           = "git"
	ReasonSegmentEnd    = "segment_end"
	ReasonSegmentLength = "segment_length"
)

// MaxSegment is the most bytes one segment of a path may hold (ADR-005 D12
// item 5, ADR-001 D2 item 5). It is a rule of the format, not a limit.
const MaxSegment = 255

// Problem is one way a path breaks the predicate.
type Problem struct {
	Kind Kind
	// Reason is one of the Reason constants, for Encoding, Shape and
	// NotPortable; empty otherwise.
	Reason string
	// CodePoint is the first code point that breaks the rule, for
	// Character.
	CodePoint rune
	// Observed is the path's length in bytes, for Length, or its number of
	// segments, for Depth.
	Observed int64
}

// CheckPath applies ADR-005 D12 items 1 to 5 to one path, under lim's
// path-length and path-depth, and returns every rule it breaks, each kind
// and reason once, in the order the items list them; nil when it breaks
// none. A path that is not valid UTF-8 is reported only as such, beside
// its length and depth: the other rules are about characters.
//
// The collision rules, items 6 and 7, concern a set of paths and are
// Collisions'.
func CheckPath(p string, lim limits.Set) []Problem {
	c := pathCheck{}
	if n := int64(len(p)); n > lim.Value(limits.PathLength) {
		c.add(Problem{Kind: Length, Observed: n})
	}
	if n := int64(strings.Count(p, "/") + 1); n > lim.Value(limits.PathDepth) {
		c.add(Problem{Kind: Depth, Observed: n})
	}
	if !utf8.ValidString(p) {
		c.add(Problem{Kind: Encoding, Reason: ReasonInvalidUTF8})
		return c.problems
	}
	body := p
	if strings.HasPrefix(p, "\uFEFF") {
		c.add(Problem{Kind: Encoding, Reason: ReasonByteOrderMark})
		body = strings.TrimPrefix(p, "\uFEFF")
	}
	if !IsNFC(p) {
		c.add(Problem{Kind: NotNFC})
	}
	switch {
	case p == "":
		c.add(Problem{Kind: Shape, Reason: ReasonEmpty})
		return c.problems
	case strings.HasPrefix(p, "/"):
		c.add(Problem{Kind: Shape, Reason: ReasonLeadingSlash})
	}
	if strings.HasSuffix(p, "/") {
		c.add(Problem{Kind: Shape, Reason: ReasonTrailingSlash})
	}
	inner := strings.TrimPrefix(body, "/")
	inner = strings.TrimSuffix(inner, "/")
	for seg := range strings.SplitSeq(inner, "/") {
		if seg == "" {
			c.add(Problem{Kind: Shape, Reason: ReasonEmptySegment})
			continue
		}
		c.segment(seg)
	}
	return c.problems
}

type pathCheck struct {
	problems []Problem
}

// add records pr unless a problem of its kind and reason is already
// recorded.
func (c *pathCheck) add(pr Problem) {
	for _, have := range c.problems {
		if have.Kind == pr.Kind && have.Reason == pr.Reason {
			return
		}
	}
	c.problems = append(c.problems, pr)
}

// segment applies the per-segment rules to one non-empty segment.
func (c *pathCheck) segment(seg string) {
	if seg == "." || seg == ".." {
		c.add(Problem{Kind: Shape, Reason: ReasonDotSegment})
		return
	}
	for _, r := range seg {
		switch {
		case r == '\\':
			c.add(Problem{Kind: Shape, Reason: ReasonBackslash})
		case r == ':':
			c.add(Problem{Kind: Shape, Reason: ReasonColon})
		case strings.ContainsRune(`<>"|?*`, r), r >= 0xFFF0 && r <= 0xFFFF, !IsPathChar(r):
			c.add(Problem{Kind: Character, CodePoint: r})
		}
	}
	if strings.HasSuffix(seg, " ") || strings.HasSuffix(seg, ".") {
		c.add(Problem{Kind: NotPortable, Reason: ReasonSegmentEnd})
	}
	if isReservedName(seg) {
		c.add(Problem{Kind: NotPortable, Reason: ReasonReservedName})
	}
	if isGit(seg) {
		c.add(Problem{Kind: NotPortable, Reason: ReasonGit})
	}
	if len(seg) > MaxSegment {
		c.add(Problem{Kind: NotPortable, Reason: ReasonSegmentLength})
	}
}

// isReservedName reports whether seg is a Windows device name (ADR-005 D12
// item 4): the part before its first ".", with trailing U+0020 removed,
// matching CON, PRN, AUX or NUL, or COM or LPT followed by one of 0–9, ¹, ²
// or ³, ASCII-case-insensitively.
func isReservedName(seg string) bool {
	base, _, _ := strings.Cut(seg, ".")
	base = strings.TrimRight(base, " ")
	if len(base) < 3 {
		return false
	}
	head, tail := asciiLower(base[:3]), base[3:]
	switch head {
	case "con", "prn", "aux", "nul":
		return tail == ""
	case "com", "lpt":
		switch tail {
		case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "¹", "²", "³":
			return true
		}
	}
	return false
}

// hfsIgnorable reports whether r is a code point HFS+ ignores when it
// compares names, which Git ignores when it decides whether a segment is
// ".git" (ADR-009 D8 item 6).
func hfsIgnorable(r rune) bool {
	switch {
	case r >= 0x200C && r <= 0x200F, r >= 0x202A && r <= 0x202E, r >= 0x206A && r <= 0x206F, r == 0xFEFF:
		return true
	}
	return false
}

// isGit reports whether seg is a segment Git treats as ".git": ".git" in
// any letter case once the code points HFS+ ignores are removed, or the
// NTFS short name "git~1" (ADR-009 D8 item 6). A ".git" followed by dots
// or spaces, which NTFS strips, already breaks item 3.
func isGit(seg string) bool {
	s := asciiLower(strings.Map(func(r rune) rune {
		if hfsIgnorable(r) {
			return -1
		}
		return r
	}, seg))
	return s == ".git" || s == "git~1"
}

// asciiLower lowers the ASCII letters of s and nothing else, so the result
// does not depend on any locale or on Unicode's case tables.
func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}
