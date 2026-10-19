// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package xref

import (
	"strings"
	"unicode/utf8"
)

// maxDistance is the farthest a suggestion may be from the name read
// (ADR-007 D3 items 3 and 4.6: "the nearest within edit distance 2").
const maxDistance = 2

// nearest returns the candidate nearest s, and false when none is near
// enough: one equal to s under simple case folding once leading and
// trailing white space is trimmed from both, else the one at the smallest
// Levenshtein distance over code points, at most maxDistance. Ties go to
// the first in byte order. The suggestion is never s itself.
func nearest(s string, candidates []string) (string, bool) {
	best, found := "", false
	trimmed := strings.TrimSpace(s)
	for _, c := range candidates {
		if c != s && strings.EqualFold(strings.TrimSpace(c), trimmed) && (!found || c < best) {
			best, found = c, true
		}
	}
	if found {
		return best, true
	}
	n := utf8.RuneCountInString(s)
	var rs []rune
	bestDist := 0
	for _, c := range candidates {
		if c == s || abs(n-utf8.RuneCountInString(c)) > maxDistance {
			continue
		}
		if rs == nil {
			rs = []rune(s)
		}
		d := levenshtein(rs, []rune(c))
		if d > maxDistance {
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
