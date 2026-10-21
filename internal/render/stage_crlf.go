// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

// crlf is ADR-005 D10 item 1: every LF not immediately preceded by CR
// becomes CR LF. A CR LF and a lone CR are left as they are, so the
// conversion is idempotent.
func crlf(b []byte) []byte {
	n := 0
	for i, c := range b {
		if c == '\n' && (i == 0 || b[i-1] != '\r') {
			n++
		}
	}
	if n == 0 {
		return b
	}
	out := make([]byte, 0, len(b)+n)
	for i, c := range b {
		if c == '\n' && (i == 0 || b[i-1] != '\r') {
			out = append(out, '\r')
		}
		out = append(out, c)
	}
	return out
}
