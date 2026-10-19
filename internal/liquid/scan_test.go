// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/kottera/metais/internal/testenv"
)

// TestLineIndex pins ADR-006 D5 item 3's counting: lines end at LF, CR LF
// or a lone CR, and columns count code points.
func TestLineIndex(t *testing.T) {
	testenv.Home(t)
	src := "a\nb\r\nc\rdé€😀x"
	x := newLineIndex(src)
	for _, c := range []struct{ off, line, col int }{
		{0, 1, 1}, {1, 1, 2}, {2, 2, 1}, {3, 2, 2}, {5, 3, 1}, {7, 4, 1},
		{strings.Index(src, "é"), 4, 2}, {strings.Index(src, "€"), 4, 3},
		{strings.Index(src, "😀"), 4, 4}, {strings.Index(src, "x"), 4, 5}, {len(src), 4, 6},
	} {
		if l, col := x.at(c.off); l != c.line || col != c.col {
			t.Errorf("at(%d) = %d:%d, want %d:%d", c.off, l, col, c.line, c.col)
		}
	}
}

// TestLineIndexBounded pins that a lookup's work does not grow with its
// line: on a single 1 MiB line, each lookup scans at most two chunks,
// counted rather than timed (ADR-010 D14 item 7). Counting from the start
// of the line instead scans half the line per lookup on average, and
// fails here.
func TestLineIndexBounded(t *testing.T) {
	testenv.Home(t)
	src := strings.Repeat("aé{{ x }}", 1<<17)
	x := newLineIndex(src)
	const lookups = 1000
	runes, prev := 0, 0
	for i := range lookups {
		off := i * (len(src) / lookups)
		runes += utf8.RuneCountInString(src[prev:off])
		prev = off
		if _, col := x.at(off); col != runes+1 {
			t.Fatalf("at(%d): column %d, want %d", off, col, runes+1)
		}
	}
	if limit := lookups * 2 * chunkSize; x.scanned > limit {
		t.Errorf("%d lookups scanned %d bytes, above %d", lookups, x.scanned, limit)
	}
}
