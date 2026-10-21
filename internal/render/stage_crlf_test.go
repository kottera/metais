// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import "testing"

// TestCRLF is ADR-005 D10 item 1: every LF not immediately preceded by CR
// becomes CR LF, and a CR LF and a lone CR are left, so the conversion is
// idempotent.
func TestCRLF(t *testing.T) {
	for in, want := range map[string]string{
		"":                 "",
		"a":                "a",
		"\n":               "\r\n",
		"a\nb\n":           "a\r\nb\r\n",
		"a\r\nb\n":         "a\r\nb\r\n",
		"a\rb":             "a\rb",
		"\r\r\n\n":         "\r\r\n\r\n",
		"\n\n":             "\r\n\r\n",
		"é\nü\r\n\r":       "é\r\nü\r\n\r",
		"tail without eol": "tail without eol",
	} {
		got := string(crlf([]byte(in)))
		if got != want {
			t.Errorf("crlf(%q) = %q, want %q", in, got, want)
		}
		if again := string(crlf([]byte(got))); again != got {
			t.Errorf("crlf is not idempotent on %q: %q", got, again)
		}
	}
}
