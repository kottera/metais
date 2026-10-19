// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sample

import (
	"regexp/syntax"
	"strings"

	"github.com/kottera/metais/internal/regexprofile"
)

// maxPatternAttempts is how many strings a pattern is walked for before
// its field is unsatisfiable (item 18).
const maxPatternAttempts = 8

// preferred are the sets a character class yields from, in order of
// preference (item 18): a–z, A–Z, 0–9, space, printable ASCII. A class
// meeting none yields its lowest character.
var preferred = [][2]rune{{'a', 'z'}, {'A', 'Z'}, {'0', '9'}, {' ', ' '}, {' ', '~'}}

// patternText walks the tree regexprofile translates pattern to (ADR-009
// D10 item 6, ADR-010 D10 item 2), the tree the validator matches with,
// drawing every choice from st (item 18): literals as written; a
// character class yields a character of its intersection with the first
// preferred set it meets; alternation picks a branch; a repetition
// {min,max} repeats min plus a draw of 0 to 2 times, capped at max; begin
// and end anchors yield nothing. The k-th attempt (from 0) raises every
// repetition toward its maximum by 2k more.
//
// Because a pattern is unanchored, a result shorter than minLength is
// padded with corpus words after it unless the pattern is end-anchored,
// else before it unless start-anchored, else left to the next attempt,
// whose repetitions are raised. ok is false for a pattern with a
// construct the walk does not support, a word-boundary assertion.
func (s *Generator) patternText(pattern string, st *SplitMix64, k int, minLength int64) (string, bool) {
	re, err := regexprofile.Compile(pattern, s.lim, nil)
	if err != nil {
		// The compiler accepted every pattern of the package.
		return "", false
	}
	tree := re.Syntax()
	w := &walker{st: st, raise: 2 * k}
	var b strings.Builder
	if !w.walk(&b, tree) {
		return "", false
	}
	out := b.String()
	if length(out) < minLength {
		start, end := anchored(tree)
		words := newWords(st)
		switch {
		case !end:
			for length(out) < minLength {
				out += " " + words.next()
			}
		case !start:
			for length(out) < minLength {
				out = words.next() + " " + out
			}
		}
	}
	return out, true
}

// anchored reports whether tree begins with a begin anchor and ends with
// an end anchor.
func anchored(tree *syntax.Regexp) (start, end bool) {
	first, last := tree, tree
	for first.Op == syntax.OpConcat && len(first.Sub) > 0 {
		first = first.Sub[0]
	}
	for last.Op == syntax.OpConcat && len(last.Sub) > 0 {
		last = last.Sub[len(last.Sub)-1]
	}
	start = first.Op == syntax.OpBeginText || first.Op == syntax.OpBeginLine
	end = last.Op == syntax.OpEndText || last.Op == syntax.OpEndLine
	return start, end
}

// walker is one walk of a pattern's tree.
type walker struct {
	st    *SplitMix64
	raise int
}

// walk writes a string re matches to b, and reports false at a construct
// it does not support.
func (w *walker) walk(b *strings.Builder, re *syntax.Regexp) bool {
	switch re.Op {
	case syntax.OpNoMatch:
		// Nothing matches; the result fails the matcher, and the attempts
		// are spent.
		return true
	case syntax.OpEmptyMatch, syntax.OpBeginLine, syntax.OpEndLine, syntax.OpBeginText, syntax.OpEndText:
		return true
	case syntax.OpWordBoundary, syntax.OpNoWordBoundary:
		return false
	case syntax.OpLiteral:
		for _, r := range re.Rune {
			b.WriteRune(r)
		}
		return true
	case syntax.OpCharClass:
		if len(re.Rune) > 0 {
			b.WriteRune(w.class(re.Rune))
		}
		return true
	case syntax.OpAnyCharNotNL, syntax.OpAnyChar:
		b.WriteRune(rune('a' + w.st.Below(26)))
		return true
	case syntax.OpCapture:
		return w.walk(b, re.Sub[0])
	case syntax.OpConcat:
		for _, sub := range re.Sub {
			if !w.walk(b, sub) {
				return false
			}
		}
		return true
	case syntax.OpAlternate:
		return w.walk(b, re.Sub[w.st.Below(uint64(len(re.Sub)))])
	case syntax.OpStar, syntax.OpPlus, syntax.OpQuest, syntax.OpRepeat:
		lo, hi := re.Min, re.Max
		switch re.Op {
		case syntax.OpStar:
			lo, hi = 0, -1
		case syntax.OpPlus:
			lo, hi = 1, -1
		case syntax.OpQuest:
			lo, hi = 0, 1
		}
		n := lo + int(w.st.Below(3)) + w.raise
		if hi >= 0 {
			n = min(n, hi)
		}
		for range n {
			if !w.walk(b, re.Sub[0]) {
				return false
			}
		}
		return true
	}
	return false
}

// class yields a character of the class whose ranges are set, sorted lo,
// hi pairs: one drawn uniformly from its intersection with the first
// preferred set it meets, or its lowest character.
func (w *walker) class(set []rune) rune {
	for _, p := range preferred {
		var parts [][2]rune
		total := uint64(0)
		for i := 0; i+1 < len(set); i += 2 {
			lo, hi := max(set[i], p[0]), min(set[i+1], p[1])
			if lo <= hi {
				parts = append(parts, [2]rune{lo, hi})
				total += uint64(hi - lo + 1)
			}
		}
		if total == 0 {
			continue
		}
		at := w.st.Below(total)
		for _, r := range parts {
			n := uint64(r[1] - r[0] + 1)
			if at < n {
				return r[0] + rune(at)
			}
			at -= n
		}
	}
	return set[0]
}
