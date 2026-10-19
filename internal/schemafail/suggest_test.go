// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schemafail

import (
	"fmt"
	"strings"
	"testing"
)

// TestSuggest is ADR-006 D9 item 7's rule: case folding after trimming
// first, then the smallest Levenshtein distance over code points within
// max(1, ⌊n/3⌋), ties to the first in byte order, never the value itself.
func TestSuggest(t *testing.T) {
	for _, c := range []struct {
		s          string
		candidates []string
		want       string
	}{
		{" Draft\t", []string{"final", "draft"}, "draft"},
		{"ΣΊΣΥΦΟΣ", []string{"σίσυφος"}, "σίσυφος"},
		{"KELVIN", []string{"kelvin"}, "kelvin"},
		{"drft", []string{"draft", "final"}, "draft"},
		{"colr", []string{"color", "colour"}, "color"},
		{"cat", []string{"bat", "car"}, "bat"},
		{"naïv", []string{"naïve"}, "naïve"},
		{"ab", []string{"xy"}, ""},
		{"abcdef", []string{"abcxyz"}, ""},
		{"abcdefghi", []string{"abcdefgxy", "abcdefhxy"}, "abcdefgxy"},
		{"abcdefghi", []string{"abcdewxyz"}, ""},
		{"draft", []string{"draft"}, ""},
		{"x", nil, ""},
	} {
		got, ok := suggest(c.s, c.candidates, true)
		if got != c.want || ok != (c.want != "") {
			t.Errorf("suggest(%q, %q) = %q, %v, want %q", c.s, c.candidates, got, ok, c.want)
		}
	}

	// An enum beyond the bounds gives no suggestion; member names have
	// none.
	var many []string
	for i := range 1001 {
		many = append(many, fmt.Sprintf("v%04d", i))
	}
	long := []string{"draft", strings.Repeat("a", 257)}
	for _, c := range []struct {
		s          string
		candidates []string
		bounded    bool
		want       string
	}{
		{"v0000x", many, true, ""},
		{"v0000x", many[:1000], true, "v0000"},
		{"v0000x", many, false, "v0000"},
		{"drafx", long, true, ""},
		{"drafx", long[:1], true, "draft"},
		{"drafx", long, false, "draft"},
	} {
		if got, _ := suggest(c.s, c.candidates, c.bounded); got != c.want {
			t.Errorf("%d candidates, bounded %v: %q, want %q", len(c.candidates), c.bounded, got, c.want)
		}
	}
}

// TestSuggestBounded holds the work of a suggestion to the candidates' size,
// not the value's: a value of a million code points against a thousand
// candidates computes no distance.
func TestSuggestBounded(t *testing.T) {
	var cands []string
	for i := range 1000 {
		cands = append(cands, fmt.Sprintf("value-%04d", i))
	}
	huge := strings.Repeat("x", 1<<20)
	if n := testing.AllocsPerRun(3, func() { suggest(huge, cands, true) }); n != 0 {
		t.Errorf("%v allocations for a huge value: a distance was computed", n)
	}
}
