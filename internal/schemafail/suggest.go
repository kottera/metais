// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schemafail

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// The bounds on an enum's candidates (ADR-006 D9 item 7): beyond them an
// enum gives no suggestion.
const (
	maxCandidates     = 1000
	maxCandidateRunes = 256
)

// suggest returns the candidate closest to s by ADR-006 D9 item 7's rule,
// and false when none is close enough. A candidate equal to s under simple
// case folding, once leading and trailing white space is trimmed from
// both, is closest; otherwise the one at the smallest Levenshtein distance
// over code points, at most max(1, ⌊n/3⌋) for a candidate of n code
// points. Ties go to the first in UTF-8 byte order. bounded applies an
// enum's bounds on its candidates.
//
// The suggestion is always a candidate, never s: a candidate is package
// content, and s may be a value from the inputs (D9 items 4 and 6).
func suggest(s string, candidates []string, bounded bool) (string, bool) {
	if bounded {
		if len(candidates) > maxCandidates {
			return "", false
		}
		for _, c := range candidates {
			if utf8.RuneCountInString(c) > maxCandidateRunes {
				return "", false
			}
		}
	}
	best, found := "", false
	better := func(c string) bool { return !found || c < best }
	trimmed := strings.TrimSpace(s)
	for _, c := range candidates {
		if c != s && strings.EqualFold(strings.TrimSpace(c), trimmed) && better(c) {
			best, found = c, true
		}
	}
	if found {
		return best, true
	}
	rs := []rune(s)
	bestDist := 0
	for _, c := range candidates {
		cr := []rune(c)
		bound := max(1, len(cr)/3)
		if c == s || abs(len(rs)-len(cr)) > bound {
			continue
		}
		d := levenshtein(rs, cr)
		if d > bound {
			continue
		}
		if !found || d < bestDist || (d == bestDist && c < best) {
			best, bestDist, found = c, d, true
		}
	}
	return best, found
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// levenshtein is the edit distance between a and b: insertions, deletions
// and substitutions of one code point, each costing one.
func levenshtein(a, b []rune) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

// jsonStrings returns the strings among texts, each a value's JSON text:
// an enum's allowed strings.
func jsonStrings(texts []string) []string {
	var out []string
	for _, t := range texts {
		var s string
		if strings.HasPrefix(t, `"`) && json.Unmarshal([]byte(t), &s) == nil {
			out = append(out, s)
		}
	}
	return out
}
