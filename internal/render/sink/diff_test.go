// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sink

import (
	"math/rand/v2"
	"slices"
	"testing"
)

// TestEditScript holds the diff's edit script to what it must be: the
// kept and removed lines are the expected file, the kept and added lines
// the rendered one, for random edits and for edits beyond the search's
// bound.
func TestEditScript(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for n := range 200 {
		var a []string
		for range rng.IntN(40) {
			a = append(a, string(rune('a'+rng.IntN(4)))+"\n")
		}
		b := slices.Clone(a)
		edits := rng.IntN(10)
		if n%50 == 0 {
			edits = 3 * maxEditDistance
		}
		for range edits {
			switch i := rng.IntN(len(b) + 1); rng.IntN(2) {
			case 0:
				b = slices.Insert(b, i, string(rune('a'+rng.IntN(4)))+"\n")
			default:
				if i < len(b) {
					b = slices.Delete(b, i, i+1)
				}
			}
		}
		var gotA, gotB []string
		for _, e := range editScript(a, b) {
			if e.op != '+' {
				gotA = append(gotA, e.line)
			}
			if e.op != '-' {
				gotB = append(gotB, e.line)
			}
		}
		if !slices.Equal(gotA, a) || !slices.Equal(gotB, b) {
			t.Fatalf("case %d: the script does not rebuild its files", n)
		}
	}
	d, cut := unifiedDiff([]byte("a\nb\nc\n"), []byte("a\nB\nc"), "x", "y", MaxDiffJSON)
	want := "--- x\n+++ y\n@@ -1,3 +1,3 @@\n a\n-b\n-c\n+B\n+c\n\\ No newline at end of file\n"
	if d != want || cut {
		t.Errorf("unifiedDiff = %q, want %q", d, want)
	}
}
