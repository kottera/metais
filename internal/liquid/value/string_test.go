// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package value

import (
	"errors"
	"testing"
)

// TestStringScalarValues pins ADR-003 D8 item 4: a String holds Unicode
// scalar values and nothing else.
func TestStringScalarValues(t *testing.T) {
	for _, s := range []string{
		"", "plain", "café", "\u0000", "\t\n", "\U0001F600", "\uFFFD", "\U0010FFFF",
		// Noncharacters are scalar values; refusing them is a reader's finding.
		"\uFFFE", "\uFDD0", "\U0001FFFF",
	} {
		v, err := NewString(s)
		if err != nil {
			t.Errorf("NewString(%q): %v", s, err)
			continue
		}
		if v.Text() != s {
			t.Errorf("NewString(%q).Text() = %q", s, v.Text())
		}
	}
	for _, s := range []string{
		"\xff",                       // not a UTF-8 byte
		"a\x80b",                     // a lone continuation byte
		"\xc3",                       // truncated sequence
		"\xc0\xaf",                   // overlong encoding
		"\xed\xa0\x80",               // U+D800, a surrogate
		"\xed\xbf\xbf",               // U+DFFF, a surrogate
		"ok\xed\xa0\xbd\xed\xb8\x80", // a surrogate pair encoded as two UTF-8 sequences (CESU-8)
		"\xf4\x90\x80\x80",           // above U+10FFFF
	} {
		if _, err := NewString(s); !errors.Is(err, ErrNotScalarValues) {
			t.Errorf("NewString(%q): err = %v, want ErrNotScalarValues", s, err)
		}
		if _, err := NewObject(Member{Name: s, Value: Null{}}); !errors.Is(err, ErrNotScalarValues) {
			t.Errorf("NewObject with name %q: err = %v, want ErrNotScalarValues", s, err)
		}
	}
}
