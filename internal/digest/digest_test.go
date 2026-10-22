// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"crypto/sha256"
	"testing"

	"metais.org/metais/internal/limits"
)

// TestVerifyValues holds the digest file to the values its form can hold,
// entries plus 4, counted as it is decoded (ADR-009 Amendment A6 item 2):
// a file of exactly that many values is read and compared, and one value
// more is refused as not in its form, reason shape, whatever it holds.
func TestVerifyValues(t *testing.T) {
	lim := limits.Defaults()
	if f, ok := lim.Lower("entries=3"); !ok {
		t.Fatal(f)
	}
	entries := []Entry{{Name: "a", Sum: sha256.Sum256([]byte("a"))}, {Name: "b", Sum: sha256.Sum256([]byte("b"))}}
	file := func(names ...string) []byte {
		var listed []Entry
		for _, n := range names {
			listed = append(listed, Entry{Name: n, Sum: sha256.Sum256([]byte(n))})
		}
		return Encode(listed)
	}
	for _, c := range []struct {
		name   string
		src    []byte
		code   string
		reason string
	}{
		// 4 + 3 values: read, and the name that is no entry refused.
		{"at the bound", file("a", "b", "c"), string(CodeEntriesMismatch), reasonNoEntry},
		// 4 + 4 values: refused before the fourth digest is decoded.
		{"one value over", file("a", "b", "c", "d"), string(CodeNotCanonical), reasonShape},
		// The values need not be digests to be counted.
		{"nested values", []byte(`{"algorithm":"sha256","entries":{},"package":[[[0,0,0,0]]]}`), string(CodeNotCanonical), reasonShape},
	} {
		t.Run(c.name, func(t *testing.T) {
			fs := Verify(c.src, entries, lim)
			if len(fs) == 0 {
				t.Fatal("the file verified")
			}
			v, _ := fs[0].Param("reason")
			got, _ := v.Str()
			if string(fs[0].Code()) != c.code || got != c.reason {
				t.Errorf("%s %v, want %s %s", fs[0].Code(), got, c.code, c.reason)
			}
		})
	}
}
