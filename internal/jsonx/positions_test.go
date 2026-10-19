// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package jsonx

import (
	"testing"
	"unicode/utf8"

	"github.com/kottera/metais/internal/finding"
)

// TestPositions: every value and every member name has a byte offset, and
// a line and column in ADR-006 D5 item 3's units (ADR-006 D13 item 2).
func TestPositions(t *testing.T) {
	src := "\xef\xbb\xbf{\"é\": [1,\r\n  {\"a/b\": \"x\",\r \"~\": null}],\n\t\"z\": {\"\": true}}"
	d := mustParse(t, src)
	pos := d.Positions()
	values := []struct {
		ptr       string
		off       int
		line, col int
	}{
		{"", 3, 1, 1},
		{"/é", 10, 1, 7},
		{"/é/0", 11, 1, 8},
		{"/é/1", 17, 2, 3},
		{"/é/1/a~1b", 25, 2, 11},
		{"/é/1/~0", 36, 3, 7},
		{"/z", 50, 4, 7},
		{"/z/", 55, 4, 12},
	}
	for _, v := range values {
		off, ok := d.ValueOffset(v.ptr)
		p, okPos := pos.Value(v.ptr)
		if !ok || !okPos || off != v.off || p != (finding.Pos{Line: v.line, Column: v.col}) {
			t.Errorf("value %q: offset %d (%v), position %+v (%v); want %d at %d:%d", v.ptr, off, ok, p, okPos, v.off, v.line, v.col)
		}
	}
	keys := []struct {
		ptr       string
		off       int
		line, col int
	}{
		{"/é", 4, 1, 2},
		{"/é/1/a~1b", 18, 2, 4},
		{"/é/1/~0", 31, 3, 2},
		{"/z", 45, 4, 2},
		{"/z/", 51, 4, 8},
	}
	for _, k := range keys {
		off, ok := d.KeyOffset(k.ptr)
		p, okPos := pos.Key(k.ptr)
		if !ok || !okPos || off != k.off || p != (finding.Pos{Line: k.line, Column: k.col}) {
			t.Errorf("key %q: offset %d (%v), position %+v (%v); want %d at %d:%d", k.ptr, off, ok, p, okPos, k.off, k.line, k.col)
		}
	}
	for _, ptr := range []string{"/nope", "/é/2", "/é/01", "/é/-", "/é/0/x", "é", "/z//"} {
		if _, ok := pos.Value(ptr); ok {
			t.Errorf("value %q is found", ptr)
		}
	}
	for _, ptr := range []string{"", "/é/0", "/é/1"} {
		if _, ok := pos.Key(ptr); ok {
			t.Errorf("key %q is found, though it names no member", ptr)
		}
	}
}

// TestPathPositions: a value and a member name read by their Path are
// where Value and Key say they are, read once or again, in any order.
func TestPathPositions(t *testing.T) {
	src := "{\"é\": [1,\r\n  {\"a/b\": \"x\",\r \"~\": null}],\n\t\"z\": {\"\": true}}"
	pos := mustParse(t, src).Positions().(finding.PathPositions)
	ptrs := []string{"/z/", "", "/é/1/~0", "/é", "/é/0", "/é/1", "/é/1/a~1b", "/z", "/nope", "/é/2", "/é/01", "/é/-", "/é/0/x", "/z//"}
	for range 2 {
		for _, ptr := range ptrs {
			p, ok := finding.ParsePath(ptr)
			if !ok {
				t.Fatalf("%q is not a pointer", ptr)
			}
			wantV, wantOK := pos.Value(ptr)
			if v, ok := pos.ValueAt(p); v != wantV || ok != wantOK {
				t.Errorf("value %q at %+v (%v), want %+v (%v)", ptr, v, ok, wantV, wantOK)
			}
			wantK, wantOK := pos.Key(ptr)
			if k, ok := pos.KeyAt(p); k != wantK || ok != wantOK {
				t.Errorf("key %q at %+v (%v), want %+v (%v)", ptr, k, ok, wantK, wantOK)
			}
		}
	}
}

