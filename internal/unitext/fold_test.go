// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package unitext

import "testing"

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
