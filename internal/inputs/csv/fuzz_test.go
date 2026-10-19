// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package csv

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
)

var delimiters = [...]Delimiter{Comma, Semicolon, Pipe, Tab}

// FuzzCSVRead is ADR-010 D8 item 5 and D14 item 6. Under every delimiter
// the reader never panics, and every position it gives is within data.
// For the record set data encodes, reading back what Write writes yields
// the same fields, quoted where Write's rule quotes them.
//
// data encodes a record set as: its first byte, whose low two bits are the
// width less one and whose next two choose the delimiter; then the fields
// in order, separated by 0x1F, the last record padded with empty fields.
func FuzzCSVRead(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		for _, d := range delimiters {
			tbl, fs := Read(data, d, testSource, limits.Defaults())
			checkPositions(t, data, tbl, fs)
		}
		if len(data) == 0 {
			return
		}
		width := 1 + int(data[0]&3)
		delim := delimiters[data[0]>>2&3]
		fields := strings.Split(string(data[1:]), "\x1f")
		var set [][]string
		for len(fields) > 0 {
			rec := make([]string, width)
			n := copy(rec, fields)
			fields = fields[n:]
			set = append(set, rec)
		}
		out, err := Write(set, rune(delim))
		if err != nil {
			t.Fatalf("Write(%q): %v", set, err)
		}
		tbl, fs := Read(out, delim, testSource, limits.Defaults())
		for _, fd := range fs {
			if c := fd.Code(); c != CodeHeaderEmpty && c != CodeHeaderDuplicate {
				t.Fatalf("reading back %q: %s", out, fd.Message())
			}
		}
		read := rows(tbl)
		if len(read) != len(set) {
			t.Fatalf("wrote %d records as %q and read back %d", len(set), out, len(read))
		}
		for i, rec := range set {
			if len(read[i]) != width {
				t.Fatalf("record %d of %q: read %d fields, want %d", i, out, len(read[i]), width)
			}
			for j, text := range rec {
				wantQuoted := strings.ContainsRune(text, rune(delim)) || strings.ContainsAny(text, "\"\r\n") || (width == 1 && text == "")
				if g := read[i][j]; g.Text != text || g.Quoted != wantQuoted {
					t.Fatalf("field %d of record %d of %q: read %q quoted %v, want %q quoted %v", j, i, out, g.Text, g.Quoted, text, wantQuoted)
				}
			}
		}
	})
}

// checkPositions fails when a field or a finding is placed outside data.
func checkPositions(t *testing.T, data []byte, tbl Table, fs []finding.Finding) {
	t.Helper()
	lines := bytes.Split(data, []byte("\n"))
	within := func(what string, line, col int) {
		if line < 1 || line > len(lines) || col < 1 || col > utf8.RuneCount(lines[line-1])+1 {
			t.Fatalf("%s at %d:%d is outside %q", what, line, col, data)
		}
	}
	for i, rec := range rows(tbl) {
		for _, fd := range rec {
			within(fmt.Sprintf("a field of record %d", i), fd.Line, fd.Col)
		}
	}
	for _, fd := range fs {
		l, _ := fd.Location()
		within(string(fd.Code()), l.Line, l.Column)
		for _, r := range fd.Related() {
			within(string(fd.Code())+"'s related location", r.Location.Line, r.Location.Column)
		}
	}
}