// TestDuplicatePositions: a duplicate name keeps its first occurrence's
// positions, so a later finding about the value points where the value
// the reader would keep is.
func TestDuplicatePositions(t *testing.T) {
	d, fs := parse(t, `{"a": [1], "a": [2, 3]}`)
	if len(fs) != 1 {
		t.Fatalf("%s", describe(fs))
	}
	if off, ok := d.KeyOffset("/a"); !ok || off != 1 {
		t.Errorf("key /a at %d (%v), want 1", off, ok)
	}
	if off, ok := d.ValueOffset("/a/0"); !ok || off != 7 {
		t.Errorf("value /a/0 at %d (%v), want 7", off, ok)
	}
	if _, ok := d.ValueOffset("/a/1"); ok {
		t.Error("value /a/1, of the duplicate, is found")
	}
}

// TestNoPositionsWithoutParse: a Doc that read nothing finds nothing.
func TestNoPositionsWithoutParse(t *testing.T) {
	var d Doc
	if _, ok := d.Positions().Value(""); ok {
		t.Error("the zero Doc has a root")
	}
	d, _ = parse(t, "\xff")
	if _, ok := d.ValueOffset(""); ok {
		t.Error("a Doc refused before reading has a root")
	}
}

// TestLineIndex pins ADR-006 D5 item 3: LF, CRLF and a lone CR each end
// one line; a column counts code points; a byte order mark is not
// counted.
func TestLineIndex(t *testing.T) {
	x := newLineIndex([]byte("a\nb\r\nc\rdé\xffz"))
	for _, c := range []struct{ off, line, col int }{
		{0, 1, 1}, {1, 1, 2}, {2, 2, 1}, {3, 2, 2}, {4, 2, 3}, {5, 3, 1}, {7, 4, 1}, {8, 4, 2}, {10, 4, 3}, {11, 4, 4}, {12, 4, 5},
	} {
		if p := x.pos(c.off); p != (finding.Pos{Line: c.line, Column: c.col}) {
			t.Errorf("offset %d: %+v, want %d:%d", c.off, p, c.line, c.col)
		}
	}
	x = newLineIndex([]byte("\xef\xbb\xbfab"))
	for _, c := range []struct{ off, col int }{{0, 1}, {2, 1}, {3, 1}, {4, 2}} {
		if p := x.pos(c.off); p != (finding.Pos{Line: 1, Column: c.col}) {
			t.Errorf("after a byte order mark, offset %d: %+v, want 1:%d", c.off, p, c.col)
		}
	}
}

// TestLineIndexCheckpoints pins that a column counted from a checkpoint
// is the column counted from the start of its line, at every code point
// and at the end of a text, whatever its length against the checkpoint's
// and wherever a line or a multi-byte character falls across one.
func TestLineIndexCheckpoints(t *testing.T) {
	for _, n := range []int{checkpoint - 1, checkpoint, checkpoint + 1, 3*checkpoint + 7} {
		for _, prefix := range []string{"", bom} {
			src := []byte(prefix)
			for i := 0; len(src) < n; i++ {
				src = append(src, []string{"é", "a", "\U0001F600", "\r\n", "€", "\n", "b"}[i%7]...)
			}
			x := newLineIndex(src)
			line, start := 1, len(prefix)
			for off := 0; off <= len(src); off++ {
				if off < len(src) && !utf8.RuneStart(src[off]) {
					continue
				}
				if off > start && (src[off-1] == '\n' || src[off-1] == '\r' && (off == len(src) || src[off] != '\n')) {
					line, start = line+1, off
				}
				want := finding.Pos{Line: line, Column: utf8.RuneCount(src[start:max(off, start)]) + 1}
				if p := x.pos(off); p != want {
					t.Fatalf("length %d, prefix %q, offset %d: %+v, want %+v", len(src), prefix, off, p, want)
				}
			}
		}
	}
}
