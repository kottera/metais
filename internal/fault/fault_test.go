// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package fault

import "testing"

func TestHas(t *testing.T) {
	const a, b, c Set = 1 << 0, 1 << 1, 1 << 63
	tests := []struct {
		name string
		set  Set
		f    Set
		want bool
	}{
		{"zero set holds nothing", 0, a, false},
		{"zero set holds no high bit", 0, c, false},
		{"empty query is never held", a | b, 0, false},
		{"empty query on zero set", 0, 0, false},
		{"member", a | b, a, true},
		{"other member", a | b, b, true},
		{"high bit member", c, c, true},
		{"non-member", a, b, false},
		{"all of several", a | b | c, a | c, true},
		{"not all of several", a, a | b, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.set.Has(tt.f); got != tt.want {
				t.Errorf("Set(%#x).Has(%#x) = %v, want %v", uint64(tt.set), uint64(tt.f), got, tt.want)
			}
		})
	}
}

// Each fault's bit is the one the plan assigns it, so that no two faults
// share a bit.
func TestFaultBits(t *testing.T) {
	for name, tt := range map[string]struct{ got, want Set }{
		"SkipDigest":           {SkipDigest, 1 << 0},
		"FirstFinding":         {FirstFinding, 1 << 1},
		"UnpackToTemp":         {UnpackToTemp, 1 << 2},
		"SkipConfine":          {SkipConfine, 1 << 3},
		"MapOrder":             {MapOrder, 1 << 4},
		"ReadExisting":         {ReadExisting, 1 << 5},
		"RemoveForeign":        {RemoveForeign, 1 << 6},
		"StdoutNotice":         {StdoutNotice, 1 << 7},
		"UpdateBeyondExpected": {UpdateBeyondExpected, 1 << 8},
		"PackMapOrder":         {PackMapOrder, 1 << 9},
		"ColourJSON":           {ColourJSON, 1 << 10},
	} {
		if tt.got != tt.want {
			t.Errorf("%s = %#x, want %#x", name, uint64(tt.got), uint64(tt.want))
		}
	}
}
