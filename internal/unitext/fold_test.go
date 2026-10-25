// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package unitext

import (
	"testing"
	"unicode"
	"unicode/utf8"
)

// TestFold pins Fold to full case folding with no normalization: what
// CommonMark matches link labels by. A fold that was simple, or that
// normalized as CaselessKey does, fails a row.
func TestFold(t *testing.T) {
	for in, want := range map[string]string{
		"Foo":        "foo",
		"\u1e9e":     "ss", // full folding: ẞ folds to "ss"
		"Straße":     "strasse",
		"ΑΓΑΠΗ":      "αγαπη",
		"e\u0301":    "e\u0301", // a decomposed é stays decomposed
		"\u00c9":     "\u00e9",  // and a composed É folds to a composed é
		"already ok": "already ok",
	} {
		if got := Fold(in); got != want {
			t.Errorf("Fold(%+q) = %+q, want %+q", in, got, want)
		}
	}
	if Fold("\u00e9") == Fold("e\u0301") {
		t.Error("Fold matched a composed and a decomposed é; it must not normalize")
	}
}

// TestAppendNFKD holds a Decomposer to NFKD for every code point, since
// the slug filter decomposes one at a time and any one could differ, and
// for a string whose marks NFKD reorders. Into a buffer with room, it
// allocates nothing.
func TestAppendNFKD(t *testing.T) {
	var d Decomposer
	var buf []byte
	var one [utf8.UTFMax]byte
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if !utf8.ValidRune(r) {
			continue
		}
		buf = d.AppendNFKD(buf[:0], utf8.AppendRune(one[:0], r))
		if want := NFKD(string(r)); string(buf) != want {
			t.Fatalf("U+%04X: AppendNFKD gives %q, NFKD %q", r, buf, want)
		}
	}
	const marks = "a\u0302\u0323\ufb01\uac01"
	if got, want := string(d.AppendNFKD([]byte("x"), []byte(marks))), "x"+NFKD(marks); got != want {
		t.Fatalf("AppendNFKD gives %q, not %q", got, want)
	}
	buf = make([]byte, 0, 64)
	src := []byte("\u01c5")
	if n := testing.AllocsPerRun(100, func() { buf = d.AppendNFKD(buf[:0], src) }); n != 0 {
		t.Fatalf("AppendNFKD allocates %v times a run into a buffer with room", n)
	}
}
